# Release checklist

Release versions use stable `major.minor.patch` numbers, starting with **1.0.0**. Known bugs can be documented and fixed in patch releases; stable naming does not mean bug-free.

## Before starting

- Merge the intended changes into `main`. There is no need to bump version files or create a tag manually.
- Run `npm ci`, `npm run check`, `npm run frontend:build`, `go vet ./...`, and `npm audit --audit-level=high`. Require successful Windows and Linux CI.
- Test on Windows with temporary/test profiles: standalone launch, per-user install/uninstall, WebView2 setup, native file/folder dialogs, import, save/restart, preview, export, and loading the filter in PoE 1. Confirm uninstall preserves user data.
- Smoke-test in-app updating from an older build in a writable temporary folder: download, restart, and confirm the new version and preserved test profile. Check that unsaved edits prevent restart. A major-version increase must show the migration/recreation warning and require confirmation.
- Test without FFmpeg: startup, editing and preview work; TTS generation explains the missing dependency. Then install FFmpeg on `PATH`, restart, and test TTS generation/export. Confirm no FFmpeg executable or DLL is shipped.
- Review third-party redistribution permissions and required license/copyright texts for all shipped code and assets, including Go/frontend dependencies, PoE data and game alert audio. A source link or attribution alone is not necessarily sufficient permission or a complete license notice. Verify the Google Translate TTS usage terms separately; availability is not guaranteed.

## Run the workflow

- After release approval, open **Actions → Release → Run workflow**, select **main**, and enter a version such as `1.0.0` (without the `v`). Only stable `major.minor.patch` versions are accepted. Existing tags and versions below the current package version are rejected.
- If the version differs from `main`, the workflow prepares `package.json`, both root versions in `package-lock.json`, and `build/config.yml` on a `release/v…` branch. Open the PR using the link in the job summary, review it, wait for CI, and merge it. CI verifies version-only PRs without repeating native builds; changes to anything besides the version fields require full CI. The merged commit always receives full Windows/Linux tests and builds. The merge automatically resumes the release workflow; **do not run Release again**. The workflow never pushes directly to `main` and does not create or approve PRs. Alternatively, maintainers can update the version files directly on `main` before running Release.
- If a proposed branch already exists, automation leaves it untouched and links to it. Update it through a reviewed change if necessary; no automated force pushes are used.
- When the version already matches `main`, Release waits for successful push CI on that exact commit and reuses its Windows and Linux packages; it does not repeat checks or builds. CI builds artifacts and checksums with read-only repository permissions and retains them for seven days. If CI failed or artifacts expired, rerun CI for that commit before retrying Release. This is also the automatic continuation after a reviewed release-version PR is merged. New pushes cancel older CI runs on the same branch; active releases are not automatically cancelled.
- Rerunning failed CI does not restart a failed Release run. If fixing CI requires a new commit, push the fix first, wait for its push CI to pass, then start a fresh Release run on `main`. Keep the same version if its tag has not been created; old runs retain their original workflow and commit. If the tag already exists, follow the recovery instructions below instead of moving it.

- A fresh, minimal write-enabled runner downloads artifacts from that successful CI run, verifies checksums, creates an annotated tag for the checked commit, and creates a **draft** release. It does not install dependencies or execute built artifacts. Before tagging, it checks that `main` has not advanced; start a fresh run if it has.
- Releases are serialized and publishing stays manual. Intermediate artifacts expire after seven days. No bot has a branch-protection bypass; bot PR creation/approval remains disabled.

## In-app update compatibility

- Publish breaking filter-format or behavior changes as a **major** version. The app compares the installed and offered major versions, including skipped releases, and requires acknowledgement before downloading across that boundary. Explain what changed and how to migrate/recreate filters in the release notes before publishing; no automatic migration is supplied.
- Published stable GitHub releases are the update source; drafts and prereleases are not offered. Keep `Filtertool.exe` and `SHA256SUMS.txt` attached. The updater selects the standalone EXE (never the setup), requires its SHA-256 checksum, and replaces only the application binary on an explicit restart.
- Checksum verification detects corrupted downloads, but relies on GitHub/TLS for authenticity. This is not artifact signing or Windows Authenticode signing. Keep release permissions protected; stronger pinned-key signing can be added separately.

## Review and publish

- Download and smoke-test the draft artifacts. Review checksums, license/notice attachments, release notes and any known limitations. Standalone downloads need the accompanying license/notice files too.
- Draft notes contain download links, requirements, and links to the versioned README and changelog (a comparison with the previous tag, or commit history for the first release). Add a concise bullet-point release description and any release-specific warnings **above `## Download`** before publishing. The app displays only that description; if it is empty, it displays a fallback message. Download and setup information remain available on GitHub. Publish the draft only after explicit approval and completed checks above.
- If draft creation fails after tagging succeeded, do not delete or move the tag. Recover by creating a draft for the existing tag and attaching the `release-assets` and `release-assets-linux` artifacts from the successful CI run for that commit; the normal workflow intentionally rejects duplicate tags.

See [Security maintenance](SECURITY.md) for repository policies and the remaining supply-chain risks.
