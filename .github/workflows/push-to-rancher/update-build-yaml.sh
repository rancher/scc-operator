#!/usr/bin/env bash
# Updates defaultSccOperatorImage in build.yaml to the specified tag.
#
# Required env vars:
#   TAG          - SCC Operator tag (e.g. v0.4.2)
#   RANCHER_DIR  - Path to rancher/rancher clone
#
# Exit codes:
#   0 - Update successful
#   1 - Error occurred
#   2 - No update needed (version already matches)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/common.sh"

require_var TAG
require_rancher_dir

BUILD_YAML="$RANCHER_DIR/build.yaml"

if [ ! -f "$BUILD_YAML" ]; then
  echo "ERROR: build.yaml not found at $BUILD_YAML" >&2
  exit 1
fi

# Remove leading 'v' if present for consistency
TAG_NO_V="${TAG#v}"
TARGET_IMAGE="rancher/scc-operator:v${TAG_NO_V}"

# Read current value from build.yaml
CURRENT_IMAGE=$(yq eval '.defaultSccOperatorImage' "$BUILD_YAML")

# Check if version already matches
if [ "$CURRENT_IMAGE" = "$TARGET_IMAGE" ]; then
  log "  ℹ️  build.yaml already has \`${TARGET_IMAGE}\` - no update needed"
  exit 2
fi

# Update the defaultSccOperatorImage field
# Use yq for safe YAML editing
yq eval ".defaultSccOperatorImage = \"${TARGET_IMAGE}\"" -i "$BUILD_YAML"

log "  - Updated build.yaml: \`defaultSccOperatorImage: ${TARGET_IMAGE}\`"
