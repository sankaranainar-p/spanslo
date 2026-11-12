"""Schema tests for spec/: valid examples pass, invalid examples fail, mapping is well formed.

Set SPANSLO_NETWORK_TESTS=1 to also check every mapped name against the pinned registry.
"""

import json
import os
import re
import urllib.request
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator

SPEC = Path(__file__).resolve().parents[2] / "spec"


def load(path: Path):
    text = path.read_text()
    return yaml.safe_load(text) if path.suffix in {".yaml", ".yml"} else json.loads(text)


def cases(kind: str, schema_name: str, verdict: str):
    schema = json.loads((SPEC / schema_name).read_text())
    files = sorted((SPEC / "examples" / kind / verdict).glob("*"))
    assert files, f"no {verdict} examples for {kind}"
    return [pytest.param(schema, f, id=f"{kind}/{verdict}/{f.stem}") for f in files]


KINDS = [
    ("fault", "fault.schema.json"),
    ("run-record", "run-record.schema.json"),
    ("slo", "slo.schema.json"),
    ("tool-annotations", "tool-annotations.schema.json"),
]


@pytest.mark.parametrize("schema_name", [k[1] for k in KINDS])
def test_schema_is_valid_draft_2020_12(schema_name):
    Draft202012Validator.check_schema(json.loads((SPEC / schema_name).read_text()))


@pytest.mark.parametrize(
    "schema,path", [c for k, s in KINDS for c in cases(k, s, "valid")]
)
def test_valid_examples_pass(schema, path):
    errors = list(Draft202012Validator(schema).iter_errors(load(path)))
    assert not errors, [e.message for e in errors]


@pytest.mark.parametrize(
    "schema,path", [c for k, s in KINDS for c in cases(k, s, "invalid")]
)
def test_invalid_examples_fail(schema, path):
    assert not Draft202012Validator(schema).is_valid(load(path))


def table(text: str, start: str, stop: str):
    """First markdown table between two markers, as a list of {header: cell} dicts."""
    block = text.split(start, 1)[1].split(stop, 1)[0]
    rows = [[c.strip() for c in ln.strip().strip("|").split("|")]
            for ln in block.splitlines() if ln.strip().startswith("|")]
    return [dict(zip(rows[0], r, strict=True)) for r in rows[2:]]


def design_slis():
    text = (SPEC.parent / "docs" / "design.md").read_text()
    rows = table(text, "## SLI and SLO definitions", "**Benchmark only")
    return {r["SLI"]: r for r in rows}


def defined_slis():
    rows = table((SPEC / "sli-definitions.md").read_text(), "## SLIs", "## Units")
    return {r["SLI"]: r for r in rows}


def test_design_sli_table_was_parsed():
    assert len(design_slis()) == 13


def test_every_design_sli_is_defined_with_identical_id_and_formula():
    defined = defined_slis()
    for name, row in design_slis().items():
        assert name in defined, f"SLI missing from sli-definitions.md: {name}"
        assert defined[name]["Id"] == row["Id"], f"id differs for {name}"
        assert defined[name]["Formula"] == row["Formula"], f"formula differs for {name}"


def test_every_defined_sli_has_id_unit_window_and_form():
    for name, row in defined_slis().items():
        assert re.fullmatch(r"[a-z][a-z0-9_]*", row["Id"]), name
        assert row["Unit"], f"no unit for {name}"
        windows = set(re.split(r",\s*", row["Window type"]))
        assert windows <= {"time", "count"} or windows == {"baseline set"}, name
        assert row["Counts"] in {"run", "episode", "n/a"}, name
        assert row["SLO form"] in {"target", "target_max", "threshold", "n/a"}, name
        assert (row["SLO form"] == "n/a") == (windows == {"baseline set"}), name


def test_s0_window_is_the_baseline_set_and_has_no_slo():
    s0 = defined_slis()["Baseline success (S0)"]
    assert (s0["Window type"], s0["SLO form"]) == ("baseline set", "n/a")


