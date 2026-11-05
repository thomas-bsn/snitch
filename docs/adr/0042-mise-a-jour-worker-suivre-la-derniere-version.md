# 0042 — Mise à jour du Worker : suivre la dernière version stable

Statut : accepté. Complète l'[ADR 0036](0036-mise-a-jour-automatique-du-worker.md) : la cible n'est plus seulement une version choisie, elle peut suivre les publications. Ne change ni l'agent local, ni l'épinglage par digest, ni le rollback.

## Contexte

L'ADR 0036 écartait « suivre `latest` » : le tag Docker `latest` est réécrit à chaque publication, sans contrôle ni version lisible. Le besoin réel est autre : ne plus avoir à choisir à la main chaque `vX.Y.Z`. Or une étiquette `vX.Y.Z` n'est créée par la CI qu'à la promotion, une fois la porte `check` verte (ADR 0040) : « publiée sous `vX.Y.Z` » signifie déjà « validée par la CI ».

## Décision

- **Trois stratégies** (`worker_update_target.mode`) : `disabled`, `pinned` (version choisie, comportement de l'ADR 0036), `latest` (suivi). Les configurations existantes avec une version deviennent `pinned`, celles sans version `disabled`.
- **Version effective** : la colonne `version` reste la cible effective dans les trois modes. En `latest`, c'est la plus haute version déjà vue ; elle ne fait que monter. La version installée reste celle que chaque Worker rapporte.
- **Résolution paresseuse, sans tâche de fond ni écriture inutile** : en `latest`, `GET /api/worker/update` (le polling existant des Workers) et la vue d'administration lisent le catalogue en cache (5 min, ADR 0036) et prennent la plus haute version SemVer `vX.Y.Z` (`workerupdate.Newest`, indépendante de l'ordre du registre). La base n'est écrite que si cette version est strictement supérieure à celle stockée, par un `UPDATE … WHERE version IS NOT DISTINCT FROM <lue>` : concurrence et plusieurs Workers sont idempotents. Aucune requête GHCR de plus que le catalogue actuel. Pas de planificateur côté serveur : sans Worker ni administrateur, personne n'a besoin de la valeur, et il n'y a aucun état d'un second processus à garder cohérent.
- **Jamais de rétrogradation automatique** : le passage à `latest` conserve la version courante comme plancher ; un tag retiré du registre ne baisse pas la cible. Seul un choix `pinned` explicite revient en arrière.
- **GHCR indisponible** : la dernière cible valide est conservée. Les échecs sont espacés d'une minute (le catalogue ne réessaie plus à chaque polling), et l'interface affiche la date de la dernière interrogation (mémoire du processus) et l'échec.
- **Catalogue paginé** : GHCR pagine la liste des étiquettes, et chaque publication ajoute `sha-*` et `vX.Y.Z` ; la dernière version pouvait se trouver hors de la première page. Le catalogue suit l'en-tête `Link` (même hôte uniquement, 20 pages au plus ; un dépassement ou un hôte étranger est un échec, jamais une liste partielle).
- **Pas de boucle de redémarrages** : après un `unhealthy` ou un digest inconnu, l'agent mémorise la version (`update.failed`) et ne la retente pas pendant 24 h, sauf si la cible devient une autre version. Sans cela, une cible `latest` défaillante serait tirée et relancée toutes les 10 minutes.
- **Drainage** : `GET /api/worker/update` renvoie `idle` (aucun job sous bail pour ce Worker). L'agent ne remplace pas un conteneur occupé ; il attend au plus 2 heures (`update.deferred`), puis met à jour quand même : un job interrompu est repris par Snitch à l'expiration du bail.
- **Image** : toujours `<dépôt>:<vX.Y.Z>` résolue, jamais le tag mutable `latest` ; l'agent démarre l'image par digest.
- **Contrat** : `PUT /api/admin/worker-updates` accepte `mode` ; sans `mode`, il est déduit de `version` (compatibilité). `latest` ignore `version`.

## Alternatives écartées

- **Tâche de fond côté serveur** qui interroge GHCR et écrit la cible : un second chemin de résolution, de l'état à protéger, et des écritures à chaque tour sans changement.
- **Résolution unique au clic** : n'est pas un suivi, contredit le besoin.
- **Résolution par l'agent local** (il interroge GHCR lui-même) : N Workers contre un registre, aucune vue centrale ni plancher, et une logique de choix de version dans un script shell.
- **Tag `latest` Docker** : voir l'ADR 0036.

## Conséquences

- Le délai entre une publication et la mise à jour est au plus de 5 minutes (cache) + 10 minutes (cron de l'agent) + l'attente d'un job en cours.
- En `latest`, une version publiée mais défectueuse atteint tout le parc ; le rollback par Worker et la mise de côté de 24 h limitent les dégâts à un redémarrage par Worker, pas à une boucle. Choisir `pinned` reste le moyen de figer.
- La date de dernière vérification est celle du processus serveur : elle est vide après un redémarrage, jusqu'au prochain polling.
- Une version échouée peut être retentée avant 24 h en supprimant `update.failed` dans le dossier du Worker, ou en publiant une version suivante.
- Les agents déjà installés n'ont ni drainage ni mémoire des échecs tant que le lanceur n'a pas été rejoué ; ils suivent quand même la cible sans erreur (`idle` est ignoré).
- Vérifications : tests Go du catalogue (pagination, hôte étranger, reprise espacée), de l'API (`TestFoundation/worker_latest_update` : bascules, aucune écriture sans nouveauté, panne, pas de rétrogradation, plancher, compatibilité de la requête) et de l'agent (attente d'un Worker occupé, version échouée non retentée) ; test e2e de l'écran.
