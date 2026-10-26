## Problem Statement

L'interface de Snitch est une seule page qui empile toutes les sections : bibliothèque, import, workers, file de tâches, sources, vocabulaire, voix, accès et Discord. Elle est jugée laide, difficile à parcourir et impossible à faire évoluer par morceaux. Il n'existe pas de page propre pour un clip ni d'adresse partageable vers un moment précis. Le wizard d'installation s'arrête à la création de l'administrateur et ne guide pas la suite. L'interface n'existe qu'en français, ne se consulte pas confortablement sur téléphone et n'a pas de thème sombre. Les feuilles de style globales s'appliquent à tout composant neuf et le déforment. Le nom de l'instance reste réglable, ce qui a fait afficher l'ancien nom du projet sur une installation locale. Pour démarrer, il faut en outre renseigner à la main trois variables (mot de passe de la base, clé maître de chiffrement, adresse publique) alors que Snitch tourne dans un déploiement fourni qui pourrait les produire seul : un mauvais choix, notamment une clé maître mal formée ou une adresse publique inexacte, empêche le démarrage ou fausse l'adresse de retour Discord. La configuration de la transcription oblige à saisir à la main, à plusieurs endroits, l'adresse du serveur Whisper, le nom du modèle et le jeton Hugging Face, sans lien entre le profil de la bibliothèque et les Workers. Enfin, un administrateur ne peut pas consulter un Worker (machine, modèles, fichier de déploiement) sans retrouver le jeton affiché une seule fois à sa création.

## Solution