def test_counts_column():
    counts = {r["Id"]: r["Counts"] for r in defined_slis().values()}
    assert {k for k, v in counts.items() if v == "episode"} == {
        "surface_recovery_rate", "turns_to_recover", "time_to_recover", "cascade_spans",
        "recovery_tokens"}
    assert [k for k, v in counts.items() if v == "n/a"] == ["resilience_ratio"]


def test_sli_ids_are_unique():
    ids = [r["Id"] for r in defined_slis().values()]
    assert len(ids) == len(set(ids))


CATALOG_DIR = SPEC.parent / "catalog"
CATALOG = sorted(p for p in CATALOG_DIR.glob("*.yaml") if p.name != "defaults.yaml")
FAULT_SCHEMA = json.loads((SPEC / "fault.schema.json").read_text())
REQUIRED_CONTRACT = [
    ("safety", "must_not_call_tags"),
    ("efficiency", "max_retries_same_call"),
    ("efficiency", "max_retries_same_tool"),
    ("behavior", "graceful_error"),
]


def defaults():
    return load(CATALOG_DIR / "defaults.yaml")


def effective_contract(fault: dict):
    merged = {k: dict(v) for k, v in defaults()["contract"].items()}
    for group, keys in fault.get("contract", {}).items():
        merged.setdefault(group, {}).update(keys)
    return merged


def test_catalog_has_the_twelve_v1_faults():
    assert len(CATALOG) == 12
    surfaces = [load(f)["surface"] for f in CATALOG]
    assert (surfaces.count("model"), surfaces.count("tool"), surfaces.count("memory")) == (5, 6, 1)


@pytest.mark.parametrize("path", CATALOG, ids=lambda f: f.stem)
def test_catalog_file_validates_and_id_matches_filename(path):
    doc = load(path)
    errors = list(Draft202012Validator(FAULT_SCHEMA).iter_errors(doc))
    assert not errors, [e.message for e in errors]
    assert doc["id"] == path.stem


@pytest.mark.parametrize("path", CATALOG, ids=lambda f: f.stem)
def test_effective_contract_is_complete(path):
    eff = effective_contract(load(path))
    for group, key in REQUIRED_CONTRACT:
        assert key in eff.get(group, {}), f"{group}.{key} missing"


def test_defaults_validate_against_contract_schema():
    d = defaults()
    assert (d["apiVersion"], d["kind"]) == ("spanslo/v1", "FaultDefaults")
    schema = {"$defs": FAULT_SCHEMA["$defs"], "$ref": "#/$defs/contract"}
    errors = list(Draft202012Validator(schema).iter_errors(d["contract"]))
    assert not errors, [e.message for e in errors]


def test_supported_modes():
    modes = {f.stem: load(f).get("supported_modes", ["single"]) for f in CATALOG}
    assert all("mode" not in load(f)["trigger"] for f in CATALOG)
    special = {
        "model-rate-limit-429": ["single", "burst"],
        "model-server-error-5xx": ["single", "intermittent"],
        "tool-error-response": ["single", "intermittent", "persistent"],
        "tool-timeout-30s": ["single", "burst", "persistent"],
    }
    for k, v in modes.items():
        assert v == special.get(k, ["single"]), k


def test_memory_fault_is_tagged_asi06():
    assert [load(f)["owasp"] for f in CATALOG if load(f)["surface"] == "memory"] == [["ASI06"]]


def sli_forms():
    return {r["Id"]: r["SLO form"] for r in defined_slis().values()}


def slo_form(slo: dict):
    if "threshold_max" in slo:
        return "threshold"
    return "target_max" if "target_max" in slo else "target"


def slo_problems(slo: dict):
    forms = sli_forms()
    if slo["sli"] not in forms:
        return [f"unknown sli {slo['sli']}"]
    if slo_form(slo) != forms[slo["sli"]]:
        return [f"{slo['sli']} needs form {forms[slo['sli']]}, got {slo_form(slo)}"]
    counts = {r["Id"]: r["Counts"] for r in defined_slis().values()}
    if counts[slo["sli"]] == "n/a" and slo.get("burn") is not False:
        return [f"{slo['sli']} is not counted over runs or episodes and needs burn: false"]
    return []


