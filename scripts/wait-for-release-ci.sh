#!/usr/bin/env bash
set -euo pipefail

# A merged version PR can start Release before its push-triggered CI is visible.
run_id=""
for attempt in {1..12}; do
  run_id=$(gh run list --repo "$GITHUB_REPOSITORY" --workflow ci.yml \
    --branch main --event push --commit "$GITHUB_SHA" \
    --json databaseId --jq '.[0].databaseId // empty')
  if [[ -n "$run_id" ]]; then
    break
  fi
  sleep 10
done
if [[ -z "$run_id" ]]; then
  echo "No push CI found for $GITHUB_SHA on main. Push this workflow change first, then start a fresh release."
  exit 1
fi

gh run watch "$run_id" --repo "$GITHUB_REPOSITORY" --interval 10 --exit-status
# Explicitly require success: never reuse failed, cancelled or skipped runs.
conclusion=$(gh run view "$run_id" --repo "$GITHUB_REPOSITORY" --json conclusion --jq .conclusion)
if [[ "$conclusion" != "success" ]]; then
  echo "CI run $run_id concluded with $conclusion. Rerun CI successfully before retrying the release."
  exit 1
fi
echo "run-id=$run_id" >> "$GITHUB_OUTPUT"
