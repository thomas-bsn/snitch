# Receipts — Phase 2 : file de tâches et worker

## Problem Statement

La bibliothèque de clips est prête, mais aucun traitement long ne peut encore être distribué à un ordinateur séparé. L’application doit pouvoir confier des tâches à un worker GPU sans lui donner accès à PostgreSQL, reprendre le travail après un arrêt et éviter qu’un résultat ancien ou répété écrase un résultat valide.

## Solution

Ajouter à Receipts une file durable de tâches dans PostgreSQL et un worker Python qui ne communique avec l’application que par HTTPS avec un jeton dédié, révocable et limité. Les tâches disposent de leases renouvelables, de reprises bornées et d’un état observable par l’administrateur. Un worker peut être mis en pause ou reprendre, et sa concurrence peut être réglée sans dépendre de la longueur de la file.

Cette phase construit le contrôle d’exécution générique et son worker de base. Elle n’ajoute pas encore d’étape de traitement métier pour les clips : la préparation audio et la transcription arrivent en phase 3. Un fournisseur factice sert à éprouver le protocole et les transitions sans charger de modèle ni traiter de média réel.

## User Stories

1. En tant qu’exploitant, je veux qu’une tâche soit conservée dans PostgreSQL, afin qu’un redémarrage de l’application ou du worker ne la fasse pas disparaître.
2. En tant qu’exploitant, je veux identifier la tâche par clip, étape et version d’entrée, afin de savoir quel calcul est demandé et avec quelles entrées.
3. En tant qu’exploitant, je veux qu’une même étape et une même version d’entrée ne produisent pas de tâches actives en double, afin que les déclenchements répétés restent sûrs.
4. En tant qu’administrateur, je veux consulter les tâches en attente, louées, réussies, en reprise, échouées, annulées ou mortes, afin de comprendre l’état de la file.
5. En tant qu’administrateur, je veux voir la priorité, les tentatives, l’heure du prochain essai et la dernière erreur expurgée, afin de diagnostiquer un blocage sans examiner les logs du worker.
6. En tant qu’administrateur, je veux que les workers concurrents réclament des tâches sans se les attribuer en double, afin de pouvoir augmenter le parallélisme en toute sécurité.
7. En tant qu’administrateur, je veux qu’une tâche acquise ait un lease limité dans le temps, afin qu’un arrêt brutal ne bloque pas la file indéfiniment.
8. En tant que worker, je veux renouveler le lease par heartbeat pendant un traitement actif, afin qu’une tâche longue ne soit pas reprise par un second worker.
9. En tant qu’exploitant, je veux qu’un lease expiré redevienne récupérable, afin qu’une machine arrêtée ou déconnectée ne bloque pas le traitement.
10. En tant qu’exploitant, je veux qu’une erreur transitoire programme une nouvelle tentative avec un délai exponentiel borné, afin de récupérer automatiquement après une panne temporaire.
11. En tant qu’administrateur, je veux qu’une tâche atteignant sa limite de tentatives passe à l’état dead, afin qu’une erreur persistante soit visible sans boucle infinie.
12. En tant qu’administrateur, je veux relancer explicitement une tâche dead, afin de réessayer après correction de sa cause.
13. En tant qu’administrateur, je veux annuler une tâche en attente ou en cours, afin de retirer un travail devenu inutile sans effacer son historique.
14. En tant qu’administrateur, je veux mettre en pause les nouvelles acquisitions globalement, afin de suspendre l’arrivée de tâches sur le PC GPU.
15. En tant qu’administrateur, je veux reprendre les acquisitions après une pause, afin que la file reparte sans recréer ses tâches.
16. En tant qu’administrateur, je veux régler le nombre maximal de tâches simultanées, afin d’adapter la charge au PC et à sa mémoire GPU.
17. En tant qu’exploitant, je veux que la concurrence effective soit limitée par la configuration du worker et le réglage serveur, afin qu’aucun des deux ne soit dépassé.
18. En tant qu’exploitant, je veux que la longueur de la file n’augmente pas à elle seule le nombre de calculs simultanés, afin de garder une consommation de ressources prévisible.
19. En tant qu’administrateur, je veux que le changement de version ou de configuration d’une étape invalide seulement les résultats de cette étape, afin d’éviter de retraiter inutilement les autres étapes.
20. En tant qu’exploitant, je veux que les résultats soient enregistrés de façon transactionnelle et idempotente, afin qu’un timeout ou une réponse perdue ne crée pas de doublon.
21. En tant qu’exploitant, je veux qu’un résultat associé à une ancienne version d’entrée soit refusé, afin qu’une tâche obsolète ne remplace pas un résultat plus récent.
22. En tant qu’exploitant, je veux que le serveur valide le propriétaire du lease avant d’accepter heartbeat, annulation ou résultat, afin qu’un worker ayant perdu sa tâche ne puisse plus la modifier.
23. En tant qu’administrateur, je veux générer, consulter une seule fois puis révoquer un jeton de worker, afin de contrôler séparément l’accès des machines de traitement.
24. En tant qu’exploitant, je veux que les jetons de worker soient stockés sous forme hachée et n’ouvrent que les routes worker, afin qu’ils ne donnent pas une session utilisateur ou administrateur.
25. En tant que worker, je veux obtenir uniquement les métadonnées et médias autorisés pour la tâche acquise, afin de traiter un clip sans accès général à la bibliothèque.
26. En tant que propriétaire du projet, je veux que le worker ne se connecte jamais directement à PostgreSQL, afin que le serveur demeure l’unique arbitre des états et résultats.
27. En tant qu’exploitant, je veux que le worker initie les connexions vers l’API HTTPS et n’expose aucun port entrant, afin qu’il puisse tourner derrière le réseau domestique ou un pare-feu.
28. En tant qu’administrateur, je veux voir si le worker est connecté, son identifiant, sa dernière activité et sa capacité, afin de distinguer une file inactive d’un worker absent.
29. En tant qu’exploitant, je veux que l’arrêt du worker cesse les nouvelles acquisitions et préserve la cohérence des leases, afin qu’un redémarrage soit sûr.
30. En tant que contributeur, je veux un contrat OpenAPI qui décrit les opérations du worker et de l’administration, afin que le serveur et les clients générés restent cohérents.
31. En tant que contributeur, je veux tester le cycle complet avec un fournisseur factice, afin de valider l’infrastructure sans modèle externe.
32. En tant que propriétaire du projet, je veux que les tests et le worker de cette phase n’utilisent que des tâches, médias synthétiques et fixtures dédiées, afin de ne pas démarrer le traitement de la vraie bibliothèque.

