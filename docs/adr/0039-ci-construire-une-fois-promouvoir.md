# 0039 — CI : construire une fois, valider en parallèle, promouvoir le digest validé

Statut : accepté. Amende le workflow de publication décrit dans `docs/deployment.md` ; conserve l'[ADR 0036](0036-mise-a-jour-automatique-du-worker.md) (tags `latest`, `sha-*`, SemVer du Worker).

## Contexte

`Verify and distribute` durait environ 30 minutes sur `main`. Mesures réelles (runs de `main` du 4 novembre 2026) :

- `check` : 913 à 1 201 s, dont `TestFoundation` 582 s : un seul test Go qui lance les 44 suites Playwright l'une après l'autre (443 s), avec `workers: 1`.
- `build` (séquentiel après `check`) : 536 à 550 s, dont 258 s pour l'image `snitch` et 235 s pour l'image Worker **à chaque push**, même quand elle n'avait pas changé.
- `publish-app` 81 à 95 s et `publish-worker` 333 à 468 s : une **seconde** construction des mêmes images, car l'image validée n'était pas celle qui était publiée.
- Libération du disque du runner : 47 à 97 s, dans chaque job qui la faisait.
- Le cache GitHub Actions (7,5 Go sur 10) contenait une couche torch de 4,4 Go qui évinçait les autres caches.
- Les pull requests ne construisaient aucune image : un Dockerfile cassé n'apparaissait qu'après le merge. Les 261 tests unitaires du Worker n'étaient exécutés nulle part.

## Décision

- **Construire une fois, promouvoir le même digest.** Sur `main` et les tags, `image-app` et `image-worker` poussent l'image **par digest, sans tag**. L'image `snitch` est tirée par ce digest et passée au test de fumée. Les jobs `promote-*`, qui n'existent qu'après la barrière `check`, copient le manifeste de ce digest vers `latest`, `sha-<commit>` et `vX.Y.Z` (`imagetools create`, sans reconstruction ni transfert de couches) et échouent si un tag ne résout pas vers ce digest. Une image non validée n'a donc jamais de tag, et il n'y a plus de seconde construction.
- **Tout ce qui est indépendant tourne en parallèle** : `lint`, `go-test`, `e2e`, `worker-test`, `image-app`, `image-worker`. `check` agrège : un job ignoré parce que le changement ne peut pas l'affecter est accepté, un job ignoré parce que son prérequis a échoué ne l'est pas (`changes` et `lint` doivent être en succès).
- **Détection des changements en premier, pour tous les événements**, y compris les PR (diff contre le parent de base du commit de fusion). `core` (tests Go et navigateur) est calculé par exclusion : seuls `docs/`, `worker/`, les `.md` et quelques scripts Python en sont exemptés ; un chemin inconnu lance les tests. `lint` tourne toujours.
- **E2E en 5 shards** : `SNITCH_E2E_SHARD=i/n` répartit les suites par poids décroissant sur le shard le plus léger (partition déterministe, testée : chaque suite tourne dans exactement un shard). Un poids périmé ralentit un shard, il ne retire jamais une suite. Chaque shard a son PostgreSQL ; le détecteur de course reste actif.
- **Cache de couches dans le registre** (`ghcr.io/<dépôt>-cache:app|worker`, `mode=max`, zstd) au lieu du cache GHA ; seul `main` l'écrit. Cache Go glissant par type de build (`lint`, `race`), écrit par `main` seulement.
- **Dockerfile Worker** : l'utilisateur d'exécution et l'import de NeMo passent avant la copie du code, pour qu'une modification du code Worker ne les rejoue pas.
- **Version SemVer du Worker réservée après `check`**, dans `promote-worker` : un build ou un test en échec ne consomme plus de version. Le groupe de concurrence `worker-version` est retiré : GitHub ne garde qu'un job en attente par groupe et annulerait silencieusement une publication ; le verrou reste l'API des références (création atomique, 10 tentatives).
- **Concurrence** : annulation des runs de PR remplacés uniquement. Jamais sur `main` ni sur les tags (`changes` compare au push précédent : un commit abandonné perdrait sa publication).
- **Mesure** : `scripts/ci_metrics.py` écrit les durées par job, le temps réel, le temps de calcul et les minutes facturées dans le résumé de chaque run, et compare les derniers runs de `main` (`--history N`).

## Alternatives écartées

- **Publier l'image avant la barrière sous un tag temporaire** : une image non validée serait tirable. Le push par digest sans tag l'évite.
- **Transférer l'image en artefact entre jobs** : plusieurs gigaoctets pour le Worker, plus lent que la copie de manifeste.
- **Image de base préconstruite avec torch/CUDA** : le cache de registre rend déjà la couche lourde gratuite tant que les verrous ne changent pas ; une image de base ajouterait un artefact à maintenir pour un gain nul en régime établi.
- **Runners auto-hébergés ou plus gros** : non nécessaires ; le dépôt est public (minutes gratuites) et le chemin critique est désormais dominé par des étapes fixes.
- **Contrôle `pip check` exécuté dans un conteneur** : déjà fait par le Dockerfile (`RUN … pip check`), dupliquer coûtait un chargement de plusieurs gigaoctets.

## Conséquences

- Le premier run de `main` après la fusion construit le Worker à froid (le cache de registre n'existe pas encore) ; les suivants le réutilisent. Les PR lisent le cache sans l'écrire.
- Des digests sans tag s'accumulent dans GHCR (un par construction publiée) ; un nettoyage périodique reste à prévoir.
- Deux runs de `main` rapprochés peuvent encore terminer dans le désordre et déplacer `latest` vers un commit plus ancien : le risque existait déjà (même fenêtre, plus courte maintenant) ; une garde par ancêtre de commit est possible.
- Un tag `v*` reconstruit et revérifie tout (cache chaud) au lieu de promouvoir le `sha-*` déjà publié : plus sûr, moins rapide qu'il ne pourrait l'être.
- Les tests unitaires du Worker (261) entrent dans la CI : un échec qui existait déjà devient visible.