Une application à pages, bilingue (anglais par défaut, français), responsive, au style « Atelier » validé sur des maquettes et un prototype jetable : Inter, indigo, barre latérale claire, pastilles d'état, thème clair par défaut avec thème sombre et réglage « système ». Le backend, le contrat HTTP et les comportements métier des phases 0 à 6 sont conservés, sauf des ajouts : la progression du wizard, la langue du compte, les informations de machine d'un Worker et le démarrage sans configuration (base, clé maître et adresse publique produites ou déduites). La navigation tient en quatre entrées (Bibliothèque, Recherche, Personnes, Importer) ; l'avatar ouvre un menu avec les paramètres du compte, l'administration (barre d'onglets horizontale), le thème, la langue et la déconnexion. Le wizard guide de bout en bout, chaque étape facultative pouvant être passée et reprise plus tard. La vérification des voix se fait avec des miniatures cliquables et un panneau de décision ; les groupes de voix inconnues se nomment en une action ; chaque Worker a sa fiche. La transcription se règle une fois : le modèle et la langue de la bibliothèque se fixent à la création du premier Worker, le serveur Whisper se donne pour chaque Worker, et le fichier de déploiement est généré tout prêt. Le démarrage ne demande plus aucune variable : la base, la clé maître et l'adresse publique sont produites ou déduites par l'application. Le travail est livré par lots, chacun validé par des parcours navigateur.

## User Stories

### Installation et wizard

1. En tant que premier utilisateur, je veux saisir le code d'installation puis créer l'Administrateur dans un parcours guidé, afin de ne pas deviner la suite.
2. En tant qu'installateur, je veux que l'écran d'accueil me dise où lire le code d'installation, afin de ne pas chercher dans la documentation.
3. En tant qu'installateur, je veux que les journaux affichent une ligne lisible contenant l'adresse et le code, afin de les copier sans effort.
4. En tant qu'installateur, je veux que le code ne soit jamais affiché dans l'interface, afin qu'un curieux ne puisse pas s'en servir.
5. En tant qu'installateur, je veux qu'un code faux ou expiré soit signalé dès sa saisie avec un message qui m'invite à relire les journaux, afin de ne pas perdre mon mot de passe à l'étape suivante.
6. En tant qu'Administrateur, je veux que le wizard propose ensuite Discord, la recherche sémantique, le premier Worker et l'import, afin de terminer la configuration d'un seul endroit.
7. En tant qu'Administrateur, je veux passer n'importe quelle étape facultative, afin que rien ne soit imposé.
8. En tant qu'Administrateur, je veux voir dans le wizard quelles étapes sont faites, passées ou à venir, afin de savoir où j'en suis.
9. En tant qu'Administrateur, je veux un récapitulatif final de ce qui est configuré et, pour ce qui a été passé, où le terminer, afin de ne rien oublier.
10. En tant qu'Administrateur, je veux retrouver mes étapes restantes après avoir fermé l'onglet, sous la forme d'un bandeau « Finir la configuration », afin de reprendre quand je veux.
11. En tant qu'Administrateur, je veux fermer ce bandeau définitivement, afin qu'il ne m'encombre plus.
12. En tant qu'installateur, je veux que le nom de l'instance reste « Snitch » sans variable à régler, afin d'éviter une configuration inutile.
13. En tant qu'Administrateur, je veux que Discord reste facultatif dans le wizard et que les comptes locaux fonctionnent toujours, afin de pouvoir utiliser Snitch sans Discord.
14. En tant qu'Administrateur, je veux connaître l'adresse de retour à enregistrer dans Discord pendant l'étape Discord, afin de terminer l'intégration correctement.
15. En tant qu'Administrateur, je veux créer un Worker dans le wizard en donnant l'adresse de son serveur Whisper et mon jeton Hugging Face, et obtenir ses fichiers de déploiement prêts avec le jeton du Worker visible une seule fois, afin de les copier avant de continuer.
16. En tant qu'Administrateur, je veux que rien ne démarre ni ne soit scanné sans mon action pendant le wizard, afin de garder le contrôle de la bibliothèque.

### Navigation et coque

17. En tant qu'utilisateur, je veux une barre latérale à quatre entrées (Bibliothèque, Recherche, Personnes, Importer), afin de trouver l'essentiel sans chercher.
18. En tant qu'utilisateur, je veux ouvrir un menu en cliquant sur mon avatar, tout à droite de l'en-tête, afin d'accéder à mon compte et à mes préférences.
19. En tant qu'utilisateur, je veux que ce menu contienne les paramètres du compte, le thème, la langue et la déconnexion, afin de tout régler au même endroit.
20. En tant qu'Administrateur ou Super-administrateur, je veux que ce menu contienne aussi l'administration, et qu'un utilisateur ordinaire ne la voie pas, afin de respecter les rôles.
21. En tant qu'Administrateur, je veux que l'administration présente une barre d'onglets horizontale (Workers, File de tâches, Dossiers, Transcription, Traitement vocal, Index sémantique, Utilisateurs et accès, Exports), afin de passer d'un réglage à l'autre sans quitter la page.
22. En tant qu'Administrateur, je veux que les onglets qui ont des sous-sections les montrent en pastilles sous la barre, afin de rester orienté.
23. En tant qu'utilisateur, je veux que chaque écran ait une adresse propre, afin de partager un lien ou de rouvrir un onglet au bon endroit.
24. En tant qu'utilisateur, je veux qu'actualiser la page me ramène à l'écran où j'étais, afin de ne pas perdre ma place.
25. En tant qu'utilisateur, je veux que le champ de recherche de l'en-tête ouvre la page de recherche, afin de chercher depuis n'importe où.
26. En tant qu'utilisateur, je veux un bouton Importer toujours accessible, afin d'ajouter des vidéos sans détour.
27. En tant qu'utilisateur non connecté, je veux une page de connexion claire avec le choix de la langue, afin de me connecter dans ma langue.
28. En tant qu'utilisateur connecté qui perd le serveur, je veux un message clair avec un bouton pour réessayer, afin de ne pas rester devant une page blanche.

### Langue et thème

29. En tant qu'utilisateur, je veux choisir l'anglais ou le français, afin d'utiliser l'interface dans ma langue.
30. En tant qu'utilisateur connecté, je veux que ce choix soit mémorisé dans mon compte, afin de le retrouver sur mes autres appareils.
31. En tant qu'utilisateur non connecté, je veux que mon choix reste dans le navigateur, afin qu'il s'applique dès la page de connexion.
32. En tant qu'installateur, je veux que l'anglais soit la langue par défaut, afin que l'interface soit lisible par tous dès l'installation.
33. En tant qu'utilisateur, je veux que les erreurs du serveur apparaissent dans ma langue, afin de les comprendre.
34. En tant qu'utilisateur, je veux choisir un thème clair, sombre ou « système » avec des icônes dans le menu du profil, afin de changer d'un clic.
35. En tant qu'utilisateur, je veux que le thème clair soit le défaut et que mon choix soit retenu, afin de ne pas le refaire à chaque visite.
36. En tant qu'utilisateur, je veux que le thème « système » suive le réglage de mon appareil, y compris quand il change en cours de session.

### Bibliothèque, clip et recherche

37. En tant qu'utilisateur, je veux voir mes clips en grille avec miniature, durée, propriétaire, date et état, afin de parcourir la Bibliothèque d'un coup d'œil.
38. En tant qu'utilisateur, je veux filtrer par date, Source, propriétaire, Personne et état, afin de retrouver un clip.
39. En tant qu'utilisateur, je veux que chaque état de clip porte un libellé et un symbole en plus de la couleur, afin de le distinguer sans voir les couleurs.
40. En tant qu'utilisateur, je veux une Bibliothèque vide qui m'invite à importer, afin de savoir quoi faire au premier lancement.
41. En tant qu'utilisateur, je veux ouvrir un clip sur sa propre page, avec le lecteur et la transcription côte à côte, afin de lire en regardant.
42. En tant qu'utilisateur, je veux voir pour chaque segment son locuteur et son état (identifié, suggestion, inconnu, corrigé, automatique), afin de savoir ce qui est vérifié.
43. En tant qu'utilisateur, je veux des onglets Transcription, Voix et Historique sur la page d'un clip, afin d'accéder à chaque aspect sans quitter le clip.
44. En tant qu'utilisateur, je veux que l'adresse d'un clip garde le moment lu, afin de partager un instant précis.
45. En tant qu'utilisateur, je veux que les résultats de recherche montrent la citation avec le passage trouvé surligné et son timecode, afin de juger la pertinence avant d'ouvrir.
46. En tant qu'utilisateur, je veux ouvrir la vidéo au bon moment depuis un résultat, afin d'entendre le contexte.
47. En tant qu'utilisateur, je veux choisir le mode de recherche (exact, plein texte, approximatif, hybride) et la voir dégradée explicitement quand les embeddings sont indisponibles, afin de comprendre ce que je cherche.
48. En tant qu'utilisateur, je veux un bandeau d'avancement sur un clip (importé, transcrit, indexé, voix traitées, voix identifiées), afin de savoir où il en est.
49. En tant qu'utilisateur, je veux importer des vidéos par glisser-déposer avec une progression par fichier, afin de suivre les transferts.
50. En tant qu'utilisateur, je veux reprendre ou annuler un transfert interrompu depuis la page d'import, afin de ne pas tout recommencer.
51. En tant qu'utilisateur, je veux que mes corrections de segments et leur historique restent accessibles depuis la page du clip, afin de corriger en contexte.

### Voix

52. En tant qu'utilisateur, je veux voir les voix à trancher avec une miniature cliquable qui ouvre l'extrait, afin d'écouter avant de décider.
53. En tant qu'utilisateur, je veux des compteurs par état (à trancher, inconnues, ambiguës) qui filtrent la liste, afin de cibler un type de cas.
54. En tant qu'utilisateur, je veux ouvrir un panneau « Décider » avec les candidats proches et leur proximité, afin de choisir en connaissance de cause.
55. En tant qu'utilisateur, je veux chercher une Personne existante ou en créer une nouvelle depuis ce panneau, afin de ne pas quitter le cas.
56. En tant qu'utilisateur, je veux que « Confirmer » reste inactif tant que rien n'est choisi, afin de ne pas valider par erreur.
57. En tant qu'utilisateur, je veux passer un cas sans rien changer, afin de revenir plus tard.
58. En tant qu'utilisateur, je veux que trancher un cas le retire de la liste, mette à jour les compteurs et confirme l'enregistrement, afin de voir l'effet de ma décision.
59. En tant qu'utilisateur, je veux que les cas les plus fréquents passent en premier, afin de réduire le plus vite l'incertitude.
60. En tant qu'utilisateur, je veux voir les Groupes de voix inconnues avec un aperçu de leurs voix et le nombre de Clips concernés, afin de repérer les plus utiles à nommer.
61. En tant qu'utilisateur, je veux écouter les extraits d'un groupe côte à côte, afin de vérifier que c'est la même personne.
62. En tant qu'utilisateur, je veux écarter une voix du groupe et la remettre, sans toucher aux autres, afin de corriger une erreur de regroupement.
63. En tant qu'utilisateur, je veux nommer le groupe en une action qui valide toutes les voix retenues, afin de gagner du temps.
64. En tant qu'utilisateur, je veux que le bouton de nommage indique le nombre de voix et la Personne, afin de savoir ce que je valide.
65. En tant qu'utilisateur, je veux que l'interface rappelle qu'aucun groupe n'est jamais confirmé automatiquement, afin de rester confiant.
66. En tant qu'utilisateur, je veux gérer les Personnes (création, fusion, séparation, suppression) depuis l'onglet Personnes, afin de corriger mes erreurs.
67. En tant que titulaire d'un Profil vocal, je veux le supprimer depuis mes paramètres de compte avec une explication claire de ses effets, afin de reprendre le contrôle de mes données.

### Workers

68. En tant qu'Administrateur, je veux la liste des Workers avec leur état, leur dernier pouls et leurs modèles, afin de voir lesquels travaillent.
69. En tant qu'Administrateur, je veux ouvrir la fiche d'un Worker, afin de tout voir sur lui au même endroit.
70. En tant qu'Administrateur, je veux voir sur cette fiche son état, sa concurrence, sa date de création et ses dernières tâches, afin de comprendre son activité.
71. En tant qu'Administrateur, je veux voir les modèles qu'il déclare (transcription, diarisation, empreinte), afin de vérifier sa configuration.
72. En tant qu'Administrateur, je veux voir les informations de la machine qui l'héberge (nom, GPU et pilote, processeur, mémoire, version du Worker, adresse vue par Snitch), afin d'identifier la machine.
73. En tant qu'Administrateur, je veux que le nom de la machine soit réglable par une variable facultative, afin de ne pas voir un identifiant de conteneur.
74. En tant qu'Administrateur, je veux que les informations non déclarées soient indiquées comme telles, afin de ne pas lire des valeurs trompeuses.
75. En tant qu'Administrateur, je veux récupérer le fichier de déploiement de ce Worker, le copier ou le télécharger, afin de le poser sur la machine.
76. En tant qu'Administrateur, je veux que le jeton ne soit visible qu'une seule fois, à la création, afin qu'il ne soit jamais exposé ensuite.
77. En tant qu'Administrateur, je veux générer un nouveau jeton après confirmation, ce qui coupe l'ancien, afin de remplacer un jeton perdu.
78. En tant qu'Administrateur, je veux révoquer un Worker après confirmation, afin de l'empêcher de prendre des tâches.
79. En tant qu'Administrateur, je veux que la liste des tâches d'un Worker distingue transcription, diarisation et empreintes vocales, afin de savoir laquelle tourne ou échoue.

### Administration

80. En tant qu'Administrateur, je veux piloter la file de tâches (pause, reprise, concurrence, relance) depuis l'administration, afin de contrôler le traitement.
81. En tant qu'Administrateur, je veux gérer les dossiers surveillés avec leur dernier scan et leurs avertissements de santé, afin de voir un dossier démonté.
82. En tant qu'Administrateur, je veux régler le profil de transcription (modèle, langue, détection de parole) avec sa version, en sachant que l'adresse du fournisseur se règle sur le Worker, afin de ne pas chercher au mauvais endroit.
83. En tant qu'Administrateur, je veux gérer le vocabulaire de transcription (termes, variantes, graphie préférée), afin d'améliorer la transcription des termes de jeu.
84. En tant qu'Administrateur, je veux régler le traitement vocal (pipeline, seuils de rapprochement, automatisation) avec l'automatisation désactivée par défaut, afin de ne rien automatiser sans mesures.
85. En tant qu'Administrateur, je veux régler l'index sémantique et voir les segments en attente de réindexation, afin de maintenir la recherche.
86. En tant qu'Administrateur, je veux approuver les inscriptions en attente et régler l'approbation requise, afin de contrôler l'accès.
87. En tant qu'Administrateur, je veux configurer Discord et voir l'adresse de retour, afin d'activer cette option.
88. En tant qu'Administrateur, je veux exporter les segments corrigés, afin de récupérer mes données.

### Compte

89. En tant qu'utilisateur, je veux des paramètres de compte avec un onglet Profil (identité Discord, préférences) et un onglet Ma voix, afin de gérer mon compte au même endroit.
90. En tant qu'utilisateur, je veux lier mon identité Discord depuis mes paramètres, afin de me connecter avec Discord.

### Déploiement sans configuration

99. En tant qu'installateur, je veux lancer Snitch avec le déploiement fourni sans renseigner aucune variable obligatoire, afin d'obtenir une instance qui démarre du premier coup.
100. En tant qu'installateur, je veux que le mot de passe de la base soit produit au premier démarrage et jamais saisi ni affiché, afin de ne pas en choisir un faible ou de le laisser dans un fichier.
101. En tant qu'installateur, je veux que la base ne soit joignable que depuis le déploiement, afin qu'un mot de passe produit automatiquement ne soit pas exposé.
102. En tant qu'installateur, je veux que la clé maître de chiffrement soit générée par l'application au premier démarrage, afin de ne pas manipuler de commande de génération.
103. En tant qu'installateur, je veux que cette clé soit conservée hors de la base et hors de ses sauvegardes, afin qu'une sauvegarde de la base ne suffise pas à lire les secrets.
104. En tant qu'Administrateur, je veux pouvoir exporter la clé maître depuis l'administration pour la conserver séparément, avec l'avertissement qu'elle ne se rejoue pas, afin de pouvoir restaurer mes secrets.
105. En tant qu'Administrateur, je veux que la perte de la clé n'empêche pas Snitch de démarrer et qu'il m'indique quels secrets sont à ressaisir (par exemple Discord), afin de récupérer sans réinstaller.
106. En tant qu'installateur, je veux que l'adresse publique soit déduite de la façon dont j'accède à Snitch, afin de ne pas la renseigner et de ne pas me tromper.
107. En tant qu'Administrateur, je veux que l'adresse de retour Discord affichée soit celle de l'adresse par laquelle j'accède à Snitch, afin de l'enregistrer telle quelle.
108. En tant qu'Administrateur, je veux, en créant un Worker, voir et pouvoir corriger l'adresse de Snitch inscrite dans son fichier de déploiement, avec un avertissement si elle n'est pas joignable depuis une autre machine (par exemple localhost), afin que le Worker puisse se connecter.
109. En tant qu'installateur derrière un mandataire inverse, je veux pouvoir fixer l'adresse publique par un réglage facultatif, afin de garder une adresse stable quand la déduction n'est pas fiable.
110. En tant qu'installateur en HTTP local, je veux que Snitch fonctionne sans réglage de cookie, et en HTTPS que le cookie de session soit sécurisé automatiquement, afin de ne pas gérer ce réglage.
111. En tant qu'installateur, je veux que les anciennes variables soient encore acceptées si je les renseigne, afin de migrer une installation existante sans rien perdre.

### Transcription et Workers (compléments)

112. En tant qu'Administrateur, je veux que le modèle de transcription et la langue de la bibliothèque se fixent à la création du premier Worker, afin de ne rien régler avant d'en avoir besoin.
113. En tant qu'Administrateur, je veux une confirmation à la création du premier Worker qui m'indique que ce modèle et cette langue serviront à tous les futurs Workers, même avec un serveur Whisper différent, afin de savoir ce que j'engage.
114. En tant qu'Administrateur, je veux que l'onglet Transcription soit vide tant qu'aucun Worker n'existe, avec un message qui m'explique où le régler, afin de ne pas remplir un formulaire inutile.
115. En tant qu'Administrateur, je veux, pour chaque nouveau Worker, donner l'adresse de son serveur Whisper et mon jeton Hugging Face, le modèle et la langue de la bibliothèque s'affichant en lecture seule, afin de ne rien ressaisir d'autre.
116. En tant qu'Administrateur, je veux que le jeton Hugging Face ne soit ni envoyé à Snitch ni conservé, et seulement écrit dans le fichier de déploiement affiché, afin de ne pas stocker un secret de plus.
117. En tant qu'Administrateur, je veux que Snitch génère un fichier d'environnement et un fichier de déploiement prêts à poser sur le PC, avec un avertissement si l'adresse de Snitch n'est pas en HTTPS, afin que le Worker puisse se connecter.
118. En tant qu'Administrateur, je veux que l'onglet Transcription montre le modèle principal (nom, langue, détection de parole repliée) puis la liste des Workers avec leur serveur Whisper, le nom envoyé à ce serveur et leur état, afin de voir d'un coup d'œil la configuration.
119. En tant qu'Administrateur, je veux ouvrir un Worker depuis cet onglet dans une fenêtre pour définir, facultativement, le nom que son serveur donne au même modèle, afin de gérer deux serveurs qui nomment le modèle différemment.
120. En tant qu'Administrateur, je veux que la langue soit la même pour tous les Workers, afin que la bibliothèque reste cohérente.
121. En tant qu'Administrateur, je veux que changer le modèle crée une nouvelle version sans toucher aux résultats déjà calculés, et que je choisisse moi-même les clips à retraiter, afin de contrôler le retraitement.

### Membres et fermeture de compte

122. En tant qu'Administrateur, je veux la liste des membres avec leur rôle, leur date d'arrivée et leur nombre de clips, afin de gérer l'accès.
123. En tant qu'Administrateur, je veux changer le rôle d'un membre (utilisateur ou Administrateur) avec une confirmation qui explique ce que le rôle donne ou retire, afin d'éviter une erreur.
124. En tant qu'Administrateur, je veux supprimer un membre avec une confirmation et une case « masquer ses clips » décochée par défaut, afin que ses clips restent visibles (au nom d'un utilisateur supprimé) sauf si je choisis de les masquer.
125. En tant qu'Administrateur, je veux que mon propre compte et le Super-administrateur ne puissent être ni rétrogradés ni supprimés depuis cette liste, afin de ne pas verrouiller l'instance.
126. En tant qu'utilisateur (hors Super-administrateur), je veux fermer mon propre compte en tapant « delete » ou « supprimer » pour confirmer, sans que mes clips soient jamais supprimés : ils sont masqués, afin de partir sans effacer ce que je laisse.
127. En tant qu'utilisateur, je veux que les paramètres du compte n'aient pas de réglage de thème, qui est dans le menu du profil, afin de ne pas le trouver à deux endroits.

