# Security policy

## Supported versions

spanslo is pre release. Only the latest commit on `main` and the latest tag are supported.

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Report it through GitHub private vulnerability reporting: open the repository's Security tab and choose "Report a vulnerability".

Include what you found, the steps to reproduce it, and the version or commit. You should get a first reply within 7 days. This is a one person project, so fix times vary with severity.

## Scope

In scope: the Go scoring core, the CLI, the collector connector, the benchmark harness and the CI configuration.

Out of scope: vulnerabilities in third party dependencies with no spanslo specific impact (report those upstream), and results of running spanslo on traces you do not control.

## Handling of trace data

spanslo reads trace files you give it. Traces from agents can hold prompts, tool arguments and outputs. Treat them as sensitive. Do not attach real production traces to public issues.
