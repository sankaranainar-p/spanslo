# Changelog

All notable changes are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions are not stable before 0.2.0.

## [Unreleased]

### Added
- Names mapping read from the pinned semantic convention file, with a test that forbids hardcoded `gen_ai.*` literals.
- RFC 8785 canonical JSON and argument hashing in the core.
- OTLP JSON reader and grouping of spans into runs.
- Turn and step indexing, and the turn mismatch count.
- Trace fixtures for indexing edge cases, and a purity check for the core.
- Hand written Go run record types with a schema drift test (ADR 0004).

### Known limits
- Duplicate JSON object keys keep the last value. RFC 8785 says to reject them.

## [0.1.1-spec]

### Added
- Spec frozen: JSON Schemas for fault, run record, SLO and tool annotations, the SLI definitions and the semantic convention mapping.
- ADR 0002 pins the GenAI semantic conventions to a commit.

## [0.1.0-spec]

### Added
- First spec draft and the week 1 research record (`docs/g0/`).