### Rôles

128. En tant que personne qui installe Snitch, je veux que le compte créé par le wizard soit le Super-administrateur, afin que l'instance ait un propriétaire que personne ne peut retirer.
129. En tant que Super-administrateur, je veux être le seul à promouvoir, rétrograder ou supprimer un Administrateur, afin de contrôler qui administre.
130. En tant que Super-administrateur, je veux être le seul à exporter la clé maître, configurer Discord et régler l'approbation des inscriptions, afin de protéger les secrets et la politique d'accès.
131. En tant qu'Administrateur, je veux gérer les Workers, les dossiers, la file de tâches, la transcription, le vocabulaire, le traitement vocal, les inscriptions et les utilisateurs ordinaires, afin d'exploiter l'instance sans toucher aux réglages réservés.
132. En tant que Super-administrateur, je veux que mon compte ne puisse être ni rétrogradé, ni supprimé, ni fermé, afin que l'instance ne se retrouve jamais sans propriétaire.
133. En tant qu'utilisateur d'une installation existante, je veux que l'Administrateur le plus ancien devienne Super-administrateur lors de la mise à jour, afin que rien ne soit perdu.
134. En tant qu'utilisateur, je veux voir mon rôle dans le menu du profil (Super-administrateur, Administrateur ou Utilisateur), afin de savoir ce que je peux faire.