## Implementation Decisions

- Le monolithe Go demeure seul propriétaire de PostgreSQL, de l’ordonnancement et de la validation des transitions. Le worker Python est un client HTTPS sortant et ne reçoit aucun accès réseau direct à la base.
- La table de tâches conserve au minimum : identifiant, clip, étape, version d’entrée, état, priorité, nombre de tentatives et maximum, propriétaire et expiration du lease, prochain essai, erreur exploitable expurgée et dates de création/mise à jour. Les contraintes nécessaires empêchent les doublons actifs pour une même identité de travail.
- L’acquisition atomique sélectionne les tâches disponibles selon priorité et ancienneté, en verrouillant les lignes afin que plusieurs workers ne reçoivent pas la même tâche. Les tâches annulées, en attente de retry ou déjà louées ne sont pas acquises prématurément.
- Les états couvrent pending, leased, done, failed, retry_wait, cancelled et dead. Les transitions autorisées, notamment expiration, perte de lease, annulation et relance manuelle, sont documentées et vérifiées en base/API.
- Les appels heartbeat, résultat et erreur incluent l’identité de tâche et du lease. Le serveur rejette une mise à jour si le lease a expiré, changé de propriétaire ou si l’input_version ne correspond plus.
- Les retries sont bornés par un maximum configurable et un délai exponentiel plafonné. La tâche dead conserve sa cause expurgée ; sa relance par admin enregistre l’action et remet la tâche dans un état exécutable.
- La pause globale empêche les nouvelles acquisitions. Les tâches déjà louées peuvent terminer et renouveler leur lease ; la pause ne détruit pas les résultats ni l’état de la file. La reprise réautorise les acquisitions.
- La concurrence maximale est configurable côté worker et bornée côté serveur ; son réglage est indépendant du nombre d’éléments en attente. La configuration ne promet pas un débit ni une utilisation GPU avant le benchmark de phase 7.
- Chaque étape expose une identité et une version d’entrée dérivée de ses paramètres pertinents. Un changement de version permet d’invalider et recréer uniquement le travail concerné ; les étapes ultérieures décideront de leurs dépendances en phase 3 et suivantes.
- Les résultats sont validés et persistés transactionnellement par le serveur. La répétition identique d’un résultat est idempotente ; un résultat ancien ne remplace pas un résultat de version plus récente.
- L’authentification worker utilise un secret aléatoire dédié, haché dans la base, révocable par l’administrateur et distinct des sessions utilisateur. Le secret brut n’est affiché qu’à sa création et n’est jamais journalisé.
- Les routes worker sont limitées à l’état/capacité worker, acquisition, heartbeat, accès aux entrées de la tâche en cours, dépôt de résultat et signalement d’échec. Un jeton ne peut pas lire la bibliothèque, modifier les réglages, administrer les comptes ou réclamer les médias d’une autre tâche.
- Le worker télécharge les seules entrées autorisées pour son lease, puis soumet des résultats au serveur. Il n’écoute aucun port entrant. TLS est vérifié ; le mode production ne désactive pas la validation des certificats.
- L’interface administrateur expose l’état de la file, les erreurs expurgées, pause/reprise, concurrence, annulation et relance des tâches dead. Elle ne permet pas de saisir des secrets déjà créés ni d’exécuter arbitrairement des commandes sur le worker.
- Le worker livré est un squelette d’exécution avec un fournisseur factice pour tests. Cette phase ne livre ni modèle, ni extraction audio métier, ni transcription, ni embeddings, ni diarisation. Les étapes de production sont ajoutées par leurs phases dédiées.
- Le worker reste absent du Compose Forgeyard/NAS. Son image autonome est construite séparément pour le PC GPU ; aucun compose ne suppose un GPU côté NAS.
- OpenAPI reste la source du contrat et les clients générés sont vérifiés. Les migrations sont versionnées et séparées du démarrage de l’API.

