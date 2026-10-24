# Bilan de phase 0

Le socle de phase 0 est implémenté. La validation utilisateur de fin de phase est attendue ; la phase 1 n’a pas commencé. Ce bilan porte sur les sources et vérifications locales, sans publication, déploiement NAS, modèle ou média réel.

## Réalisé

Les tickets #2–#6 livrent Go/SPA embarquée, PostgreSQL/pgvector, migrations séparées et rejouables, premier administrateur transactionnel, sessions locales, inscription/approbation, politique d’accès et Discord facultatif avec liaison explicite et secrets chiffrés. Les refus sont appliqués par l’API et les parcours OAuth utilisent un fournisseur simulé.

Le ticket #7 ajoute le contrat de distribution Forgeyard à quatre services, les images/configurations portables, les mots de passe via `PGPASSWORD`, les limites locales, les dépendances verrouillées et notices complètes, le smoke Compose réel, la CI et la publication future conditionnelle, les guides d’exploitation et de contribution. Le Compose n’expose que app:8080 dans Forgeyard et utilise des volumes nommés ; aucun worker GPU n’est présent sur NAS.

Le mot de passe de la fixture contient `:@/?#%+` et un espace. Il fonctionne pour le job goose/pgx et le pool app : une URL sans composante mot de passe laisse pgx utiliser `PGPASSWORD`. Sources vérifiées : pgx v5.10.0 `pgconn/config.go` lit les variables d’environnement puis superpose uniquement les champs présents de l’URL ; `stdlib/sql.go` appelle ParseConfig ; goose v3.28.0 `db.go` utilise ce pilote via sql.Open. L’ancienne interpolation directe dans l’URL échouait au smoke ; la correction passe sans imposer de restriction artificielle au mot de passe.

## Corrections de revue

La revue finale a relevé un retour JSON brut lors d’une panne interne du callback OAuth en navigation HTML. Les erreurs de lecture/déchiffrement des réglages, de transaction et de création de session reviennent désormais vers une notice française de la SPA ; les clients API conservent JSON/503 et les logs restent expurgés. Des fixtures couvrent le déchiffrement impossible après consommation du state et le refus d’écriture d’une session.

Le magasin de réglages est réutilisé dans la transaction du contrôleur : chiffrement et UPSERT ont un propriétaire unique, sans connexion supplémentaire ni changement des verrous et de l’atomicité de la politique d’inscription. Un échec de mise à jour de politique vérifie le rollback des réglages. `client_secret:null` est refusé conformément au contrat ; omission et chaîne vide conservent le secret existant.

## Vérifications

- Format Go, go vet, format frontend et build TypeScript/Vite ; génération Go/TypeScript/Python/sqlc identique octet pour octet.
- Tests Go avec race detector et vrai PostgreSQL/pgvector Testcontainers, contrat HTTP, concurrence, sécurité, migrations, binaire et arrêt ; cinq suites navigateur avec vraie API et Discord simulé.
- Validation statique du workflow avec actionlint v1.7.7 ; aucune exécution distante.
- 23 fixtures négatives Forgeyard : options interdites, montage hôte/anonyme, volume externe, exposition supplémentaire, job en boucle, dépendances et santé invalides.
- Builds locaux app et worker arm64 et cross-compilation du binaire Go Linux/amd64 ; `pip check`, commande du squelette worker et présence des textes de licences vérifiés dans les images (24 fichiers app, 26 worker).
- Smoke Compose isolé : installation vierge réelle par HTTP, nom configurable, SPA, absence de modèle, disponibilité après arrêt d’embed, persistance après redémarrage, sortie 0 sur SIGTERM ; projet séparé où une commande de migration volontairement invalide bloque l’API et ne redémarre pas. Les projets et leurs volumes sont supprimés après test. Les logs de bootstrap et secrets ne sont pas conservés dans les résultats.

La CI distante n’a pas été exécutée : aucun push n’est autorisé dans cette session. Le workflow existe et bloque la publication derrière les contrôles/tests et les builds app/worker ; seul un futur tag `v*` peut publier. Les branches/tags n’ont pas été poussés. La publication cible amd64, plateforme native des contrôles CI et du NAS/PC prévus ; les builds locaux mesurés sont arm64.

