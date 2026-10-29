# Retrouver HTTPS derrière plusieurs mandataires

Statut : accepté pour le correctif d'installation Forgeyard. Précise le mécanisme d'adresse publique de l'ADR 0013 : les métadonnées du navigateur deviennent une source de repli pour le schéma, en plus des en-têtes de mandataire de confiance.

## Contexte

Un accès HTTPS via Caddy puis un Traefik non configuré pour faire confiance à Caddy peut arriver en HTTP dans Snitch avec `X-Forwarded-Proto: http`. L'hôte est correct mais la vérification d'origine refuse le navigateur (`403 origin_rejected`). Imposer `PUBLIC_URL` à chaque installation contredit le démarrage sans configuration. La configuration locale de Forgeyard présente cette chaîne ; la requête navigateur observée contient `Origin` et `Sec-Fetch-Site: same-origin`. Les en-têtes reçus dans le conteneur de production n'ont pas été capturés.

## Décision

Derrière un pair TCP déjà reconnu par `TRUSTED_PROXIES`, Snitch peut relever le schéma HTTP vers HTTPS si `Sec-Fetch-Site` vaut exactement `same-origin` et si l'origine HTTPS a le même hôte et port que ceux déjà établis par la requête. Les ports HTTPS implicite et 443 sont équivalents. L'origine doit être valide, sans identifiants, chemin, requête ni fragment. Pour GET/HEAD sans `Origin`, le `Referer` sert au même contrôle, en autorisant son chemin et sa requête. Un `Origin` présent reste prioritaire, même invalide. Les mutations sans origine restent refusées.

La règle alimente la vérification d'origine, les cookies Secure et les adresses Discord/Worker. Elle ne mémorise rien entre requêtes, ne change jamais l'hôte et ne rétrograde jamais HTTPS. `PUBLIC_URL` fournie reste prioritaire. Les navigateurs contrôlent les en-têtes Fetch Metadata ; un site tiers ne peut pas déclarer sa requête `same-origin`. Comme pour tout contrôle d'origine, un client HTTP hors navigateur peut fabriquer ces en-têtes : cela ne lui donne ni session, ni code d'installation, ni jeton CSRF.

## Alternatives et conséquences

Faire confiance à tous les proxies ou comparer seulement l'hôte affaiblirait le contrôle. Corriger la confiance Caddy → Traefik est possible côté plateforme, mais Snitch peut prendre en charge ce cas sans variable par installation. La déduction dépend des métadonnées du navigateur : pour un client qui les omet et un proxy qui perd le schéma, corriger la chaîne de proxies ou fixer `PUBLIC_URL` reste nécessaire. Aucun traitement de média n'est lié à cette correction.
