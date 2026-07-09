#!/usr/bin/env bash
# Updates defaultSccOperatorImage in build.yaml to the specified tag.
#
# Required env vars:
#   TAG          - SCC Operator tag (e.g. v0.4.2)
#   RANCHER_DIR  - Path to rancher/rancher clone

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

# Update the defaultSccOperatorImage field
# Use yq for safe YAML editing
yq eval ".defaultSccOperatorImage = \"rancher/scc-operator:v${TAG_NO_V}\"" -i "$BUILD_YAML"

summary "  - Updated build.yaml: \`defaultSccOperatorImage: rancher/scc-operator:v${TAG_NO_V}\`"
