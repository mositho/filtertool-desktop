#!/usr/bin/env bash
set -euo pipefail

# Fresh publishing runner only: never install dependencies or execute downloaded data.
# This branch is automation-owned; review changes in the PR rather than editing it.
branch=chore/update-basetypes
message="chore: update PoE base data"
paths=(internal/data/BaseTypes.csv internal/data/reference.json internal/data/weapons.json)
bot_email='41898282+github-actions[bot]@users.noreply.github.com'

[[ "$GITHUB_REF" == refs/heads/main ]]
[[ "$(git rev-parse HEAD)" == "$GITHUB_SHA" ]]
export GIT_CONFIG_COUNT=1
export GIT_CONFIG_KEY_0=http.https://github.com/.extraheader
export GIT_CONFIG_VALUE_0="AUTHORIZATION: basic $(printf 'x-access-token:%s' "$GH_TOKEN" | base64 | tr -d '\r\n')"

# Don't publish data prepared against an obsolete main if it moved during the run.
main=$(git ls-remote origin refs/heads/main)
if [[ "${main%%$'\t'*}" != "$GITHUB_SHA" ]]; then
  echo "main advanced during preparation; rerun Update Base Types." >&2
  exit 1
fi
remote=$(git ls-remote origin "refs/heads/$branch")
expected="${remote%%$'\t'*}"
if [[ -n "$expected" ]]; then
  git fetch --no-tags origin "refs/heads/$branch"
  [[ "$(git rev-parse FETCH_HEAD)" == "$expected" ]]
  # Refuse to overwrite human edits or unrelated files on the reserved branch.
  while IFS= read -r email; do
    if [[ "$email" != "$bot_email" ]]; then
      echo "Proposal contains non-bot commits; reconcile it manually." >&2
      exit 1
    fi
  done < <(git log --format=%ae "HEAD..$expected")
  while IFS= read -r path; do
    case "$path" in
      internal/data/BaseTypes.csv|internal/data/reference.json|internal/data/weapons.json) ;;
      *) echo "Unexpected proposal file: $path" >&2; exit 1 ;;
    esac
  done < <(git diff --name-only "HEAD...$expected")
fi

for path in "${paths[@]}"; do
  [[ -f "prepared/$path" && ! -L "prepared/$path" ]]
  cp "prepared/$path" "$path"
done
git add -- "${paths[@]}"
if git diff --cached --quiet; then
  echo "No changes to propose." >> "$GITHUB_STEP_SUMMARY"
  exit 0
fi

# Keep the existing commit if its entire tree already matches the proposal.
if [[ -n "$expected" ]] && git diff --cached --quiet "$expected"; then
  echo "Proposal is already current." >> "$GITHUB_STEP_SUMMARY"
else
  git config user.name 'github-actions[bot]'
  git config user.email "$bot_email"
  git -c core.hooksPath=/dev/null commit -m "$message"
  # An explicit lease also refuses branch creation if another actor created it.
  git push "--force-with-lease=refs/heads/$branch:$expected" origin "HEAD:refs/heads/$branch"
fi

number=$(gh pr list --repo "$GITHUB_REPOSITORY" --head "$branch" --base main --state open --json number --jq '.[0].number // empty')
if [[ -z "$number" ]]; then
  gh pr create --repo "$GITHUB_REPOSITORY" --base main --head "$branch" \
    --title "$message" \
    --body "Updates PoE 1 base data from FilterBlade and regenerates desktop references. This automation-owned branch is refreshed on subsequent runs. Review data and rule behavior, then merge manually only after Linux and Windows CI pass."
fi

# GITHUB_TOKEN-created PRs/pushes don't trigger ordinary PR/push workflows.
# Dispatch full CI explicitly on the proposal branch, including on retries.
gh workflow run ci.yml --repo "$GITHUB_REPOSITORY" --ref "$branch"
echo "Review [the base-data PR](https://github.com/$GITHUB_REPOSITORY/pulls?q=is%3Apr+is%3Aopen+head%3A$branch). Full CI dispatched; merge manually only after it passes." >> "$GITHUB_STEP_SUMMARY"
