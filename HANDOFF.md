# Snitch — reprise de session

> Archive de reprise : les mentions ci-dessous concernant l'état du projet, le blocage réseau et l'autorisation de coder sont obsolètes. Lire AGENTS.md, ROADMAP.md et la spec de phase 0 pour l'état actuel. Conserver les détails des phases futures jusqu'à leur reprise dans les specs correspondantes.

Ce document transmet le contexte de la conversation précédente à une nouvelle session Codex. Le nom « Receipts » est provisoire. L'utilisateur travaille avec **Codex via Orca** sur Mac ; ne pas le confondre avec l'application Codex.

## État et autorisation

- Répertoire : `/Users/thomasboisson/dev/snitch`.
- Le dossier était entièrement vide, sans dépôt Git. Aucun code applicatif n'a encore été écrit. Ce document est le seul livrable de reprise.
- L'utilisateur a demandé un plan avant le code, puis **validé le plan de phase 0 et ses trois décisions complémentaires**. L'implémentation de cette phase est autorisée ; ne pas redemander l'approbation du plan.
- **Travailler phase par phase. À la fin de chaque phase : tests verts, README à jour, bilan du réalisé, du restant et des écarts ; arrêter et attendre la validation avant la suivante.**
- Blocage actuel : `npm ping --fetch-retries=0 --fetch-timeout=10000` échoue avec `ENOTFOUND registry.npmjs.org`. La session déclare le réseau restreint. L'utilisateur a mis `[sandbox_workspace_write] network_access = true` dans `~/.codex/config.toml` ; sa présence a été vérifiée, mais cette session reste bloquée. Ne pas affirmer sans preuve qu'Orca expose un menu particulier ou que l'erreur DNS prouve seule sa cause.
- Commencer par retester `npm ping`. Si le réseau fonctionne, poursuivre la phase 0. En cas de véritable blocage, s'arrêter et expliquer, sans contournement.
- `go version` local : `go1.27.1 darwin/arm64`. Node local : `v24.21.0`. Docker : `29.5.2` avec Colima. La commande isolée `docker info --format '{{.ServerVersion}}'` avait réussi ; combinée à d'autres commandes, elle avait rencontré une permission refusée. Vérifier les accès nécessaires sans annoncer tous les tests disponibles à partir d'un seul succès.
- Aucun test de l'application n'a été exécuté, aucune image construite ni publiée, aucun dépôt distant créé, aucun traitement vidéo lancé.
- L'utilisateur préfère des explications brèves, concrètes et sans jargon. Il a été frustré par des questions techniques longues ; expliquer le comportement proposé et prendre les décisions courantes dans le périmètre autorisé.

## Mission et règles impératives

Construire from scratch une application open source **MIT** qui indexe des clips gaming pour les retrouver par ce qui a été dit, qui l'a dit, quand et dans quel clip ; lecture au timestamp correspondant. Français avec environ 20 % de jargon anglais. Pas de chatbot en V1.

- Monolithe modulaire simple. Ni Redis, ni RabbitMQ, ni base vectorielle séparée.
- Toute décision technique non triviale : ADR `docs/adr/NNNN-titre.md` avec contexte, décision, alternatives écartées et conséquences.
- Aucun fournisseur ou modèle IA codé en dur dans le cœur : configuration et interfaces remplaçables.
- **Aucun traitement sur la vraie bibliothèque avant validation du benchmark de phase 7.** Les tests utilisent des données synthétiques ou des fixtures dédiées. Ne pas scanner spontanément la vraie bibliothèque.
- Aucun secret dans le dépôt, les logs ou le frontend ; `.env.example` documenté. Exception explicitement demandée : code d'installation initial à usage unique affiché dans les logs.
- Tous les comptages affichés proviennent de requêtes en base, jamais d'une estimation de modèle.
- Tests obligatoires : unitaires, intégration avec un vrai PostgreSQL via Testcontainers, E2E des parcours clés. Ne pas remplacer PostgreSQL par SQLite.

## Matériel et architecture décidée

