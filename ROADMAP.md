# Receipts — Roadmap V1

Receipts indexera des clips de jeu pour retrouver ce qui a été dit, qui l'a dit et quand, puis ouvrir le bon clip au bon instant. Le nom est provisoire. Cette roadmap reprend les phases et contraintes du handoff ; elle ne remplace pas leurs futures specs détaillées.

## État actuel

- Phase 0 : implémentée et vérifiée localement ; [bilan et mesures](docs/phase-0-report.md) disponibles. Les issues #1 à #7 sont fermées. La CI distante, publication et validation NAS ne sont pas exécutées.
- Phase 1 : implémentée et vérifiée localement ; [bilan](docs/phase-1-report.md) disponible. Validation utilisateur de fin de phase reçue.
- Phase 2 : implémentée et vérifiée localement ; [bilan](docs/phase-2-report.md) disponible. La validation utilisateur de fin de phase est attendue.
- Phases 3 à 7 : non implémentées.
- Aucun média réel traité, aucun déploiement effectué, aucun code ni image publié. Le dépôt public et le suivi des issues existent sur GitHub.

## Règles de progression

Une phase à la fois. Chaque phase se termine par des tests verts, un README à jour et un bilan du réalisé, du restant et des écarts. Attendre la validation utilisateur avant la suivante.

Avant de reprendre le code de phase 0, faire valider sa spec. Pour les phases suivantes, détailler la spec au moment de les aborder, sans rouvrir les décisions déjà actées sans motif concret.

Aucun scan ni traitement de la vraie bibliothèque avant validation du benchmark de phase 7. Jusqu'alors, utiliser uniquement des données synthétiques ou des fixtures dédiées. Le benchmark sur un échantillon réel nécessite lui-même la validation utilisateur avant tout lancement ; son résultat conditionne le batch complet.

## Phases

| Phase | Livrable | Validation principale |
| --- | --- | --- |
| **0 — Installation et accès** | Socle Go/SPA, PostgreSQL, installation initiale, comptes locaux, approbation, Discord facultatif, secrets chiffrés, contrat HTTP, CI et déploiement Forgeyard. | Installation vierge, concurrence premier administrateur, permissions, Discord simulé, migrations et SPA embarquée, empreinte du socle documentée. |
| **1 — Ingestion et bibliothèque** | Sources dossier et upload reprenable, détection de fichiers stables, métadonnées, dédoublonnage SHA-256, provenance, bibliothèque et streaming Range authentifié. | Fichier incomplet ou illisible, doublon, renommage, disparition et permissions ; originaux toujours conservés. |
| **2 — File de tâches et worker** | Jobs PostgreSQL, leases et heartbeat, reprises et retries, worker Python via HTTPS, jeton dédié limité et révocable, pause/reprise et parallélisme. | Crash et reprise sans doublon, lease expiré, dead/relance, résultats transactionnels, worker sans accès direct à PostgreSQL. |
| **3 — Transcription et recherche textuelle** | Préparation audio adaptée aux pistes, transcription française avec jargon et timestamps par mot, recherche exacte, FTS et floue, ouverture au timestamp. | Qualité sur fixtures, sélection des pistes, comptages SQL exacts et couverture des clips non traités. |
| **4 — Recherche sémantique** | Embeddings CPU NAS, pgvector, recherche hybride RRF, filtres, changement de modèle avec reconstruction ciblée et bascule d'index. | Pertinence sur fixtures, lexical maintenu si embeddings indisponibles, cohérence modèle/version et absence de retraitement vidéo. |
| **5 — Voix** | Locuteurs locaux, personnes, empreintes et échantillons validés, groupes d'inconnues, validation humaine, historique et confidentialité. | Concurrence des validations, propagation sans PC ni vidéo, ambiguïtés, suppression des données vocales ; activation automatique réservée à l'admin après mesures. |
| **6 — Corrections** | Édition des transcriptions et attributions, historique et réindexation ciblée. | Une correction ne réindexe que les segments concernés ; auteur et changements sont traçables. |
| **7 — Benchmark et durcissement** | Échantillon représentatif de 200–500 clips après accord, mesures PC/NAS, qualité transcription/voix/recherche, reprise, sécurité et restauration. | Rapport go/no-go validé avant batch réel complet, extrapolation à 1 To avec hypothèses explicites. |

