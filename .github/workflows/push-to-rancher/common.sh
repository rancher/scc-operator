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

# Target branches in rancher/rancher to update (default)
# Ordered newest-first: main, then descending release versions
# This ensures branches with the latest fixes are processed first
DEFAULT_RANCHER_BRANCHES=(
  "main"
  "release/v2.15"
  "release/v2.14"
  "release/v2.13"
  "release/v2.12"
)

# Allow override via RANCHER_BRANCHES_OVERRIDE (comma or space separated)
if [ -n "${RANCHER_BRANCHES_OVERRIDE:-}" ]; then
  # Convert to array, handling both comma and space separators
  IFS=',' read -ra RANCHER_BRANCHES <<< "${RANCHER_BRANCHES_OVERRIDE// /,}"
else
  RANCHER_BRANCHES=("${DEFAULT_RANCHER_BRANCHES[@]}")
fi
export RANCHER_BRANCHES

# Docker registry to validate image existence
IMAGE_REGISTRY="${IMAGE_REGISTRY:-docker.io}"
IMAGE_REPO="${IMAGE_REPO:-rancher/scc-operator}"

# Prime registry URLs (allow anonymous read access for validation)
PRIME_STG_REGISTRY="${PRIME_STG_REGISTRY:-stgregistry.suse.com}"
PRIME_REGISTRY="${PRIME_REGISTRY:-registry.suse.com}"

# Write to GitHub step summary if available, and always print to stdout
summary() {
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    echo "$@" >> "$GITHUB_STEP_SUMMARY"
  fi
  echo "$@"
}

# Write only to stdout (detailed logs, not in GH summary)
log() {
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

# Detect if tag is a stable release (no prerelease suffix)
# Returns 0 if stable, 1 if prerelease
is_stable_release() {
  local tag="$1"
  # Strip leading 'v' if present
  tag="${tag#v}"
  # Check if tag matches semver with prerelease (has dash after version numbers)
  if echo "$tag" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+-'; then
    return 1  # Has prerelease suffix
  else
    return 0  # Stable release
  fi
}

# Validate that the SCC Operator image exists in required registries
# All registries checked support anonymous read access via docker manifest inspect
validate_image_exists() {
  local tag="$1"
  local validation_failed=0

  summary ""
  summary "## Image Validation"

  # Determine which registries to check based on release type
  local registries=()
  local registry_labels=()

  # Always validate Docker Hub
  registries+=("${IMAGE_REGISTRY}")
  registry_labels+=("Docker Hub")

  # Always validate Prime Staging
  registries+=("${PRIME_STG_REGISTRY}")
  registry_labels+=("Prime Staging")

  # Validate Prime Production only for stable releases
  if is_stable_release "$tag"; then
    registries+=("${PRIME_REGISTRY}")
    registry_labels+=("Prime Production")
    log "ℹ️  Stable release detected - will validate Prime Production registry"
  else
    log "ℹ️  Prerelease detected - skipping Prime Production validation"
  fi

  # Validate each registry (no authentication needed - anonymous read access)
  for i in "${!registries[@]}"; do
    local registry="${registries[$i]}"
    local label="${registry_labels[$i]}"
    local full_image="${registry}/${IMAGE_REPO}:${tag}"

    summary "- **${label}**: \`${full_image}\`"

    if docker manifest inspect "$full_image" >/dev/null 2>&1; then
      summary "  ✓ Image found"
    else
      summary "  ✗ Image NOT found"
      validation_failed=1
    fi
  done

  if [ $validation_failed -eq 1 ]; then
    echo "" >&2
    echo "ERROR: Image validation failed for one or more registries" >&2
    echo "ERROR: Cannot proceed with PR creation until images are published" >&2
    exit 1
  fi

  summary ""
  summary "✅ All registry validations passed"
}

# Commit all changes in RANCHER_DIR if any exist. Returns 1 if no changes, 0 on success.
commit_if_changed() {
  local message="$1"
  if git -C "$RANCHER_DIR" diff --quiet --exit-code && [ -z "$(git -C "$RANCHER_DIR" status --porcelain)" ]; then
    return 1
  fi
  git -C "$RANCHER_DIR" add .
  git -C "$RANCHER_DIR" commit -m "$message"
}
