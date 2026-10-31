# Contributing

Use Go 1.26+ and Node.js 24, either on Windows or in WSL. WebView2 is required on Windows to run the app.

```sh
npm ci
npm run dev
```

One command opens the Windows app with UI hot reload and Go rebuilds. In WSL, keep Go, Node and the checkout in WSL; Windows interop and localhost forwarding must be enabled. The launcher forwards Wails' dev-server URL automatically—no Windows Go/Node installation is needed. Development uses the same AppData folder as the built app; back up valuable filters before testing changes.

### Editing bundled templates

Run `npm run dev:templates` (or launch `Filtertool.exe --templates`) to open **Template Editor**. It uses `%APPDATA%\Filtertool\profiles\templates` for filters, settings, sounds and WebView data, with a separate single-instance identity. Normal launch returns to your personal filters. The workspace contains only Caster Template and Attack Template; names are fixed, and personal imports/global-default edits are unavailable. No game export folder is selected automatically.

**Save** persists edits only in that workspace. **Export template bundle…** saves both templates as `templates.json`, saving the current filter first if needed. To include reviewed edits in a build, explicitly save the bundle to `internal/data/templates.json` in this checkout, run `npm run format` and `npm run check`, and review the diff. Reopening the workspace preserves edits rather than reseeding from the bundled file. Existing personal filter copies are never renamed, removed or overwritten.

Normal mode does not automatically add templates to the filter list. When creating a filter, choose Blank, Caster Template or Attack Template as the starting template; the resulting personal copy is freely editable.

```sh
npm run check
npm run build
```

The build creates `bin/Filtertool.exe`. `npm run format` formats Vue/TypeScript and all Go files using standard `gofmt` formatting; `check` enforces formatting. `go vet ./...` runs in CI. Verify native dialogs, audio, shortcuts, save/reopen, installer/uninstaller and both distribution formats on Windows before releasing.

Use Conventional Commits. For a release, update `package.json` and `build/config.yml` together, run checks, then tag the checked commit `v<version>`. The workflow builds both executables, checksums and a **draft** release. Publish manually only after the desktop app is stable; include any changes to generated rules, defaults or sounds in the release notes. Do not promise compatibility for unreleased desktop data.

Never commit personal filters, settings or generated audio. Do not use online TTS or personal game files in automated tests.
