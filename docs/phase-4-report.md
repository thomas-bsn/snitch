# Bilan de phase 4 — implémentation sur branche d’intégration

La phase 4 est implémentée sur `integration/phase-4-semantic-search`. Les tickets #27 à #29 couvrent l’indexation, la recherche hybride et la reconstruction d’index. La validation complète reste à exécuter dans un environnement avec Docker ; aucun média réel ni fournisseur externe n’a été utilisé.

## Réalisé

- Les transcriptions terminées alimentent une file durable dans la même transaction que leur achèvement. Quatre consommateurs reprennent les tâches expirées après un arrêt ; les segments sont envoyés par lots au fournisseur compatible embeddings configuré.
- PostgreSQL stocke les vecteurs par version de transcription et profil. Une transcription obsolète ne devient pas la source des résultats courants. Les index HNSW halfvec sont propres à chaque profil.
- L’administrateur peut voir l’état, relancer l’indexation des transcriptions existantes et préparer un profil candidat. L’index actif continue de répondre jusqu’à la fin du candidat ; la bascule est atomique et un candidat en échec ne remplace pas l’ancien index.
- La recherche hybride fusionne par RRF jusqu’à 100 candidats de chaque classement, conserve filtres, permissions, extrait et timestamp, et signale le repli plein texte lorsque les embeddings échouent. La recherche exacte et le plein texte existants restent disponibles.
- Le contrat OpenAPI, les clients Go/TypeScript/Python, la migration, l’interface, le README et les instructions d’accès sont à jour. Les fixtures couvrent indexation, reconstruction réussie/échouée, disponibilité de l’ancien index, fusion et repli.

## Vérifications

- Réussies : `go test -race ./...`, `go vet ./...`, 11 tests worker, `pip check`, génération OpenAPI (`scripts/generate.sh`), vérification des artefacts générés, vérification Compose, lint et build web, et énumération des 15 tests Playwright. Le test d’intégration compilé avec `-tags=integration` passe la compilation.
- Le test d’intégration PostgreSQL/pgvector compile, mais son exécution échoue avant le lancement : Testcontainers ne trouve aucun moteur Docker rootless dans l’environnement courant. Les tests Playwright n’ont pas été exécutés contre l’application.
- Aucun benchmark de qualité, débit, coût ou bibliothèque réelle n’a été réalisé ; ces mesures restent en phase 7.

## À valider

Exécuter `SNITCH_E2E=1 bash scripts/test.sh` dans un environnement avec Docker fonctionnel, puis valider la phase 4 avant de commencer la phase 5. Les tickets restent ouverts jusqu’à cette vérification de bout en bout.
