# Receipts — Phase 1 : ingestion et bibliothèque

## Problem Statement

L'administrateur a des clips dans des dossiers et les utilisateurs peuvent vouloir en importer depuis leur ordinateur. Sans ingestion fiable, une copie partielle peut être prise pour un clip complet, un même contenu peut apparaître plusieurs fois, et un fichier déplacé ou temporairement indisponible peut faire perdre son historique. Les membres n'ont pas encore de bibliothèque où parcourir et lire les clips auxquels ils ont accès.

## Solution

Ajouter deux façons d'apporter des clips à la bibliothèque : surveiller des dossiers accessibles en lecture seule et envoyer des fichiers par upload reprenable. Receipts attend qu'un fichier soit stable, vérifie qu'il s'agit d'une vidéo lisible, extrait ses métadonnées, calcule son SHA-256 et rattache ses emplacements au clip correspondant. Un fichier incomplet ou illisible reste visible avec un état compréhensible, sans interrompre l'ingestion des autres.

Les utilisateurs connectés peuvent parcourir la bibliothèque partagée, lire un clip et retrouver ses différentes provenances. L'accès aux originaux est authentifié et prend en charge la lecture vidéo par plages HTTP afin que la navigation temporelle fonctionne. Les originaux ne sont jamais modifiés, renommés, déplacés ni supprimés par Receipts.

## User Stories

