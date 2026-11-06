# Ticket #150 : suivi des phases d'installation et des capacités d'un Worker

Parent : #89. Décision : ADR 0021. Données synthétiques uniquement ; aucune machine réelle, aucun scan de la vraie bibliothèque.

## Réalisé

- Contrat : `POST /api/worker/install-events` (jeton du Worker), schémas `WorkerInstall*`, champ `install` sur `Worker` et `WorkerDetail`. Un événement = `phase`, `status`, `cause_code`, `occurred_at` d'origine : vocabulaire fermé, aucun texte libre ni secret (champ inconnu refusé, lot invalide refusé en entier, `invalid_install_event`).
- Phases : celles de l'ADR (`prerequisites`, `image`, `permissions`, `gpu`, `voice-models`, `service`) plus `registration` (étape du lanceur) et `voice-test` (inférence synthétique). Codes de cause : les onze codes stables du lanceur (#124), rejoints au contrat et à la contrainte SQL.
- Persistance : migration `00053` (table `worker_install_events`, colonne `workers.first_pulse_at`). Le « premier pouls » est un vrai pouls : les appels authentifiés du lanceur (configuration, événements) mettent à jour `last_seen_at` mais pas `first_pulse_at`.
- États calculés côté serveur : `installing`, `connected` (premier pouls), `voice_ready` (connecté et événement `voice-test` réussi), `failed` (phase et cause, tant qu'aucun pouls), `never_connected` (aucun pouls 24 h après la création). La voix (`ready` / `unavailable` + cause / `pending`) est indépendante de l'installation ; la transcription reste lue dans `transcription_config` (une transcription non configurée n'est pas un échec).
- Dernier événement toujours visible ; `no_news` (10 min sans événement ni pouls en installation ou en échec) est séparé. `received_later` si l'événement est reçu plus de 2 min après son horodatage d'origine.
- Lanceur : `emit` écrit `phase statut cause horodatage` dans `events.queue` (mode 600, 50 lignes au plus) puis tente l'envoi ; hors ligne, la file attend la prochaine exécution authentifiée et part avec les horodatages d'origine. `fail` émet l'échec avec son code stable.
- Voix prête : le lanceur exécute `python -m snitch_worker.voice_selftest` (audio sinusoïdal synthétique, diarisation réelle du fournisseur) après un bootstrap réussi, seulement si un GPU est présent ; `voice-test` réussi n'est émis qu'à ce moment.
- Interface d'administration (EN + FR) : colonne « Installation » dans la liste, section « Installation » dans la fiche (état, échec avec action de reprise, dernier événement et « reçu plus tard », nouvelles, transcription, voix, historique).

## Preuves de test

Exécutés dans cette branche (Colima, Testcontainers PostgreSQL 18) :

- `cd app && go test ./internal/workers` : calcul des états (installation, échec, reprise, connecté, voix refusée, modèles téléchargés mais voix non prête, voix prête après inférence synthétique, jamais connecté, pouls plus fort que 24 h) et validation du vocabulaire.
- `go test -race -tags integration ./... ` avec `SNITCH_E2E=1 SNITCH_E2E_WORKERS=1` : toute la suite verte, dont
  - `worker install progress` : cycle de vie vu par l'administrateur (liste et détail concordants), vocabulaire fermé, lot invalide non stocké, jeton absent refusé, échec avec phase/cause, reprise, événement hors ligne « reçu plus tard », connecté au pouls, voix indisponible (`hf_access_denied`) sans quitter « connecté », `voice_ready` seulement après `voice-test`, aucune trace du jeton en base ;
  - `worker install never connected and silence` : dernier événement visible avec `no_news`, `never_connected` après 25 h, Worker supprimé refusé ;
  - `HTTP contract and permissions` : l'opération `install-events` est couverte et les réponses respectent le schéma ;
  - `TestInstallerEmitsPhaseEventsWithoutSecrets`, `TestInstallerQueuesEventsOfflineAndSendsThemWithOriginalTimestamp`, `TestInstallerReportsFailuresToSnitchOnceReachable` : lanceur rendu et exécuté avec faux `docker`, `curl`, `uname` ;
  - Playwright `admin-workers.spec.ts` (nouveau scénario) : échec, action de reprise, « reçu plus tard », voix indisponible, accessibilité axe.
- `cd worker && uv run --frozen python -m unittest discover -s tests` : 82 tests verts, dont `test_voice_selftest` (inférence synthétique requise, modèles absents ou inférence en échec = non prêt).
- `gofmt -l app`, `go vet ./...`, `python3 scripts/check_generated.py`, `python3 scripts/check_error_codes.py`, `npm --prefix web run lint` et `npm --prefix web run build` (tsc) : verts.

## Écarts et limites

- Le chemin vocal du lanceur (invite TTY, `voice-models`, `voice-test`) n'est pas exécutable dans les tests sans terminal ni GPU : seul son parsing bash est vérifié (le script est analysé en entier à l'exécution). Il sera validé sur la machine neuve de la démonstration P0.
- Le test synthétique ne tourne qu'avec un GPU (la voix y est supposée indisponible sinon, comme dans #124) ; sans lui l'état « voix prête » n'est jamais atteint.
- Les événements ne sont rejoués que par une nouvelle exécution du lanceur ; le Worker en service n'émet pas d'événement.
- Une reprise du lanceur avec modèles déjà présents ne redemande pas de test vocal (le comportement « ne pas redemander le jeton » reste celui de #124/#126).
- Les événements sont conservés par cascade tant que l'identité Worker existe (200 au plus par Worker) ; la suppression d'identité reste une révocation, l'audit minimal relève du ticket de suppression.
