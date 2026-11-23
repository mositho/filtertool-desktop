# Security maintenance

- Review dependency and workflow updates manually. Dependabot opens PRs with a seven-day cooldown for routine updates; security updates are not delayed.
- Maintainers can push directly to `main`. CI and dependency review are advisory, not required merge gates; review results before releases. Force pushes and branch deletion are blocked.
- Actions default to read-only, use allowlisted SHA-pinned actions, and require approval for external fork workflows. GitHub's combined setting permitting Actions to create/approve PRs is enabled for base-data PR creation; workflows never approve or auto-merge PRs.
- Builds run read-only; fresh, minimal write-enabled jobs prepare proposals or draft releases. Release-version and Go-patch updates provide a link to open a PR manually. Base-data updates automatically open or refresh a PR and explicitly dispatch full CI; review and merge manually. Release tags cannot be changed or deleted.
- Keep secret scanning/push protection enabled and protect maintainer accounts with MFA/passkeys. Periodically review tokens, collaborators and installed apps.

CI scans known vulnerabilities, but green tests and checksums cannot prove dependencies or binaries are safe. Signing, attestations and reproducible-build verification are not currently provided.

Repository settings are separate from workflow files. The action allowlist is recorded in `.github/actions-policy.json`. See [Releasing](RELEASING.md) for the release process.

## Base-data updates

**Update Base Types** runs weekly on Monday or manually on `main`. When data changes, it refreshes the automation-owned `chore/update-basetypes` branch from the run's `main` commit and opens one PR. Identical proposals reuse the existing commit. Review changes in the PR; do not edit this reserved branch. Publishing refuses non-bot commits or unrelated files rather than overwriting them, and uses an explicit force-with-lease when refreshing the branch.

The publishing job alone has contents, PR and Actions write permissions. It copies only the three expected prepared data files and never installs dependencies or executes downloaded data. It dispatches `CI` on the proposal branch because `GITHUB_TOKEN`-created PRs do not trigger ordinary PR CI. Check that both Linux and Windows checks pass on the latest proposed commit before merging. If `main` advances during preparation, rerun the update workflow.
