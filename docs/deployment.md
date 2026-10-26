# Déploiement

Les médias importés sont stockés dans le volume nommé `media`, séparé de PostgreSQL et accessible uniquement à l’application (UID 10001). Les variables `MAX_UPLOAD_BYTES` (20 GiB), `UPLOAD_CHUNK_BYTES` (4 MiB) et `UPLOAD_SESSION_TTL` (168 h) règlent les limites et la durée de reprise. Le nom d’origine est conservé comme métadonnée ; le chemin est toujours généré côté serveur. Le volume `media` contient les fichiers importés et doit être sauvegardé séparément de la base.

## Forgeyard

Importer **compose.yaml seul**, jamais compose.override.yaml. Fournir dans les variables chiffrées du projet :

- `SNITCH_IMAGE` : facultative. Par défaut `ghcr.io/thomas-bsn/snitch:latest`, mise à jour à chaque push sur `main` ; « Redéployer » retire `latest` et relance migrate puis app (même modèle que pingme). Pour figer une version, définir `ghcr.io/<propriétaire>/<dépôt>:vX.Y.Z` ou `@sha256:…`. Les forks utilisent leur propre namespace et doivent définir cette variable.
- `POSTGRES_IMAGE`, `EMBED_IMAGE` : facultatives. Par défaut, le compose tire directement les images amont Docker Hub (`pgvector/pgvector` et `ollama/ollama`) épinglées par digest, comme dans `deploy/upstream-images.json`. Le projet ne les republie pas sur GHCR ; Forgeyard accepte les images Docker Hub (confirmé par l’utilisateur, comme pour `postgres:17-alpine` dans pingme). Ne les surcharger que pour changer de version.
- Les images GHCR créées par Actions sont privées par défaut : régler leur visibilité dans GitHub (Packages), ou configurer sur la plateforme un accès `docker login ghcr.io` avec un jeton `read:packages`. Aucune image de mise à jour n'existe.
- `POSTGRES_PASSWORD` : mot de passe aléatoire ; les caractères réservés d’URL sont admis, car l’application et la migration passent le mot de passe par `PGPASSWORD`, hors URL.
- `MASTER_KEY` : 32 octets aléatoires encodés base64 ; conserver la même clé entre redémarrages et la sauvegarder séparément.
- `PUBLIC_URL=https://<projet>.<domaine>`, `COOKIE_SECURE=true`, `DISPLAY_NAME` facultatif.
- `CLIPS_ROOTS` peut rester vide. `EMBEDDINGS_URL` vise par défaut le service privé embed ; aucun modèle n’est préchargé.

Seule app est exposée sur son port HTTP interne 8080. PostgreSQL, migration et embeddings restent privés. Les volumes nommés `database` (PostgreSQL 18 : `/var/lib/postgresql`), `embeddings` et `media` persistent. Les clips provenant de dossiers reste un accès distinct, en lecture seule, via `CLIPS_ROOTS`. Les métadonnées mémoire sont des plafonds proposés : PG 512, app 384, embed 1536, migration transitoire 256 MiB.

PostgreSQL sain précède migrate, puis app attend sa réussite. Un job échoué bloque l’API, sans boucle de redémarrage. Lire ses logs localement, corriger, puis **Redéployer** le projet : cette action recrée tous les services et relance les jobs. Une modification du compose ou des variables ne redéploie que les services dont la configuration change ; une modification sans effet sur migrate ne garantit pas son rejeu. Les migrations sont rejouables ; app ne migre jamais implicitement et refuse un schéma incompatible.

## Docker Compose local

Le README utilise compose.yaml + l’override automatique. L’override autorise les builds, ports loopback et `mem_limit` ; Docker ignore les plafonds `x-forgeyard`. Aucun volume hôte n’est utilisé. Sur une VM de 2 Go, les plafonds additionnés ne sont pas des réservations : un modèle chargé pourrait dépasser les ressources physiques.

Un `up` local peut réutiliser le conteneur migrate déjà terminé. Pour une nouvelle version : arrêter app, construire/télécharger les images, exécuter `docker compose run --rm migrate`, puis `docker compose up -d app` uniquement après succès. Pour rejouer l’ordonnancement complet : `docker compose up -d --force-recreate` ; il conserve les volumes. Sauvegarder avant une mise à jour. SIGTERM laisse dix secondes aux requêtes avant fermeture des connexions.

## Publication

Le workflow `Verify and distribute` suit un seul chemin : `verify` → `changes` → `publish-app` et `publish-worker`. `verify` exécute toutes les vérifications sur PR, `main` et tags `v*`. Les jobs de publication passent toujours après son succès et jamais depuis une PR ou une autre branche :

- `changes` (git seul, aucune action tierce) décide quoi publier et sous quels tags.
  - **Push sur `main`** (ou `workflow_dispatch` sur `main`, entrée `all` pour tout publier) : il compare `github.event.before` à `github.sha` et ne retient que ce qui a changé. Les tags sont `latest` et `sha-<commit court>`. Un `before` nul, introuvable ou non ancêtre (force-push) publie tout.
  - **Tag `v*`** : il publie toujours tout, sous le tag de version figé (versions cohérentes entre l'application et le worker).
- `publish-app` (`snitch`) : si `app/`, `web/`, `contracts/`, `deploy/docker/app.Dockerfile`, `compose.yaml`, les licences ffmpeg ou les scripts de build changent.
- `publish-worker` (`snitch-worker`, image unique du worker) : si `worker/`, `contracts/`, `deploy/docker/worker.Dockerfile`, `scripts/collect_python_notices.py` ou `deploy/licenses/` changent. Cache de couches buildx `type=gha`, espace disque du runner libéré pour cette image.
- Les images amont (PostgreSQL/pgvector, Ollama) ne sont pas publiées par le projet : le compose les référence directement, épinglées par digest. Pour les mettre à jour, modifier `deploy/upstream-images.json`, `compose.yaml` (défauts de `POSTGRES_IMAGE` et `EMBED_IMAGE`) ensemble ; `scripts/check_compose.py` vérifie qu’ils concordent. Cette modification redéclenche la publication de l’image app uniquement via `compose.yaml`.

Premier lancement : `workflow_dispatch` avec `all`.

L'image worker exige un GPU NVIDIA pour des performances raisonnables (`SNITCH_VOICE_DEVICE=cpu` reste possible mais lent, non mesuré). Les licences NVIDIA de l'image worker sont à relire avant de la rendre publique (voir deploy/NOTICES.md). Les actions et bases sont épinglées ; les images applicatives amd64 (cible NAS N100 et PC GPU) partagent le namespace du dépôt, y compris les forks. Aucun secret packages:write n’est donné aux PR. Cette configuration ne constitue ni un push ni une publication ; aucune exécution distante n’a été observée ici.

Le volume nommé `media` contient les originaux importés ; une sauvegarde PostgreSQL ne contient ni ces vidéos ni les clés qui les désignent. Sauvegardez le volume séparément (ou les originaux dédiés), avec une rétention et une destination adaptées. La bibliothèque conserve ses métadonnées et l’historique des sources lorsqu’un original devient indisponible, mais la lecture exige qu’au moins une copie existe toujours. Les dossiers surveillés restent des originaux externes montés en lecture seule et leur sauvegarde dépend de leur stockage d’origine.
