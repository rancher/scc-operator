#!/usr/bin/env bash
# Shared setup for push-to-rancher scripts. Source this file: source "$(dirname "$0")/common.sh"

# Determine SCC_DIR (scc-operator root) from this script's location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCC_DIR="${SCC_DIR:-$(cd "$SCRIPT_DIR/../../.." && pwd)}"

# Required: path to a local rancher/rancher clone
RANCHER_DIR="${RANCHER_DIR:-}"

# Remote name for rancher/rancher in RANCHER_DIR (may differ locally if using a fork)
RANCHER_REMOTE="${RANCHER_REMOTE:-origin}"

# Skip git commits, push, and PR creation when true
DRY_RUN="${DRY_RUN:-false}"

# Target branches in rancher/rancher to update
RANCHER_BRANCHES=(
  "release/v2.12"
  "release/v2.13"
  "release/v2.14"
#  "release/v2.15"
  "main"
)

# Docker registry to validate image existence
IMAGE_REGISTRY="${IMAGE_REGISTRY:-docker.io}"
IMAGE_REPO="${IMAGE_REPO:-rancher/scc-operator}"

# Write to GitHub step summary if available, and always print to stdout
summary() {
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    echo "$@" >> "$GITHUB_STEP_SUMMARY"
  fi
  echo "$@"
}

require_var() {
  local var="$1"
  if [ -z "${!var:-}" ]; then
    echo "ERROR: $var is required" >&2
    exit 1
  fi
}

require_rancher_dir() {
  require_var RANCHER_DIR
  if [ ! -d "$RANCHER_DIR" ]; then
    echo "ERROR: RANCHER_DIR '$RANCHER_DIR' does not exist" >&2
    exit 1
  fi
}

# Validate that the SCC Operator image exists in the registry
validate_image_exists() {
  local tag="$1"
  local full_image="${IMAGE_REGISTRY}/${IMAGE_REPO}:${tag}"

  summary "- Validating image exists: \`$full_image\`"

  if ! docker manifest inspect "$full_image" >/dev/null 2>&1; then
    echo "ERROR: Image $full_image does not exist in registry" >&2
    echo "ERROR: Cannot proceed with PR creation until image is published" >&2
    exit 1
  fi

  summary "  ✓ Image validated"
}

# Commit all changes in RANCHER_DIR if any exist. Does nothing if tree is clean.
commit_if_changed() {
  local message="$1"
  if git -C "$RANCHER_DIR" diff --quiet --exit-code && [ -z "$(git -C "$RANCHER_DIR" status --porcelain)" ]; then
    return 0
  fi
  git -C "$RANCHER_DIR" add .
  git -C "$RANCHER_DIR" commit -m "$message"
}
