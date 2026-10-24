# Receipts — Phase 0 : installation, accès et socle de déploiement

## Problem Statement

L'administrateur veut héberger Receipts sur un NAS aux ressources limitées, puis ouvrir un accès contrôlé à ses amis. Il lui faut une installation vérifiable, des comptes sécurisés, une configuration récupérable après restauration et un déploiement compatible avec Forgeyard avant d'ajouter des clips ou du traitement IA.

Sans ce socle, les prochaines phases risquent de dépendre d'une installation manuelle fragile, de secrets mal conservés ou de règles d'accès incohérentes. Cette phase ne doit ni parcourir ni traiter la vraie bibliothèque.

## Solution

Livrer une première application accessible dans un navigateur, installable sur une base vierge, avec création du premier administrateur, comptes locaux, approbation des inscriptions et connexion Discord facultative. L'administrateur peut consulter les étapes disponibles et celles qui arriveront dans les phases suivantes.

L'interface est servie par un binaire Go autonome. Le déploiement sépare les migrations de l'API et conserve les données dans PostgreSQL avec pgvector. Des vérifications automatisées couvrent les parcours réels, les permissions et le démarrage. La documentation permet d'installer, développer et préparer la restauration sans promettre de performances non mesurées.

## User Stories

1. En tant qu'administrateur, je veux installer Receipts sur une base vierge, afin de disposer d'un espace prêt pour les phases suivantes.
2. En tant qu'administrateur, je veux obtenir un code temporaire d'installation dans les logs, afin de réserver la création du premier compte à la personne qui exploite le serveur.
3. En tant qu'administrateur, je veux que ce code expire et ne serve qu'une fois, afin qu'un ancien code ne permette pas de reprendre l'installation.
4. En tant qu'administrateur, je veux qu'une seule création de premier administrateur aboutisse même en cas de demandes simultanées, afin d'éviter des comptes privilégiés inattendus.
5. En tant qu'administrateur, je veux que l'installation initiale se ferme après sa réussite, afin que personne ne puisse la rejouer.
6. En tant qu'administrateur, je veux connaître la procédure à suivre si le code expire ou si l'installation est interrompue, afin de pouvoir terminer sans manipuler manuellement la base.
7. En tant qu'administrateur, je veux créer mon compte avec un mot de passe personnel, afin de ne pas dépendre d'identifiants par défaut.
8. En tant qu'utilisateur, je veux me connecter avec mon compte local, afin d'accéder aux fonctions réservées aux membres.
9. En tant qu'utilisateur, je veux recevoir une explication exploitable en cas de connexion refusée, afin de comprendre que mes identifiants ou mon approbation peuvent être en cause sans exposer les comptes existants.
10. En tant qu'utilisateur, je veux me déconnecter, afin que la session courante ne permette plus d'accéder à mon compte.
11. En tant qu'utilisateur, je veux qu'une session expirée soit refusée, afin de ne pas conserver indéfiniment un accès oublié.
12. En tant que visiteur, je veux demander un compte local, afin de rejoindre l'espace sans obtenir automatiquement un accès approuvé.
13. En tant que demandeur, je veux être informé lorsque mon inscription attend une approbation, afin de savoir pourquoi je ne peux pas encore me connecter.
14. En tant qu'administrateur, je veux voir les inscriptions en attente, afin de décider qui peut rejoindre l'espace.
15. En tant qu'administrateur, je veux approuver une inscription, afin que la personne puisse ensuite se connecter.
16. En tant qu'administrateur, je veux que l'approbation soit requise par défaut, afin de garder le contrôle des nouveaux accès.
17. En tant qu'administrateur, je veux pouvoir configurer ce réglage d'inscription, afin d'appliquer la même politique aux comptes locaux et Discord.
18. En tant qu'utilisateur ordinaire, je veux que les réglages d'administration soient réservés aux administrateurs, afin de préserver la configuration collective.
19. En tant qu'administrateur, je veux pouvoir utiliser Receipts sans Discord, afin que ce fournisseur reste facultatif.
20. En tant qu'administrateur, je veux configurer la connexion Discord depuis l'application, afin d'activer cette possibilité sans exposer le secret au navigateur après enregistrement.
21. En tant qu'administrateur, je veux connaître l'URL de retour nécessaire à la configuration Discord, afin de terminer l'intégration correctement.
22. En tant qu'utilisateur, je veux savoir que la connexion Discord ouvre un service externe et ne demande que mon identité, afin de comprendre l'autorisation accordée.
23. En tant que nouvel utilisateur Discord, je veux suivre la politique d'approbation des inscriptions, afin que ce mode de connexion ne contourne pas les règles d'accès.
24. En tant qu'utilisateur déjà connecté, je veux lier explicitement mon identité Discord à mon compte, afin de choisir cette association.
25. En tant qu'utilisateur, je veux que mon pseudo ne déclenche jamais de fusion automatique de comptes, afin de conserver une identité distincte de celles des autres membres.
26. En tant qu'utilisateur, je veux qu'une identité Discord déjà liée ne soit pas récupérée par un autre compte, afin de préserver la propriété de cette association.
27. En tant qu'utilisateur, je veux qu'un retour Discord invalide, expiré ou déjà utilisé soit refusé, afin de protéger ma connexion.
28. En tant qu'utilisateur, je veux qu'un visiteur anonyme ne puisse pas appeler directement les fonctions privées, afin que la protection ne dépende pas seulement de l'affichage de l'interface.
29. En tant qu'utilisateur, je veux que les actions effectuées avec ma session soient protégées contre les demandes forgées par un autre site, afin de conserver le contrôle de mon compte.
30. En tant qu'administrateur, je veux limiter les tentatives de connexion abusives, afin de protéger les comptes et les ressources du NAS.
31. En tant qu'administrateur, je veux que les mots de passe et jetons de session soient conservés sous une forme non réutilisable directement, afin de réduire l'impact d'une fuite de la base.
32. En tant qu'administrateur, je veux que les secrets de configuration soient chiffrés en base avec une clé conservée séparément, afin qu'une sauvegarde de la base seule ne les révèle pas.
33. En tant qu'administrateur, je veux que les secrets ne soient pas renvoyés après saisie ni écrits dans les logs, afin de ne pas les diffuser pendant l'exploitation.
34. En tant qu'administrateur, je veux comprendre que restaurer les secrets exige la base et la clé maître, afin de préparer une sauvegarde réellement utilisable.
35. En tant qu'administrateur, je veux voir les étapes d'installation reportées aux phases suivantes, afin de ne pas confondre le socle avec une bibliothèque déjà fonctionnelle.
36. En tant qu'administrateur, je veux que l'application reste disponible lorsque les embeddings sont indisponibles, afin que cette dépendance facultative n'empêche pas l'installation et les connexions.
37. En tant qu'exploitant, je veux distinguer un processus vivant d'une application prête à répondre, afin de détecter les problèmes de base ou de démarrage.
38. En tant qu'exploitant, je veux qu'une migration en échec bloque le démarrage de l'API, afin d'éviter une application fonctionnant sur un schéma incompatible.
39. En tant qu'exploitant, je veux pouvoir rejouer la commande de migration sans réappliquer les changements déjà réussis, afin de redéployer de manière prévisible.
40. En tant qu'exploitant, je veux servir l'interface sans processus Node en production, afin de limiter la consommation et la complexité sur le NAS.
41. En tant qu'exploitant, je veux arrêter proprement l'application, afin de terminer les requêtes en cours et libérer les connexions.
42. En tant qu'exploitant, je veux déployer avec le sous-ensemble Compose accepté par Forgeyard, afin d'utiliser mon hébergement existant.
43. En tant qu'exploitant, je veux des images versionnées sur GHCR et des données persistantes dans des volumes nommés, afin de maîtriser les déploiements.
44. En tant qu'exploitant, je veux connaître les limites actuelles de Forgeyard pour monter les dossiers du NAS, afin de ne pas croire que l'accès aux clips est déjà disponible.
45. En tant qu'exploitant, je veux des mesures de l'empreinte du socle avec leur environnement et leurs limites, afin de préparer le déploiement sans confondre des résultats locaux avec ceux du NAS.
46. En tant que contributeur, je veux un contrat HTTP unique et des artefacts Go, TypeScript et Python cohérents, afin de faire évoluer l'application sans divergence entre clients et serveur.
47. En tant que contributeur, je veux que la CI refuse un contrat ou des artefacts générés incohérents, afin de détecter les divergences avant publication.
48. En tant que contributeur, je veux des tests sur un vrai PostgreSQL avec pgvector, afin de vérifier les contraintes et transactions du système réellement utilisé.
49. En tant que contributeur, je veux que les images ne soient publiées qu'après réussite des contrôles, afin de ne pas distribuer un socle dont les vérifications ont échoué.
50. En tant qu'administrateur, je veux qu'aucun média réel ne soit parcouru ou traité dans cette phase, afin de conserver la validation du benchmark avant toute exploitation de ma bibliothèque.
51. En tant qu'utilisateur, je veux une interface en français qui explique les étapes et les refus d'accès, afin de pouvoir utiliser l'installation sans connaissances techniques particulières.
52. En tant que propriétaire du projet, je veux un bilan de phase qui distingue réalisé, restant et écarts, afin de décider explicitement du passage à la phase suivante.

