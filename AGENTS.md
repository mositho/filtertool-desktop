# Agent guide

Filtertool is a Windows desktop Path of Exile **1** leveling-filter builder. The internal repository/module name is `filtertool-desktop`; the public app name is Filtertool.

- Read `GLOSSARY.md` for filter terminology. Preserve rule order, exact matching, unset versus empty values, and the final unknown-item fallback.
- `frontend/` is the existing Vue editor; `src/config/` contains UI-only form metadata and validation, not a TypeScript backend.
- `internal/compiler/` is the Go compiler. `testdata/` contains output captured from the original web compiler with user overrides and TTS disabled, plus the corresponding frozen weapon dataset. Keep parity tests meaningful; normal builds use current generated data.
- `internal/data/` embeds built-in defaults, styles, rule instructions, reference data and the sound manifest. Edit source data, not generated reference/weapon JSON. Base updates must be reviewed, not silently released.
- User filters, settings and sounds belong in `%APPDATA%\Filtertool`; never use personal data or game exports as test fixtures. Tests use temporary directories.
- Only existing web JSON import is supported. No TypeScript config execution or pre-stable desktop migration framework.
- Run `npm run check` and `npm run frontend:build`; use Windows CI/manual testing for native dialogs, WebView2, installer and UI behavior.
- Do not commit, push, tag, create a remote repository, or publish without asking the user. Use Conventional Commits when authorized.