- NAS UGREEN DXP2800, Intel N100 4 cœurs, 8 Go partagés, pas de GPU, **budget app total environ 2–3 Go**, deux disques sans miroir.
- Environ 1 To de clips Medal, souvent 15 secondes, déjà sur le NAS. Medal écrit les nouveaux clips dans un dossier du NAS. Audio variable : pistes séparées jeu/micro/vocal, ou piste mixée.
- PC Windows WSL2 + Docker, RTX 4070 12 Go VRAM, 32 Go RAM, pas toujours allumé.
- NAS : binaire Go unique (REST, SPA embarquée, streaming Range, ingestion, jobs serveur, recherche, matching vocal), PostgreSQL + pgvector, service d'embeddings CPU.
- Front : Vite + React + TypeScript, compilé puis embarqué avec `embed.FS`. Aucun Node en production, aucun Next.js.
- SQL : pgx + sqlc ; migrations versionnées goose ou équivalent dans un **job séparé**, jamais au démarrage de l'API.
- `contracts/openapi.yaml` est la source de vérité. Serveur Go et clients TS/Python générés ou testés contre le contrat ; CI échoue en cas de divergence.
- Worker séparé Python + CUDA sur le PC : ffmpeg/ffprobe, faster-whisper, pyannote, empreintes vocales. Il **ne se connecte jamais à PostgreSQL** : pull de jobs avec lease, téléchargement des médias et dépôt des résultats par HTTPS. Jeton dédié haché, révocable et limité aux routes worker et au média de sa tâche.
- Recherche disponible PC éteint. Même modèle/version d'embeddings pour indexation et requêtes, endpoint `/v1/embeddings` compatible OpenAI sur NAS. Ollama CPU par défaut, pas de modèle définitif avant benchmark. Panne embeddings : recherche lexicale maintenue. Candidats : multilingual-e5-small, EmbeddingGemma, bge-m3.

## Forgeyard — contraintes vérifiées

Source à relire au besoin : dépôt `thomas-bsn/forgeyard`, `docs/apps/projects.md`. Le contenu a été obtenu via `gh api repos/thomas-bsn/forgeyard/contents/docs/apps/projects.md --jq .content | base64 -d`.

- `compose.yaml` racine compatible Forgeyard et Docker Compose : **image uniquement, pas de build**, volumes nommés uniquement.
- Interdits : `ports`, `networks`, `network_mode`, `env_file`, `container_name`, `privileged`, `cap_add`, `devices`, `extends`, bind mounts, volumes anonymes et tmpfs.
- Un service exposé via `x-forgeyard: { expose: { port: 8080 } }` : `app`.
- Mémoire : `x-forgeyard: { memory: <Mo> }` ; défaut Forgeyard 512 Mo. PostgreSQL : `x-forgeyard: { backup: { type: postgres } }`.
- Services proposés : `postgres`, `migrate`, `app`, `embed`. `migrate` utilise la même image que l'app avec une commande dédiée et `restart: "no"`.
- `depends_on` avec `service_healthy` et `service_completed_successfully` est pris en charge. Une migration échouée bloque l'API. Un job est exécuté une fois par version, et relancé lors du redéploiement approprié.
- Clés admises : image, command, entrypoint, user, environment, volumes nommés, depends_on, healthcheck, restart, x-forgeyard. Volumes racine sans options. Variables `${VAR}` et formes usuelles prises en charge.
- Images publiées GHCR par GitHub Actions. Le plan validé prévoit de republier les images amont PostgreSQL/pgvector et Ollama épinglées en conservant les licences, pour respecter la contrainte GHCR de l'utilisateur.
- `compose.override.yaml` de dev peut utiliser build, ports et bind mounts **en lecture seule de fixtures**.
- Forgeyard ne sait pas encore monter des dossiers hôte dans les projets. **Ne pas inventer de syntaxe**. L'app lit `CLIPS_ROOTS` ; documenter dans un ADR le besoin futur de montage RO validé par un admin.
- Worker absent du compose NAS ; image `…-worker` séparée sur PC avec `docker run --gpus all` ou app Forgeyard GPU seule.
- Réglages initiaux proposés PostgreSQL : shared_buffers 128 MB, work_mem 4 MB, pool limité. Enveloppes proposées : app 384 Mo, PostgreSQL 512 Mo, embed 1536 Mo, migration transitoire 256 Mo. **Budgets non mesurés**, pas garanties pour les modèles. Docker Compose ignore x-forgeyard : limites locales à définir dans l'override.
- Sauvegarde DB Forgeyard par pg_dump vers S3 ; documenter restauration et absence de sauvegarde des vidéos par l'app, NAS sans miroir.

