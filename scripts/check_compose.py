"""Vérifie le sous-ensemble Forgeyard, sans inventer de montages hôte."""
import json
import subprocess
from pathlib import Path

root = Path(__file__).resolve().parents[1]
# YAML parser supplied by the pinned OpenAPI generator environment.
from ruamel.yaml import YAML

def check(document):
    assert set(document) <= {"services", "volumes"}
    allowed = {"image", "command", "entrypoint", "user", "environment", "volumes", "depends_on", "healthcheck", "restart", "x-forgeyard"}
    exposed = []
    for name, service in document["services"].items():
        assert set(service) <= allowed, (name, set(service) - allowed)
        assert service.get("image", "").startswith(("ghcr.io/", "${"))
        for mount in service.get("volumes", []):
            source, target = mount.split(":")
            assert source in document["volumes"] and target.startswith("/")
        meta = service.get("x-forgeyard", {})
        assert set(meta) <= {"memory", "backup", "expose"}
        if "expose" in meta:
            exposed.append((name, meta["expose"]["port"]))
    assert exposed == [("app", 8080)]
    assert all(v is None or v == {} for v in document["volumes"].values())
    assert document["services"]["app"]["depends_on"]["migrate"]["condition"] == "service_completed_successfully"
    assert document["services"]["migrate"]["restart"] == "no"
    assert document["services"]["postgres"]["x-forgeyard"]["backup"] == {"type": "postgres"}

raw = YAML(typ="safe").load((root / "compose.yaml").read_text())
check(raw)
# Negative fixtures make sure the validator itself rejects common invalid deployments.
from copy import deepcopy
for invalid in ("ports", "networks", "network_mode", "env_file", "container_name", "privileged", "cap_add", "devices", "extends", "build", "tmpfs"):
    doc = deepcopy(raw)
    doc["services"]["app"][invalid] = True
    try:
        check(doc)
    except AssertionError:
        continue
    raise AssertionError(f"Accepted forbidden key: {invalid}")
print("Forgeyard compose: valid; forbidden-key fixtures rejected")
