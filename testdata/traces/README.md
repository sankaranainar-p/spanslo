# testdata/traces

Golden trace fixtures: one raw OTLP JSON trace per fault in `catalog/` that the benchmark runs (5 `model-*`, 6 `tool-*` and `memory-irrelevant-retrieval`), 12 in all. Each `<fault>.json` has a hand written `<fault>.expected.json` next to it. An expected record lists only the fields checked by hand, and a run record must match it as a subset.

The fault event in each trace is named `spanslo.fault`, the canonical name (R-057), so the files go straight through `spanslo score`.

Two tests read them: `core/golden_model_test.go` and `core/golden_tool_test.go` (through `ScoreRun`) and `cmd/spanslo/golden_test.go` (through the `score` command). `indexing/` holds the turn indexing fixtures.
