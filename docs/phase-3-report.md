# Bilan de phase 3 — état de l’implémentation

La phase 3 est implémentée sur `main` pour les parcours livrés dans les issues #20 à #25. La VAD et le versionnement du pipeline sont inclus. Aucun clip de la vraie bibliothèque n’a été traité, aucun modèle définitif téléchargé et aucun worker déployé.

## Réalisé

- Le worker télécharge le média uniquement sous lease valide, sélectionne une piste audio, prépare un WAV mono 16 kHz et appelle un fournisseur configurable compatible OpenAI. Les mots et segments horodatés, la piste et les métadonnées du fournisseur sont enregistrés ; un résultat vide est valide.
- L’administrateur configure une règle de sélection par source, un repli explicite et des exceptions par clip. Les clips multipistes ambigus restent sans tâche tant qu’une piste n’est pas choisie.
- Le vocabulaire de transcription gère termes, variantes, graphies préférées et activation. Une version du vocabulaire est figée dans l’entrée de chaque tâche.
- La recherche exacte, française plein texte et approximative utilise PostgreSQL, avec filtres, compteurs, pagination, permissions private/shared, extraits et lecture au timestamp.
- La bibliothèque permet de prévisualiser puis confirmer un lot filtré. Les clips sans original lisible ou piste résolue sont exclus ; les tâches restent soumises à la file durable existante.
- Contrat OpenAPI, clients Go/TypeScript/Python, migrations, documentation worker et interface ont été mis à jour.

## Vérifications

- `SNITCH_E2E=1 bash scripts/test.sh` réussit : `go test -race` avec PostgreSQL/pgvector via Testcontainers et les 15 suites Playwright, données synthétiques uniquement.
- `go test ./... -count=1`, `go vet ./...`, tests Python du worker (11), build web, génération reproductible, validation Compose, format Go et `git diff --check` réussissent.

## Restant

Aucun critère des tickets #20 à #25 ne reste ouvert. Le choix final du modèle, les mesures de qualité et de débit, le benchmark PC/NAS, le batch réel et tout traitement nocturne restent soumis à la phase 7 et à son autorisation distincte. La recherche plein texte utilise une sous-chaîne normalisée SQL qui tolère accents et casse ; l’approximation `pg_trgm` est également disponible.
