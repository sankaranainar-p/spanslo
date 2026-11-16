# Stage 8: Intermediate Zoom Out of spec.md (lines 1 to 97)

Reviewer: egm-zoom-out, second pass, after the Comprehension and Critic stages. Read: spec.md, zoom_out_report.md, comprehension_report.md, critic_report.md. design.md was not supplied.

Verdict: pass with Major findings. No Blockers from the Zoom Out view. The Critic's FAIL verdicts come mostly from checklist sections (Alternatives, Risks, Problem Statement, Architecture). Adding them in full would make the spec heavier without making the build safer.

## Stage 5 findings after Stages 6 and 7

| Stage 5 # | Status | What Stages 6 and 7 add |
| --- | --- | --- |
| 1 (adapters, R-056 to R-059) | Stands, Major | Comprehension 7 and Critic M8 find the four adapters unexplained. Cutting to bench only (R-057, R-058, the bench adapter) removes those findings along with the scope. |
| 2 (wall clock windows and multi window burn, R-052 to R-054) | Stands, Major | Critic M7 independently finds no anchor for "now" and no ordering for "last N runs". For G2, evaluate over the whole dataset or one count window. |
| 3 (pipeline ownership, shared SLO evaluation, undefined agent-sre format) | Stands, Major | Comprehension 8 and Critic M9 confirm R-062 has no definition. Add one sentence of data flow. `slo`, `report` and `gate` should share one evaluation. |
| 4 (test criteria mixed into behavior) | Stands, Major | Move R-009, R-033, R-047, R-055, R-064 and R-065 into a short acceptance list per week. R-032, R-033 and R-035 duplicate other rows, so drop them. |
| 5 to 7 (Nits) | Stand | Naming the canonicalization standard (RFC 8785) in one clause settles R-005. |

## Evaluation of the remediations the earlier stages suggest

| Suggested addition | Source | Zoom Out call | Severity if left out |
| --- | --- | --- | --- |
| Alternatives Considered section | Critic B1 | Reject. Retrospective justification of choices design.md already owns; it would duplicate or fork the design record. One line of "see design.md section X" per decision is enough. | Nit |
| Risks and Mitigations section | Critic B2 | Mostly reject. Memory use, hostile input and clock skew are premature for an offline CLI in weeks 5 to 8. Keep two items as requirements: exit codes (0 pass, 1 SLO breach, 2 input or usage error) and `gate` behavior on empty or unparseable input. | Major for those two, Nit for the rest |
| Problem Statement | Critic B3, Comprehension 1 | Partly accept. Two sentences of why, a link to docs/design.md, and a one line definition of G2. No glossary. | Nit |
| Proposed Architecture | Critic B3 | Accept only as the single pipeline sentence. No diagram or component section. | Nit |
| Glossary of episode, oracle, SLI, SLO and so on | Comprehension 1, 2, 3, 5, 6 | Reject. The spec is a citation layer over design.md. Link it once, pin v2.5, and define only terms ambiguous in the spec itself: "qualifying recovery", "policy deny verdict", "task identity known". | Nit |
| Extra requirements for absent or duplicate `task_success`, malformed adapter records, OTLP version pinning, SLO precedence, default mode | Critic M1, M8, M10, N6, N7 | Reject as a batch. Pick the ones that block correctness (default mode when absent, value for a zero baseline or unknown task, target 1.0 divide by zero) and add them as clauses on existing rows. | Major for those three, Nit for the rest |
| Enumerate fixtures per requirement | Critic M11 | Reject. R-064 plus per week acceptance is enough. Naming fixtures belongs in the plan. | Nit |
| Rename to "Goals & Non-Goals", enumerated goals, version, date, approver | Critic M12, M13, N1 | Reject enumerated goals (the weekly done criteria are the success criteria). Add a version line and a named approver. | Nit |
| More non-goals (live ingestion, storage, migration, concurrency) | Critic N5 | Accept, one line. | Nit |

## Net new findings

| # | Severity | Scope | Finding |
| --- | --- | --- | --- |
| 1 | Major | Whole file | Do not let Stages 6 and 7 turn the spec into a second design.md. Together they ask for about 8 new sections or requirement groups. Cut scope (Stage 5 findings 1 and 2), then add a few clauses, not sections. |
| 2 | Major | R-017, R-018, R-046 | One clause for unrecovered episodes: they report null and are excluded from the distribution. |
| 3 | Major | R-014, R-027, R-028, R-035 | One clause: a run with zero triggered episodes is excluded from recovery SLIs. |
| 4 | Nit | R-042 | "Policy deny verdict" has no source in this spec. Define it in a clause or drop it from R-042. |

## Elephant note (not part of the reviewer's report)
Critic B1 and B2 conflict with the request's constraint of adding no new requirements and covering weeks 5 to 8 only. Cutting scope is a human decision. If the human keeps all four adapters and the burn windows, Stage 5 findings 1 and 2 downgrade to Nit.
