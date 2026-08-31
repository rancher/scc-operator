#!/usr/bin/env bash
# GHA entry point: orchestrates the full rancher/rancher update workflow.
# Called from push-to-rancher.yml after token generation and rancher checkout.
#
# Required env vars (set by push-to-rancher.yml):
#   TAG          - SCC Operator tag (e.g. v0.4.2)
#   GH_TOKEN     - GitHub app token with access to rancher/rancher
#   APP_USER     - GitHub app slug for commit attribution (e.g. "my-app[bot]")
#   SOURCE_REPO  - source repo (github.repository)
#   SCC_DIR      - path to scc-operator workspace ($GITHUB_WORKSPACE)
#   RANCHER_DIR  - path where rancher/rancher was cloned (must exist before script runs)
#
# Optional env vars:
#   RANCHER_BRANCHES_OVERRIDE - space or comma-separated list of branches to process

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/common.sh"

require_var TAG
require_var GH_TOKEN
require_var APP_USER
require_var SOURCE_REPO
require_var SCC_DIR
require_var RANCHER_DIR # Expected to be cloned before this script is run

export SCC_DIR RANCHER_DIR DRY_RUN RANCHER_BRANCHES_OVERRIDE

summary "## Push to rancher/rancher"
summary "- Tag: \`$TAG\`"
summary "- Target branches: \`${RANCHER_BRANCHES[*]}\`"
summary ""

# Validate image exists before proceeding
validate_image_exists "$TAG"

# Configure git identity for commits (GHA only - fresh clone deleted at workflow end)
user_id=$(gh api "/users/$APP_USER" --jq .id)
git -C "$RANCHER_DIR" config user.name "$APP_USER"
git -C "$RANCHER_DIR" config user.email "${user_id}+${APP_USER}@users.noreply.github.com"

summary ""
summary "## Creating PRs"

export SOURCE_REPO="${SOURCE_REPO:-rancher/scc-operator}"
bash "$SCRIPT_DIR/create-prs.sh"

summary ""
summary "## Workflow Complete"
