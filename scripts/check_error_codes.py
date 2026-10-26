"""Check that every error code the HTTP handlers can return has a message in each language.

The server answers errors with stable codes; the interface translates them (web/src/app/i18n).
This lists the codes written in the Go handlers and compares them with the messages of
the English and French files, so a new code cannot ship without its texts.
"""
import re
import sys
from pathlib import Path

root = Path(__file__).resolve().parents[1]
handlers = sorted((root / "app/internal/httpapi").glob("*.go"))

patterns = [
    re.compile(r'\bfail\(\s*w\s*,\s*[\w.]+\s*,\s*"([a-z][a-z0-9_]*)"'),
    re.compile(r'\boauthFailure\(\s*w\s*,\s*r\s*,\s*[\w.]+\s*,\s*"([a-z][a-z0-9_]*)"'),
    re.compile(r'"error"\s*:\s*"([a-z][a-z0-9_]*)"'),
]
# Codes sent in the address after a redirect rather than in a JSON answer.
redirect_codes = {"discord_denied"}

server_codes: set[str] = set(redirect_codes)
for path in handlers:
    if path.name.endswith("_test.go"):
        continue
    text = path.read_text()
    for pattern in patterns:
        server_codes.update(pattern.findall(text))


def messages(name: str) -> set[str]:
    text = (root / "web/src/app/i18n" / name).read_text()
    return set(re.findall(r"^\s+([a-z][a-z0-9_]*):", text, flags=re.MULTILINE))


english, french = messages("errors.en.ts"), messages("errors.fr.ts")
problems = []
for code in sorted(server_codes - english):
    problems.append(f"no English message for the server code {code}")
for code in sorted(server_codes - french):
    problems.append(f"no French message for the server code {code}")
for code in sorted(english ^ french):
    problems.append(f"{code} exists in only one of the two language files")
if problems:
    print("\n".join(problems))
    sys.exit(1)
print(f"{len(server_codes)} server error codes have an English and a French message")