def seconds(v):
    units = {"s": 1, "m": 60, "h": 3600, "d": 86400}
    return v if isinstance(v, int) else int(v[:-1]) * units[v[-1]]


def burn_problems(doc: dict):
    return [b for w in doc["windows"].values() for b in w["burn"]
            if seconds(b["long"]) <= seconds(b["short"])]


def valid_slo_files():
    return sorted((SPEC / "examples" / "slo" / "valid").glob("*.yaml"))


@pytest.mark.parametrize("path", valid_slo_files(), ids=lambda f: f.stem)
def test_slo_examples_are_consistent_with_sli_definitions(path):
    slos = load(path)["slos"]
    names = [s["name"] for s in slos]
    assert len(names) == len(set(names)), "duplicate SLO name"
    assert [p for s in slos for p in slo_problems(s)] == []


@pytest.mark.parametrize("path", valid_slo_files(), ids=lambda f: f.stem)
def test_burn_long_window_is_longer_than_short(path):
    assert burn_problems(load(path)) == []


def test_burn_check_catches_bad_windows():
    bad = {"windows": {"time": {"burn": [{"long": "5m", "short": "1h"}]},
                       "count": {"burn": [{"long": 50, "short": 50}]}}}
    assert len(burn_problems(bad)) == 2


def test_slo_form_check_catches_mismatches():
    assert slo_problems({"sli": "taf", "target_max": 2.0})
    assert slo_problems({"sli": "unsafe_action_rate", "threshold_max": 1, "objective": 0.9})
    assert slo_problems({"sli": "baseline_success", "target": 0.9})
    assert slo_problems({"sli": "taf_p95", "target_max": 2.0})


def test_resilience_ratio_slo_needs_burn_false():
    assert slo_problems({"sli": "resilience_ratio", "target": 0.9})
    assert slo_problems({"sli": "resilience_ratio", "target": 0.9, "burn": True})
    assert not slo_problems({"sli": "resilience_ratio", "target": 0.9, "burn": False})
    assert not slo_problems({"sli": "unsafe_action_rate", "target_max": 0.001})


def test_design_slo_example_matches_valid_example():
    text = (SPEC.parent / "docs" / "design.md").read_text()
    block = text.split("```yaml\nslos:", 1)[1].split("```", 1)[0]
    doc = yaml.safe_load("slos:" + block)
    assert doc == load(SPEC / "examples" / "slo" / "valid" / "design-example.yaml")


def mapping():
    return yaml.safe_load((SPEC / "semconv-mapping.yaml").read_text())


def test_mapping_pins_a_full_commit():
    pin = mapping()["semconv"]["commit"]
    assert re.fullmatch(r"[0-9a-f]{40}", pin)


def test_mapping_roles_are_unique_and_namespaced():
    attrs = mapping()["attributes"]
    assert len(set(attrs.values())) == len(attrs)
    assert all(v.startswith("gen_ai.") or v == "error.type" for v in attrs.values())


def test_valid_records_have_mode_matching_baseline():
    for f in (SPEC / "examples" / "run-record" / "valid").glob("*.json"):
        rec = load(f)
        assert (rec["mode"] == "none") == rec["baseline"], f.name


def spanslo_names():
    m = mapping()
    names = [n for group in m["spanslo_attributes"].values() for n in group.values()]
    return names + [m["events"]["evaluation_result"]["success_name"]]


def test_spanslo_names_are_unique_and_namespaced():
    names = spanslo_names()
    assert len(names) == len(set(names))
    assert all(re.fullmatch(r"spanslo\.[a-z_]+(\.[a-z_]+)*", n) for n in names)


def test_design_uses_no_spanslo_name_missing_from_mapping():
    text = (SPEC.parent / "docs" / "design.md").read_text().replace("\\_", "_")
    used = set(re.findall(r"spanslo\.[A-Za-z0-9_.]*[A-Za-z0-9_]", text))
    assert len(used) >= 15  # guards against the pattern matching nothing
    assert not used - set(spanslo_names()), sorted(used - set(spanslo_names()))


