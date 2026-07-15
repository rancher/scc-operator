#!/usr/bin/env bash
# Updates defaultSccOperatorImage in build.yaml to the specified tag.
#
# This script is called by the rancher/ecm-distro-tools create-pr action to
# update the SCC Operator image version in rancher/rancher's build.yaml file
# and regenerate any dependent code.
#
# Required env vars:
#   TAG          - SCC Operator tag (e.g. v0.4.2)
#   RANCHER_DIR  - Path to rancher/rancher clone
#
# Optional env vars (provided by PRBUILDER):
#   PRBUILDER_TAG          - The release tag (e.g., "v0.4.2")
#   PRBUILDER_VERSION      - Parsed version based on version_mapping_type
#   PRBUILDER_TARGET_DIR   - Absolute path to the cloned target repository
#   PRBUILDER_TARGET_REPO  - Target repository in "owner/repo" format
#   PRBUILDER_TARGET_BRANCH - Target branch being updated (e.g., "dev-v2.14")
#   PRBUILDER_SOURCE_DIR   - Absolute path to the source repository

set -euo pipefail

# =============================================================================
# ENVIRONMENT SETUP
# =============================================================================

# Use PRBUILDER_* variables if available, fall back to legacy names
TAG=${TAG:-${PRBUILDER_TAG}}
RANCHER_DIR=${RANCHER_DIR:-${PRBUILDER_TARGET_DIR}}
TARGET_BRANCH=${PRBUILDER_TARGET_BRANCH:-unknown}  # For error message context only

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

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

require_binary() {
  local bin="$1"
  if ! command -v "$bin" &> /dev/null; then
    echo "ERROR: Required binary '$bin' not found in PATH" >&2
    exit 1
  fi
}

# =============================================================================
# PREFLIGHT CHECKS
# =============================================================================

# Check required binaries
require_binary yq
require_binary grep
require_binary sed

# Check required environment variables
require_var TAG

require_var RANCHER_DIR
if [ ! -d "$RANCHER_DIR" ]; then
  echo "ERROR: RANCHER_DIR '$RANCHER_DIR' does not exist" >&2
  exit 1
fi

# Check that build.yaml exists
BUILD_YAML="$RANCHER_DIR/build.yaml"
if [ ! -f "$BUILD_YAML" ]; then
  echo "ERROR: build.yaml not found at $BUILD_YAML" >&2
  exit 1
fi

# =============================================================================
# UPDATE BUILD.YAML
# =============================================================================

# Remove leading 'v' if present for consistency
TAG_NO_V="${TAG#v}"

# Update the defaultSccOperatorImage field using yq for safe YAML editing
yq eval ".defaultSccOperatorImage = \"rancher/scc-operator:v${TAG_NO_V}\"" -i "$BUILD_YAML"

summary "  - Updated build.yaml: \`defaultSccOperatorImage: rancher/scc-operator:v${TAG_NO_V}\`"

# =============================================================================
# RUN GO GENERATE
# =============================================================================

# Check that generate.go exists
GENERATE_FILE="$RANCHER_DIR/generate.go"
if [ ! -f "$GENERATE_FILE" ]; then
  summary "  ⚠️  generate.go not found in branch '$TARGET_BRANCH' - cannot proceed"
  exit 1
fi

# Extract first //go:generate directive
GENERATE_CMD=$(grep -m 1 '^//go:generate' "$GENERATE_FILE" | sed 's|^//go:generate ||')
if [ -z "$GENERATE_CMD" ]; then
  summary "  ⚠️  No go:generate directive found in generate.go on branch '$TARGET_BRANCH' - cannot proceed"
  exit 1
fi

# Run the generate command
summary "  - Running: \`$GENERATE_CMD\`"
if ! (cd "$RANCHER_DIR" && eval "$GENERATE_CMD"); then
  summary "  ⚠️  go generate failed on branch '$TARGET_BRANCH'"
  exit 1
fi

summary "  ✓ Successfully updated SCC Operator to v${TAG_NO_V}"