# Déploiement

Les médias importés sont stockés dans le volume nommé `media`, séparé de PostgreSQL et accessible uniquement à l’application (UID 10001). Les variables `MAX_UPLOAD_BYTES` (20 GiB), `UPLOAD_CHUNK_BYTES` (4 MiB) et `UPLOAD_SESSION_TTL` (168 h) règlent les limites et la durée de reprise. Le nom d’origine est conservé comme métadonnée ; le chemin est toujours généré côté serveur. Le volume `media` contient les fichiers importés et doit être sauvegardé séparément de la base.

## Forgeyard

Importer **compose.yaml seul**, jamais compose.override.yaml. Fournir dans les variables chiffrées du projet :

- `RECEIPTS_IMAGE`, `POSTGRES_IMAGE`, `EMBED_IMAGE` : références publiées immuables, idéalement `ghcr.io/<propriétaire>/<dépôt>[-postgres|-embed]@sha256:…`. Les forks utilisent leur propre namespace.
- `POSTGRES_PASSWORD` : mot de passe aléatoire ; les caractères réservés d’URL sont admis, car l’application et la migration passent le mot de passe par `PGPASSWORD`, hors URL.
- `MASTER_KEY` : 32 octets aléatoires encodés base64 ; conserver la même clé entre redémarrages et la sauvegarder séparément.
- `PUBLIC_URL=https://<projet>.<domaine>`, `COOKIE_SECURE=true`, `DISPLAY_NAME` facultatif.
- `CLIPS_ROOTS` peut rester vide. `EMBEDDINGS_URL` vise par défaut le service privé embed ; aucun modèle n’est préchargé.

Seule app est exposée sur son port HTTP interne 8080. PostgreSQL, migration et embeddings restent privés. Les volumes nommés `database` (PostgreSQL 18 : `/var/lib/postgresql`), `embeddings` et `media` persistent. Les clips provenant de dossiers reste un accès distinct, en lecture seule, via `CLIPS_ROOTS`. Les métadonnées mémoire sont des plafonds proposés : PG 512, app 384, embed 1536, migration transitoire 256 MiB.

PostgreSQL sain précède migrate, puis app attend sa réussite. Un job échoué bloque l’API, sans boucle de redémarrage. Lire ses logs localement, corriger, puis **Redéployer** le projet : cette action recrée tous les services et relance les jobs. Une modification du compose ou des variables ne redéploie que les services dont la configuration change ; une modification sans effet sur migrate ne garantit pas son rejeu. Les migrations sont rejouables ; app ne migre jamais implicitement et refuse un schéma incompatible.

## Docker Compose local

Le README utilise compose.yaml + l’override automatique. L’override autorise les builds, ports loopback et `mem_limit` ; Docker ignore les plafonds `x-forgeyard`. Aucun volume hôte n’est utilisé. Sur une VM de 2 Go, les plafonds additionnés ne sont pas des réservations : un modèle chargé pourrait dépasser les ressources physiques.

Un `up` local peut réutiliser le conteneur migrate déjà terminé. Pour une nouvelle version : arrêter app, construire/télécharger les images, exécuter `docker compose run --rm migrate`, puis `docker compose up -d app` uniquement après succès. Pour rejouer l’ordonnancement complet : `docker compose up -d --force-recreate` ; il conserve les volumes. Sauvegarder avant une mise à jour. SIGTERM laisse dix secondes aux requêtes avant fermeture des connexions.

## Publication ultérieure

Le workflow `Verify and distribute` exécute les contrôles sur PR, main et tags `v*`. Seul un push de tag version, après succès du job verify, publie app/worker et copie les manifestes amont PostgreSQL/pgvector et Ollama intacts. Les actions et bases sont épinglées ; les images applicatives amd64 (cible NAS N100 et PC GPU) partagent le namespace du dépôt, y compris les forks. Le workflow conserve les références amont en artifact ; relever les digests GHCR résultants pour configurer l’installation. Aucun secret packages:write n’est donné aux PR. Cette configuration ne constitue ni un push ni une publication ; aucune exécution distante n’a été observée ici.

Le volume nommé `media` contient les originaux importés ; une sauvegarde PostgreSQL ne contient ni ces vidéos ni les clés qui les désignent. Sauvegardez le volume séparément (ou les originaux dédiés), avec une rétention et une destination adaptées. La bibliothèque conserve ses métadonnées et l’historique des sources lorsqu’un original devient indisponible, mais la lecture exige qu’au moins une copie existe toujours. Les dossiers surveillés restent des originaux externes montés en lecture seule et leur sauvegarde dépend de leur stockage d’origine.
