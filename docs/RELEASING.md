# Release checklist

Release versions use stable `major.minor.patch` numbers, starting with **1.0.0**. Known bugs can be documented and fixed in patch releases; stable naming does not mean bug-free.

## Before starting

- Merge the intended changes into `main`. There is no need to bump version files or create a tag manually.
- Run `npm ci`, `npm run check`, `npm run frontend:build`, `go vet ./...`, and `npm audit --audit-level=high`. Require successful Windows and Linux CI.
- Test on Windows with temporary/test profiles: standalone launch, per-user install/uninstall, WebView2 setup, native file/folder dialogs, import, save/restart, preview, export, and loading the filter in PoE 1. Confirm uninstall preserves user data.
- Test without FFmpeg: startup, editing and preview work; TTS generation explains the missing dependency. Then install FFmpeg on `PATH`, restart, and test TTS generation/export. Confirm no FFmpeg executable or DLL is shipped.
- Review third-party redistribution permissions and required license/copyright texts for all shipped code and assets, including Go/frontend dependencies, PoE data and game alert audio. A source link or attribution alone is not necessarily sufficient permission or a complete license notice. Verify the Google Translate TTS usage terms separately; availability is not guaranteed.

## Run the workflow

- After release approval, open **Actions → Release → Run workflow**, select **main**, and enter a version such as `1.0.0` (without the `v`). Only stable `major.minor.patch` versions are accepted. Existing tags and versions below the current package version are rejected. The current version is allowed for the initial release or a retry before tagging.
- The workflow applies the version to `package.json`, both root versions in `package-lock.json`, and `build/config.yml`. Windows executable metadata and the NSIS version use that same version. Dependencies are not updated.
- Linux and Windows CI check the version-updated source before packaging. The workflow then builds the executable and installer and generates SHA-256 checksums. No version commit or tag is pushed until these steps succeed.
- If metadata changed, the workflow creates a `chore(release): v…` commit. It atomically pushes `main` and the matching annotated tag, then creates a **draft** release. If the version was already correct, it tags the checked source without an empty commit.
- Releases are serialized. If `main` advances during the build, the atomic push fails without publishing the tag; start a fresh workflow run against the latest `main`. No force pushes are used.
- The workflow needs `contents: write` in its packaging job, and branch/tag rules must permit the Actions bot's release commit/tag. It uses the built-in `GITHUB_TOKEN`; pushes made with that token do not trigger another CI/release workflow. CI already ran inside the release workflow.

## Review and publish

- Download and smoke-test the draft artifacts. Review checksums, license/notice attachments, release notes and any known limitations. Standalone downloads need the accompanying license/notice files too.
- Replace the pre-publication README note when publishing. Publish the draft only after explicit approval and completed checks above.
- If draft creation fails after the atomic push succeeded, the version commit/tag remain. Do not delete or move that tag. Recover by creating a draft for the existing tag and attaching artifacts from that exact commit; the normal workflow intentionally rejects duplicate tags.
