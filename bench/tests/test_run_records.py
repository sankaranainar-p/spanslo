"""R-031: every run record `spanslo score` emits validates against the run record schema."""

import json
import re
import subprocess
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator, ValidationError

ROOT = Path(__file__).resolve().parents[2]
SCHEMA = json.loads((ROOT / "spec" / "run-record.schema.json").read_text())
NAMES = yaml.safe_load((ROOT / "spec" / "semconv-mapping.yaml").read_text())
VALIDATOR = Draft202012Validator(SCHEMA)
ATTR = NAMES["attributes"]
OP = ATTR["operation_name"]
OPS = NAMES["operations"]
ID = NAMES["spanslo_attributes"]["run_identity"]
ER = NAMES["events"]["evaluation_result"]
BASE_MS = 1_700_000_000_000
# ponytail: golden fixtures from T-017 are picked up by name; expected records are skipped.
GOLDEN = sorted(
    p
    for p in (ROOT / "testdata" / "traces").glob("*.json*")
    if "expected" not in p.name and not p.name.endswith(".invalid.jsonl")
)


def val(v):
    if isinstance(v, bool):
        return {"boolValue": v}
    if isinstance(v, int):
        return {"intValue": str(v)}
    if isinstance(v, float):
        return {"doubleValue": v}
    return {"stringValue": v}


def attrs(d):
    return [{"key": k, "value": val(v)} for k, v in d.items()]


def span(run, sid, start_ms, end_ms, a, events=(), error=False):
    s = {
        "traceId": f"t-{run}",
        "spanId": f"{run}-{sid}",
        "name": "s",
        "startTimeUnixNano": str((BASE_MS + start_ms) * 1_000_000),
        "endTimeUnixNano": str((BASE_MS + end_ms) * 1_000_000),
        "attributes": attrs(a),
        "events": list(events),
    }
    if error:
        s["status"] = {"code": 2}
    return s


def success_event():
    names = ER["attributes"]
    a = {names["evaluation_name"]: ER["success_name"], names["score_value"]: 1.0}
    return {"name": ER["name"], "timeUnixNano": "1", "attributes": attrs(a)}


def fault_event_name():
    # ponytail: the event name is a Go constant, not in the mapping; read it from core/.
    for p in (ROOT / "core").glob("*.go"):
        m = re.search(r'FaultEventName\s*=\s*"([^"]+)"', p.read_text())
        if m:
            return m.group(1)
    raise AssertionError("FaultEventName not found in core/")


def fault_event():
    a = {
        "spanslo.fault.id": "tool-error-response",
        "spanslo.fault.surface": "tool",
        "spanslo.fault.target": "search",
        "spanslo.fault.turn": 1,
        "spanslo.fault.call_index": 0,
        "spanslo.fault.mode": "single",
        "spanslo.fault.triggered": True,
    }
    return {"name": fault_event_name(), "timeUnixNano": "1", "attributes": attrs(a)}


def trace(kind):
    """One run: baseline, faulted (error then recovery) or not_applied (no fault event)."""
    root = {OP: OPS["run"]}
    root.update({ID["baseline"]: True} if kind == "baseline" else {ID["mode"]: "single"})
    res = {
        ID["run_id"]: kind,
        ID["experiment_id"]: "e",
        ID["framework"]: "fw",
        ID["model"]: "m",
        ID["task_id"]: "task1",
    }
    chat = {OP: OPS["model_call"], ATTR["input_tokens"]: 10, ATTR["output_tokens"]: 5}
    tool = {
        OP: OPS["tool_call"],
        ATTR["tool_name"]: "search",
        ATTR["tool_call_arguments"]: '{"q":"x"}',
    }
    spans = [
        span(kind, "r", 0, 10_000, root, [success_event()]),
        span(kind, "c", 1_000, 1_100, chat),
    ]
    if kind == "faulted":
        spans.append(span(kind, "t1", 1_200, 1_300, tool, [fault_event()], error=True))
        spans.append(span(kind, "t2", 2_000, 2_100, tool))
    else:
        spans.append(span(kind, "t1", 1_200, 1_300, tool))
    resource = {"attributes": attrs(res)}
    return {"resourceSpans": [{"resource": resource, "scopeSpans": [{"spans": spans}]}]}


@pytest.fixture(scope="session")
def spanslo(tmp_path_factory):
    exe = tmp_path_factory.mktemp("bin") / "spanslo"
    subprocess.run(["go", "build", "-o", str(exe), "./cmd/spanslo"], cwd=ROOT, check=True)
    return exe


def score(exe, traces, out):
    cmd = [str(exe), "score", "--traces", str(traces), "--out", str(out)]
    cmd += ["--catalog", str(ROOT / "catalog")]
    subprocess.run(cmd, check=True)
    return [json.loads(line) for line in out.read_text().splitlines() if line]


@pytest.fixture
def inline(spanslo, tmp_path):
    src = tmp_path / "traces.jsonl"
    kinds = ("baseline", "faulted", "not_applied")
    src.write_text("".join(json.dumps(trace(k)) + "\n" for k in kinds))
    out = tmp_path / "runs.jsonl"
    records = score(spanslo, src, out)
    return records, out


def test_inline_records_validate(inline):
    records, out = inline
    assert {r["run_id"] for r in records} == {"baseline", "faulted", "not_applied"}
    assert out.with_name(out.name + ".invalid.jsonl").read_text() == ""
    faulted = next(r for r in records if r["run_id"] == "faulted")
    assert len(faulted["episodes"]) == 1 and faulted["episodes"][0]["surface_recovered"]
    for record in records:
        VALIDATOR.validate(record)


def test_missing_required_field_fails(inline):
    for record in inline[0]:
        VALIDATOR.validate(record)
        for field in SCHEMA["required"]:
            broken = {k: v for k, v in record.items() if k != field}
            with pytest.raises(ValidationError):
                VALIDATOR.validate(broken)


@pytest.mark.parametrize("fixture", GOLDEN, ids=lambda p: p.name)
def test_golden_records_validate(spanslo, fixture, tmp_path):
    for record in score(spanslo, fixture, tmp_path / "runs.jsonl"):
        VALIDATOR.validate(record)
