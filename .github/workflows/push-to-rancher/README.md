# push-to-rancher

Automates opening PRs against [rancher/rancher](https://github.com/rancher/rancher) when a new SCC Operator release is published.

## Workflows

### `.github/workflows/push-to-rancher.yml`

Standalone workflow that can be:
- **Manually triggered** via `workflow_dispatch` with any tag (including RCs)
- **Called** from other workflows (e.g., `release.yml` for stable releases)

### `.github/workflows/release.yml`

Automatically calls `push-to-rancher.yml` for stable releases only (after images are published and release is un-drafted).

## How it works

For each target Rancher branch:
1. Clone rancher/rancher
2. Update `defaultSccOperatorImage` in `build.yaml`
3. Run `go generate ./pkg/...` to update generated files
4. Commit changes
5. Push branch and create PR

## Target branches

Edit `common.sh` and update `RANCHER_BRANCHES` array:

```bash
RANCHER_BRANCHES=(
  "release/v2.12"
  "release/v2.13"
  "release/v2.14"
  "release/v2.15"
  "main"
)
```

Add new Rancher versions as they're released. Remove EOL versions.

## Local usage

```bash
./.github/workflows/push-to-rancher/run-local.sh \
  --tag v0.4.2 \
  --rancher-dir /path/to/rancher \
  [--dry-run] \
  [--remote upstream]
```

`--dry-run` runs all local git work (commits to your rancher clone) but skips push and PR creation.

## Step sequence

| Script | What it does |
|---|---|
| `update-build-yaml.sh` | Updates `defaultSccOperatorImage` in `build.yaml` using `yq` |
| `create-prs.sh` | For each target branch: checkout, update, `go generate`, commit, push, create PR |
| `run-gha.sh` | GHA entry point: validates image exists, clones rancher/rancher, calls create-prs.sh |
| `run-local.sh` | Local entry point: parses args, sets up env, calls create-prs.sh |

## Key env vars

| Var | Description |
|---|---|
| `TAG` | SCC Operator tag (e.g. `v0.4.2`) |
| `RANCHER_DIR` | Path to local rancher/rancher clone |
| `RANCHER_REMOTE` | Remote name in `RANCHER_DIR` (default: `origin`) |
| `DRY_RUN` | Set to `true` to skip push and PR creation |
| `SOURCE_REPO` | Source repo for PR body (default: `rancher/scc-operator`) |

## GHA prerequisites

The workflow reads a GitHub App credential from Vault at:

```
secret/data/github/repo/rancher/scc-operator/github/app-credentials
```

The app must have write access to `rancher/rancher` to push branches and open PRs.

## Error handling

The workflow continues processing branches even if one fails. Failed branches are logged at the end of the run. Common failure modes:

- Branch doesn't exist (e.g., when a new Rancher version isn't released yet)
- `go generate` fails (rare, usually indicates build.yaml format change)
- PR already exists for this tag on this branch