## Détails à préserver dans les prochaines specs

### Phase 0

La [spec de phase 0](docs/specs/phase-0.md) détaille le périmètre et les tests. Le wizard commence par l'administrateur et Discord facultatif ; les étapes des phases futures sont clairement reportées.

Découpage validé, suivi dans les sous-issues de la spec :

| Ticket | Bloqué par |
| --- | --- |
| [#2 — Démarrer une instance vierge](https://github.com/thomas-bsn/snitch/issues/2) | Aucun |
| [#3 — Installer le premier admin et se connecter](https://github.com/thomas-bsn/snitch/issues/3) | #2 |
| [#4 — Inscrire et approuver les utilisateurs](https://github.com/thomas-bsn/snitch/issues/4) | #3 |
| [#5 — Configurer Discord et protéger ses secrets](https://github.com/thomas-bsn/snitch/issues/5) | #3 |
| [#6 — Se connecter avec Discord et lier un compte](https://github.com/thomas-bsn/snitch/issues/6) | #4, #5 |
| [#7 — Distribuer une installation open source vérifiée](https://github.com/thomas-bsn/snitch/issues/7) | #4, #6 |

Chaque ticket inclut ses tests et sa documentation. Les dépendances sont enregistrées nativement dans GitHub ; traiter un ticket seulement après résolution de ses blocages.

### Phase 1

Surveillance de dossiers et réconciliation légère taille/mtime, sans rehash systématique des fichiers connus. Les fichiers doivent être stables et valides au probe avant ingestion ; exclusions configurables, retries puis état illisible. Un contenu peut avoir plusieurs emplacements sans multiplier les clips. Les originaux ne sont jamais modifiés, déplacés, renommés ou supprimés. Un original disparu conserve son index et sa transcription, avec état explicite et alerte en cas de disparitions nombreuses.

Conserver provenance, pistes, codecs, durée et date d'enregistrement. Prévoir propriétaire et visibilité private/shared, avec bibliothèque partagée en V1. L'upload n'utilise jamais le nom fourni comme chemin. L'accès aux dossiers hôte reste tributaire d'un montage en lecture seule réellement pris en charge par Forgeyard ; ne pas en inventer la configuration.

### Phase 2

Acquisition serveur transactionnelle, tentatives bornées, lease et heartbeat, récupération après expiration, retry exponentiel, dead et relance administrateur. Résultats remplacés sans doublon ; version modèle/configuration par étape pour invalider seulement l'étape concernée. Le worker charge ses modèles une seule fois et enchaîne les traitements sous la contrainte des 12 Go de VRAM. Aucun retraitement global nocturne.

### Phase 3

Préparation WAV 16 kHz mono sur le worker, pistes vocales conservées et piste jeu ignorée seulement lorsqu'elle est identifiable. Demander un échantillon des titres Medal au moment utile. Langue française, vocabulaire personnalisable et VAD. Départager les modèles de transcription au benchmark. Recherche française avec unaccent et pg_trgm ; résultats sûrs/approximatifs et leurs listes issus de SQL.

### Phase 4

Même modèle/version d'embeddings pour indexation et requêtes, endpoint compatible embeddings OpenAI. Les modèles restent configurables et non définitifs avant benchmark. Index halfvec HNSW, fusion RRF et filtres par locuteur, date, source et état. L'ancien index reste utilisable pendant la reconstruction. Recherche disponible PC éteint ; en panne embeddings, recherche lexicale maintenue.

### Phase 5

Distinguer locuteur local, personne, échantillon validé et attribution. La piste micro dédiée est attribuée à son propriétaire sans diarisation. Une voix inconnue ne bloque pas le pipeline ; deux voix proches ne sont pas fusionnées sans validation. Les extraits sont écoutables par les utilisateurs connectés.

La première validation fait foi sous contrainte en base ; les suivantes voient l'attribution et son auteur et peuvent la corriger explicitement. Historique complet et propagation par rapprochement en base, sans retraitement vidéo.

Objectif précisé par l'utilisateur : identifier les personnes sur un lot initial de clips représentatifs, puis attribuer automatiquement les occurrences suffisamment fiables dans le reste de la bibliothèque. La confirmation initiale concerne la constitution et l'évaluation des références de chaque personne, pas une obligation de confirmer chaque occurrence de chaque clip. Une fois la qualité mesurée et l'automatisation activée par l'administrateur, présenter surtout les cas inconnus ou ambigus à vérifier. Les critères et le volume de calibration restent à déterminer au benchmark ; aucun nombre d'extraits ne garantit la qualité à lui seul.

Les échantillons validés et les corrections humaines enrichissent les références vocales. Cela ne suppose pas de réentraîner le modèle. Les attributions automatiques restent distinctes des validations humaines et ne deviennent pas d'elles-mêmes de nouveaux échantillons validés. Vérifier au benchmark qu'un ensemble de références évalué sur des clips distincts permet effectivement de réduire la vérification manuelle. Voir [ADR 0008](docs/adr/0008-identification-progressive.md).

Une personne avec compte voit les clips où elle est identifiée et peut supprimer son profil vocal, ses empreintes et échantillons ; les attributions correspondantes redeviennent inconnues.

### Phase 6

Toute correction conserve ancienne valeur, nouvelle valeur, auteur et date. Réindexation des seuls segments concernés, sans relancer la vidéo entière.

### Phase 7

Mesurer distributions de temps par étape, clips/heure et concurrence worker 1–4, pics RAM/VRAM/CPU, WER sur références humaines, qualité de diarisation et d'identification, pertinence des requêtes, taille et mémoire des index, latence NAS, erreurs et reprise. Aucun débit ni durée pour 1 To promis avant ces mesures. Tester restauration de la base et des secrets avec la clé conservée séparément. Demucs peut être évalué au benchmark, sans devenir une fonction V1.

## Décisions transversales

- NAS Intel N100, 8 Go partagés, sans GPU ; budget total application d'environ 2–3 Go à mesurer.
- Monolithe Go et SPA embarquée ; PostgreSQL avec pgvector ; service d'embeddings CPU. Aucun Node en production, Redis, RabbitMQ ou base vectorielle séparée.
- Worker Python/CUDA sur PC RTX 4070 12 Go ; API HTTPS uniquement, aucun accès direct à PostgreSQL.
- OpenAPI comme contrat unique ; tests PostgreSQL réel via Testcontainers, tests unitaires et parcours navigateur.
- Secrets chiffrés en base avec clé maître externe. Sauvegarde PostgreSQL et clé conservées séparément ; vidéos non sauvegardées par l'application.
- Fournisseurs et modèles remplaçables ; comptages toujours issus de la base.
- Une décision technique non triviale est documentée par un ADR avec contexte, choix, alternatives et conséquences.

## Choix laissés ouverts

Nom définitif, titres réels des pistes Medal, modèles définitifs et seuils vocaux. Ne pas les considérer comme décidés par les brouillons de code.

## Hors V1

Chatbot, keyframes, analyse visuelle/VLM, résumés et événements de jeu, séparation vocale comme fonctionnalité, agent desktop amis, stockage multi-utilisateur avancé et enregistreur maison.

## Documents

- [Spec détaillée de phase 0](docs/specs/phase-0.md)
- [Handoff historique](HANDOFF.md)
- [Glossaire métier](GLOSSARY.md)