1. En tant qu'administrateur, je veux enregistrer un dossier source parmi les chemins rendus accessibles en lecture seule à Receipts, afin que ses clips soient découverts sans copier les originaux.
2. En tant qu'administrateur, je veux voir si une source est accessible et quand elle a été réconciliée pour la dernière fois, afin de distinguer un dossier vide d'un montage indisponible.
3. En tant qu'administrateur, je veux que les changements de fichiers soient détectés rapidement et qu'une réconciliation périodique complète rattrape les événements manqués, afin que la bibliothèque reste à jour.
4. En tant qu'administrateur, je veux exclure des motifs configurables de fichiers temporaires, cachés ou annexes, afin d'éviter de traiter les fichiers qui accompagnent les clips.
5. En tant qu'utilisateur, je veux qu'un clip en cours d'écriture soit ignoré jusqu'à ce que sa taille et sa date de modification soient stables, afin qu'une copie incomplète n'apparaisse pas comme prête.
6. En tant qu'utilisateur, je veux que Receipts vérifie qu'un fichier contient une vidéo lisible avant de l'ajouter à la bibliothèque, afin d'éviter les entrées inutilisables.
7. En tant qu'utilisateur, je veux que la découverte d'un fichier illisible ne bloque pas les autres fichiers de la source, afin de conserver une ingestion robuste.
8. En tant qu'administrateur, je veux voir les fichiers en attente de stabilité, à réessayer ou illisibles, avec une raison exploitable, afin de corriger un problème sans examiner les logs.
9. En tant qu'utilisateur, je veux que l'ingestion calcule une empreinte SHA-256 du contenu stable, afin de reconnaître le même clip à plusieurs emplacements.
10. En tant qu'utilisateur, je veux qu'un même contenu ne crée qu'un seul clip même s'il est découvert dans plusieurs dossiers ou reçu par upload, afin d'éviter les doublons dans la bibliothèque.
11. En tant qu'utilisateur, je veux que le déplacement ou renommage d'un fichier stable ajoute son nouvel emplacement au clip existant, afin de conserver son historique sans créer un second clip.
12. En tant qu'administrateur, je veux que les fichiers déjà connus ne soient pas rehashés à chaque réconciliation lorsqu'ils n'ont pas changé de taille ou de date de modification, afin de limiter les lectures disque.
13. En tant qu'utilisateur, je veux qu'un fichier modifié au même emplacement soit vérifié à nouveau, afin que la bibliothèque reflète le nouveau contenu sans écraser l'ancien clip.
14. En tant qu'utilisateur, je veux pouvoir envoyer un fichier vidéo par morceaux et reprendre un upload interrompu, afin que les gros fichiers ne nécessitent pas un transfert unique ininterrompu.
15. En tant qu'utilisateur, je veux voir les uploads inachevés et les reprendre ou les annuler, afin de savoir où en est un transfert interrompu.
16. En tant qu'utilisateur, je veux que le nom envoyé par mon navigateur ne soit jamais interprété comme un chemin sur le serveur, afin de protéger les fichiers et les autres utilisateurs.
17. En tant qu'administrateur, je veux définir une limite de taille d'upload, afin de contrôler l'espace persistant consommé par les fichiers importés.
18. En tant qu'utilisateur, je veux que seul un upload complet et vérifié soit présenté comme un clip lisible, afin de ne jamais lire un fichier partiel.
19. En tant qu'utilisateur, je veux que chaque clip garde la date de découverte, les sources et emplacements connus, ainsi que l'identité du membre ayant effectué un upload, afin de comprendre sa provenance.
20. En tant qu'utilisateur, je veux consulter la durée, les codecs et les pistes audio/vidéo détectés, avec les titres et langues disponibles, afin de connaître le contenu technique du clip.
21. En tant qu'utilisateur, je veux que la date d'enregistrement soit extraite des métadonnées lorsqu'elle existe et sinon estimée depuis la date du fichier, afin de pouvoir situer les clips dans le temps.
22. En tant qu'utilisateur connecté, je veux parcourir la bibliothèque partagée avec pagination et filtres simples par date, source, propriétaire et état, afin de retrouver un clip sans une recherche textuelle qui appartient à une phase ultérieure.
23. En tant qu'utilisateur, je veux ouvrir un clip et naviguer dans sa vidéo, afin de vérifier son contenu et accéder au bon instant.
24. En tant qu'utilisateur, je veux que la lecture prenne en charge les requêtes HTTP Range et le déplacement dans la vidéo, afin que le navigateur ne télécharge pas nécessairement tout le fichier avant lecture.
25. En tant qu'utilisateur, je veux que les clips et sources puissent porter un propriétaire et une visibilité privée ou partagée, afin que les bases d'accès de la bibliothèque soient en place dès V1.
26. En tant qu'utilisateur, je veux que les clips partagés soient visibles aux membres connectés et que mes clips privés restent accessibles à moi et aux administrateurs, afin que la visibilité soit appliquée côté serveur.
27. En tant qu'administrateur, je veux que la bibliothèque soit partagée par défaut dans V1, afin que les membres puissent consulter les clips communs sans réglage supplémentaire.
28. En tant qu'utilisateur, je veux que l'original reste lisible depuis un autre emplacement si une copie identique existe encore, afin qu'une disparition locale ne rende pas le clip indisponible à tort.
29. En tant qu'utilisateur, je veux que le clip et ses métadonnées restent dans la bibliothèque si tous ses originaux deviennent introuvables, afin de préserver l'historique.
30. En tant qu'administrateur, je veux être averti si une réconciliation réussie constate un nombre inhabituellement élevé de disparitions, afin de détecter une source vidée ou un montage erroné.
31. En tant qu'administrateur, je veux qu'une source inaccessible ne fasse pas passer en masse ses clips pour disparus, afin qu'une panne de montage ne dégrade pas l'état de la bibliothèque.
32. En tant qu'administrateur, je veux pouvoir désactiver une source sans effacer les fichiers, clips ou provenances déjà connus, afin de suspendre sa surveillance sans perte d'historique.
33. En tant que membre, je veux que toute lecture, liste et métadonnée de clip exige une session et respecte les permissions du clip, afin qu'un lien média direct ne contourne pas l'accès.
34. En tant que contributeur, je veux que le contrat HTTP de l'ingestion et de la bibliothèque soit publié dans OpenAPI et génère des clients cohérents, afin que l'interface et le serveur partagent les mêmes règles.
35. En tant qu'exploitant, je veux que les uploads et originaux importés soient conservés dans un volume persistant distinct de PostgreSQL, afin qu'un redémarrage de l'application ne les efface pas.
36. En tant que propriétaire du projet, je veux utiliser uniquement des médias de test synthétiques ou dédiés avant le benchmark de phase 7, afin qu'aucune véritable bibliothèque ne soit scannée prématurément.

## Implementation Decisions

