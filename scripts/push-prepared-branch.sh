#!/usr/bin/env bash
set -euo pipefail

# Run only on a fresh write-enabled runner, without npm/Go dependencies installed.
case "${1:-}" in
  release)
    [[ "${RELEASE_VERSION:-}" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || exit 1
    branch="release/v$RELEASE_VERSION"
    message="chore(release): v$RELEASE_VERSION"
    paths=(package.json package-lock.json build/config.yml)
    ;;
  basetypes)
    branch=chore/update-basetypes
    message="chore: update PoE base data"
    paths=(internal/data/BaseTypes.csv internal/data/reference.json internal/data/weapons.json)
    ;;
  *) echo "Expected release or basetypes" >&2; exit 1 ;;
esac

export GIT_CONFIG_COUNT=1
export GIT_CONFIG_KEY_0=http.https://github.com/.extraheader
export GIT_CONFIG_VALUE_0="AUTHORIZATION: basic $(printf 'x-access-token:%s' "$GH_TOKEN" | base64 | tr -d '\r\n')"
remote=$(git ls-remote origin "refs/heads/$branch")
if [[ -z "$remote" ]]; then
  for path in "${paths[@]}"; do
    [[ -f "prepared/$path" && ! -L "prepared/$path" ]] || exit 1
    cp "prepared/$path" "$path"
  done
  git add -- "${paths[@]}"
  if git diff --cached --quiet; then
    echo "No changes to propose." >> "$GITHUB_STEP_SUMMARY"
    exit 0
  fi
  git config user.name 'github-actions[bot]'
  git config user.email '41898282+github-actions[bot]@users.noreply.github.com'
  git -c core.hooksPath=/dev/null commit -m "$message"
  git push origin "HEAD:refs/heads/$branch"
else
  echo "Existing branch left untouched; review it before opening or updating its PR." >> "$GITHUB_STEP_SUMMARY"
fi
echo "Review changes and [open a PR](https://github.com/$GITHUB_REPOSITORY/compare/main...$branch?expand=1). Merge only after CI passes." >> "$GITHUB_STEP_SUMMARY"
