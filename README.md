# scc-operator

`scc-operator` is a Kubernetes operator that manages **SUSE Customer Center (SCC) integration for Rancher Prime**. It handles registration and activation of a cluster with SCC — online via a registration code, or offline (air-gapped) via certificate exchange — keeps the registration alive, manages SCC credentials as Kubernetes Secrets, and reports telemetry.

It is built on the standard Rancher controller stack ([wrangler](https://github.com/rancher/wrangler)/[lasso](https://github.com/rancher/lasso)) and integrates with [SUSE/connect-ng](https://github.com/SUSE/connect-ng) to talk to SCC.

## How it fits into Rancher

`scc-operator` ships as the SCC registration component of [Rancher Prime](https://www.rancher.com/). It is **not** installed by the user directly: Rancher core deploys and manages the operator at runtime.

- Rancher core contains a deployer (`pkg/scc/deployer` in [rancher/rancher](https://github.com/rancher/rancher)) that installs the operator into the `cattle-scc-system` namespace on the local cluster, wiring in the Rancher version, private registry/pull secrets, and the operator image (configurable via the `scc-operator-image` setting).
- The operator waits for Rancher prerequisites — the `server-url` and `install-uuid` settings — before becoming ready, and resolves cluster identity from Rancher state.
- Once running, it watches the cluster-scoped `scc.cattle.io` `Registration` resource (see below). The Rancher UI writes the registration input (regcode, offline certificate, RMT endpoint) to the `scc-registration` Secret and the `Registration` CR; the operator reconciles it against SCC and surfaces status back through the same resource, which the registration page in the Rancher dashboard displays.
- It also produces SCC telemetry for Rancher (`rancher-scc-metrics`), gated by Rancher's telemetry settings.

In short: **Rancher deploys it, feeds it a `Registration` intent, and the operator owns the entire lifecycle with SCC** — announce, activate, keepalive, sync, deactivate, and credential management.

## What it does

- **Online registration**: activates the system in SCC using a regcode (optionally via an [RMT](https://documentation.suse.com/sles/15-SP6/html/SLES-all/cha-rmt.html) server with a custom registration URL and CA), stores the returned system credentials in a Secret, and runs periodic jittered keepalives.
- **Offline (air-gapped) registration**: generates an offline registration request for the user to exchange with SCC, then validates the returned offline certificate locally and tracks its expiry.
- **Credential management**: SCC system credentials (`scc-system-credentials-*`) and registration codes (`registration-code-*`) are persisted as Secrets in the `cattle-scc-system` namespace; secrets are rotated and cleaned up as part of the lifecycle.
- **Subscription reporting**: records subscription info (name, limits, expiry, product classes) and the SCC system ID on the `Registration` status.
- **Version awareness**: re-syncs with SCC when the product version changes (e.g. after a Rancher upgrade).

## The `Registration` resource

The operator's public API is the cluster-scoped `Registration` CRD (`scc.cattle.io/v1`):

```yaml
apiVersion: scc.cattle.io/v1
kind: Registration
metadata:
  name: scc-registration
spec:
  mode: online # or offline
  registrationRequest:
    registrationCodeSecretRef:
      name: registration-code
    # optional: register through RMT instead of SCC directly
    registrationAPIUrl: https://rmt.example.com
    registrationAPICertificateSecretRef:
      name: rmt-ca
  # optional: trigger a re-sync
  syncNow: false
```

Status reported by the operator includes the SCC system ID, activation state, registration expiry, subscription info, and references to the system credentials and offline request Secrets. Progress is tracked via conditions (`RegistrationAnnounced`, `RegistrationActivated`, `RegistrationKeepalive`, `OfflineRequestReady`, `OfflineCertificateReady`, …) — check `kubectl get registration -o yaml` for the current state.

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for build, test, and contribution notes, and for the repository layout.
