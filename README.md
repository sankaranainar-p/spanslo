# spanslo

spanslo scores how well AI agents survive injected faults using standard OpenTelemetry traces, and publishes a benchmark of real recovery behavior. It computes SLIs, SLOs and error budgets from `gen_ai.*` spans independently of any one fault injector. It measures; it does not inject faults for other people's systems or govern agents.

Status: design phase

## Non-goals

- A general purpose fault injector or proxy for other people's systems
- A competing SLO specification (align with Microsoft's AGENT-SRE-GOVERNANCE-1.0 instead)
- Security red teaming; prompt injection faults appear only as tool payload faults
- Software delivery or coding agent tasks, and anything involving payments or employer data

See [docs/design.md](docs/design.md).
