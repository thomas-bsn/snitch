#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
(cd app && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate)
(cd app && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0 --config ../contracts/generators/go.yaml ../contracts/openapi.yaml)
(cd web && npm run generate)
.venv/bin/openapi-python-client generate --path contracts/openapi.yaml --config contracts/generators/python.yaml --output-path worker/receipts_worker/api --overwrite
python3 scripts/normalize_generated_python.py