## Testing Decisions

Un bon test vérifie un résultat observable aux points d’entrée publics et la cohérence persistée, pas les appels internes ni la structure choisie. La couture principale est l’API réelle avec PostgreSQL/pgvector Testcontainers et un worker Python de test qui communique par HTTPS selon le même contrat public. Les tests utilisent un certificat de test explicitement approuvé, un fournisseur factice et uniquement des tâches/médias synthétiques. Playwright couvre les réglages et actions de l’administrateur.

1. À travers l’API et PostgreSQL réels, vérifier l’enqueue, l’acquisition concurrente, la priorité et l’absence de double lease pour plusieurs workers.
2. Exercer le protocole worker HTTPS : authentification dédiée, acquisition, heartbeat, accès média limité à la tâche, dépôt de résultat, erreur, refus des routes utilisateur/admin et révocation du jeton.
3. Vérifier le cycle pending → leased → done, la perte de lease après expiration, la reprise par un autre worker et le rejet des actions de l’ancien propriétaire.
4. Vérifier les erreurs transitoires, le délai de retry, le maximum de tentatives, l’état dead, sa relance admin et la conservation de l’erreur expurgée.
5. Vérifier pause/reprise : aucune nouvelle acquisition en pause, heartbeats et fin des tâches déjà en cours, puis reprise sans duplication.
6. Vérifier annulation avant acquisition et pendant un lease, arrêt propre du worker et comportement après redémarrage de l’application ou du worker.
7. Vérifier idempotence des résultats, transaction de persistance, rejet d’une input_version obsolète et qu’une invalidation ne recrée que le travail de l’étape concernée.
8. Vérifier en concurrence que le nombre de leases actifs n’excède ni le plafond serveur ni celui annoncé par le worker, indépendamment de la taille de la file.
9. Parcours Playwright : un administrateur consulte la file, règle la concurrence, met en pause puis reprend, annule ou relance une tâche et voit les états actualisés.
10. Construire et démarrer l’image worker avec fournisseur factice ; vérifier `pip check`, absence de connexion PostgreSQL et comportement si l’API HTTPS est indisponible ou son certificat invalide.
11. Vérifier le contrat OpenAPI et la génération Go/TypeScript/Python, les migrations vierges/rejouées, `go test -race`, `go vet` et les builds app/worker.
12. Ne lancer aucun worker sur la vraie bibliothèque, aucun téléchargement de modèle et aucun appel fournisseur externe. Les jobs de test utilisent seulement des identifiants et fixtures isolés.

Le code existant fournit déjà les coutures à prolonger : serveur HTTP Go, PostgreSQL réel via Testcontainers, OpenAPI et clients générés, suites Playwright avec API réelle, image worker Python avec client généré, et scripts locaux de build/validation. La vérification système complète utilise un faux worker HTTPS plutôt qu’une nouvelle base ou une dépendance de service externe.

## Out of Scope

- Préparation audio de production, sélection des pistes, nettoyage, VAD et transcodage : phase 3.
- Modèle de transcription, timestamps, recherche exacte/floue, jargon et vocabulaire administrable : phase 3.
- Embeddings, index vectoriel et recherche sémantique : phase 4.
- Diarisation, empreintes vocales, identité de personnes, validation et attribution : phase 5.
- Édition des transcriptions/attributions et historique de correction : phase 6.
- Choix définitif d’un modèle ou promesse de débit/qualité, mesure soutenue sur le PC/NAS, benchmark de 200–500 clips et batch réel de 1 To : phase 7 et autorisations distinctes.
- Entraînement ou fine-tuning de modèles, lancement automatique d’un traitement nocturne réel, accès direct worker→PostgreSQL, broker séparé, Redis/RabbitMQ, GPU sur le NAS, orchestration multi-PC avancée.
- Déploiement et restauration complète sur Forgeyard/NAS, ouverture de ports entrants sur le PC, CI distante ou publication d’images.

## Further Notes

- Le PC GPU prévu dispose d’une RTX 4070 avec 12 Go de VRAM ; la charge réelle et le parallélisme soutenable restent à mesurer au benchmark.
- Les tâches peuvent être durablement mises en file avant qu’une étape métier soit fournie par la phase suivante ; la présence d’une file ne signifie pas que la transcription ou l’analyse des clips est déjà livrée.
- Cette spec décrit la phase 2 et autorise sa préparation. Elle ne déclenche pas l’implémentation : la validation de fin de phase 1 reste préalable, et aucun traitement réel n’est autorisé avant les décisions de benchmark de phase 7.
