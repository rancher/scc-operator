#!/usr/bin/env bash
# Creates PRs to rancher/rancher branches with updated SCC Operator image.
#
# Required env vars:
#   TAG          - SCC Operator tag (e.g. v0.4.2)
#   RANCHER_DIR  - Path to rancher/rancher clone
#   GH_TOKEN     - GitHub token for PR creation
#   SOURCE_REPO  - Source repo for PR body (e.g. rancher/scc-operator)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/common.sh"

require_var TAG
require_var GH_TOKEN
require_rancher_dir

SOURCE_REPO="${SOURCE_REPO:-rancher/scc-operator}"

# Configure git in rancher clone
git -C "$RANCHER_DIR" config user.name "github-actions[bot]"
git -C "$RANCHER_DIR" config user.email "github-actions[bot]@users.noreply.github.com"

summary ""
summary "## Processing branches"

FAILED_BRANCHES=()

for TARGET_BRANCH in "${RANCHER_BRANCHES[@]}"; do
  summary ""
  summary "### Branch: \`$TARGET_BRANCH\`"

  # Fetch and checkout target branch
  if ! git -C "$RANCHER_DIR" fetch "$RANCHER_REMOTE" "$TARGET_BRANCH" 2>&1; then
    summary "  ⚠️  Failed to fetch branch \`$TARGET_BRANCH\` - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (fetch failed)")
    continue
  fi

  git -C "$RANCHER_DIR" checkout -B "$TARGET_BRANCH" "$RANCHER_REMOTE/$TARGET_BRANCH"

  # Create feature branch
  BRANCH_NAME="bot/scc-operator-${TAG}-$(date +%s)"
  git -C "$RANCHER_DIR" checkout -b "$BRANCH_NAME"

  # Update build.yaml
  export TAG RANCHER_DIR
  if ! bash "$SCRIPT_DIR/update-build-yaml.sh"; then
    summary "  ⚠️  Failed to update build.yaml - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (update failed)")
    git -C "$RANCHER_DIR" checkout "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  # Run go generate (extract first directive from generate.go)
  GENERATE_FILE="$RANCHER_DIR/generate.go"
  if [ ! -f "$GENERATE_FILE" ]; then
    summary "  ⚠️  generate.go not found - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (no generate.go)")
    git -C "$RANCHER_DIR" checkout "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  # Extract first //go:generate directive
  GENERATE_CMD=$(grep -m 1 '^//go:generate' "$GENERATE_FILE" | sed 's|^//go:generate ||')
  if [ -z "$GENERATE_CMD" ]; then
    summary "  ⚠️  No go:generate directive found in generate.go - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (no generate directive)")
    git -C "$RANCHER_DIR" checkout "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  summary "  - Running: \`$GENERATE_CMD\`"
  if ! (cd "$RANCHER_DIR" && eval "$GENERATE_CMD"); then
    summary "  ⚠️  go generate failed - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (go generate failed)")
    git -C "$RANCHER_DIR" checkout "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  # Commit changes
  COMMIT_MSG="Update SCC Operator to ${TAG}

Automated update from ${SOURCE_REPO} release ${TAG}"

  if ! commit_if_changed "$COMMIT_MSG"; then
    summary "  ℹ️  No changes detected - skipping"
    git -C "$RANCHER_DIR" checkout "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  if [ "$DRY_RUN" = "true" ]; then
    summary "  ✓ Changes committed (dry-run, not pushing)"
    git -C "$RANCHER_DIR" checkout "$TARGET_BRANCH"
    continue
  fi

  # Push branch
  summary "  - Pushing branch \`$BRANCH_NAME\`"
  if ! git -C "$RANCHER_DIR" push -u "$RANCHER_REMOTE" "$BRANCH_NAME"; then
    summary "  ⚠️  Failed to push branch - skipping PR creation"
    FAILED_BRANCHES+=("$TARGET_BRANCH (push failed)")
    git -C "$RANCHER_DIR" checkout "$TARGET_BRANCH"
    continue
  fi

  # Create PR
  summary "  - Creating PR..."

  # Format branch name for title: strip "release/" prefix
  BRANCH_LABEL="${TARGET_BRANCH#release/}"

  PR_BODY="## Summary
Update SCC Operator image to [\`${TAG}\`](https://github.com/${SOURCE_REPO}/releases/tag/${TAG})

## Changes
- Updated \`defaultSccOperatorImage\` in \`build.yaml\`
- Ran \`go generate ./pkg/...\` to update generated files"

  if gh pr create \
    --repo rancher/rancher \
    --base "$TARGET_BRANCH" \
    --head "$BRANCH_NAME" \
    --title "[${BRANCH_LABEL}] Update SCC Operator to ${TAG}" \
    --body "$PR_BODY" \
    --label "status/auto-created" 2>&1; then
    summary "  ✓ PR created successfully"
  else
    summary "  ⚠️  Failed to create PR"
    FAILED_BRANCHES+=("$TARGET_BRANCH (PR creation failed)")
  fi

  # Return to target branch for next iteration
  git -C "$RANCHER_DIR" checkout "$TARGET_BRANCH"
done

summary ""
summary "## Summary"

if [ ${#FAILED_BRANCHES[@]} -eq 0 ]; then
  summary "✅ All branches processed successfully"
else
  summary "⚠️  Some branches failed:"
  for branch in "${FAILED_BRANCHES[@]}"; do
    summary "  - $branch"
  done
  exit 1
fi
