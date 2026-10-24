"""Remove trailing whitespace from the generated OpenAPI Python client."""
from pathlib import Path


root = Path(__file__).resolve().parents[1] / "worker/receipts_worker/api"
for path in root.rglob("*.py"):
    lines = path.read_bytes().splitlines()
    while lines and not lines[-1].strip():
        lines.pop()
    normalized = b"\n".join(line.rstrip(b" \t") for line in lines) + b"\n"
    path.write_bytes(normalized)
