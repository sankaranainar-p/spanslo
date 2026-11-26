# cmd/spanslo

Offline CLI wrapping `core/`. Each command takes flags only; Go accepts `-flag` or `--flag`.

| Command | Flags | What it does |
| --- | --- | --- |
| `score` | `--traces <file> --catalog <dir> --out <file>` | Reads one raw OTLP JSON traces file and writes one run record per run to `<out>` (JSON lines). Invalid runs go to `<out>.invalid.jsonl`. The names file is read from `<catalog>/../spec/semconv-mapping.yaml`. |
| `slo` | `--runs <file> --slo <file>` | Prints, per SLO, whether it is met, the budget remaining and each burn-rate alert. |
| `report` | `--runs <file> --out <file> [--slo <file>]` | Writes spanslo's own JSON report: SLI values, percentiles, unrecovered counts and, with `--slo`, each SLO result. |
| `gate` | `--runs <file> --slo <file>` | Like `slo`, but fails the build: exit 1 when an SLO is not met or a burn alert fires. |

Exit codes: `0` success. `1` only from `gate`, for a breached SLO or firing alert. `2` for a usage error or bad input to the whole command (unreadable or unparseable file, unknown flag, unparseable `ended_at`); `gate` also exits 2 when `<runs>.invalid.jsonl` is missing or non empty. A problem with one run never makes `score` exit 2: that run is listed in the sidecar and `score` exits 0. `slo` and `report` exit 0 even when SLOs are breached.

To score several trace files into one runs file, run `score` once per file and concatenate the outputs and their sidecars.