- Phase 1 ajoute les sources dossier, l'upload reprenable, la bibliothèque et la lecture. Elle ne crée pas la file de tâches ni le worker de phase 2 ; l'ingestion et l'extraction de métadonnées restent dans les limites du service Receipts.
- Une source dossier référence un répertoire déjà accessible au processus et autorisé par les racines de configuration. L'administrateur peut enregistrer une source et la désactiver depuis Receipts. Les dossiers sont montés en lecture seule.
- Les événements filesystem accélèrent la détection, mais ne constituent pas la seule source de vérité : une réconciliation périodique compare chemins relatifs, taille et date de modification.
- La stabilité est établie par plusieurs observations concordantes de taille et de date de modification pendant une durée configurée, puis par une vérification vidéo réussie. Les durées, périodicités et backoff sont documentés et configurables sans être des seuils produit imposés ici.
- Les fichiers temporaires, cachés et annexes sont filtrés par des motifs configurables. L'extraction utilise ffprobe sans transcodage. Les formats acceptés sont les fichiers que le probe identifie comme contenant au moins une piste vidéo lisible.
- Un fichier qui change pendant le probe ou le calcul de SHA-256 retourne à l'état en attente ; ses métadonnées ne sont pas validées à partir d'une lecture incohérente.
- Un SHA-256 unique identifie le contenu du clip. Les emplacements sont des enregistrements séparés et peuvent pointer vers une seule entité clip. Une nouvelle provenance candidate doit être vérifiée avant rattachement. Les fichiers connus ne sont pas rehashés à chaque cycle si taille et date de modification n'ont pas changé ; une nouvelle copie ou un fichier modifié est hashé après stabilité.
- Le renommage ou déplacement dans un dossier ne peut être rapproché d'un clip existant qu'après vérification du nouveau contenu. La disparition d'un chemin n'efface ni le clip ni ses métadonnées. Le clip reste lisible si une autre provenance est encore disponible ; sinon son état indique que l'original est introuvable.
- Une source inaccessible ou une réconciliation interrompue ne marque aucun chemin comme disparu. Les fortes disparitions détectées pendant une réconciliation complète et réussie déclenchent une alerte administrateur ; elles ne provoquent pas une suppression automatique des clips.
- L'upload est découpé en sessions identifiées par le serveur, stocké temporairement sous un volume nommé et reprenable après reconnexion. Chaque morceau est idempotent ou son offset attendu est vérifié ; un upload concurrent ne peut pas écrire les mêmes octets deux fois. L'utilisateur peut reprendre ou annuler ses propres sessions. Les sessions abandonnées sont purgées après une période documentée.
- Le nom de fichier d'origine est une métadonnée d'affichage nettoyée, jamais un chemin disque. Le stockage utilise des identifiants générés par le serveur et vérifie les limites et chemins avant toute lecture ou écriture.
- Une limite d'upload configurable par l'administrateur est vérifiée avant et pendant le transfert. La complétion entraîne stabilité, probe et hash avant publication comme clip ; un contenu identique à un clip connu ajoute une provenance au lieu de créer un doublon.
- Les métadonnées conservées incluent les dates de découverte et d'enregistrement, codecs, durée, pistes et leurs titres/langues lorsqu'ils existent, provenance et auteur d'upload. Une date d'enregistrement absente des métadonnées est indiquée comme estimée depuis la date de modification du fichier.
- Les clips et sources portent un propriétaire et une visibilité private/shared. Les clips et la bibliothèque créée par les sources sont partagés par défaut dans V1. La politique est appliquée à chaque requête API et de streaming : un clip privé n'est visible qu'à son propriétaire et aux administrateurs.
- La bibliothèque fournit une liste paginée, des filtres simples par date, source, propriétaire et état, les métadonnées d'un clip et un lecteur vidéo. Elle ne fournit pas encore de recherche plein texte, sémantique, transcription, locuteur ou étiquetage par modèle.
- Le streaming authentifié prend en charge les requêtes Range HTTP et les réponses partial-content nécessaires aux navigateurs ; les plages invalides suivent le contrat HTTP. Les réponses ne divulguent pas de chemin système.
- Les migrations PostgreSQL restent versionnées et séparées du démarrage API. Le schéma conserve clips, sources, emplacements, métadonnées, uploads et états d'ingestion, avec contraintes d'unicité/transactions protégeant les dédoublements concurrents. OpenAPI demeure la source du contrat HTTP.
- Le stockage persistant des uploads et copies reçues est un volume nommé distinct de la base. La sauvegarde PostgreSQL existante ne sauvegarde pas ces médias ; l'exploitation est documentée comme telle.
- Le compose Forgeyard n'invente pas de bind mount. Le scan de dossier n'est opérationnel que si l'hébergeur rend réellement un chemin accessible au conteneur, en lecture seule ; cette limite est affichée et documentée. Les tests et le développement local utilisent uniquement des répertoires temporaires/fixtures dédiés montés en lecture seule.
- Le wizard progressif présente l'ajout de dossiers et l'état des sources, ainsi que l'upload/bibliothèque, sans prétendre réaliser les étapes IA qui seront ajoutées plus tard.

## Testing Decisions

Un bon test passe par un comportement observable et les points d'entrée publics ; il ne vérifie pas les appels internes ni la structure choisie. La couture principale validée pour cette spec est l'API réelle avec PostgreSQL/pgvector via Testcontainers, des médias synthétiques et des dossiers temporaires. Playwright couvre les parcours utilisateur visibles avec la vraie API.