## Implementation Decisions

- Périmètre : phase 0 uniquement. La cible générale est une application open source sous licence MIT, provisoirement nommée Receipts. Le nom d'affichage et les références d'images doivent permettre un renommage ultérieur.
- Architecture : monolithe Go modulaire avec API REST et SPA Vite, React et TypeScript compilée puis embarquée. Aucun Node en production ; aucun Next.js, Redis, RabbitMQ ou stockage vectoriel séparé.
- Modules du socle : configuration, authentification, installation initiale, réglages, accès aux données et exposition HTTP. Le squelette Python ne fournit qu'une base d'intégration au contrat ; il ne traite aucun clip.
- Persistance : PostgreSQL avec pgvector, pgx pour les connexions et sqlc pour les requêtes typées. Les migrations sont versionnées et exécutées par une commande et un job distincts, jamais implicitement au démarrage de l'API.
- Schéma du socle : état d'installation et de politique d'inscription, utilisateurs avec rôle et état d'approbation, informations d'authentification locale et de liaison Discord, sessions hachées avec expiration, état temporaire OAuth et réglages chiffrés. Les règles d'unicité et la transaction d'installation protègent les demandes concurrentes.
- Installation : code temporaire à usage unique affiché dans les logs, création atomique du premier administrateur et fermeture du wizard initial après réussite. L'expiration et la récupération d'une installation interrompue doivent être documentées et testées. Les durées précises ne sont pas des décisions produit déjà validées.
- Wizard progressif : administrateur et Discord facultatif en phase 0 ; dossiers en phase 1 ; worker en phase 2 ; vérifications embeddings et Hugging Face dans les phases correspondantes. Les étapes futures sont explicitement présentées comme reportées.
- Authentification locale : Argon2id, sessions opaques hachées en base, cookies HttpOnly et SameSite=Lax, Secure en production. Protection CSRF des actions authentifiées et limitation des tentatives de connexion. Les paramètres précis de durée, de longueur et de limitation doivent être documentés lors de l'implémentation, sans être déduits du brouillon existant.
- Autorisation : rôles administrateur et utilisateur ; fonctions métier protégées par défaut ; réglages et approbation réservés aux administrateurs. Les refus sont appliqués sur le serveur, indépendamment de l'interface.
- Inscriptions : approbation administrateur requise par défaut, réglage commun aux comptes locaux et aux nouvelles inscriptions Discord. Une inscription en attente n'ouvre pas de session autorisée.
- Discord : OAuth2 avec la portée identify uniquement, validation de state et protection contre sa réutilisation. Une liaison exige une action explicite depuis un compte déjà connecté ; aucune fusion sur pseudo. L'identité liée reste unique. Les appels sortants à Discord sont signalés à l'utilisateur.
- Secrets : chiffrement authentifié en base avec la bibliothèque standard Go et une clé maître fournie par l'environnement Forgeyard. La clé est distincte de la sauvegarde PostgreSQL. Aucun secret dans le dépôt, le frontend ou les logs, sauf le code initial temporaire explicitement demandé. Après saisie, l'interface peut indiquer qu'un secret est configuré mais ne le restitue pas.
- Contrat : OpenAPI est la source de vérité et ne décrit que les fonctions livrées : santé, état et finalisation de l'installation, connexion/inscription/déconnexion, session courante, configuration et connexion/liaison Discord, réglages administrateur et approbation des utilisateurs. Les noms précis des routes ne sont pas figés par cette spec.
- Cohérence du contrat : types/serveur Go et clients TypeScript/Python sont générés ou vérifiés contre le contrat. La régénération est reproductible et la CI détecte les écarts. Les erreurs et permissions sont incluses dans les vérifications contre l'API réelle.
- Exploitation : configuration validée, logs structurés expurgés, arrêt propre, vivacité et disponibilité distinctes. Une base indisponible est signalée ; un service d'embeddings indisponible donne un état dégradé sans empêcher l'accès au socle. Aucun modèle n'est téléchargé implicitement.
- Frontières futures : aucun fournisseur ou modèle IA codé en dur dans le cœur. Le worker GPU sera distinct du NAS et n'accédera jamais directement à PostgreSQL. La recherche devra fonctionner PC éteint lorsqu'elle sera livrée.
- Déploiement NAS : quatre services prévus, PostgreSQL, migration, application et embeddings. Le worker est absent du compose NAS. Seule l'application est exposée, sur le port interne 8080.
- Forgeyard : compose racine avec images uniquement et volumes nommés. Pas de build, ports publiés, réseaux personnalisés, mode réseau spécial, fichier d'environnement externe déclaré par Compose, nom de conteneur imposé, privilèges, capacités supplémentaires, périphériques, héritage de service, bind mount, volume anonyme ou tmpfs. L'override local peut autoriser les besoins de développement et des fixtures montées en lecture seule.
- Images : GHCR, versions et références amont épinglées, licences conservées. PostgreSQL/pgvector et Ollama sont republiés en conservant leurs notices. Les versions candidates du plan doivent être vérifiées avant verrouillage ; une présence dans le brouillon ne suffit pas à les approuver.
- Ordonnancement : PostgreSQL sain avant migration ; migration terminée avec succès avant API. Le job de migration ne redémarre pas en boucle. Les contrôles CI précèdent la publication d'images.
- Ressources : cible totale NAS d'environ 2–3 Go. Enveloppes initiales proposées : application 384 Mo, PostgreSQL 512 Mo, embeddings 1536 Mo, migration transitoire 256 Mo ; PostgreSQL avec shared buffers 128 Mo, work memory 4 Mo et pool limité. Ce sont des budgets à vérifier, pas des garanties. Les limites locales doivent être explicites, car Docker Compose n'applique pas les métadonnées mémoire Forgeyard.
- Clips : configuration des racines prévue pour la suite, mais aucun scanner ni lecture de la vraie bibliothèque en phase 0. Forgeyard ne fournit pas encore les montages de dossiers hôte nécessaires ; le besoin de montage en lecture seule doit être documenté sans inventer de syntaxe.
- Documentation : licence, guide de démarrage, configuration d'exemple sans secrets, glossaire métier, décisions d'architecture, développement, déploiement, tests et préparation à la restauration. Les décisions techniques non triviales explicitent contexte, choix, alternatives et conséquences.
- Restauration : documenter la sauvegarde PostgreSQL via Forgeyard vers S3, la conservation séparée de la clé maître et les étapes de restauration. Les vidéos ne sont pas sauvegardées par Receipts ; le NAS sans miroir ne constitue pas une protection des originaux. Le test complet de restauration reste au programme de la phase 7.

