#!/usr/bin/env bash
# Comparaison visuelle de l'application réelle avec les prototypes (ticket #251).
#
# Construit l'interface, lance l'application sur un jeu de données synthétique (harnais
# d'intégration : Postgres testcontainers + Playwright), capture les mêmes états que
# docs/ui-references/ et compose une image « référence | application » par état.
#
#   scripts/ui-compare.sh [écran]       écran : fiche-video, people, traitement, bibliotheque, import
#
# Sortie (hors dépôt, ignorée par git) : ui-compare-out/current/ et ui-compare-out/diff/.
# Docker : Colima (DOCKER_HOST déduit du contexte docker, sinon à fournir).
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -z "${DOCKER_HOST:-}" ]]; then
  export DOCKER_HOST="$(docker context inspect --format '{{.Endpoints.docker.Host}}')"
fi
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE="${TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE:-/var/run/docker.sock}"
export TESTCONTAINERS_RYUK_DISABLED="${TESTCONTAINERS_RYUK_DISABLED:-true}"

export SNITCH_UI_OUT="${SNITCH_UI_OUT:-$PWD/ui-compare-out}"
export SNITCH_UI_COMPARE=1
export SNITCH_E2E=1
export SNITCH_E2E_UI_COMPARE=1
[[ $# -gt 0 ]] && export SNITCH_UI_ONLY="$1"
rm -rf "$SNITCH_UI_OUT"

(cd web && npm ci --silent && npm run build --silent)
(cd app && go test -tags integration ./internal/httpapi -run 'TestFoundation/browser/ui-compare' -count=1 -timeout 30m -v 2>&1 | grep -v '^=== ' | tail -n 60)

echo "Comparaisons : $SNITCH_UI_OUT/diff/<écran>/<état>-<thème>-<largeur>.png"
