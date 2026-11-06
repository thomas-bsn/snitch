# Adresse de retour Discord figée au démarrage, `PUBLIC_URL` supprimée

Statut : accepté. Remplace la surcharge `PUBLIC_URL` de l'ADR 0013 et complète l'ADR 0016.

## Contexte

La connexion Discord échouait au callback avec `token_exchange`, HTTP 400, `invalid_grant`. L'adresse de retour (`redirect_uri`) était recalculée par `publicOrigin(r)` au démarrage puis au callback. Derrière Cloudflare → Caddy → Traefik, le démarrage (POST du navigateur, avec `Origin` et `Sec-Fetch-Site`) retrouvait HTTPS, alors que le callback (GET de navigation, sans `Origin`, avec un schéma perdu par un relais) retombait sur HTTP. Discord exige la même valeur aux deux étapes. `PUBLIC_URL` masquait le défaut en court-circuitant la déduction.

## Décision

- L'adresse est calculée une fois, au démarrage, et stockée dans `oauth_states.redirect_uri` avec l'état, le cookie navigateur et la session de liaison. Le callback la lit dans la ligne consommée (`DELETE … RETURNING`) et ne la reconstruit jamais.
- `PUBLIC_URL` est supprimée (configuration, Compose, documentation, tests). Une valeur résiduelle dans l'environnement est ignorée.
- Sans adresse déductible, le démarrage répond `400 public_address_unknown` au lieu de produire une URL invalide.
- La déduction (`publicOrigin`, `trustedPeer`, ADR 0016) est inchangée : en-têtes de mandataire acceptés seulement d'un pair TCP de confiance.
- Migration 00093 : les états en vol (10 minutes de vie) sont supprimés, car ils n'ont pas d'adresse enregistrée ; la colonne est `NOT NULL`.

## Conséquences

Le callback n'a plus besoin d'`Origin`, de `Referer` ni d'en-têtes de mandataire. Pas de nouvelle surface : Discord n'accepte que les adresses enregistrées dans son portail, et l'adresse stockée n'est jamais contrôlée par le callback. Limite : le démarrage doit encore se faire sur une chaîne qui livre le bon schéma (ou des métadonnées Fetch same-origin) ; les installations qui comptaient sur `PUBLIC_URL` pour une chaîne qui perd HTTPS doivent corriger le mandataire (confiance, `X-Forwarded-Proto`). Les adresses du fichier de déploiement Worker restent déduites de la requête de création.
