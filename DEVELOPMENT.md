# Development

Requires Go (see `go.mod`) and Docker for the Makefile-driven workflow (CI runs inside `ghcr.io/rancher/ci-image` unless `CI=true` bypasses it).

```sh
make build        # build the binary
make test         # run tests
make lint         # golangci-lint
make ci           # full CI: build, test, validate

make generate     # regenerate clients & CRDs (go generate ./...)
make validate     # check for generated-code drift

make build-image  # build the container image for the current platform
make push-image   # build multi-arch and push (REPO/TAG overridable)
```

To run the operator locally against a development cluster:

```sh
go run ./cmd/operator \
  --kubeconfig "$KUBECONFIG" \
  --operator-namespace cattle-scc-system
```

Set `--debug`/`--trace` for verbose logging, `--log-format`/`--log-level` to tune output.

## Repository layout

| Path | Purpose |
|---|---|
| `cmd/operator/` | Operator binary and version subcommand |
| `pkg/apis/scc.cattle.io/` | CRD API types (`Registration`) |
| `pkg/controllers/` | Reconciliation: online/offline modes, dispatcher, lifecycle |
| `pkg/operator/` | Bootstrap, setup, readiness |
| `pkg/crds/` | CRD install manager + generated YAML |
| `pkg/generated/` | Generated clientset, controllers, listers |
| `internal/` | Private libraries: `suseconnect`, `telemetry`, `config`, `consts`, `repos`, … |
| `scripts/`, `hack/`, `Makefile` | Build, test, validation, packaging |
| `.github/workflows/` | CI, lint, release, head builds, rancher/rancher image bumps |
| `updatecli/`, `gotools/` | Dependency automation and pinned tool modules |
