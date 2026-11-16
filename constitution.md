# Constitution

Fixed rules for every agent on every task in this project. Only the human edits this file. Copied from CLAUDE.md; if they ever differ, CLAUDE.md wins and this file is stale.

## Language boundaries
- Go owns scoring: `core/`, `core/adapters/`, `collector/`, `cmd/spanslo/`. Python owns `bench/` and the user facing Python package.
- `core/` is pure functions over spans. No network or file I/O inside it.
- The boundary between Go and Python is files and the CLI, never linked code. No cgo, no ctypes, no embedded interpreters.

## Spec
- `spec/` is the shared contract and is frozen at G1 (tag v0.1.0-spec). Do not edit it. Go types come from the schemas, never hand edited.
- Never hardcode `gen_ai.*` names. Read them from `spec/semconv-mapping.yaml`.
- If code and `docs/design.md` disagree, stop and ask.

## Dependencies and license
- Ask before adding any dependency. Record its name, version and license in `docs/dependencies.md`.
- The project is Apache 2.0.

## Testing
- Every change ships with a test that proves it. Never weaken or delete a test to make it pass.
- Before finishing: `go vet ./...`, `go test ./...`, `uv run pytest`, `uv run ruff check`.

## Scope and data
- Build only what a task asks for, and cite the R-IDs it satisfies. Out of scope findings are listed at the end, not fixed.
- Public data and the local deterministic bench tools only. No employer data, names, code, prompts or designs.

## Git
- Small commits with clear imperative messages.
- No `Co-Authored-By` lines, AI attribution or "Generated with" footers in commits, pull requests or docs. The `.githooks/commit-msg` hook rejects them.
- Agents never merge their own work. Do not push, tag or release unless the human asks.

## Never do
- Report success without evidence.
- Change this file.
