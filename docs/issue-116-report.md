# Ticket #116 : révoquer un Worker et remettre ses tâches en file

Parent : #89. Décision : ADR 0021, section « Mise en œuvre de la suppression (#116) ». Données synthétiques uniquement.

## Réalisé

- `DELETE /api/admin/workers/{id}` (corps facultatif `{reason}`, 200 caractères au plus), en une transaction : le jeton cesse d'authentifier, les tâches en cours repassent en `pending` avec l'événement `worker_revoked` (tentative interrompue non comptée), l'inventaire, la configuration de transcription, les invitations et les événements d'installation sont supprimés. Reste : nom, `revoked_at`, `revoke_reason` (migration `00054_worker_revocation_audit.sql`).
- Seconde suppression sans effet (premier audit conservé) ; aucune réactivation possible.
- `GET /api/worker/revocation` : seul endroit où un jeton révoqué est accepté, répond `{revoked, revoked_at}` ; renseigne `revocation_acknowledged_at`. Jeton inconnu : `worker_token_invalid`.
- Lanceur : sur 401, interroge ce point ; sur confirmation, arrête le service (`docker rm -f`), affiche en anglais et français que volume, modèles et fichiers locaux ne sont pas effacés.
- Interface : détail du Worker révoqué, mention « nettoyage local en attente » tant que le lanceur n'a pas confirmé, motif facultatif.
- Le renouvellement d'un jeton perdu réutilise cette même révocation (#89 revue) : l'ancienne identité reçoit le motif `token_renewed`.

## Preuves

Résultat de la dernière exécution complète : voir la fin de ce document.

- `TestFoundation/worker_revocation` (`app/internal/httpapi/worker_revocation_test.go`) : révocation, tâche remise en file puis prise par un autre Worker, résultats tardifs refusés, confirmation étroite, audit, détails supprimés, Worker hors ligne.
- `TestInstallerStopsServiceOnRevocationConfirmationWithoutErasingLocalData` : faux docker/curl, service arrêté, aucun `volume rm`.
- Couverture du contrat OpenAPI de l'endpoint de suppression (`integration_test.go`).

## Écarts et limites

- Aucun test sur machine réelle : le conteneur d'un Worker hors ligne n'a pas été vu s'arrêter à son retour ; seul le faux `docker` l'atteste.
- Le conteneur déjà lancé d'un Worker révoqué reste actif jusqu'à la prochaine exécution du lanceur et reçoit des 401 sans effet.
- Aucune suppression physique des données locales, volontairement (ADR 0021).

## Résultat de la dernière exécution (branche review-fixes-89)

`SNITCH_E2E=1 SNITCH_E2E_WORKERS=1 bash scripts/test.sh` : code 0, `ok snitch/internal/httpapi` (282 s) et les huit autres paquets Go ; Playwright `admin-workers.spec.ts` : 6 passés. `worker` : 90 tests unittest verts. `gofmt -l app` vide, `go vet ./...`, `scripts/check_generated.py`, `scripts/check_error_codes.py` (166 codes) et `npm run lint` verts, `npm run build` frais réussi. Aucun test sur machine réelle.
