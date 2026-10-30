# File de tâches par clip — #98

Direction E validée par l’utilisateur, prototype conservé dans le commit 93bb11a de la branche prototype/issue-98-task-queue. Les fichiers de prototype et le contournement de session de développement sont retirés de l’implémentation.

La file regroupe les dernières tâches de chaque étape par clip. Une tâche annulée reste visible et ne compte pas comme une réussite. « Terminé » signifie que tous les traitements planifiés ont réussi, pas que toutes les fonctions de Snitch ont été exécutées. Le tableau affiche les trois clips terminés les plus récents. L’historique pagine dix clips côté serveur, et les travaux actifs vingt clips. Les titres originaux viennent de l’API des clips ; le jeu est extrait uniquement des noms au format jeu/date/heure reconnu.

La planification propose la transcription (étape 0) et l’analyse vocale (étape 1) selon les capacités déclarées par les workers non révoqués. Le serveur résout la piste et la version. Un worker compatible acquiert la tâche ; aucun choix arbitraire de machine n’est demandé. La transcription conserve la confirmation de remplacement des corrections. L’analyse vocale exige une transcription ; le serveur applique cette condition et la configuration des modèles. Les autres étapes dérivées sont gérées par leurs pipelines existants. Priorité et tentatives utilisent les valeurs par défaut du serveur.

La file affiche le nom du worker et son GPU près du flux de clips. Les réglages de concurrence restent en haut, les filtres viennent avant les colonnes de progression, et la pagination utilise deux flèches avec le numéro de page au centre. Le prototype autonome est disponible dans [docs/prototypes/issue-98-queue.html](prototypes/issue-98-queue.html).

Vérifications : compilation web ; tests Go jobs/workers/httpapi ; test PostgreSQL TestFoundation/job_queue avec Testcontainers/Colima ; Chromium avec API synthétique (route authentifiée, historique page 2, fermeture de fenêtre, choix des étapes, largeur 390 px, aucune erreur JavaScript). Les vérifications navigateur ne lancent ni transcription réelle ni parcours OAuth.