## Plan de phase 0 déjà validé

Arborescence :

```text
LICENSE, README.md, GLOSSARY.md, .env.example
compose.yaml, compose.override.yaml
.github/workflows/
app/
  go.mod
  cmd/receipts/            # serve, migrate, healthcheck
  internal/{config,auth,setup,settings,httpapi,database}/
  migrations/, queries/, sqlc.yaml
  ui/                     # SPA compilée et embed.FS
web/src/, web/e2e/
worker/pyproject.toml, worker/snitch_worker/api/
contracts/openapi.yaml, contracts/generators/
deploy/docker/
docs/adr/, docs/development.md, docs/deployment.md, docs/testing.md
```

- Nom d'affichage et références aux images configurables, renommage facilité.
- Configuration validée, logs structurés expurgés, arrêt propre, vivacité/disponibilité. Embeddings indisponibles : état dégradé sans bloquer l'app.
- Auth locale Argon2id, sessions opaques hachées en base, cookies HttpOnly/Secure/SameSite=Lax, CSRF, rate limiting login, rôles admin/user. Routes métier protégées par défaut.
- Discord OAuth2 `identify` seulement, vérification state, tests avec fournisseur simulé.
- Création du premier admin atomique et résistante à deux demandes concurrentes. Code bootstrap à usage unique, expirant. Wizard fermé après initialisation.
- OpenAPI uniquement pour les routes effectivement livrées. Régénération reproductible Go/TS/Python ; absence de diff et tests contre API réelle, erreurs et permissions comprises.
- CI format/lint, tests, SPA, binaire, images et publication GHCR seulement après succès. Épingler versions et images, conserver licences.
- Tests de sortie : unitaires ; PostgreSQL+pgvector Testcontainers ; E2E installation vierge/login/logout/refus anonymes/permissions/Discord simulé ; concurrence premier admin ; migrations rejouables et échec bloquant ; clés compose Forgeyard strictes/volumes/service exposé unique ; SPA embarquée sans Node.
- Mesurer et documenter l'empreinte du socle sur l'environnement disponible, en distinguant clairement des mesures NAS et du benchmark complet restant à réaliser.

### Trois décisions complémentaires expressément acceptées

1. **Wizard progressif** : phase 0 admin et configuration Discord facultative ; dossiers en phase 1 ; worker en phase 2 ; vérifications embeddings et Hugging Face dans les phases qui introduisent ces fonctions. Étapes futures clairement indiquées comme reportées ; wizard complet avant benchmark.
2. **Secrets chiffrés en base** avec clé maître fournie par variable Forgeyard, distincte de la sauvegarde DB. Chiffrement authentifié via bibliothèque standard Go. Restauration nécessite base + clé ; sauvegarde de la clé à documenter.
3. **Approbation admin des inscriptions par défaut**. Liaison Discord explicite depuis un compte déjà connecté, jamais de fusion automatique sur pseudo. Les nouvelles inscriptions Discord suivent le réglage d'inscription.

### ADR prévus

0001 Monolithe Go / SPA / frontières ; 0002 PostgreSQL / pgx / sqlc / migrations séparées ; 0003 contrat OpenAPI ; 0004 Forgeyard / GHCR / montages hôte manquants ; 0005 auth / sessions / liaison Discord ; 0006 bootstrap / conservation secrets ; 0007 fournisseurs configurables / verrou avant benchmark.

### Versions proposées dans le plan

Ces versions ont été annoncées lors de recherches précédentes. **Vérifier leur existence effective, leurs licences et leur compatibilité sur les registres avant installation ; ne pas inventer de verrouillage ou d'exécution réussie.** Si une version annoncée est inexistante ou incompatible, signaler l'écart et choisir une version stable vérifiée sans cacher le problème.

