# Web compiler reference

These ten JSON configurations and `.filter` outputs were captured from the original Filtertool TypeScript compiler with user overrides and TTS generation disabled. `_defaults` and `_styles` are test-only inputs for the global-override case.

`data/weapons.json` freezes the matching source dataset. Parity tests give both implementations the same inputs; normal desktop builds use the current generated data in `internal/data`. Do not regenerate expected output merely to make a compiler regression pass.
