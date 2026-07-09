#!/usr/bin/env bash
# Local entry point for testing push-to-rancher workflow.
#
# Usage:
#   ./run-local.sh --tag v0.4.2 --rancher-dir /path/to/rancher [--dry-run] [--remote upstream]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Defaults
TAG=""
RANCHER_DIR=""
DRY_RUN="false"
RANCHER_REMOTE="origin"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --tag)
      TAG="$2"
      shift 2
      ;;
    --rancher-dir)
      RANCHER_DIR="$2"
      shift 2
      ;;
    --dry-run)
      DRY_RUN="true"
      shift
      ;;
    --remote)
      RANCHER_REMOTE="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1" >&2
      echo "Usage: $0 --tag TAG --rancher-dir DIR [--dry-run] [--remote REMOTE]" >&2
      exit 1
      ;;
  esac
done

# Validate required args
if [ -z "$TAG" ]; then
  echo "ERROR: --tag is required" >&2
  exit 1
fi

if [ -z "$RANCHER_DIR" ]; then
  echo "ERROR: --rancher-dir is required" >&2
  exit 1
fi

if [ ! -d "$RANCHER_DIR" ]; then
  echo "ERROR: rancher-dir '$RANCHER_DIR' does not exist" >&2
  exit 1
fi

# Set up environment
export TAG RANCHER_DIR RANCHER_REMOTE DRY_RUN
export GH_TOKEN="${GH_TOKEN:-$(gh auth token)}"

# Source common for validation functions
source "$SCRIPT_DIR/common.sh"

echo "## Push to rancher/rancher (local)"
echo "- Tag: $TAG"
echo "- Rancher dir: $RANCHER_DIR"
echo "- Remote: $RANCHER_REMOTE"
echo "- Dry run: $DRY_RUN"
echo ""

# Validate image exists
validate_image_exists "$TAG"

echo ""
echo "## Creating PRs"

export SOURCE_REPO="${SOURCE_REPO:-rancher/scc-operator}"
bash "$SCRIPT_DIR/create-prs.sh"

echo ""
echo "## Workflow Complete"

if [ "$DRY_RUN" = "true" ]; then
  echo ""
  echo "NOTE: This was a dry run. Changes were committed locally but not pushed."
  echo "Review the commits in $RANCHER_DIR and run without --dry-run to push and create PRs."
fi
