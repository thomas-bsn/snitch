# Contrat HTTP versionné

Statut : accepté dans le plan de phase 0.

## Contexte et décision

contracts/openapi.yaml décrit seulement les routes livrées. La génération des types et clients est déterministe et vérifiée en CI, avec validation des réponses réelles.

## Alternatives écartées

Des contrats écrits séparément par langage finiraient par diverger.

## Conséquences

Toute évolution HTTP modifie d’abord le contrat et régénère les artefacts.
