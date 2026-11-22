# spanslo

[![ci](https://github.com/sankaranainar-p/spanslo/actions/workflows/ci.yml/badge.svg)](https://github.com/sankaranainar-p/spanslo/actions/workflows/ci.yml)
![Go 1.27](https://img.shields.io/badge/go-1.27-00ADD8)
![Python 3.12](https://img.shields.io/badge/python-3.12-3776AB)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

spanslo scores how well AI agents survive injected faults, using standard OpenTelemetry traces. It computes SLIs, SLOs and error budgets from `gen_ai.*` spans, independently of any one fault injector, and publishes a benchmark of real recovery behavior.

It measures. It does not inject faults into other people's systems, and it does not govern agents.

**Status: pre release.** The scoring core is landing in stages toward `v0.2.0`. The spec is frozen at tag `v0.1.1-spec`. There is no stable CLI yet.

## What it does

- Reads recorded OpenTelemetry GenAI traces (OTLP JSON) and groups spans into runs.
- Indexes turns and steps with fixed, documented rules, so the same trace always gives the same result.
- Computes recovery SLIs per fault episode, such as turns to recover and time to recover.
- Evaluates SLOs and burn rates over time windows or run count windows.
- Keeps scoring pure: the Go core has no I/O and never reads the clock.

## Design in one view

- `core/` (Go): scoring. Pure functions over parsed traces.
- `cmd/spanslo/` (Go): the offline CLI.
- `collector/` (Go): a collector connector, planned for later weeks.
- `bench/` (Python 3.12, uv): the benchmark harness and schema tests.
- `spec/`: the shared contract. JSON Schema 2020-12, SLI definitions and the semantic convention mapping.
- `catalog/`: fault definitions used by the benchmark.
- `testdata/`: trace fixtures.
- `docs/`: design, architecture decision records and dependency list.

The GenAI semantic conventions are pinned to a commit in `spec/semconv-mapping.yaml`. See `docs/adr/0002-semconv-pin.md`.

## Build and test

```
go vet ./... && go test ./... && (cd bench && uv run pytest && uv run ruff check)
```

Needs Go 1.27 and [uv](https://docs.astral.sh/uv/) with Python 3.12.

## Limits

- Pre release. Interfaces and output formats can change before `v0.2.0`.
- Output is spanslo's own JSON. Other report formats come later.
- Runs are read from JSON lines. Parquet is deferred.
- The semantic conventions are in development upstream. The pin makes results reproducible, but it will need re-checking at each gate.
- Adapters for external fault injectors need real captured fixtures. They are not built yet.
- Benchmark results do not exist yet. Do not read any number in this repo as a published result.

## Non-goals

- A general purpose fault injector or proxy for other people's systems.
- A competing SLO specification. spanslo aligns with Microsoft's AGENT-SRE-GOVERNANCE-1.0.
- Security red teaming. Prompt injection faults appear only as tool payload faults.
- Software delivery or coding agent tasks, and anything involving payments or employer data.

## Documentation

- [Design](docs/design.md)
- [Spec](spec.md) and [spec/](spec/)
- [Decisions](docs/adr/)
- [Dependencies](docs/dependencies.md)
- [Contributing](CONTRIBUTING.md), [Security](SECURITY.md), [Changelog](CHANGELOG.md)

## Cite

See [CITATION.cff](CITATION.cff).

## Disclaimer

spanslo is an independent personal project. It is not affiliated with, endorsed by or derived from any employer's systems, data or designs. It uses public data and local deterministic test tools only.

## License

Apache License 2.0. See [LICENSE](LICENSE).
