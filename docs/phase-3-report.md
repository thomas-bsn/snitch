# Bilan de phase 3 — état de l’implémentation

La phase 3 est implémentée sur `integration/phase-3` pour les parcours livrés dans les issues #20 à #25. Elle n’est pas déclarée terminée : la revue de spec a repéré des exigences explicites qui restent à réaliser, listées ci-dessous. Aucun clip de la vraie bibliothèque n’a été traité, aucun modèle définitif téléchargé et aucun worker déployé.

## Réalisé

- Le worker télécharge le média uniquement sous lease valide, sélectionne une piste audio, prépare un WAV mono 16 kHz et appelle un fournisseur configurable compatible OpenAI. Les mots et segments horodatés, la piste et les métadonnées du fournisseur sont enregistrés ; un résultat vide est valide.
- L’administrateur configure une règle de sélection par source, un repli explicite et des exceptions par clip. Les clips multipistes ambigus restent sans tâche tant qu’une piste n’est pas choisie.
- Le vocabulaire de transcription gère termes, variantes, graphies préférées et activation. Une version du vocabulaire est figée dans l’entrée de chaque tâche.
- La recherche exacte, française plein texte et approximative utilise PostgreSQL, avec filtres, compteurs, pagination, permissions private/shared, extraits et lecture au timestamp.
- La bibliothèque permet de prévisualiser puis confirmer un lot filtré. Les clips sans original lisible ou piste résolue sont exclus ; les tâches restent soumises à la file durable existante.
- Contrat OpenAPI, clients Go/TypeScript/Python, migrations, documentation worker et interface ont été mis à jour.

## Vérifications

- `RECEIPTS_E2E=1 bash scripts/test.sh` réussit : `go test -race` avec PostgreSQL/pgvector via Testcontainers et les 15 suites Playwright, données synthétiques uniquement.
- `go test ./... -count=1`, `go vet ./...`, tests Python du worker (10), build web, génération reproductible, validation Compose, format Go et `git diff --check` réussissent.

## Restant et écarts à la spec

- VAD configurable et versionné, avec conservation des offsets temporels lorsque les silences sont réduits : non implémenté. La préparation actuelle ne fait que sélectionner et convertir la piste.
- L’identifiant d’entrée ne comprend pas la version de préparation ni le fournisseur/modèle du worker. Le modèle est configuré côté worker et annoncé dans le résultat, mais son changement ne rend pas automatiquement l’ancien résultat obsolète.
- La recherche « plein texte » actuelle s’appuie sur la normalisation et des correspondances SQL de type sous-chaîne ; elle n’utilise pas `tsvector`/`tsquery`.
- Aucun modèle final, seuil de qualité, débit, benchmark PC/NAS, batch réel ou traitement nocturne n’est inclus. Ces éléments restent soumis au benchmark et à l’autorisation de phase 7.

La validation de fin de phase 3 reste à demander après résolution ou décision explicite sur les écarts ci-dessus. Les issues concernées restent ouvertes.
