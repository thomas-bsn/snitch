# Bilan de la phase 6 bis

## Livré

La refonte de l’interface et du wizard prévue par la phase 6 bis est intégrée dans la branche de livraison. Les tickets livrés sont #84, #85, #69, #74, #75, #76, #77, #80, #78 et #79 : navigation du wizard, recherche sémantique, corrections de segments, informations des Workers, file et dossiers, transcription et traitement vocal, membres et exports, fermeture de compte, retrait de l’ancienne interface, puis finition responsive et accessibilité.

L’ancienne application `/legacy/` et ses ressources ont été retirées. Les routes historiques renvoient vers l’application actuelle. Les pages principales ont été contrôlées en thème clair et sombre, ainsi qu’à une largeur mobile de 390 px ; Playwright axe-core contrôle les routes principales.

## Vérifications

- `npm --prefix web run lint` et `npm --prefix web run build` passent. Vite signale un chunk JavaScript de 711 Ko après minification.
- `bash scripts/generate.sh`, `scripts/check_generated.py`, `scripts/check_error_codes.py`, `go vet -tags integration ./...` et `gofmt` passent.
- `SNITCH_E2E=1 bash scripts/test.sh` passe : les tests Go et les parcours Playwright terminent en 349,9 secondes.
- Les contrôles axe-core ciblés passent : 5 contrôles, sans violation, débordement horizontal ni cible tactile visible sous 44 px.

La suite démarre PostgreSQL/pgvector avec Testcontainers, remet la base à l’état de fixtures avant chaque scénario HTTP, puis lance les fichiers Playwright sélectionnés contre l’API locale et la SPA embarquée. Les parcours utilisent des clips, comptes, Workers et fournisseurs synthétiques. Les E2E retirés étaient des doublons qui visaient l’ancienne interface ; leurs parcours sont exercés par `people.spec.ts`, `people-groups.spec.ts`, `clip-corrections.spec.ts` et `admin-voice.spec.ts`. La réindexation, la recherche après correction et l’export gardent leurs propres scénarios ciblés.

## Restant et écarts

Le ticket #81 reste en attente de validation (`needs-triage`) et n’est pas inclus. Le benchmark réel de phase 7 n’a pas été réalisé. La vérification mobile utilise Chromium à 390 px ; aucun appareil physique ni autre moteur de navigateur n’a été testé. Les données de test sont synthétiques.
