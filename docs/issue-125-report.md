# Ticket #125 : support WSL2 et diagnostics réseau du Worker

Parent : #89. Décision : ADR 0021 (WSL2 en P1). Détails des diagnostics : `docs/worker-wsl2.md`. Données synthétiques uniquement.

## Réalisé

- Détection WSL2 par le lanceur (`uname -r` ou `/proc/version`) ; Linux natif (P0) n'exécute aucun diagnostic WSL ; WSL1 refusé (`unsupported_platform`).
- Diagnostics nommés dans le terminal (`wsl-storage`, `wsl-docker`, `snitch-reachability`, `transcription-reachability`, `docker-cuda`), chacun avec un code de cause stable existant et une reprise affichée en anglais et français. Aucun nouveau code ni changement de contrat.
- `/mnt/<lettre>/` refusé pour l'état du lanceur ; indice `host.docker.internal` pour un point d'accès `localhost`.

## Preuves

Résultat de la dernière exécution complète : voir la fin de ce document.

- `app/internal/httpapi/worker_install_launcher_test.go` (faux `docker`, `uname`, `curl`, `nvidia-smi`, faux `/proc/version`, préfixe de montage simulé) : `TestInstallerNativeLinuxRunsNoWSLPreflight`, `TestInstallerWSL2DetectionAndHappyPath`, `TestInstallerWSL2DetectionFromProcVersion`, `TestInstallerWSL1IsRefused`, `TestInstallerWSL2DiagnosticsNamePreflightAndRecovery`, `TestInstallerWSL2LoopbackEndpointGetsAHint`, `TestInstallerWSL2RefusesWindowsMountStorage`.

## Écarts et limites

- **Validation de bout en bout non faite** : aucun hôte Windows/WSL2 n'était disponible. Aucun Worker WSL2 réel n'a été installé, enregistré ni n'a traité de charge synthétique ; le critère « un Worker WSL2 s'enregistre et traite une charge » reste non validé. La procédure à jouer est dans `docs/worker-wsl2.md`.
- CUDA sous WSL2 (`libcuda.so.1` dans un conteneur `--gpus all`) et accessibilité réseau depuis le runtime Docker de Windows ne sont simulés que par des faux exécutables.
- Le chemin vocal réel (modèles Hugging Face, inférence pyannote, GPU) reste non validé, comme pour #126.
- macOS et Docker Desktop hors périmètre.

## Résultat de la dernière exécution (branche review-fixes-89)

`SNITCH_E2E=1 SNITCH_E2E_WORKERS=1 bash scripts/test.sh` : code 0, `ok snitch/internal/httpapi` (282 s) et les huit autres paquets Go ; Playwright `admin-workers.spec.ts` : 6 passés. `worker` : 90 tests unittest verts. `gofmt -l app` vide, `go vet ./...`, `scripts/check_generated.py`, `scripts/check_error_codes.py` (166 codes) et `npm run lint` verts, `npm run build` frais réussi. Aucun test sur machine réelle.
