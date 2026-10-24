# Monolithe modulaire et SPA

Statut : accepté dans le plan de phase 0.

## Contexte et décision

Le NAS dispose de peu de mémoire. Un binaire Go sert une SPA React embarquée ; le worker Python reste sur le PC GPU.

## Alternatives écartées

Next.js et des microservices permanents multiplieraient les processus et les déploiements.

## Conséquences

Aucun Node en production. Les modules communiquent dans le processus ; le worker ne connaît que HTTPS.