## Empreinte locale

Mesures du 24 octobre 2026, Mac arm64 macOS 26.4.1, Colima/Linux Ubuntu 24.04.4 aarch64, Docker 29.5.2, VM limitée à 2 053 644 288 octets (~1958 MiB). Projets neufs avec secrets synthétiques, installation admin puis redémarrage, sans modèle ni média. Après 10 s de chauffe, cinq relevés `docker stats --no-stream`, espacés de 2 s en plus de la durée des commandes. Cette fenêtre courte mesure l’inactivité après installation, pas une charge soutenue ni les pics de connexion Argon2.

Les valeurs Docker stats sont le working set de conteneur selon Docker (traitement du cache propre à Docker/Linux), pas le RSS du processus ni la mémoire totale de la VM. Les budgets NAS (app 384, PG 512, embed 1536, migrate 256 MiB transitoires) sont des plafonds, pas des réservations. Ils dépassent la RAM de cette VM s’ils sont tous consommés ; aucun modèle chargé n’a été évalué.

| Service | Mémoire idle min–max (MiB) | `image ls` | `image inspect .Size` (octets) |
| --- | ---: | ---: | ---: |
| app arm64 `snitch-smoke:local` | 34,82–34,85 | 30,5 MB | 8 635 471 |
| PostgreSQL/pgvector arm64 | 31,00–33,64 | 660 MB | 159 402 643 |
| Ollama arm64 sans modèle | 11,06–11,06 | 7,08 GB | 2 818 184 304 |
| worker squelette arm64, non démarré sur NAS | non mesurée | 229 MB | 49 298 190 |

Total des services idle observés : environ 76,85–79,98 MiB ; ce chiffre n’inclut ni VM/daemon, ni processus de build, ni pic Argon2. Le rejeu de migration terminé réussit avec un pic cgroup v2 `memory.peak` de 4 108 288 octets (~3,92 MiB), mesuré dans un conteneur temporaire, pas via stats après sa sortie. Le pic de migration initiale n’a pas été capturé. Volumes : PostgreSQL 47 944 KiB, Ollama 28 KiB sans fichier modèle. Le worker n’est qu’un contrat sans modèles/CUDA.

[Relevés expurgés et protocole](measurements/phase-0-arm64.json). App construit : `sha256:750a3833d88098a4994a538d85279fced46aac000bc19f800366f083039505db` ; worker : `sha256:ad5de75a157cca431ffb107af62b3ce667daad90e6a096b35a04bc1c4ab884c7`. Ce sont des images locales, sans référence GHCR publiée.

Les références amont exactes sont dans [upstream-images.json](../deploy/upstream-images.json). Les tailles `docker image ls` et `docker image inspect .Size` sont rapportées séparément : avec le stockage containerd, elles diffèrent et ne représentent pas le même agrégat. Les couches partagées empêchent de sommer ces chiffres comme espace unique ; ni transfert réseau registry ni espace alloué au disque VM n’ont été mesurés. Les volumes sont mesurés séparément par `du -sk` dans leur service.

## Restant et écarts

Aucun écart fonctionnel déclaré au périmètre de phase 0. Les artefacts sont construits et vérifiés localement ; les images GHCR restent non publiées. Les bases amont sont épinglées par digest et la publication prévue conserve leurs couches/notices, mais aucune republication réelle n’a eu lieu. Les contrôles distants, builds d’images amd64 et leur exécution restent à observer après un push autorisé ; la cross-compilation Go Linux/amd64 locale est verte.

L’empreinte NAS N100 8 Go reste non mesurée. La mesure transitoire de migration porte sur un rejeu, pas le pic du premier schéma ; le pic de build, les modèles chargés, GPU/worker actif, débit, index de recherche, restauration complète et extrapolation à 1 To ne sont pas évalués. La procédure de sauvegarde/restauration est documentée ; l’exercice complet reste en phase 7. Forgeyard ne permet pas encore les montages hôte nécessaires aux futures sources dossier. Aucun traitement réel avant autorisation du benchmark de phase 7.

La prochaine décision est la validation utilisateur de la fin de phase 0, puis la préparation de la spec de phase 1. Aucune phase suivante n’est autorisée par ce bilan.