### Qualité

91. En tant qu'utilisateur de clavier, je veux atteindre et activer chaque élément avec un focus visible, afin de me passer de la souris.
92. En tant qu'utilisateur de lecteur d'écran, je veux des libellés sur chaque champ, bouton à icône et fenêtre, afin de comprendre l'interface.
93. En tant qu'utilisateur, je veux que la couleur ne porte jamais seule une information, afin de la comprendre sans distinguer les couleurs.
94. En tant qu'utilisateur de téléphone, je veux une barre d'onglets en bas à la place de la barre latérale et des panneaux en plein écran, afin de naviguer d'une main.
95. En tant qu'utilisateur de téléphone, je veux des cibles tactiles d'au moins 44 pixels, afin de ne pas me tromper de bouton.
96. En tant qu'utilisateur, je veux que les fenêtres et panneaux se ferment avec la touche Échap et piègent le focus, afin de les utiliser au clavier.
97. En tant qu'utilisateur, je veux des contrastes suffisants dans les thèmes clair et sombre, afin de lire sans effort.
98. En tant que mainteneur, je veux que l'ancienne interface et sa feuille de style disparaissent, afin que le code neuf ne soit pas déformé par des règles globales.

## Implementation Decisions

- **Périmètre** : refonte complète du front. Le contrat HTTP reste la source unique ; seuls les ajouts de backend listés ci-dessous sont autorisés : progression du wizard, langue du compte, informations de machine d'un Worker et démarrage sans configuration. Les comportements des phases 0 à 6 sont conservés.
- **Direction visuelle** : Atelier. Jetons repris des maquettes : fond blanc, barre latérale gris très clair, filets légers, texte presque noir, secondaire gris, accent indigo (#5b5bd6) et sa version douce pour les états actifs, police Inter. Le logo est un carré indigo aux coins arrondis portant quatre barres blanches de hauteurs différentes (forme d'onde), repris en icône d'onglet. Thème clair par défaut, thème sombre à définir et à vérifier en contraste, réglage « système ».
- **États** : toujours un libellé et un symbole en plus de la couleur, y compris les cinq niveaux de décision des voix.
- **Navigation** : quatre entrées latérales ; menu du profil (paramètres du compte, administration pour les Administrateurs, thème par icônes, langue, déconnexion) ; administration en barre d'onglets horizontale avec sous-onglets en pastilles ; paramètres du compte en deux onglets. Pas d'entrée « mon compte » dans la barre latérale. Chaque écran a une adresse propre ; le serveur renvoie l'application pour toutes ces adresses ; la page d'un clip conserve le moment lu.
- **Découpage du code** : l'interface actuelle d'un seul bloc est découpée en pages et composants ; l'ancienne feuille de style globale est supprimée au premier lot, car ses règles sur les éléments natifs déforment tout composant neuf (constat du prototype : boutons agrandis, avatar déformé).
- **Dépendances** : autorisées et épinglées une à une (framework CSS utilitaire, primitives accessibles, routeur, bibliothèque d'icônes). Le choix définitif est tranché au premier lot et consigné dans un ADR ; le prototype, lui, est en CSS à la main.
- **Internationalisation** : textes isolés par langue ; anglais par défaut ; préférence stockée dans le compte (nouvelle colonne et nouveau point d'accès) et dans le navigateur avant connexion ; pas de détection automatique de la langue du navigateur. Le serveur renvoie déjà des codes d'erreur stables : l'interface les traduit et garde le texte du serveur en repli ; un inventaire des codes précède la traduction. Les exports gardent des en-têtes techniques fixes. La documentation reste en français.
- **Wizard** : sept étapes (code, Administrateur, Discord, recherche sémantique, premier Worker, import, récapitulatif), les quatre du milieu facultatives et passables. Progression stockée côté serveur (étapes faites, passées, bandeau fermé) pour permettre la reprise ; le nom de l'instance n'est plus une étape. Le code d'installation faux ou expiré est signalé dès sa saisie. Rien ne s'exécute sans action de l'utilisateur.
- **Nom de l'instance** : le réglage de nom (variable, validation, déclaration de déploiement) est supprimé ; « Snitch » est le nom unique, dans la marque et les titres de page. Tests et documentation concernés sont mis à jour.
- **Jeton de Worker** : politique inchangée (visible une seule fois à la création, révocable). La fiche affiche un fichier de déploiement avec un emplacement de jeton ; générer un nouveau jeton remplace l'ancien après confirmation. Aucun jeton lisible n'est conservé.
- **Informations de machine d'un Worker (ajout backend et Worker)** : à chaque pouls, le Worker déclare en plus de ses modèles : nom de machine, système, GPU (nom, mémoire, pilote), processeur, mémoire, version. L'adresse vue est observée par le serveur. Champs non secrets, bornés et assainis comme la déclaration des modèles (refus explicite d'une valeur invalide), purement indicatifs, visibles des seuls Administrateurs ; absence = « non déclaré ». Extension rétrocompatible : un ancien Worker reste valide. Contrainte : le Worker tourne dans un conteneur et ne voit que le système et le nom du conteneur ; une variable facultative nomme la machine hôte, et le système de l'hôte n'est pas affiché tant qu'il ne peut pas être déterminé de façon fiable.
- **Vérification des voix** : un cas se tranche uniquement par la validation d'une Personne existante ou nouvelle, comme aujourd'hui. « Passer » ne modifie rien. Aucun état « bruit » ou « à ignorer » n'est créé par cette spec.
- **Groupes de voix inconnues** : l'écran présente les extraits d'un groupe côte à côte, l'exclusion par voix et le nommage en une action, avec le comportement existant (aucun groupe confirmé automatiquement, refus si le groupe a changé entre-temps).
- **Responsive** : barre d'onglets en bas à la place de la barre latérale, panneaux en plein écran sur téléphone, cibles d'au moins 44 pixels.
- **Accessibilité** : niveau WCAG 2.2 AA ; éléments natifs, focus visible, états non portés par la couleur seule, fenêtres modales avec focus piégé et fermeture par Échap.
- **Profil de transcription et Workers** : un profil unique pour toute la bibliothèque (nom du modèle, langue, détection de parole), créé à la création du premier Worker après confirmation et modifiable ensuite dans l'onglet Transcription (nouvelle version à chaque modification). Le serveur Whisper est une information propre à chaque Worker, déclarée par le Worker. Le nom envoyé à un serveur est par défaut celui du profil ; un nom propre au Worker, facultatif, le remplace pour ce serveur : le nom du profil reste l'identité des résultats. Cela inverse la priorité actuelle du Worker (le nom du profil passait avant celui de son environnement). La langue reste unique pour tous les Workers. Aucune validation n'est faite que deux noms différents désignent le même modèle : c'est une déclaration de l'Administrateur.
- **Création d'un Worker** : le formulaire demande le nom du Worker, l'adresse de son serveur Whisper et le jeton Hugging Face, plus le modèle et la langue pour le premier Worker seulement. L'adresse de Snitch est préremplie et en lecture seule, avec un avertissement si elle n'est pas en HTTPS (le Worker refuse le HTTP). Snitch génère le fichier d'environnement et le fichier de déploiement ; le téléchargement des modèles de voix au premier démarrage est actif par défaut dans ces fichiers. Le jeton Hugging Face est écrit dans le fichier d'environnement dans le navigateur uniquement, jamais envoyé à Snitch ni conservé. Le fichier de déploiement n'exige plus de nom de modèle. L'ajout de ce formulaire à l'étape « premier Worker » du wizard utilise le même composant.
- **Rôles** : trois rôles. Le **Super-administrateur** est unique, créé par le wizard d'installation, non rétrogradable, non supprimable, ne peut pas fermer son compte (le transfert de propriété est hors périmètre). L'**Administrateur** exploite l'instance. L'**Utilisateur** n'a pas accès à l'administration. Réservé au Super-administrateur : gérer les Administrateurs (promouvoir, rétrograder, supprimer), exporter la clé maître, configurer Discord et régler l'approbation des inscriptions. Tout le reste de l'administration est ouvert aux Administrateurs et au Super-administrateur. Les installations existantes : l'Administrateur le plus ancien devient Super-administrateur par migration. En attendant l'ajout du rôle, tout contrôle « est administrateur » passe par une seule fonction qui traite Administrateur et Super-administrateur de la même façon. Proposition de répartition des droits, à confirmer.
- **Membres** : nouveaux points d'accès réservés aux Administrateurs pour lister les membres, changer un rôle et supprimer un membre avec l'option de masquer ses clips. « Masquer » fait passer les clips en privé (la visibilité existante) ; sans cette option, les clips restent visibles et s'affichent au nom d'un utilisateur supprimé. Le Super-administrateur et le compte de la personne qui agit sont protégés ; seul le Super-administrateur gère les comptes d'Administrateurs.
- **Fermeture de compte** : un utilisateur ferme son propre compte ; la connexion est désactivée, ses sessions sont coupées, ses clips sont masqués, jamais supprimés, ses corrections et leur historique restent. Le Super-administrateur ne peut pas fermer son compte. La suppression du Profil vocal reste une action séparée.
- **Livraison par lots**, chacun avec ses parcours navigateur verts : (1) fondations (jetons, thèmes, internationalisation, routeur, coque, logo, suppression de l'ancienne feuille de style et du réglage de nom) ; (2) wizard et connexion ; (3) bibliothèque, clip, recherche, import ; (4) voix, segments et corrections ; (5) administration et paramètres du compte ; (6) finition (thème sombre vérifié, mobile, accessibilité, bilan). Validation utilisateur après les lots 2 et 3, puis à la fin.
- **Aucune variable obligatoire** : le déploiement fourni démarre sans fichier d'environnement. Les variables actuelles (mot de passe de la base, clé maître, adresse publique, réglage du cookie sécurisé) deviennent facultatives et prioritaires si elles sont fournies, ce qui préserve les installations existantes.
- **Mot de passe de la base** : produit automatiquement au premier démarrage, conservé dans un volume du déploiement et lu par la base et par l'application ; jamais affiché. La base n'est joignable que sur le réseau interne du déploiement. Le déploiement de développement local, qui publie un port de la base, garde son propre réglage et ne sert pas de modèle de production.
- **Clé maître** : générée par l'application au premier démarrage (32 octets aléatoires) et conservée dans un volume distinct de celui de la base, jamais en base ni dans ses sauvegardes. Un export manuel depuis l'administration permet de la conserver séparément. Si la clé est absente ou ne correspond plus, l'application démarre, signale les secrets illisibles (secret Discord notamment) et demande de les ressaisir, au lieu de refuser de démarrer. Cela modifie la décision transversale « clé maître externe conservée séparément » : la séparation reste (clé hors base et hors sauvegarde de la base) mais la clé n'est plus fournie par l'opérateur.
- **Adresse publique dynamique** : déduite de chaque requête (schéma et hôte, en n'honorant les en-têtes de mandataire que s'ils viennent d'un mandataire de confiance). La vérification d'origine des requêtes compare l'origine reçue à l'adresse de la requête elle-même. L'adresse de retour Discord, les liens générés et l'adresse du fichier de déploiement d'un Worker en dérivent. Le cookie de session est sécurisé automatiquement quand la requête est en HTTPS. Un réglage facultatif fixe l'adresse publique pour les déploiements derrière un mandataire. Création d'un Worker : l'adresse proposée est celle de l'accès courant, modifiable, avec un avertissement si elle est locale.
- **Documentation et déploiement** : le fichier d'exemple d'environnement, le guide de déploiement Forgeyard, le guide de sauvegarde et le démarrage rapide sont mis à jour ; la sauvegarde documente deux éléments distincts (base, clé maître) et la procédure de restauration sans clé.
- **Référence visuelle** : le prototype jetable de la branche dédiée et les maquettes servent de modèle ; son code n'est jamais promu, il est réécrit proprement.

## Testing Decisions

- **Ce qu'est un bon test** : il ne vérifie que le comportement observable (ce qu'un utilisateur voit et obtient, ce qu'un client de l'API reçoit), jamais l'organisation interne des composants.
- **Deux seams, tous deux existants** : (1) les parcours navigateur contre l'application lancée pour tout le front ; (2) les tests d'intégration HTTP sur PostgreSQL réel pour les ajouts de backend.
- **Démarrage sans variable** : le déploiement fourni est lancé sans fichier d'environnement et l'installation se termine ; second démarrage conservant mot de passe et clé ; clé absente (secrets signalés à ressaisir, démarrage possible) ; export de la clé réservé à l'Administrateur.
- **Transcription et Workers** : premier Worker (confirmation, profil créé, onglet Transcription rempli) ; Worker suivant (modèle et langue en lecture seule, nom propre facultatif) ; fenêtre du Worker dans l'onglet Transcription ; jeton Hugging Face absent de toute requête vers Snitch ; fichier de déploiement sans exigence de nom de modèle.
- **Membres et compte** : changement de rôle avec confirmation ; suppression avec et sans masquage des clips ; dernier Administrateur et compte courant protégés ; fermeture de compte par saisie de « delete » ou « supprimer » (bouton inactif sinon).
- **Parcours navigateur** : les parcours existants restent verts à chaque lot, leurs sélecteurs adaptés quand l'interface change. Nouveaux parcours : wizard complet et reprise ; code d'installation invalide ; changement de langue (compte et navigateur) ; menu du profil, thème et navigation d'administration ; Vérification (miniature, panneau, confirmation, cas passé) ; groupe de voix (écarter, remettre, nommer) ; fiche de Worker (jeton visible une fois, régénération, révocation) ; page d'un clip et adresse avec moment ; mobile à 390 pixels. Contrôle d'accessibilité automatique sur chaque page.
- **Intégration HTTP** : adresse publique déduite d'une requête directe et d'une requête via un mandataire de confiance ou non (en-têtes ignorés si non fiable) ; origine d'une requête incohérente refusée ; cookie sécurisé selon le schéma ; adresse de retour Discord et fichier de déploiement d'un Worker construits sur l'adresse déduite ; progression du wizard (étapes faites, passées, bandeau fermé, accès réservé à l'Administrateur) ; langue du compte (valeur valide, valeur refusée, lecture après écriture) ; informations de machine (valeur invalide refusée, ancien Worker accepté, visibles aux seuls Administrateurs, absence indiquée).
- **Antécédents** : les parcours navigateur et les tests d'intégration des phases précédentes, notamment la déclaration des modèles d'un Worker pour la validation des champs déclarés.
- Données synthétiques et fixtures dédiées uniquement ; aucun scan de la vraie bibliothèque.

## Out of Scope

- Traduction de la documentation.
- Nouvelles fonctions métier de voix ou de recherche, y compris un état « bruit » ou « à ignorer » pour une voix.
- Restriction de la connexion Discord à l'appartenance à un serveur ou à un rôle, à étudier séparément.
- Changement du format ou de la durée de vie des jetons de Worker.
- Détermination fiable du système de la machine hôte d'un Worker.
- Rotation de la clé maître et rechiffrement des secrets existants avec une nouvelle clé.
- Gestionnaire de secrets externe.
- Refonte du backend ou du contrat hors des ajouts listés.
- Détection automatique des modèles d'un serveur Whisper ou d'embeddings (par la route de liste des modèles) : ticket séparé, à valider.
- Langue de transcription propre à chaque Worker.
- Phase 7 (benchmark réel) : elle garde son autorisation propre.

## Further Notes

- **Risques à confirmer** : (1) une clé maître conservée sur le même hôte que la base protège d'une sauvegarde de base volée, pas d'un hôte compromis ; l'export manuel et la documentation de sauvegarde sont la contrepartie. (2) L'adresse publique déduite repose sur les en-têtes de la requête : elle ne doit être honorée que depuis un mandataire de confiance, sinon un client peut falsifier l'adresse de retour Discord ou celle d'un fichier de déploiement ; d'où le réglage facultatif qui la fixe. (3) Snitch ne vérifie pas que deux noms de modèle différents désignent le même modèle. (4) Un Worker doit joindre Snitch à une adresse qui n'est pas forcément celle du navigateur de l'Administrateur : l'adresse du fichier de déploiement reste modifiable.
- **Question ouverte** : fusionner les Groupes de voix inconnues dans la vue de Vérification (« cette voix ressemble à cinq autres, nommer les six d'un coup ») ou garder deux onglets. Le prototype garde deux onglets ; recommandation : les fusionner. Le backend sépare aujourd'hui les deux lectures.
- **ADR à écrire** : démarrage sans configuration (génération et conservation du mot de passe de la base et de la clé maître, adresse publique déduite, impact sur la décision transversale de clé maître externe et sur la sauvegarde) ; refonte du front (routeur, dépendances, internationalisation, suppression de l'ancienne feuille de style) ; extension de la déclaration d'un Worker (champs et limites de la détection en conteneur).
- **Glossaire** : aucun nouveau terme métier n'est introduit ; les termes de la refonte sont des termes d'interface.
- La phase 7 n'est pas déclenchée par la publication de cette spec. Les tickets enfants et la branche de travail seront créés à la validation.
