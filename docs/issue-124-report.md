# Ticket #124 : installation Linux d'un Worker en une commande

Parent : #89. Décision : ADR 0021. Données synthétiques uniquement ; aucune machine réelle.

## Réalisé

- Invitation d'installation à usage unique (migration `00052_worker_install_invites_single_use.sql`) : consommée dès la demande acceptée de téléchargement du lanceur ; un rejeu répond 410 `install_invite_used` sans jamais renvoyer le jeton. Téléchargements concurrents : un seul réussit.
- La commande copiée ne contient pas le jeton du Worker (scellé côté serveur, livré par le lanceur). Le lanceur local conserve le jeton durable en mode 600 et reprend avec lui, en relisant la configuration courante de Snitch.
- Refus de générer une commande pour une adresse `http://` sauf choix explicite (`allow_insecure_install`) ; la réponse porte `install_command_unavailable`.
- Erreurs par étape (nom de l'étape, code de cause stable, action de reprise) sans secret. Transcription non configurée : l'installation et la connexion aboutissent.
- Reprise idempotente ; échec de démarrage du service : restauration de la version précédente.
- Jeton perdu : nouvelle invitation et nouvelle identité (voir « Correctif de revue » ci-dessous).

## Preuves

Dernière exécution complète (branche `review-fixes-89`) : voir le résultat en fin de document.

- `app/internal/httpapi/worker_install_invite_test.go` : invitation unique, rejeu, suppression d'une identité jamais connectée sans réactivation, téléchargements concurrents, refus `http://`, jeton perdu.
- `app/internal/httpapi/worker_install_launcher_test.go` (faux `docker`, `curl`, `nvidia-smi` dans le `PATH`) : `TestInstallerFirstRunAndResumeWithFakeDocker`, `TestInstallerUnconfiguredTranscriptionStillInstalls`, `TestInstallerFailuresNameStepCauseAndRecovery`, `TestInstallerFailedServiceStartRestoresPreviousVersion`.

## Correctif de revue (US6)

Un jeton perdu est irrécupérable : `POST /api/admin/workers/{id}/token` révoque l'identité (tâches en cours remises en file, invitations non consommées supprimées, audit `token_renewed`) et crée en une transaction une nouvelle identité de même nom et de même concurrence, avec sa commande à usage unique. L'id de la réponse est celui de la nouvelle identité. L'interface propose « Jeton perdu : remplacer ce Worker… » et affiche la commande et un lien vers le nouveau Worker. Tests : ancienne identité refusée, ancienne invitation non consommée inutilisable, nouvelle invitation à usage unique, double renouvellement refusé (404). L'option HTTP non chiffré est désormais une case à cocher visible, avec avertissement, à la création et au remplacement ; son comportement serveur est inchangé.

## Écarts et limites

- Aucune machine neuve réelle : démonstration P0 non faite (faux `docker`). Le lanceur n'a jamais tiré l'image publiée réelle.
- L'image par défaut reste `ghcr.io/thomas-bsn/snitch-worker:latest` alors que l'ADR demande une image épinglée par version : décision utilisateur en attente.
- Le jeton du Worker reste en clair dans `worker.env` (mode 600), conséquence acceptée par l'ADR.

## Résultat de la dernière exécution (branche review-fixes-89)

`SNITCH_E2E=1 SNITCH_E2E_WORKERS=1 bash scripts/test.sh` : code 0, `ok snitch/internal/httpapi` (282 s) et les huit autres paquets Go ; Playwright `admin-workers.spec.ts` : 6 passés. `worker` : 90 tests unittest verts. `gofmt -l app` vide, `go vet ./...`, `scripts/check_generated.py`, `scripts/check_error_codes.py` (166 codes) et `npm run lint` verts, `npm run build` frais réussi. Aucun test sur machine réelle.
