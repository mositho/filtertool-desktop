# Contributing

Use Go 1.26+ and Node.js 24, either on Windows or in WSL. WebView2 is required on Windows to run the app.

```sh
npm ci
npm run dev
```

One command opens the Windows app with UI hot reload and Go rebuilds. In WSL, keep Go, Node and the checkout in WSL; Windows interop and localhost forwarding must be enabled. The launcher forwards Wails' dev-server URL automatically—no Windows Go/Node installation is needed. Development uses the same AppData folder as the built app; back up valuable filters before testing changes.

```sh
npm run check
npm run build
```

The build creates `bin/Filtertool.exe`. `npm run format` formats Vue/TypeScript and all Go files using standard `gofmt` formatting; `check` enforces formatting. `go vet ./...` runs in CI. Verify native dialogs, audio, shortcuts, save/reopen, installer/uninstaller and both distribution formats on Windows before releasing.

Use Conventional Commits. For a release, update `package.json` and `build/config.yml` together, run checks, then tag the checked commit `v<version>`. The workflow builds both executables, checksums and a **draft** release. Publish manually only after the desktop app is stable; include any changes to generated rules, defaults or sounds in the release notes. Do not promise compatibility for unreleased desktop data.

Never commit personal filters, settings or generated audio. Do not use online TTS or personal game files in automated tests.
