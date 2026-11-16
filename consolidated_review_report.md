# Consolidated review of spec.md (weeks 5 to 8, gate G2)

Mode: consolidated report, no checkpoints. `spec.md` was not edited.
Ranges: FeatureRange lines 1 to 97, DetailedRange N/A (no implementation plan section). Stage 9 (Implementation Readiness) was not run.
Context: none supplied. No reviewer read docs/design.md, so the Source citations in spec.md are unverified by this review.

| Stage | Report | Reviewers | Result |
| --- | --- | --- | --- |
| 5 Zoom Out | `zoom_out_report.md` | egm-zoom-out | Pass, 4 Major, 3 Nit |
| 6 Comprehension | `comprehension_report.md` | egm-comprehension | 2 Blocker, 8 Major, 4 Nit |
| 7 Critic | `critic_report.md` | egm-design, egm-structural, egm-style | Design FAIL, structural FAIL, style no Blockers |
| 8 Zoom Out (second pass) | `intermediate_zoom_out_report.md` | egm-zoom-out | Pass, Major findings |
| 9 Implementation Readiness | not run | n/a | No implementation plan in the document |

## The main disagreement, for you to decide
The reviewers split on one question. Comprehension and Critic read spec.md as a standalone document and mark the missing Alternatives, Risks, Problem Statement, Architecture and glossary as Blockers. The Zoom Out reviewer reads it as a citation layer over design.md and recommends rejecting most of those sections. Your request asked for citations, no new requirements and weeks 5 to 8 only, which supports the Zoom Out reading. This report does not decide it.

## Actions

### 🔴 Blocker, decision needed
| # | Action | Origin |
| --- | --- | --- |
| A1 | Decide whether spec.md must be self-contained (add Alternatives, Risks, Problem Statement, Architecture, glossary) or stay a citation layer over design.md (link it once, pin v2.5, state that Source cells refer to design.md headings unless a file is named). | Critic B1, B2, B3; Comprehension 1; Zoom Out Stage 8 |

### 🟡 Major
| # | Action | Origin |
| --- | --- | --- |
| A2 | Decide scope: cut G2 to the bench adapter, R-057 and R-058, and evaluate SLOs over the whole dataset or one count window; or keep all four adapters and the burn windows and add the missing definitions (adapter input formats, window anchor, run ordering). Cutting conflicts with the design.md week 8 row. | Zoom Out Stage 5 findings 1 and 2, Stage 8; Critic M7, M8; Comprehension 7 |
| A3 | State one pipeline (reader, adapters, grouping, scorer, SLO evaluation) with one owner for normalization, and have `slo`, `report` and `gate` share one evaluation. Define the agent-sre report or cite its schema. | Zoom Out Stage 5 finding 3; Critic M9; Comprehension 8, 14 |
| A4 | Define exit codes for all four commands and `gate` behavior on empty or unparseable input. | Critic B2, M9; Zoom Out Stage 8 |
| A5 | Add one-clause behavior for gaps that change results: unrecovered episodes (null, excluded from percentiles), runs with zero triggered episodes, zero baseline or unknown task, target 1.0 zero budget, default `spanslo.mode`, negative `time_to_recover_ms`. | Critic M2, M3, M4, M6, M7, N7; Comprehension 10; Zoom Out Stage 8 |
| A6 | Define or drop "policy deny verdict" in R-042, define "qualifying recovery" and the source of "declared fallback", and say whether retry limits count per run or per episode and whether the first call counts. | Critic M2, M5; Comprehension 3, 4; Stage 8 net finding 4 |
| A7 | Tie the oracle to the `spanslo.task_success` event, and say what happens when the event is missing, duplicated or off scale. | Critic M1; Comprehension 6 |
| A8 | Move acceptance statements (R-009, R-033, R-047, R-055, R-064, R-065) into a short per week list, and drop the duplicates R-032, R-033 (repeat of R-024) and R-035 (repeat of R-027). | Zoom Out Stage 5 finding 4; Critic M11, M15, N3; Style 5, 7 |
| A9 | Name an approver and a version, define G2, and say which file each Source cell cites. | Critic M13, M14, N1, N2; Style 1, 2; Comprehension 12 |
| A10 | Write the budget rules as formulas (budget = 1 minus target) and define `burn: false`. | Critic M16 (Style 4) |
| A11 | Enforce R-002 with a test or lint, and say which OTLP JSON shapes the reader must accept. | Critic M10 |

### 🟢 Nit
| # | Action | Origin |
| --- | --- | --- |
| A12 | Name the canonicalization (RFC 8785) in R-005; say whether R-025 merges nested keys; add precedence for SLO forms; add one line of non-goals (live ingestion, storage, migration, concurrency). | Zoom Out Stage 5 finding 5; Critic N4, N5, N6 |
| A13 | Heading wording, hyphenation, "Source:" used in two senses, "laid over it", "answers wrongly" (say the oracle fails). | Critic N1, N8 |

## Traceability matrix

| Raw findings | Action |
| --- | --- |
| Stage 5: 1, 2 | A2 |
| Stage 5: 3 | A3 |
| Stage 5: 4 | A8 |
| Stage 5: 5 to 7 | A8, A12 |
| Stage 6: 1, 2 | A1, A10 |
| Stage 6: 3, 4 | A6 |
| Stage 6: 5 | A5 |
| Stage 6: 6 | A7 |
| Stage 6: 7 | A2 |
| Stage 6: 8 | A3 |
| Stage 6: 9, 10 | A5, A12 |
| Stage 6: 11 to 14 | A9, A5, A3 |
| Stage 7: B1, B2, B3 | A1, A4 |
| Stage 7: M1 | A7 |
| Stage 7: M2 to M6 | A5, A6 |
| Stage 7: M7, M8 | A2, A5 |
| Stage 7: M9 | A3, A4 |
| Stage 7: M10 | A11 |
| Stage 7: M11, M15 | A8 |
| Stage 7: M12 to M14 | A9 |
| Stage 7: M16 | A10 |
| Stage 7: N1 to N8 | A9, A12, A13 |
| Stage 8: net 1 | A1 |
| Stage 8: net 2, 3 | A5 |
| Stage 8: net 4 | A6 |

## Not done
- Stage 9 was skipped because the document has no implementation plan.
- No reviewer saw docs/design.md, so none could check a citation or a formula against it.
- No edits were made. The `egm-resolve` step that applies fixes was not run, because the consolidated mode skips it.
- The review reports (`zoom_out_report.md`, `comprehension_report.md`, `critic_report.md`, `intermediate_zoom_out_report.md`, `consolidated_review_report.md`) are new, uncommitted files in the repo root.
