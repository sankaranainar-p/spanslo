# Stage 5: Zoom Out review of spec.md (lines 1 to 97)

Reviewer: egm-zoom-out. Standard: zoom-out-standard.md. Context files: none (design.md was not supplied, so its citations were taken on trust).

Verdict: pass with Major and Nit findings. No Blockers.

**Context sufficiency:** passes. The Goal says what is built and the Non-goals bound the scope. The "why" is only implied by the design.md citations.

**Reuse check:** `core/` and `core/adapters/` hold only `doc.go` and `README.md`, so there is no Go scoring code to reuse. The spec reuses the existing schemas, `catalog/defaults.yaml` and `spec/semconv-mapping.yaml` (R-002, R-025, R-030, R-031, R-048). The reviewer's file listing was truncated, so it could not confirm that `spec/semconv-mapping.yaml` exists.

## Findings

| # | Severity | Requirements | Finding |
| --- | --- | --- | --- |
| 1 | Major | R-056 to R-059 | Four adapters (bench, tumult, deepagentlabs-chaos, chaos-security) and the run-identity heuristic are speculative while only the bench source exists. Build the canonical fault event (R-057), the bench adapter and the pass through (R-058) now. Move R-059 and the three external adapters to a later gate. |
| 2 | Major | R-052 to R-054 | Wall clock windows over recorded files need a defined "now", which the spec does not give. Multi window burn pairs are a live alerting pattern and live export is week 9 or later. Evaluate over the whole dataset or one count window for G2, or add a requirement that defines the offline reference time. |
| 3 | Major | R-001, R-060 to R-063 | The spec does not say whether the reader or the adapters normalize first, so normalization straddles weeks 5 and 8. State one pipeline (reader, adapters, grouping, scorer) with one owner. `slo`, `report` and `gate` all evaluate SLOs; share the evaluation. The agent-sre format is not defined in the spec. |
| 4 | Major | R-008, R-009, R-033, R-047, R-055, R-064, R-065 | Test and acceptance statements sit in the same table as behavior. Move them to a short acceptance section per week. |
| 5 | Nit | R-005 | SHA-256 over canonicalized JSON is an implementation detail. Name the canonicalization (for example RFC 8785) or point to design.md, and let R-006 carry the determinism rule. |
| 6 | Nit | R-023, R-049, R-053 | Ambiguity: whether the first call counts toward the retry limit; the budget formulas are readable only next to the design.md YAML; which "bad fraction" applies when the SLI counts episodes. |
| 7 | Nit | R-014, R-015, R-032, R-040, R-041 | R-014 and R-032 partly overlap, and R-032 is a test scenario that belongs in an acceptance section. |

## Elephant note (not part of the reviewer's report)
Findings 1 and 2 ask for less than the request that produced spec.md, which named adapters, SLO windows and burn rates as required coverage, and the week 8 row of design.md names four adapters. They are scope calls for the human, not defects to fix silently.
