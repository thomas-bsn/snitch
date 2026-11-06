# Ticket #126 : modèles vocaux sans exposer les secrets

Parent : #89. Décision : ADR 0021. Données synthétiques uniquement ; aucune machine réelle, aucun accès Hugging Face réel.

## Réalisé

- Bootstrap Python (`worker/snitch_worker/bootstrap.py`) : codes de cause stables (`hf_access_denied`, `network_unreachable`, `volume_not_writable`, `hf_cache_not_writable`, `gpu_unavailable`, `unknown`), identiques à ceux du contrat d'événements (#150). Ils sont déduits du type de l'erreur et du chemin concerné (volume de modèles ou cache HF/Xet), jamais de son texte. `run_bootstrap(..., report)` renseigne `cause_code`.
- Entrées du lanceur : `python -m snitch_worker.bootstrap check` (code 0 si le manifeste et les fichiers sont valides) et `provision` (téléchargement explicite ; n'imprime que `cause=<code>` sur stdout).
- Reprise : le téléchargement écrit en place dans le volume persistant (`local_dir`) avec le cache HF/Xet du volume ; un échec ne laisse jamais de manifeste, donc jamais de « modèles présents ».
- Lanceur : sans GPU, aucune étape vocale. Modèles déjà valides : ni question ni jeton, mais le test d'inférence synthétique est relancé. Sinon, et seulement avec un terminal : choix y/N, confirmation explicite des conditions (saisie du mot `accept`), jeton saisi masqué. Le jeton ne passe que par un fichier d'environnement privé (0600) supprimé ensuite, jamais en argument, journal, événement ni `worker.env`. Le code de cause lu en sortie est normalisé au vocabulaire fermé (`unknown` sinon). Un échec vocal n'arrête jamais l'installation ni la transcription.
- Voix prête : `voice-test` réussi n'est émis qu'après l'inférence synthétique ; des modèles téléchargés seuls ne suffisent pas.

## Preuves

- `cd worker && uv run --frozen python -m unittest discover -s tests` : 90 tests verts (opt-in, jeton absent ou invalide, codes de cause, cache contre volume, GPU absent, reprise d'un téléchargement partiel, réutilisation du manifeste, rédaction du secret, CLI `check`/`provision`).
- `cd app && go test ./internal/httpapi -run TestInstaller` : le chemin vocal du lanceur s'exécute désormais sous un vrai pseudo-terminal (pty Python) avec faux `docker`, `curl`, `nvidia-smi` : parcours nominal (jeton jamais affiché ni dans la ligne de commande, `worker.env`, événements ; fichier temporaire supprimé), cinq codes de cause d'échec, sortie inattendue ramenée à `unknown`, refus sans `accept`, refus du choix, réutilisation des modèles, échec d'inférence = voix non prête. Les tests ignorés si `python3` est absent.
- `go test ./internal/httpapi ./internal/workers`, `gofmt -l app`, `go vet ./...`, `python3 scripts/check_generated.py`, `python3 scripts/check_error_codes.py` : verts.

## Écarts et limites

- Aucune migration, aucun changement de contrat : le vocabulaire des causes était déjà au contrat.
- Les vrais téléchargements Hugging Face, l'inférence pyannote et le GPU ne sont pas exercés (faux `docker`) ; validation sur la machine neuve de la démonstration P0.
- `gpu_unavailable` côté Python n'est émis que si `SNITCH_VOICE_REQUIRE_GPU=1` (posé par le lanceur) ; la détection est la présence de `/dev/nvidiactl`.
- Pas de test web ni Playwright : aucun changement d'interface (le gestionnaire de modèles complet relève de #128).
