# Contributing

Thanks for looking. spanslo is pre release, and issues and discussion are the most useful contribution right now.

## Build and test

```
go vet ./... && go test ./... && (cd bench && uv run pytest && uv run ruff check)
```

Enable the commit hook once per clone:

```
git config core.hooksPath .githooks
```

## Ground rules

- `spec/` is the contract. A change to it needs a decision recorded in `docs/adr/` and a version bump in the mapping or spec.
- Go scoring code in `core/` stays pure: no I/O, no clock, no network.
- Do not hardcode `gen_ai.*` names. They come from `spec/semconv-mapping.yaml`.
- A new dependency needs approval first and a row in `docs/dependencies.md` with its license.
- Tests come with every behavior change. Prefer fixtures under `testdata/`.
- Use public data and local deterministic tools only. No employer data, names, code, prompts or designs.
- Do not add AI co-author or attribution trailers to commits. The hook rejects them.

## Pull requests

- Keep one change per pull request.
- Say which requirement in `spec.md` it satisfies, if any.
- CI must pass.

## How the project is built

Much of the Go core is written through [agent-foundry](https://github.com/sankaranainar-p/agent-foundry), a spec driven pipeline. Every change is reviewed and merged by the maintainer, and the same tests and rules apply to any change.