| Composant | Version proposée | Licence annoncée |
|---|---|---|
| Go | 1.27.3 | BSD-3-Clause |
| PostgreSQL / pgvector | 18.6 / 0.8.7 | PostgreSQL |
| pgx / sqlc / goose | 5.10.0 / 1.31.1 / 3.28.0 | MIT |
| oapi-codegen | 2.8.0 | Apache-2.0 |
| x/crypto / x/oauth2 / x/time | 0.58.0 / 0.37.0 / 0.16.0 | BSD-3-Clause |
| React / Vite | 19.3.1 / 8.3.5 | MIT |
| TypeScript | 5.9.3 | Apache-2.0 |
| openapi-typescript / openapi-fetch | 7.13.0 / 0.17.0 | MIT |
| openapi-python-client | 0.29.1 | MIT |
| Testcontainers Go | 0.44.0 | MIT |
| Playwright | 1.64.1 | Apache-2.0 |
| Ollama | 0.40.10 | MIT |

Choix TS 5.9.3 motivé par le peer TS ^5.x annoncé d'openapi-typescript. oapi-codegen/runtime >=1.6.0 annoncé nécessaire ; versions exactes à verrouiller. Ollama sans modèle préchargé en phase 0. Les bibliothèques de traitement IA et licences de modèles seront examinées dans leurs phases.

## Spécification des phases suivantes — ne pas implémenter maintenant

### Phase 1 — ingestion et bibliothèque

- Interface Source extensible ; dossiers fsnotify + réconciliation périodique légère taille/mtime sans rehash des fichiers connus ; upload par morceaux reprenable dans volume nommé, limite configurable.
- Originaux **jamais modifiés, renommés, déplacés ou supprimés**. Montages RO. Noms upload jamais utilisés comme chemins.
- Stabilité taille/mtime pendant N secondes ET ffprobe réussi avant ingestion ; retry/backoff puis illisible. Motifs d'exclusion configurables temporaires/cachés/annexes.
- SHA-256 contenu : un clip pour plusieurs emplacements et sources. Renommage/déplacement reconnu sans retraitement.
- Provenance source/chemin/découverte/utilisateur upload ; durée/codecs/nombre et titres/langues des pistes/date d'enregistrement (métadonnées sinon mtime).
- Fichier disparu : index/transcription conservés, original_introuvable, vidéo non lisible. Alerte admin si disparitions anormalement nombreuses.
- Bibliothèque et filtres de base, streaming Range authentifié. owner_id sur clips et sources, visibilité private/shared dès maintenant ; V1 tout partagé.

### Phase 2 — file de tâches et worker

- Table jobs PostgreSQL, acquisition serveur FOR UPDATE SKIP LOCKED ; worker API seulement.
- Champs : id, clip_id, step, statut, priorité, tentatives/max, lease_owner/expires_at, prochain essai, dernière erreur exploitable, input_version (hash modèle+config), created_at/updated_at.
- États pending → leased → done/failed, retry_wait/cancelled/dead ; documenter et tester les transitions. Heartbeat, lease expiré récupérable, retry exponentiel, dead relançable admin.
- Résultats remplacés transactionnellement et sans doublons. Version modèle/config par étape, relance ciblée de l'étape invalidée ; aucun retraitement global nocturne.
- Worker Python squelette, faux fournisseur tests. Modèles chargés une seule fois, concurrence configurable indépendamment de la longueur de file ; pause/reprise et parallélisme admin, modèles GPU enchaînés sous 12 Go.

### Phase 3 — transcription et recherche textuelle

- Probe NAS ingestion. audio_prepare worker WAV 16 kHz mono ; garder pistes vocales, ignorer jeu si identifiable selon règles Medal configurables.
- faster-whisper large-v3 ou large-v3-turbo à départager benchmark ; langue forcée fr, timestamps par mot, VAD, vocabulaire personnalisé prénoms/jargon depuis admin en initial prompt/hotwords.
- FTS français + unaccent, recherche exacte, floue pg_trgm, ouverture timestamp. Comptages SQL « N sûrs + M approximatifs » avec liste cliquable ; couverture des clips non traités.

### Phase 4 — recherche sémantique

- Embeddings CPU NAS, indexation arrière-plan, pgvector halfvec HNSW, fusion hybride RRF.
- Filtres locuteur/date/période/source/état ; résultats clip, transcription surlignée, timestamps, locuteur/confiance.
- Modèle/version stockés ; changement reconstruit uniquement les embeddings en arrière-plan, ancien index utilisé jusqu'à bascule. Panne service : lexical toujours disponible.

### Phase 5 — voix

