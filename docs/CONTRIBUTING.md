# Contributing

Requires Go 1.26.10+ and Node.js 26 (see `.nvmrc`). Use `nvm install && nvm use`, or `fnm use --install-if-missing`, before installing dependencies. Windows desktop builds require Windows 10+ and WebView2. Linux desktop builds use GTK 4 and WebKitGTK 6.0; on Debian or Ubuntu install `libgtk-4-dev` and `libwebkitgtk-6.0-dev` before building.

```sh
npm ci
npm run dev
npm run check
npm run build
```

On Linux, build the native Linux executable with `npm run build:linux`; run it with `./bin/Filtertool`. `npm run dev` selects the native desktop runtime for Linux and the Windows executable in WSL.

To build the Linux AppImage, run `npm run package:appimage`. It builds the Linux executable, downloads `appimagetool` into `bin/` if it is not already installed, and writes `bin/Filtertool-x86_64.AppImage`; the AppImage requires GTK 4 and WebKitGTK 6.0 from the host distribution.

Dev **Save** edits templates/defaults/styles in `internal/data/`, not personal data. Use `npm run dev:personal` for your personal profile. Windows builds produce `bin/Filtertool.exe`; Linux builds produce `bin/Filtertool`.

To build the installer locally:

1. Run `npm run build` in WSL/Linux or Windows to produce `bin/Filtertool.exe`.
2. From the repository root in Windows PowerShell (5.1 or 7), run `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/package-windows.ps1` (or use `pwsh`). The override applies only to that process, not system-wide.

The packaging step needs no Node.js, Go or global NSIS installation on Windows. It downloads checksum-pinned portable NSIS and the signed Microsoft WebView2 bootstrapper, then writes `bin/Filtertool-Setup.exe` and `bin/SHA256SUMS.txt`.

From WSL, you can run that packaging step directly with `powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(wslpath -w scripts/package-windows.ps1)"`. The execution-policy override applies only to that process.

`npm ci` installs a pre-commit hook via Husky. It runs Prettier and `gofmt` on staged files and stages the formatting changes; lint-staged protects unstaged edits in partially staged files. Existing Prettier ignores still apply. Keep Node.js/npm and Go on your Git client's `PATH`. Run `npm run format` to format the whole project; the full test suite stays in `npm run check` and CI, not the hook. For an exceptional bypass, use `HUSKY=0 git commit …` (Git Bash/WSL syntax).

Use Conventional Commits. Keep personal data out of commits and tests. Test native UI behavior on Windows before releases.

See [Releasing](RELEASING.md) for the release checklist.

Dependency updates are reviewed manually. See [Security maintenance](SECURITY.md) for protected-branch rules, workflow permissions and supply-chain safeguards.
