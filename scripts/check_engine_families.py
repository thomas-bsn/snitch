"""Check that the Worker's engine family tables agree with Snitch's.

ADR 0056: the Go table (app/internal/httpapi/voice_engine_families.go) is the single source of the on-demand
engine families. The Worker has to know the same families to install and run them, so it keeps a copy
(worker/snitch_worker/engine_runtime.py). A family added or changed on one side only would be refused by
the other at run time; this compares the two, so the build fails instead.

What is compared: family keys, engine names, pip distributions, kinds, kinds an adapter actually runs, and the
adapters that exist only as a signed Release. Sizes, versions and hardware needs live in Go only.
"""
import re
import sys
from pathlib import Path

root = Path(__file__).resolve().parents[1]
GO_FILE = root / "app/internal/httpapi/voice_engine_families.go"


def go_source() -> str:
    return GO_FILE.read_text()


def _list(block: str, field: str) -> list[str] | None:
    found = re.search(r"\b" + field + r":\s*\[\]string\{([^}]*)\}", block)
    return None if found is None else re.findall(r'"([^"]+)"', found.group(1))


def parse_go(text: str) -> tuple[dict[str, dict], set[str]]:
    table = re.search(r"var engineFamilies = map\[string\]engineFamily\{(.*?)\n\}\n", text, re.S)
    if table is None:
        raise SystemExit("engineFamilies table not found in " + str(GO_FILE))
    families: dict[str, dict] = {}
    for block in re.split(r'\n\t"', "\n" + table.group(1))[1:]:
        key = re.search(r'\bKey:\s*"([^"]+)"', block)
        engine = re.search(r'\bEngine:\s*"([^"]+)"', block)
        distribution = re.search(r'\bDistribution:\s*"([^"]+)"', block)
        kinds = _list(block, "Kinds")
        if not (key and engine and distribution and kinds is not None):
            raise SystemExit("unreadable family in " + str(GO_FILE) + ": " + block[:60])
        families[key.group(1)] = {"engine": engine.group(1), "distribution": distribution.group(1), "kinds": kinds,
                                  "active": _list(block, "ActiveKinds") or kinds}
    signed = re.search(r"var signedOnlyAdapters = map\[string\]bool\{([^}]*)\}", text)
    if signed is None:
        raise SystemExit("signedOnlyAdapters not found in " + str(GO_FILE))
    return families, set(re.findall(r'"([^"]+)":\s*true', signed.group(1)))


def worker_tables() -> tuple[dict[str, dict], set[str]]:
    sys.path.insert(0, str(root / "worker"))
    from snitch_worker import engine_runtime

    families = {}
    for key, family in engine_runtime.FAMILIES.items():
        inactive = engine_runtime._INACTIVE_KINDS.get(key, ())
        families[key] = {"engine": family["engine"], "distribution": family["distribution"], "kinds": list(family["kinds"]),
                         "active": [kind for kind in family["kinds"] if kind not in inactive]}
    return families, set(engine_runtime.SIGNED_ONLY_ADAPTERS)


def differences(text: str, worker: tuple[dict[str, dict], set[str]]) -> list[str]:
    go_families, go_signed = parse_go(text)
    worker_families, worker_signed = worker
    out = []
    for key in sorted(set(go_families) | set(worker_families)):
        if key not in worker_families:
            out.append(f"family {key}: in Snitch's table, missing from the Worker's")
        elif key not in go_families:
            out.append(f"family {key}: in the Worker's table, missing from Snitch's")
        else:
            for field in ("engine", "distribution", "kinds", "active"):
                if go_families[key][field] != worker_families[key][field]:
                    out.append(f"family {key}: {field} is {go_families[key][field]} in Snitch, {worker_families[key][field]} in the Worker")
    for adapter in sorted(go_signed ^ worker_signed):
        out.append(f"signed-only adapter {adapter}: only in " + ("Snitch" if adapter in go_signed else "the Worker"))
    return out


def main() -> int:
    problems = differences(go_source(), worker_tables())
    for problem in problems:
        print(problem, file=sys.stderr)
    if problems:
        print("Engine family tables differ: update both (Go is the source, ADR 0056).", file=sys.stderr)
        return 1
    print("Engine family tables agree")
    return 0


if __name__ == "__main__":
    sys.exit(main())
