# Stage 7: Critic Review of spec.md (lines 1 to 97)

Reviewers: egm-design and egm-structural (both against feature-spec-standard.md), egm-style (no standard). All three completed. No context files were supplied, so none of them read docs/design.md.

Verdicts: design FAIL, structural FAIL, style no Blockers.

Finding ids: D = design, S = structural, T = style. Duplicates across reviewers are merged below.

## Blockers

| # | Ids | Scope | Finding |
| --- | --- | --- | --- |
| B1 | D1, S1 | Whole file | "Alternatives Considered" is missing. Choices with no rejected alternative: offline Go core with no I/O (R-008), SHA-256 args hash as the tie-break (R-005), baseline median over the dataset (R-036), episode semantics (R-010 to R-014), adapter per source (R-056). |
| B2 | D2, S2 | Whole file | "Risks & Mitigations" is missing. Unaddressed: whole-file memory use (R-001, R-060); malformed or hostile trace input and `spanslo.run.id` values (R-003); truncated runs, missing chat spans and clock skew affecting R-018 and R-019; `spanslo gate` behavior on parse errors or empty input (R-063); no exit codes defined for any command. |
| B3 | D3, S3, S4 | Lines 5 to 9 | No Problem Statement and no Proposed Architecture or data flow. G2 is cited but never defined. Everything is delegated to design.md section names. |

## Major

| # | Ids | Scope | Finding |
| --- | --- | --- | --- |
| M1 | D4 | R-015, R-016 | The oracle and the `spanslo.task_success` event are not tied together. Behavior when the event is absent, duplicated or has a score other than 0.0 or 1.0 is not stated. |
| M2 | D5 | R-010 to R-013, R-017, R-018 | "Qualifying recovery" and the source of "declared fallback" are undefined. Overlapping episodes on different targets, a fault arriving mid recovery, and unrecovered episodes (null or zero for R-017 and R-018, which affects R-046) are open. |
| M3 | D6 | R-014, R-027, R-028, R-035 | A run whose faults were all untriggered has zero episodes and is vacuously recovered. Its treatment in numerators and denominators is not stated. |
| M4 | D7 | R-007, R-017, R-018 | The authoritative turn source (derived or fault attribute) is not stated. Negative `time_to_recover_ms` when the recovery call overlaps the fault span has no rule. |
| M5 | D8, T8 | R-021 to R-024, R-042 | "Policy deny verdict" has no definition or source. R-023 does not say whether retry counts are per run or per episode, or whether the first call counts. |
| M6 | D9 | R-036, R-039, R-044, R-045, R-047 | Value reported for an unknown task or a zero baseline is not defined (null, skipped or infinite). No minimum baseline sample size. R-036 reads as contradicting the key by task, framework and model rule. Sf and S0 are not defined in the file. |
| M7 | D10 | R-047, R-048 to R-055 | The run record timestamp source, the anchor for wall clock windows over recorded files, and the ordering for "last N runs" are not defined. A target of 1.0 gives a zero error budget and a divide by zero. R-055's "never negative before exhaustion" is unclear. |
| M8 | D11 | R-056 to R-059 | Input formats of the four sources are not described. Malformed records from a known source have no defined behavior. Unknown sources "pass through untouched" can hide failures. The R-059 time window join can misattribute faults, with no flag for heuristic matches. |
| M9 | D12 | R-060 to R-063 | No exit codes beyond "non zero" for gate. The gate's treatment of budget exhaustion, burn alerts and `burn: false` SLOs is unspecified. R-062 has no schema or reference. R-061 and R-062 have no acceptance test. |
| M10 | D13 | R-001, R-002 | OTLP JSON version and shapes (multiple spans per file, nested scope spans, attribute value types) are not pinned. R-002 is not enforced by any test or lint. |
| M11 | D14 | R-009, R-033, R-064, R-065 | Fixtures for R-010 to R-014, R-021 to R-023 and R-026 to R-030 are not enumerated. R-027 to R-030 have no direct test. "Hand checked" does not say by whom. |
| M12 | S5 | Lines 5 to 9 | "Goal" and "Non-goals" are not the standard's paired "Goals & Non-Goals", and there are no enumerated goals or success criteria. |
| M13 | T1 | Line 4 | "The human" who approves the draft is not named, and no approval mechanism is stated. |
| M14 | T2 | Source column | Source cells do not say which file they cite. Entries naming CLAUDE.md and `catalog/defaults.yaml` are mixed with design.md section names. |
| M15 | T3 | R-047, R-055 | Test requirements have no stated expected value for a zero denominator and no defined budget floor. |
| M16 | T4 | R-049, R-051 | "`target` gives 1 minus target" and `burn: false` are terse and not written as formulas. |

## Nits

| # | Ids | Scope | Finding |
| --- | --- | --- | --- |
| N1 | S6, S8, D15 | Headings, line 3 | Heading wording and capitalization are inconsistent. The status line has no version, date or approver. |
| N2 | S7, D16 | Source column | No links, and design.md is not version pinned (git log shows v2.5). |
| N3 | T5, T7 | R-033, R-035 | Repeated clauses: R-033 repeats R-024, and R-035 repeats R-027. |
| N4 | T6, D18 | R-025 | "Laid over it, key by key" is figurative and does not say whether nested keys or lists are replaced or merged. |
| N5 | D17 | Non-goals | No explicit non-goals for this scope (live ingestion, storage backend, record schema migration, concurrency). |
| N6 | D19 | R-049 | No precedence or validation rule when more than one SLO form appears. |
| N7 | D20 | R-027 | Default `spanslo.mode` when absent is not stated. |
| N8 | T9, T10, T11 | Style | "Source:" is used in two senses, compound modifier hyphenation is inconsistent, and R-032 "answers wrongly" should say the oracle fails. |

## Elephant note (not part of the reviewers' reports)
- The reviewers worked without design.md by design. B3, M1, M2, M5, M6 and M14 largely follow from spec.md citing design.md instead of restating it, which is what the request asked for ("cite the design.md section", "add NO new requirements"). Treat them as a scope decision for the human, not defects to fix by inventing requirements.
- M5 (policy deny verdict), M6 and M2 (unrecovered episodes) overlap questions 6, 8 and 9 already raised in the hand off notes. M7 overlaps question 5.
- B1 and B2 conflict with "Add NO new requirements" and the "Requirements for weeks 5 to 8 only" scope. A spec.md that adds Alternatives and Risks sections is a different document from the one requested.
- The Stage 5 report also recommended cutting the external adapters and burn window pairs. See `zoom_out_report.md`.
