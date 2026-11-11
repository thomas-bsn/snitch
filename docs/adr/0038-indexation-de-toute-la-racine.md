# 0038 — Indexation de toute la Racine vidéo, sur action explicite

Statut : accepté. Amende l'[ADR 0032](0032-explorateur-du-disque-et-indexation-a-la-demande.md), qui écartait « indexer toute la racine en arrière-plan », sans toucher à la règle de l'[ADR 0017](0017-racine-video-unique-et-publication-nommee.md) : aucune découverte automatique, aucune analyse sans action. Complète l'[ADR 0035](0035-analyse-complete-automatique.md). Ticket #275.

## Contexte

L'ADR 0032 écartait l'indexation de masse parce qu'elle lit en entier de gros fichiers sur un NAS, et la laissait aux ouvertures une à une. Au démarrage d'une instance, cela oblige à ouvrir chaque dossier, ou à lancer l'analyse d'un dossier depuis le navigateur : une boucle côté client, qui ne survit ni à la fermeture de la page ni à un redémarrage (#247).

## Décision

- **Une action d'administrateur, jamais un fond.** « Indexer les vidéos de la Racine » existe dans l'administration (Dossiers) et dans l'assistant d'initialisation (étape facultative, même API, mêmes droits). Rien ne démarre seul, et chaque démarrage passe par une estimation puis une confirmation.
- **Une indexation durable côté serveur** (`library_index_runs`) : elle étiquette les fichiers qu'elle vise (`source_files.index_run_id`) et laisse le scan géré, qui reprend après un redémarrage, les traiter 50 par 50 en sondant chaque fichier. Aucune boucle ne vit dans le navigateur ; quitter la page ne l'arrête pas.
- **Deux modes, visibles.** *Fiches seulement* : chaque vidéo reçoit sa fiche clip, sans transcription. *Fiches et analyse complète* (proposé par défaut) : chaque nouveau clip est aussi transcrit puis analysé par la file de tâches, même si le réglage « Analyse complète automatique » est désactivé, puisque l'administrateur l'a demandé. Sans Worker, les vidéos attendent avec leur raison et reprennent seules.
- **Idempotente.** Une vidéo qui a déjà un clip n'est jamais retraitée ni comptée comme à faire ; une seule indexation est active par Racine ; relancer la demande n'ajoute ni fiche ni tâche en double.
- **Contrôle.** Pause, reprise et annulation agissent sur les vidéos qui n'ont pas encore de clip ; les clips déjà créés sont conservés. La progression « N / M » et les vidéos en erreur sont lues à la demande.
- **Visibilité.** Les dossiers cachés, les liens symboliques, les fichiers hors Racine et les Dossiers privés (sauf pour un Super-administrateur, [ADR 0026](0026-dossier-prive.md)) ne sont jamais visés. L'estimation ne lit que les répertoires, jamais le contenu des vidéos.

## Alternatives écartées

- **Indexer en arrière-plan sans action** : contraire à l'ADR 0017 et coûteux sur un NAS.
- **Étendre la boucle du navigateur** : fragile, non reprenable, sans pause ni suivi partagé.
- **Un nouveau moteur de tâches** : le scan géré et la file d'analyse font déjà le travail, durablement.
- **Retraiter les clips déjà indexés mais jamais analysés** : reste possible dossier par dossier depuis la Bibliothèque ; une demande explicite est requise pour ne pas relancer du travail valide.

## Conséquences

- Le débit est celui du scan géré (sondage séquentiel, 50 fichiers par passage) : le parallélisme n'est pas réglable.
- Une Racine de plus de 50 000 vidéos est refusée (`video_selection_too_large`) : elle se découpe en plusieurs dossiers.
- L'état « terminée » est constaté à la lecture, quand plus aucune vidéo n'est demandée ; sans lecteur, l'indexation reste « en cours » dans la base.

## Amendement — « Scanner toute la racine » (#404)

Le scan de toute la Racine, lancé depuis Administration › Dossiers, ne parcourait plus rien au-delà de 10 000 fichiers, échouait entièrement sur un seul dossier illisible et dépendait du délai de la requête. Il devient une exécution de `library_index_runs` en mode `scan` : on réutilise l'exécution durable, l'étiquetage des fichiers et le scan géré plutôt qu'un nouveau moteur.

- **La requête enregistre, le serveur parcourt.** La confirmation répond aussitôt (`202`). Le parcours du disque tourne en arrière-plan par lots de 500 ; tant qu'il n'est pas fini, l'exécution reste `walking` et le serveur le recommence au démarrage. Recommencer est sans effet de bord : la mise en file est idempotente et les compteurs du parcours sont réécrits.
- **Pas de limite de nombre, pas d'arrêt sur un dossier illisible.** Un dossier illisible est sauté, compté et nommé (les 20 premiers) dans le bilan.
- **Ce qui est mis en file** : les vidéos non indexées ou modifiées, et les vidéos indexées dont le clip n'a jamais été analysé (ni exécution d'analyse ni tâche hors benchmark). Leur analyse suit ensuite les réglages de traitement, comme un nouveau clip (ADR 0035) : contrairement au mode `full`, le scan ne force pas l'analyse.
- **Ce qui est laissé de côté, pour tous les rôles** : les Dossiers privés (le Super-administrateur compris : il en scanne un en le sélectionnant explicitement), les fichiers et dossiers masqués, les dossiers système et les clips privés. Un clip déjà analysé est « déjà à jour » : la réanalyse reste une action distincte. Cette exclusion des Dossiers privés ne concerne que le scan ; l'Indexation de la Racine garde la règle ci-dessus.
- **Une seule exécution active par Racine**, scan ou indexation (`index_run_active`). Le scan s'annule mais ne se met pas en pause ; les routes de l'indexation ne le pilotent pas.
- **Aperçu et bilan** : l'aperçu parcourt les dossiers dans la requête, comme l'estimation de l'indexation, et ne renvoie que des comptes (à mettre en file, déjà à jour, exclus par motif, dossiers illisibles), jamais la liste des chemins.

Alternative écartée : garder le parcours dans la requête en relevant la limite. Le délai de la requête et du navigateur resterait la borne, et un redémarrage perdrait le travail.
