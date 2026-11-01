# Release checklist

Release versions use stable `major.minor.patch` numbers, starting with **1.0.0**. Known bugs can be documented and fixed in patch releases; stable naming does not mean bug-free.

## Before starting

- Merge the intended changes into `main`. There is no need to bump version files or create a tag manually.
- Run `npm ci`, `npm run check`, `npm run frontend:build`, `go vet ./...`, and `npm audit --audit-level=high`. Require successful Windows and Linux CI.
- Test on Windows with temporary/test profiles: standalone launch, per-user install/uninstall, WebView2 setup, native file/folder dialogs, import, save/restart, preview, export, and loading the filter in PoE 1. Confirm uninstall preserves user data.
- Test without FFmpeg: startup, editing and preview work; TTS generation explains the missing dependency. Then install FFmpeg on `PATH`, restart, and test TTS generation/export. Confirm no FFmpeg executable or DLL is shipped.
- Review third-party redistribution permissions and required license/copyright texts for all shipped code and assets, including Go/frontend dependencies, PoE data and game alert audio. A source link or attribution alone is not necessarily sufficient permission or a complete license notice. Verify the Google Translate TTS usage terms separately; availability is not guaranteed.

## Run the workflow

- After release approval, open **Actions → Release → Run workflow**, select **main**, and enter a version such as `1.0.0` (without the `v`). Only stable `major.minor.patch` versions are accepted. Existing tags and versions below the current package version are rejected.
- If the version differs from `main`, the workflow prepares `package.json`, both root versions in `package-lock.json`, and `build/config.yml` on a `release/v…` branch. Open the PR using the link in the job summary, review it, wait for CI, and merge it. **Then run Release again with the same version.** The workflow never pushes directly to `main` and does not create or approve PRs. Alternatively, maintainers can update the version files directly on `main` before running Release.
- If a proposed branch already exists, automation leaves it untouched and links to it. Update it through a reviewed change if necessary; no automated force pushes are used.
- When the version already matches `main` (including the initial `1.0.0`), Linux and Windows CI run before packaging. Executable/installer versions use the checked metadata; dependencies are not updated. The workflow builds the artifacts and checksums with read-only repository permissions.
- A fresh, minimal write-enabled runner downloads artifacts from that same run, verifies checksums, creates an annotated tag for the checked commit, and creates a **draft** release. It does not install dependencies or execute built artifacts. Before tagging, it checks that `main` has not advanced; start a fresh run if it has.
- Releases are serialized and publishing stays manual. Intermediate artifacts expire after seven days. No bot has a branch-protection bypass; bot PR creation/approval remains disabled.

## Review and publish

- Download and smoke-test the draft artifacts. Review checksums, license/notice attachments, release notes and any known limitations. Standalone downloads need the accompanying license/notice files too.
- Replace the pre-publication README note when publishing. Publish the draft only after explicit approval and completed checks above.
- If draft creation fails after tagging succeeded, do not delete or move the tag. Recover by creating a draft for the existing tag and attaching the `release-assets` artifact from that workflow run; the normal workflow intentionally rejects duplicate tags.

See [Security maintenance](SECURITY.md) for repository policies and the remaining supply-chain risks.
