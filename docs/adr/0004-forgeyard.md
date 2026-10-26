# Déploiement Forgeyard et images GHCR

Statut : accepté dans le plan de phase 0.

## Contexte et décision

Le compose racine utilise seulement des images GHCR, volumes nommés et clés Forgeyard admises. Les images amont sont republiées sans modification et conservent leurs licences.

## Alternatives écartées

Les bind mounts et les builds ne sont pas admis par Forgeyard.

## Conséquences

CLIPS_ROOTS reste une configuration future : un administrateur devra valider un montage hôte en lecture seule lorsque Forgeyard le supportera. Ne pas inventer de syntaxe. Les budgets mémoire sont des objectifs non mesurés sur NAS.

## Mise à jour

Mise à jour : les images amont (PostgreSQL/pgvector, Ollama) ne sont plus republiées sur GHCR ; Forgeyard accepte les images Docker Hub (confirmé par l'utilisateur) ; le compose les référence directement, épinglées par digest. Les images de l'application et du worker restent publiées sur GHCR.