1. Tester les sources et la réconciliation sur de vrais répertoires temporaires : racine absente ou inaccessible, source désactivée, fichiers exclus, fichiers dont taille/date changent, fichier stable, événement manqué rattrapé par réconciliation, et échec d'un fichier qui n'interrompt pas les suivants.
2. Tester la validation vidéo avec fixtures dédiées contenant une piste vidéo valide, une vidéo tronquée/illisible, un fichier sans vidéo et un fichier modifié pendant son traitement. Vérifier les états et erreurs observables, sans dépendre du détail interne du probe.
3. Tester SHA-256 avec deux chemins et un upload au contenu identique : un seul clip, plusieurs provenances, pas de rehash d'un chemin connu inchangé, nouveau hash si son contenu change, et rattachement après renommage/déplacement.
4. Tester les disparitions après une réconciliation complète réussie, un montage devenu inaccessible, une réconciliation partielle et une disparition massive. Vérifier que l'historique est conservé, que les alertes sont visibles, et que l'indisponibilité d'une source ne marque pas ses clips absents.
5. Tester l'upload via HTTP : chunks incomplets et hors ordre, reprise après interruption, session concurrente, annulation, dépassement de taille, nom contenant traversée/Unicode/caractères réservés, purge d'une session expirée et refus de publier avant complétion/probe. Vérifier que seuls les octets attendus sont reçus et que les fichiers du serveur restent inaccessibles.
6. Tester les contraintes concurrentes de dédoublonnage quand deux sources ou uploads finissent au même moment ; une seule entité clip est créée et toutes les provenances sont conservées.
7. Tester les permissions via l'API : anonyme refusé, membre autorisé pour bibliothèque partagée, accès privé du propriétaire et de l'admin, autre membre refusé, visibilité appliquée aux listes/détails et au média direct, sources réservées à l'admin.
8. Tester le streaming sur l'API réelle avec réponses complètes et Range : début, plage ouverte, suffixe, seek, fichier disparu, plage invalide, clip privé et session absente. Vérifier statut, Content-Range, longueur et octets sans divulguer le chemin disque.
9. Parcours Playwright : ajouter une source disponible, voir sa réconciliation et ses clips ; uploader un média synthétique, rafraîchir/reprendre un upload interrompu, filtrer la bibliothèque, ouvrir et déplacer la vidéo, et afficher les états en attente/illisible/introuvable.
10. Vérifier la conformité OpenAPI et la reproductibilité des artefacts Go/TypeScript/Python, les migrations vierges/rejouées et les permissions contre PostgreSQL réel. Les tests ne contactent aucun fournisseur externe.
11. Utiliser uniquement fixtures dédiées et médias synthétiques ; aucun chemin de la vraie bibliothèque, aucun scan du NAS et aucun traitement batch réel. Le benchmark réel reste soumis à sa validation préalable en phase 7.

Références de tests existantes : les tests de phase 0 utilisent déjà PostgreSQL/pgvector Testcontainers au niveau HTTP, plusieurs suites Playwright sur la vraie API, OpenAPI généré, migrations embarquées et un smoke Compose. La phase 1 prolonge ces coutures plutôt que d'introduire une base de test différente.

## Out of Scope

- File de tâches, lease, heartbeat, retries de jobs, worker Python/GPU, accès HTTPS du worker et parallélisme : phase 2.
- Extraction audio, transcription, recherche textuelle, vocabulaire, timestamps de parole et sélection de pistes pour transcription : phase 3.
- Embeddings, recherche vectorielle/hybride et modèle d'embedding : phase 4.
- Diarisation, identité de personne, empreintes vocales, échantillons, validation humaine et attribution automatique : phase 5.
- Édition des transcriptions/attributions et historique de correction : phase 6.
- Parcours, migration, montée de version, alerting de production et tests de restauration complets sur Forgeyard/NAS ; benchmark représentatif, vraie bibliothèque, batch 1 To et go/no-go : phase 7 et autorisations correspondantes.
- Réencodage, conversion, déplacement, renommage ou suppression des originaux par Receipts.
- Recherche libre par texte ou sémantique, résumés, keyframes, analyse visuelle, chatbot, événements de jeu et enregistreur maison.

## Further Notes

- Le réseau Forgeyard ne prend actuellement pas en charge les montages hôte nécessaires aux sources dossier. L'upload reste utilisable ; la surveillance d'un dossier sur NAS attend une capacité de montage officiellement disponible et validée. Aucune syntaxe de montage n'est inventée dans cette spec.
- Le volume médias de l'application n'est pas couvert par la sauvegarde PostgreSQL Forgeyard. La procédure de sauvegarde/restore complète demeure hors phase 1 et doit être résolue dans le durcissement de phase 7.
- Cette spec autorise la préparation de phase 1, pas le scan ni le traitement de médias réels. Le code de phase 1 et tous ses tests doivent se limiter à des fixtures dédiées jusqu'à la validation séparée du benchmark de phase 7.
- À la fin de la phase, README, documentation d'exploitation et bilan du réalisé/restant/écarts seront mis à jour ; aucune phase 2 ne commencera avant validation utilisateur.
