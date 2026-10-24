# Déploiement

Phase 0 uniquement. Ni média, ni montage hôte, ni GPU, ni modèle téléchargé. Le worker est un squelette séparé et ne rejoint pas le Compose NAS. Le nom `DISPLAY_NAME` est libre ; toutes les images sont configurables. Ne pas utiliser de tag GHCR inexistant : les sources actuelles doivent être construites localement jusqu’à une publication validée.

## Forgeyard

Importer **compose.yaml seul**, jamais compose.override.yaml. Fournir dans les variables chiffrées du projet :

- `RECEIPTS_IMAGE`, `POSTGRES_IMAGE`, `EMBED_IMAGE` : références publiées immuables, idéalement `ghcr.io/<propriétaire>/<dépôt>[-postgres|-embed]@sha256:…`. Les forks utilisent leur propre namespace.
- `POSTGRES_PASSWORD` : mot de passe aléatoire ; les caractères réservés d’URL sont admis, car l’application et la migration passent le mot de passe par `PGPASSWORD`, hors URL.
- `MASTER_KEY` : 32 octets aléatoires encodés base64 ; conserver la même clé entre redémarrages et la sauvegarder séparément.
- `PUBLIC_URL=https://<projet>.<domaine>`, `COOKIE_SECURE=true`, `DISPLAY_NAME` facultatif.
- Laisser `CLIPS_ROOTS` et `EMBEDDINGS_MODEL` vides. `EMBEDDINGS_URL` vise par défaut le service privé embed ; aucun modèle n’est préchargé.

Seule app est exposée sur son port HTTP interne 8080. PostgreSQL, migration et embeddings restent privés. Les volumes nommés `database` (PostgreSQL 18 : `/var/lib/postgresql`) et `embeddings` persistent. Forgeyard n’accepte pas les bind mounts nécessaires aux futurs clips : ne pas inventer de montage NAS. Les métadonnées mémoire sont des plafonds proposés : PG 512, app 384, embed 1536, migration transitoire 256 MiB ; les mesures NAS restent à faire.

PostgreSQL sain précède migrate, puis app attend sa réussite. Un job échoué bloque l’API, sans boucle de redémarrage. Lire ses logs localement, corriger, puis **Redéployer** le projet : cette action recrée tous les services et relance les jobs. Une modification du compose ou des variables ne redéploie que les services dont la configuration change ; une modification sans effet sur migrate ne garantit pas son rejeu. Les migrations sont rejouables ; app ne migre jamais implicitement et refuse un schéma incompatible.

## Docker Compose local

Le README utilise compose.yaml + l’override automatique. L’override autorise les builds, ports loopback et `mem_limit` ; Docker ignore les plafonds `x-forgeyard`. Aucun volume hôte n’est utilisé. Sur une VM de 2 Go, les plafonds additionnés ne sont pas des réservations : un modèle chargé pourrait dépasser les ressources physiques.

Un `up` local peut réutiliser le conteneur migrate déjà terminé. Pour une nouvelle version : arrêter app, construire/télécharger les images, exécuter `docker compose run --rm migrate`, puis `docker compose up -d app` uniquement après succès. Pour rejouer l’ordonnancement complet : `docker compose up -d --force-recreate` ; il conserve les volumes. Sauvegarder avant une mise à jour. SIGTERM laisse dix secondes aux requêtes avant fermeture des connexions.

## Publication ultérieure

Le workflow `Verify and distribute` exécute les contrôles sur PR, main et tags `v*`. Seul un push de tag version, après succès du job verify, publie app/worker et copie les manifestes amont PostgreSQL/pgvector et Ollama intacts. Les actions et bases sont épinglées ; les images applicatives amd64 (cible NAS N100 et PC GPU) partagent le namespace du dépôt, y compris les forks. Le workflow conserve les références amont en artifact ; relever les digests GHCR résultants pour configurer l’installation. Aucun secret packages:write n’est donné aux PR. Cette configuration ne constitue ni un push ni une publication ; aucune exécution distante n’a été observée ici.
