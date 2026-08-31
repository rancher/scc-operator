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

summary ""
summary "## Processing branches"

FAILED_BRANCHES=()
CREATED_PRS=()
CREATED_PR_BRANCHES=()

for TARGET_BRANCH in "${RANCHER_BRANCHES[@]}"; do
  summary ""
  summary "### Branch: \`$TARGET_BRANCH\`"

  # Fetch and checkout target branch
  if ! git -C "$RANCHER_DIR" fetch "$RANCHER_REMOTE" "$TARGET_BRANCH" 2>&1; then
    summary "  ⚠️  Failed to fetch branch \`$TARGET_BRANCH\` - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (fetch failed)")
    continue
  fi

  if ! git -C "$RANCHER_DIR" checkout -B "$TARGET_BRANCH" "$RANCHER_REMOTE/$TARGET_BRANCH" 2>&1; then
    summary "  ⚠️  Failed to checkout branch \`$TARGET_BRANCH\` - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (checkout failed)")
    continue
  fi

  # Create feature branch with target branch name for clarity
  # Convert release/v2.12 -> v2.12, main -> main
  TARGET_SUFFIX="${TARGET_BRANCH#release/}"
  BRANCH_NAME="bot/scc-operator-${TAG}-${TARGET_SUFFIX}-$(date +%s)"
  if ! git -C "$RANCHER_DIR" checkout -b "$BRANCH_NAME" 2>&1; then
    summary "  ⚠️  Failed to create branch \`$BRANCH_NAME\` - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (branch creation failed)")
    continue
  fi

  # Update build.yaml
  export TAG RANCHER_DIR
  UPDATE_EXIT=0
  bash "$SCRIPT_DIR/update-build-yaml.sh" || UPDATE_EXIT=$?

  if [ "$UPDATE_EXIT" -eq 2 ]; then
    # No update needed - version already matches
    summary "  ℹ️  Version already up-to-date - skipping"
    git -C "$RANCHER_DIR" checkout -f "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  elif [ "$UPDATE_EXIT" -ne 0 ]; then
    # Update failed
    summary "  ⚠️  Failed to update build.yaml - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (update failed)")
    git -C "$RANCHER_DIR" checkout -f "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  # Run go generate (extract first directive from generate.go)
  GENERATE_FILE="$RANCHER_DIR/generate.go"
  if [ ! -f "$GENERATE_FILE" ]; then
    summary "  ⚠️  generate.go not found - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (no generate.go)")
    git -C "$RANCHER_DIR" checkout -f "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  # Extract first //go:generate directive
  GENERATE_CMD=$(grep -m 1 '^//go:generate' "$GENERATE_FILE" | sed 's|^//go:generate ||')
  if [ -z "$GENERATE_CMD" ]; then
    summary "  ⚠️  No go:generate directive found in generate.go - skipping"
    FAILED_BRANCHES+=("$TARGET_BRANCH (no generate directive)")
    git -C "$RANCHER_DIR" checkout -f "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  log "  - Running: \`$GENERATE_CMD\`"
  GENERATE_OUTPUT=$(cd "$RANCHER_DIR" && eval "$GENERATE_CMD" 2>&1) || GENERATE_EXIT=$?
  if [ "${GENERATE_EXIT:-0}" -ne 0 ]; then
    summary "  ⚠️  go generate failed with exit code ${GENERATE_EXIT}"
    summary "  Error output:"
    echo "$GENERATE_OUTPUT" | sed 's/^/    /'
    FAILED_BRANCHES+=("$TARGET_BRANCH (go generate failed)")
    git -C "$RANCHER_DIR" checkout -f "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  # Commit changes
  COMMIT_MSG="Update SCC Operator to ${TAG}

Automated update from ${SOURCE_REPO} release ${TAG}

Automation: push-to-rancher
Created-by: scc-operator-release-integration"

  if ! commit_if_changed "$COMMIT_MSG"; then
    exit_code=$?
    if [ "$exit_code" -eq 1 ]; then
      summary "  ℹ️  No changes detected - skipping"
    else
      summary "  ⚠️  Failed to commit changes - skipping"
      FAILED_BRANCHES+=("$TARGET_BRANCH (commit failed)")
    fi
    git -C "$RANCHER_DIR" checkout -f "$TARGET_BRANCH"
    git -C "$RANCHER_DIR" branch -D "$BRANCH_NAME" || true
    continue
  fi

  if [ "$DRY_RUN" = "true" ]; then
    summary "  ✓ Changes committed (dry-run, not pushing)"
    git -C "$RANCHER_DIR" checkout -f "$TARGET_BRANCH"
    continue
  fi

  # Push branch
  log "  - Pushing branch \`$BRANCH_NAME\`"
  if ! git -C "$RANCHER_DIR" push -u "$RANCHER_REMOTE" "$BRANCH_NAME"; then
    summary "  ⚠️  Failed to push branch - skipping PR creation"
    FAILED_BRANCHES+=("$TARGET_BRANCH (push failed)")
    git -C "$RANCHER_DIR" checkout -f "$TARGET_BRANCH"
    continue
  fi

  # Create PR
  log "  - Creating PR..."

  # Format branch name for title: strip "release/" prefix
  BRANCH_LABEL="${TARGET_BRANCH#release/}"

  PR_BODY="## Summary
Update SCC Operator image to [\`${TAG}\`](https://github.com/${SOURCE_REPO}/releases/tag/${TAG})

## Changes
- Updated \`defaultSccOperatorImage\` in \`build.yaml\`
- Ran \`go generate ./pkg/...\` to update generated files"

  PR_OUTPUT=$(gh pr create \
    --repo rancher/rancher \
    --base "$TARGET_BRANCH" \
    --head "$BRANCH_NAME" \
    --title "[${BRANCH_LABEL}] Update SCC Operator to ${TAG}" \
    --body "$PR_BODY" \
    --label "status/auto-created" 2>&1)

  if [ $? -eq 0 ]; then
    PR_URL=$(echo "$PR_OUTPUT" | tail -1)
    summary "  ✓ PR created: $PR_URL"
    CREATED_PRS+=("$PR_URL")
    CREATED_PR_BRANCHES+=("$TARGET_BRANCH")
  else
    summary "  ⚠️  Failed to create PR"
    echo "$PR_OUTPUT" | sed 's/^/    /' >&2
    FAILED_BRANCHES+=("$TARGET_BRANCH (PR creation failed)")
  fi

  # Return to target branch for next iteration
  git -C "$RANCHER_DIR" checkout -f "$TARGET_BRANCH"
done

summary ""
summary "## Pull Requests Created"
if [ ${#CREATED_PRS[@]} -gt 0 ]; then
  for i in "${!CREATED_PRS[@]}"; do
    summary "- **${CREATED_PR_BRANCHES[$i]}**: ${CREATED_PRS[$i]}"
  done
else
  summary "_No PRs were created in this run_"
fi

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