- pyannote.audio speaker-diarization-community-1 ; token HF et conditions vérifiés par wizard. Piste micro dédiée attribuée à son propriétaire sans diarisation.
- Empreintes par locuteur/clip et éventuellement tours assez longs avec durée/qualité. Matching NAS en base.
- Distinguer locuteur local, personne globale, échantillon validé, confiance et validation humaine. Pipeline jamais bloqué par voix inconnue.
- Clustering inconnues singleton, extraits écoutables et exclusions possibles, nommage de groupe ; deux voix proches jamais considérées identiques sans validation. Chiffres des groupes issus de DB.
- Tous utilisateurs connectés peuvent valider ; **première validation fait foi via contrainte DB**, secondes voient attribution/auteur et peuvent corriger explicitement. Historique ancienne/nouvelle valeur/auteur/date et propagation par voice_match en base, sans PC ni retraitement vidéo.
- Confirmation initiale même si confiance haute. Mesurer précision par personne/globale. Au seuil configurable (exemple 95 % / au moins 50 confirmations), proposer activation automatique ; seul admin l'active. Niveaux auto/suggestion/inconnue + ambigu, seuils configurables et choix benchmark.
- Confidentialité : extraits accessibles seulement connecté ; personne avec compte voit clips où identifiée et peut supprimer profil, empreintes/échantillons effacés, attributions inconnues. Ne pas promettre une identification parfaite.

### Phase 6 — corrections

Édition transcription/locuteur, historique, réindexation ciblée des seuls segments concernés.

### Phase 7 — benchmark et durcissement

- **Avant tout batch réel**, échantillon représentatif de 200–500 clips, avec validation utilisateur du benchmark.
- Temps par étape moyenne/distribution, clips/h pour WORKER_CONCURRENCY 1–4 ; pics VRAM/RAM/CPU PC et NAS ; WER sur sous-échantillon corrigé humainement ; qualité diarisation/identification ; pertinence jeu de requêtes ; taille base/index, index vectoriel par million de segments et RAM ; latence NAS ; erreurs et reprise.
- Rapport go/no-go pour batch complet et extrapolation 1 To avec hypothèses ; aucune durée promise avant mesures.
- Test de restauration, sécurité et documentation installation. Demucs seulement éventuellement évalué au benchmark, pas fonctionnalité V1.

## Critères transversaux à tester dans les phases concernées

Fichier stable/incomplet ; doublon/renommage sans retraitement ; crash app/worker et reprise sans perte/doublon ; worker indisponible avec recherche fonctionnelle ; retry/dead/relance ; propagation des voix sans vidéo et concurrence validation ; recherches exacte/floue/sémantique et comptages DB ; correction réindexe un segment ; changement de modèle ne relance que son étape ; suppression clip efface dérivés et conserve original ; permissions anonymes et jeton worker limité ; restauration DB ; wizard vierge complet ; conformité compose Forgeyard.

## Wizard final attendu

Code one-shot des logs ; admin ; Discord facultatif ; dossiers lecture+nombre fichiers vus ; test endpoint/modèle embeddings ; génération jeton worker et commande docker run avec test connexion ; vérification token HF/conditions ; récapitulatif. Secrets jamais renvoyés après saisie ni journalisés. Signaler les appels sortants Discord/HF. Réglages avancés modèles/seuils/parallélisme dans admin.

## Questions laissées ouvertes par l'utilisateur

- Nom définitif : ne pas trancher seul.
- Titres réels des pistes Medal : demander un échantillon au moment utile, ne pas inventer.
- Modèles définitifs et seuils vocaux : benchmark.

## Hors V1

Chatbot/outils, keyframes, VLM/analyse visuelle, résumés et événements de jeu, séparation vocale fonctionnelle, agent desktop amis, stockage multi-utilisateur avancé, enregistreur maison.

## Skills de la conversation

L'utilisateur a invoqué `grill-with-docs`, situé `/Users/thomasboisson/.agents/skills/grill-with-docs/SKILL.md`, qui demande l'application de `grilling` et `domain-modeling`. Ils ont été lus ; le plan et les trois décisions ont ensuite été validés. Ne pas rouvrir l'interview sur les décisions déjà actées. Le glossaire doit rester un glossaire de termes métier, sans détails d'implémentation ; les ADR suivent les exigences explicites utilisateur ci-dessus. Utiliser les skills disponibles applicables conformément aux instructions de la nouvelle session.
