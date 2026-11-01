# Contributing

Requires Go 1.26+ and Node.js 24. Running the desktop UI requires 64-bit Windows 10+ and WebView2; development checks and builds are not Windows-only.

```sh
npm ci
npm run dev
npm run check
npm run build
```

Dev **Save** edits templates/defaults/styles in `internal/data/`, not personal data. Use `npm run dev:personal` for your personal profile. Builds produce `bin/Filtertool.exe`.

`npm ci` installs a pre-commit hook via Husky. It runs Prettier and `gofmt` on staged files and stages the formatting changes; lint-staged protects unstaged edits in partially staged files. Existing Prettier ignores still apply. Keep Node.js/npm and Go on your Git client's `PATH`. Run `npm run format` to format the whole project; the full test suite stays in `npm run check` and CI, not the hook. For an exceptional bypass, use `HUSKY=0 git commit …` (Git Bash/WSL syntax).

Use Conventional Commits. Keep personal data out of commits and tests. Test native UI behavior on Windows before releases.

See [Releasing](RELEASING.md) for the release checklist.