## Testing Decisions

Les points de test suivants ont été validés par l'utilisateur pendant la rédaction de cette spec. Un bon test vérifie un résultat observable ou une règle d'accès ; il ne reproduit pas l'implémentation ni ne se limite à vérifier des appels internes.

1. **Point principal : API réelle avec PostgreSQL et pgvector via Testcontainers.** Tester les modules installation, comptes, sessions, autorisations, réglages et intégration Discord à travers leurs comportements HTTP. Vérifier les effets persistés lorsque c'est nécessaire pour établir une garantie : une seule création de premier administrateur, un secret non stocké en clair ou une session non réutilisable après déconnexion. Ne pas remplacer PostgreSQL par SQLite.
2. **Installation et concurrence.** Base vierge, code incorrect, expiré et rejoué, interruption de l'installation, fermeture après réussite et deux tentatives simultanées de création du premier administrateur. Une seule réussit ; aucune seconde voie de création anonyme d'administrateur ne reste ouverte.
3. **Comptes et sessions.** Connexion locale réussie et refusée, compte en attente, approbation, session expirée, déconnexion, absence de cookie et refus anonyme. Vérifier que les rôles ordinaires n'accèdent ni aux réglages ni aux approbations.
4. **Sécurité observable.** Cookies et attributs attendus en production, requêtes sans protection CSRF ou depuis une origine illégitime refusées, limitation de tentatives effective et erreurs sans fuite de secrets. Compléter par des tests unitaires ciblés d'Argon2id, de chiffrement authentifié et de validation de configuration.
5. **Discord simulé.** Utiliser un fournisseur OAuth de test, sans compte réel ni appel au service externe. Couvrir portée identify, state invalide/expiré/rejoué, erreur fournisseur, nouvelle inscription soumise à approbation, reconnexion à une identité déjà liée, liaison explicite depuis une session existante et conflit d'identité. Un pseudo identique ne fusionne jamais deux comptes.
6. **Contrat HTTP.** Vérifier les opérations livrées contre l'API réelle, succès, erreurs et permissions compris. Valider les formats d'entrée et de sortie et la reproductibilité des artefacts Go, TypeScript et Python. Une divergence bloque la CI.
7. **Point navigateur : parcours utilisateur réels.** Couvrir installation vierge, connexion/déconnexion, inscription en attente, approbation, fonctions administrateur inaccessibles aux autres rôles et parcours Discord avec fournisseur simulé. Le navigateur parle à la vraie API et au vrai PostgreSQL de test.
8. **Point déploiement : démarrage et artefacts.** Vérifier migrations sur base vierge et rejouées, échec de migration bloquant l'API, SPA embarquée servie sans Node, arrêt propre, santé, état dégradé sans embeddings et validation des clés/volumes/services exposés du compose Forgeyard. Les fixtures négatives doivent démontrer que les configurations interdites sont rejetées.
9. **Empreinte.** Mesurer le socle dans l'environnement disponible, décrire protocole et conditions, et distinguer mémoire de processus, conteneurs, stockage et éventuels services non mesurés. Ne pas extrapoler ces chiffres au NAS ni au traitement IA. Aucune mesure sur les clips réels.
10. **Référence existante.** Le projet était initialement vide. Le brouillon créé prématurément contient des tests unitaires, un début de suite API/Testcontainers et un parcours Playwright. Ils ne constituent ni une validation du socle ni une contrainte de conception : la reprise doit les confronter à cette spec. Au dernier essai, la suite d'intégration échouait au chargement du driver PostgreSQL ; aucun succès E2E complet n'a été établi.

