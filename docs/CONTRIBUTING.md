# Contributing

Requires Go 1.26+ and Node.js 24. Running the desktop UI requires 64-bit Windows 10+ and WebView2; development checks and builds are not Windows-only.

```sh
npm ci
npm run dev
npm run check
npm run build
```

Dev **Save** edits templates/defaults/styles in `internal/data/`, not personal data. Use `npm run dev:personal` for your personal profile. Builds produce `bin/Filtertool.exe`.

Use Conventional Commits. Keep personal data out of commits and tests. Test native UI behavior on Windows before releases.
