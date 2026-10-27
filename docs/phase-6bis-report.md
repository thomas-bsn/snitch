# Bilan de la phase 6 bis

## Livré

La refonte de l’interface et du wizard prévue par la phase 6 bis est intégrée dans la branche de livraison. Les tickets livrés sont #84, #85, #69, #74, #75, #76, #77, #80, #78 et #79 : navigation du wizard, recherche sémantique, corrections de segments, informations des Workers, file et dossiers, transcription et traitement vocal, membres et exports, fermeture de compte, retrait de l’ancienne interface, puis finition responsive et accessibilité.

L’ancienne application `/legacy/` et ses ressources ont été retirées. Les routes historiques renvoient vers l’application actuelle. Les pages principales ont été contrôlées en thème clair et sombre, ainsi qu’à une largeur mobile de 390 px ; Playwright axe-core contrôle les routes principales.

## Vérifications

- `npm --prefix web run lint` et `npm --prefix web run build` passent. Vite signale un chunk JavaScript de 711 Ko après minification.
- `bash scripts/generate.sh`, `scripts/check_generated.py`, `scripts/check_error_codes.py`, `go vet -tags integration ./...` et `gofmt` passent.
- Les tests E2E d’accessibilité ciblés passent : 5 contrôles axe, sans violation, débordement horizontal ni cible tactile visible sous 44 px.
- Le test E2E de retranscription corrigé passe seul.
- La suite intégrale `go test -timeout 20m -race -tags integration ./... -count=1 -v` échoue encore dans des specs E2E conservées après le retrait de l’ancienne interface : certaines cherchent des libellés ou ancres de l’ancien écran (par exemple `Actualiser la transcription` ou `#voice-groups`), et `library.spec.ts` attend une suggestion alors que la fixture affiche une voix inconnue. Les tests Go et les autres specs rapportées passent. Ces E2E restants doivent être réalignés avant de considérer la suite complète verte.

## Restant et écarts

Le ticket #81 reste en attente de validation (`needs-triage`) et n’est pas inclus. Le benchmark réel de phase 7 n’a pas été réalisé. La vérification mobile utilise Chromium à 390 px ; aucun appareil physique ni autre moteur de navigateur n’a été testé. Les données de test sont synthétiques.
