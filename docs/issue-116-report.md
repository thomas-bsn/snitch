# Ticket #116 : révoquer un Worker et remettre ses tâches en file

Parent : #89. Décision : ADR 0021, section « Mise en œuvre de la suppression (#116) ». Données synthétiques uniquement.

## Réalisé

- `DELETE /api/admin/workers/{id}` (corps facultatif `{reason}`, 200 caractères au plus), en une transaction : le jeton cesse d'authentifier, les tâches en cours repassent en `pending` avec l'événement `worker_revoked` (tentative interrompue non comptée), l'inventaire, la configuration de transcription, les invitations et les événements d'installation sont supprimés. Reste : nom, `revoked_at`, `revoke_reason` (migration `00054_worker_revocation_audit.sql`).
- Seconde suppression sans effet (premier audit conservé) ; aucune réactivation possible.
- `GET /api/worker/revocation` : seul endroit où un jeton révoqué est accepté, répond `{revoked, revoked_at}` ; renseigne `revocation_acknowledged_at`. Jeton inconnu : `worker_token_invalid`.
- Lanceur : sur 401, interroge ce point ; sur confirmation, arrête le service (`docker rm -f`), affiche en anglais et français qu'il n'a rien effacé, qu'un Worker en marche efface lui-même ses modèles et caches, et que le volume Docker et le dossier restent.
- Worker (#89, révision) : sur 401 `worker_token_invalid`, interroge ce même point ; sur `revoked: true` seulement, ferme son client HTTP, écrit le marqueur `.snitch-worker-revoked` dans `/models`, efface le reste du volume (modèles, manifeste, caches Hugging Face/Xet) et reste inactif sans plus aucun appel. Au démarrage, le marqueur le garde inactif sans client réseau ni téléchargement : pas de boucle de redémarrage, et `--restart unless-stopped` reste en place pour les Workers valides (US9). Plus aucune commande à relancer après une suppression.
- Interface : détail du Worker révoqué, mention « nettoyage local en attente » tant que la machine n'a pas confirmé, motif facultatif. Une fois confirmé : le Worker en marche a effacé ses modèles et caches ; restent sur l'hôte le volume Docker vide, le conteneur inactif et le dossier du lanceur, que le conteneur ne peut pas supprimer.
- Le renouvellement d'un jeton perdu réutilise cette même révocation (#89 revue) : l'ancienne identité reçoit le motif `token_renewed`.

## Preuves

Résultat de la dernière exécution complète : voir la fin de ce document.

- `TestFoundation/worker_revocation` (`app/internal/httpapi/worker_revocation_test.go`) : révocation, tâche remise en file puis prise par un autre Worker, résultats tardifs refusés, confirmation étroite, audit, détails supprimés, Worker hors ligne.
- `TestInstallerStopsServiceOnRevocationConfirmationWithoutErasingLocalData` : faux docker/curl, service arrêté, aucun `volume rm`.
- `worker/tests/test_revocation.py` : révocation confirmée (volume vidé sauf le marqueur, client fermé, un seul pouls, aucun claim) ; 401 non confirmé, `revoked: false` ou panne de la confirmation (rien d'effacé, le Worker réessaie) ; marqueur au démarrage (aucun client réseau, état inactif) ; bootstrap révoqué (aucun téléchargement).
- Couverture du contrat OpenAPI de l'endpoint de suppression (`integration_test.go`).

## Écarts et limites

- Aucun test sur machine réelle : le conteneur d'un Worker hors ligne n'a pas été vu s'arrêter à son retour ; seul le faux `docker` l'atteste.
- Le conteneur d'un Worker révoqué reste présent, inactif, jusqu'à ce que l'utilisateur le supprime ou relance le lanceur ; le volume Docker (vide) et le dossier du lanceur (`worker.env` compris) restent aussi : le conteneur, sans socket Docker, ne peut pas les supprimer.
- Si seul le lanceur confirme (conteneur absent), le volume garde son contenu ; `revocation_acknowledged_at` ne distingue pas les deux cas, l'interface le dit.
- Le nettoyage par le Worker n'est pas encore vu sur machine réelle ; l'image `:latest` publiée depuis `main` ne l'embarque qu'après sa prochaine publication.

## Résultat de la dernière exécution (branche worker-auto-cleanup-89)

`SNITCH_E2E=1 SNITCH_E2E_WORKERS=1 bash scripts/test.sh` : code 0, `ok snitch/internal/httpapi` (256 s) et les huit autres paquets Go ; Playwright : 6 passés. `worker` : 97 tests unittest verts (dont 7 de `test_revocation.py`). `gofmt -l app` vide, `go vet ./...`, `scripts/check_generated.py`, `scripts/check_error_codes.py` (166 codes) et `npm run lint` verts, `npm run build` frais réussi. Aucun test sur machine réelle du nettoyage par le Worker.
