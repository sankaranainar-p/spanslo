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


KINDS = [("fault", "fault.schema.json"), ("run-record", "run-record.schema.json")]


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


def mapping():
    return yaml.safe_load((SPEC / "semconv-mapping.yaml").read_text())


def test_mapping_pins_a_full_commit():
    pin = mapping()["semconv"]["commit"]
    assert re.fullmatch(r"[0-9a-f]{40}", pin)


def test_mapping_roles_are_unique_and_namespaced():
    attrs = mapping()["attributes"]
    assert len(set(attrs.values())) == len(attrs)
    assert all(v.startswith("gen_ai.") or v == "error.type" for v in attrs.values())


def test_valid_records_use_pinned_semconv_version():
    pin = mapping()["semconv"]["commit"]
    for f in (SPEC / "examples" / "run-record" / "valid").glob("*.json"):
        assert load(f)["semconv_version"] == pin


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
