# Bilan de phase 1

La phase 1 est implémentée sur la branche locale `integration/phase-1`. La validation utilisateur de fin de phase est attendue avant d’entamer la phase 2. Aucun média réel, NAS, batch de production ou déploiement n’a été utilisé.

## Réalisé

- Sources de dossiers enregistrables sous les racines autorisées, montées en lecture seule ; surveillance filesystem et réconciliation périodique, stabilité, exclusions configurables et états d’ingestion visibles.
- Extraction des métadonnées par ffprobe, hash SHA-256, identité unique de contenu et provenances multiples. Une source inaccessible ou une réconciliation incomplète ne marque pas les fichiers absents ; les historiques sont conservés et les disparitions massives affichent un avertissement.
- Upload reprenable par morceaux, limite de taille, annulation et reprise. Le stockage média utilise un volume distinct ; un contenu n’est publié dans la bibliothèque qu’après complétion et vérification.
- Bibliothèque paginée avec filtres date/source/propriétaire/état, détail, provenance et auteur(s) d’upload. Les visibilités privées/partagées sont vérifiées côté serveur. La lecture authentifiée accepte les requêtes HTTP Range.
- Contrat OpenAPI et clients générés Go, TypeScript et Python ; migrations versionnées, guides actualisés et décisions documentées dans ADR 0009 et 0010.

## Vérifications

- `RECEIPTS_E2E=1 bash scripts/test.sh` : tests Go avec race detector, Testcontainers PostgreSQL/pgvector, tests API et huit suites Playwright réussis sur l’état final.
- `go vet ./...`, `npm run lint`, `npm run build`, `scripts/check_generated.py`, `scripts/check_compose.py` et `git diff --check` réussis.
- Build local de l’image app et smoke avec ffprobe réel : vidéo synthétique acceptée, fixture audio seule rejetée. Les tests utilisent uniquement des médias synthétiques et des répertoires temporaires.
- La revue finale standards/spec a été exécutée ; elle a conduit à exposer les auteurs d’upload après dédoublonnage et à corriger le nombre de suites navigateur dans la documentation.

## Restant et écarts

- Forgeyard ne permet pas actuellement les montages hôte nécessaires aux sources de dossiers sur NAS. Elles ne fonctionnent que si un chemin est réellement rendu accessible au conteneur en lecture seule. L’upload reste utilisable.
- Le volume média des uploads n’est pas inclus dans les sauvegardes PostgreSQL ; les sauvegardes/restore complètes restent à traiter et vérifier en phase 7.
- L’image app embarque ffprobe depuis le paquet Alpine `ffmpeg` GPL/LGPL. Les textes de licence et sources sont documentés ; avant toute publication d’image, archiver les sources correspondantes exactes et les informations de build, comme décrit dans ADR 0009.
- La CI distante, publication d’images, déploiement NAS, montage de vrais dossiers, benchmark réel, traitement du batch 1 To, worker et étapes IA ne sont pas exécutés ou implémentés. Aucun scan de vraie bibliothèque n’a été lancé.

La prochaine étape est la validation utilisateur de la phase 1. Aucune phase 2 ne commence avant cette validation.