Critères de sortie : contrôles applicables verts, contrat reproductible, build SPA/binaire/images réussi, documentation à jour, empreinte documentée et bilan du réalisé, du restant et des écarts. Tout blocage ou écart est explicite. Le passage à la phase 1 nécessite ensuite une validation utilisateur distincte.

## Out of Scope

- Ingestion, surveillance de dossiers, upload, dédoublonnage, bibliothèque de clips et lecture vidéo : phase 1.
- File de tâches, leases, jetons worker, téléchargement de médias et exécution GPU : phase 2.
- Préparation audio, transcription et recherche textuelle : phase 3.
- Embeddings de clips, indexation vectorielle et recherche hybride : phase 4. Le service d'embeddings de phase 0 reste sans modèle préchargé.
- Diarisation, empreintes, rapprochement des voix, validations et suppression des profils vocaux : phase 5.
- Corrections des transcriptions et attributions : phase 6.
- Benchmark sur 200–500 clips, validation go/no-go, traitement de la vraie bibliothèque, extrapolation à 1 To et test complet de restauration : phase 7 et autorisation correspondante.
- Choix définitif du nom, des modèles IA et des seuils vocaux.
- Création automatique d'un dépôt distant, déploiement sur le NAS ou publication immédiate d'images depuis cette conversation.
- Chatbot, analyse visuelle, keyframes, résumés de jeu, séparation vocale fonctionnelle, agent desktop et enregistreur maison : hors V1.

## Further Notes

- Cette spec synthétise les décisions validées de la conversation précédente transmises par le handoff, puis la validation des points de test donnée dans la présente conversation. Elle n'est pas une spécification détaillée des phases 1 à 7.
- Le code commencé avant la rédaction est arrêté et incomplet. Aucun média réel n'a été traité et aucun code ni image n'a été publié. Sa présence ne signifie pas que la phase 0 est achevée ou que ses choix de détail ont été acceptés.
- Le plan de phase 0 était validé ; après l'interruption de l'implémentation prématurée, aucune reprise du code ne doit être déduite de la seule création ou publication de cette spec. Attendre la validation de la spec et le feu vert de reprise.
- Le tracker du projet est GitHub, dépôt public thomas-bsn/snitch. Le dossier local n'est pas encore un dépôt Git ; les commandes utilisent explicitement la destination. Publier cette spec avec le label ready-for-agent demandé par le skill, sans lancer automatiquement l'implémentation.
- La configuration des skills désigne le tracker, les cinq labels standard et un contexte métier unique. Le nom snitch a été choisi pour le dépôt ; le nom d'affichage reste configurable.
- Les fixtures de test ne doivent contenir aucun secret réel, média privé ni accès à la bibliothèque du NAS.
