# Agent guide

Filtertool is a Windows desktop Path of Exile **1** leveling-filter builder. The internal repository/module name is `filtertool-desktop`; the public app name is Filtertool.

## Scope and context

- This is a small, low-criticality tool. Prefer straightforward fixes and maintainable code over exhaustive coverage, speculative hardening, or frameworks for hypothetical needs.
- Read the relevant implementation and nearby tests first. Follow the affected data flow; do not survey the whole repository or load unrelated docs by default.
- Read `docs/GLOSSARY.md` when working on filter terminology or matching behavior. Preserve rule order, exact matching, unset versus empty values, and the final unknown-item fallback.
- Keep changes within the request. Mention unrelated issues separately rather than fixing them opportunistically. Extract shared code when it materially avoids duplication, not just to add layers.
- Keep this guide short and stable. Put detailed architecture, workflows and explanations in task-specific docs, not here.

## Code map and boundaries

- `frontend/` is the Vue editor; `src/config/` contains UI-only form metadata and validation, not a TypeScript backend.
- `internal/compiler/` is the Go compiler. `testdata/` contains output captured from the original web compiler with user overrides and TTS disabled, plus the corresponding frozen weapon dataset. Keep parity tests meaningful; normal builds use current generated data.
- `internal/data/` embeds built-in defaults, styles, rule instructions, reference data and the sound manifest. Edit source data, not generated reference/weapon JSON. Base updates must be reviewed, not silently released.
- User filters, settings and sounds belong in `%APPDATA%\Filtertool`; never use personal data or game exports as test fixtures. Tests use temporary directories.
- Only existing web JSON import is supported. No TypeScript config execution or pre-stable desktop migration framework.

## UI and user experience

- Prioritize task clarity, low friction and preserved functionality over visual minimalism. Follow established modern desktop UI conventions; do not redesign unrelated areas or introduce a design system for a small change.
- Keep visible descriptions to the bare minimum. Improve grouping, layout, labels and interaction first rather than explaining a confusing UI with more text. Put optional explanations in tooltips or contextual help; keep essential warnings and errors visible.
- Prefer familiar icons for compact secondary actions such as copy, paste, reset and navigation. Use consistent SVG icons, not arbitrary Unicode glyphs. Keep text for primary or ambiguous actions when an icon alone would hurt discoverability.
- Icon-only buttons need an accessible name and a tooltip. Support keyboard operation, visible focus, sufficient contrast and comfortable click targets; never rely on color or hover alone to communicate meaning.
- Reuse existing controls and interaction patterns. Align related fields and actions, keep icon buttons proportionate to neighboring inputs, and use consistent spacing, terminology and reset behavior.
- Keep actions near the item they affect. Prefer in-place editing with live feedback; preserve scroll position and selections. Make inherited defaults, explicit overrides and the scope of shared edits clear without verbose explanations.
- Use progressive disclosure for advanced options without hiding common tasks. Keep layouts usable in narrow windows, and make empty, disabled and error states understandable. Validate the actual workflow, not just its appearance; consult targeted UX guidance when an interaction is unfamiliar.

## Proportionate testing

- Prioritize incorrect filter output or lost user data: compiler matching/priority, configuration save/load, default and override semantics, import/export, and sound references/files. Keep filesystem and path-safety checks at external boundaries.
- For a bug fix, normally add the smallest regression test that demonstrates the failure. For a feature, cover the main workflow and any materially different risky case. Reuse existing coverage; do not test every combination, trivial wrapper, or UI permutation.
- Test observable behavior and meaningful saved/output values. Avoid asserting exact DOM structure, CSS classes, decorative text, or implementation details unless those are the actual requirement.
- Test a shared behavior at its owning layer; add integration coverage only for wiring that could realistically fail. Do not duplicate the same assertions through every parent component.
- Simple visual/layout changes usually need a build and manual inspection, not new automated UI tests. Do not remove useful existing regression tests solely to reduce the test count.
- During development, run targeted tests. Before handing off code changes, run `npm run check` and `npm run frontend:build` once the changes are ready; rerun affected checks after subsequent edits. Documentation-only changes do not require the application suite.
- Use Windows CI/manual testing for native dialogs, WebView2, installer and UI behavior. State what was verified and what still needs Windows testing; do not claim unperformed checks.

## Repository actions

- Do not commit, push, tag, create a remote repository, or publish without asking the user. Use Conventional Commits when authorized.
