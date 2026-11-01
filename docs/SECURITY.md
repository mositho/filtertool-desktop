# Security maintenance

- Review dependency and workflow updates manually. Dependabot opens PRs with a seven-day cooldown for routine updates; security updates are not delayed.
- Maintainers can push directly to `main`. CI and dependency review are advisory, not required merge gates; review results before releases. Force pushes and branch deletion are blocked.
- Actions default to read-only, use allowlisted SHA-pinned actions, and require approval for external fork workflows. Bot PR creation/approval and auto-merge are disabled.
- Builds run read-only; fresh, minimal write-enabled jobs prepare proposal branches or draft releases. Version/data updates provide a link to open a PR manually. Release tags cannot be changed or deleted.
- Keep secret scanning/push protection enabled and protect maintainer accounts with MFA/passkeys. Periodically review tokens, collaborators and installed apps.

CI scans known vulnerabilities, but green tests and checksums cannot prove dependencies or binaries are safe. Signing, attestations and reproducible-build verification are not currently provided.

Repository settings are separate from workflow files. The action allowlist is recorded in `.github/actions-policy.json`. See [Releasing](RELEASING.md) for the release process.
