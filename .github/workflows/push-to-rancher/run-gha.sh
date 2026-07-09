#!/usr/bin/env bash
# GHA entry point: orchestrates the full rancher/rancher update workflow.
# Called from push-to-rancher.yaml after token generation.
#
# Required env vars (set by push-to-rancher.yaml or release.yml):
#   TAG          - SCC Operator tag (e.g. v0.4.2)
#   GH_TOKEN     - GitHub app token with access to rancher/rancher
#   SOURCE_REPO  - source repo (github.repository)
#   SCC_DIR      - path to scc-operator workspace ($GITHUB_WORKSPACE)
#   RANCHER_DIR  - path to clone rancher/rancher into

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/common.sh"

require_var TAG
require_var GH_TOKEN

export SCC_DIR RANCHER_DIR DRY_RUN

summary "## Push to rancher/rancher"
summary "- Tag: \`$TAG\`"
summary "- Target branches: \`${RANCHER_BRANCHES[*]}\`"
summary ""

# Validate image exists before proceeding
validate_image_exists "$TAG"

# Clone rancher/rancher
summary "- Cloning rancher/rancher..."
git clone "https://oauth2:${GH_TOKEN}@github.com/rancher/rancher.git" "$RANCHER_DIR"

summary ""
summary "## Creating PRs"

export SOURCE_REPO="${SOURCE_REPO:-rancher/scc-operator}"
bash "$SCRIPT_DIR/create-prs.sh"

summary ""
summary "## Workflow Complete"