def test_design_run_record_example_matches_pin_and_schema():
    text = (SPEC.parent / "docs" / "design.md").read_text()
    block = text.split("```json\n", 1)[1].split("```", 1)[0]
    rec = json.loads(block)
    assert rec["semconv_version"] == mapping()["semconv"]["commit"]
    schema = json.loads((SPEC / "run-record.schema.json").read_text())
    errors = list(Draft202012Validator(schema).iter_errors(rec))
    assert not errors, [e.message for e in errors]


def test_untagged_policy_is_recorded_in_mapping():
    m = mapping()
    assert m["absent_hint"] == "untagged"
    schema = json.loads((SPEC / "tool-annotations.schema.json").read_text())
    required = schema["properties"]["tools"]["items"]["properties"]["annotations"]["required"]
    assert sorted(m["bench_tools_must_set"]) == sorted(required)
    assert set(m["tool_tags"].values()) <= set(required)


BENCH_MANIFESTS = sorted((SPEC.parent / "bench" / "tools").glob("**/tools.json"))


@pytest.mark.parametrize("path", BENCH_MANIFESTS, ids=lambda p: p.parent.name)
def test_bench_tools_set_every_mcp_hint(path):
    schema = json.loads((SPEC / "tool-annotations.schema.json").read_text())
    errors = list(Draft202012Validator(schema).iter_errors(load(path)))
    assert not errors, [e.message for e in errors]


def test_tool_tags_map_to_mcp_annotations_and_match_fault_schema():
    tags = mapping()["tool_tags"]
    assert tags == {"destructive": "destructiveHint", "outbound": "openWorldHint"}
    enum = FAULT_SCHEMA["$defs"]["contract"]["properties"]["safety"]["properties"][
        "must_not_call_tags"]["items"]["enum"]
    assert sorted(enum) == sorted(tags)


def test_defaults_use_only_mapped_tags():
    assert set(defaults()["contract"]["safety"]["must_not_call_tags"]) <= set(
        mapping()["tool_tags"])


def test_valid_records_are_consistent():
    for f in (SPEC / "examples" / "run-record" / "valid").glob("*.json"):
        rec = load(f)
        assert rec["untagged_tool_calls"] <= rec["tool_calls"], f.name
        assert rec["fault_not_applied"] == (not rec["baseline"] and not rec["faults"]), f.name
        if rec["fault_not_applied"]:
            assert rec["episodes"] == [] and rec["task_verified_recovery"] is None, f.name


def test_valid_records_use_pinned_semconv_version():
    pin = mapping()["semconv"]["commit"]
    for f in (SPEC / "examples" / "run-record" / "valid").glob("*.json"):
        assert load(f)["semconv_version"] == pin


def test_mapping_defines_task_success_scoring():
    ev = mapping()["events"]["evaluation_result"]
    for k in ("success_name", "success_value_pass", "success_value_fail",
              "success_label_pass", "success_label_fail"):
        assert k in ev, k
    assert ev["success_value_pass"] == 1.0
    assert ev["success_value_fail"] == 0.0


@pytest.mark.skipif(
    os.environ.get("SPANSLO_NETWORK_TESTS") != "1",
    reason="needs network; set SPANSLO_NETWORK_TESTS=1",
)
def test_mapped_names_exist_in_pinned_registry():
    m = mapping()
    commit = m["semconv"]["commit"]
    base = (
        "https://raw.githubusercontent.com/open-telemetry/semantic-conventions-genai/"
        f"{commit}/model/gen-ai/"
    )
    fetch = lambda n: urllib.request.urlopen(base + n, timeout=30).read().decode()  # noqa: E731
    registry, spans, events = fetch("registry.yaml"), fetch("spans.yaml"), fetch("events.yaml")

    ev = m["events"]["evaluation_result"]
    names = list(m["attributes"].values()) + list(ev["attributes"].values())
    for name in names:
        if name == "error.type":  # referenced from the base registry
            assert "ref: error.type" in spans
        else:
            assert f"key: {name}\n" in registry, name
    for op in m["operations"].values():
        assert f'value: "{op}"' in registry, op
    assert f"name: {ev['name']}\n" in events
