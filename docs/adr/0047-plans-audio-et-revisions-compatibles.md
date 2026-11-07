# 0047 — Plans audio immuables et révisions de transcription compatibles

Statut : accepté pour le préfactoring #337 du parent #274. Complète les ADR [0012](0012-corrections-de-segments.md), [0024](0024-analyse-de-clip-durable.md) et [0035](0035-analyse-complete-automatique.md). Les interfaces sont détaillées dans [Plans audio](../audio-plans.md).

## Contexte

La piste choisie actuellement est un indice unique ; le dernier job de transcription réussi devient visible et rend les corrections précédentes obsolètes. Cette règle ne convient plus à une analyse qui diagnostique plusieurs pistes puis fusionne leurs paroles : le premier enfant réussi ne doit publier ni un résultat partiel ni des pertes de corrections. Les indices ffprobe ne sont pas les positions d'une liste. Une durée de piste inconnue n'est pas la durée du conteneur ; son décalage peut être négatif.

## Décision

Un Run d'analyse peut posséder plusieurs Plans audio immuables. Le Plan capture le mode manuel ou Auto, l'inventaire observé, les indices réels sélectionnés triés et uniques, les décisions humaines (action, choix forcé, motif, auteur, date), les versions du détecteur, de sa configuration, de la fusion, du pipeline et du vocabulaire. Son identité est le digest SHA-256 de l'entrée canonique, unique dans le Run. Répéter l'entrée retrouve le Plan ; modifier un choix ou une version crée un autre Plan. Cela prépare le remplacement de la sélection exclusivement monopiste, sans activer Auto ni changer le formulaire actuel.

Un enfant est lié au Plan, à la piste, à l'étape et à sa propre version d'entrée **dans la transaction de création du job, avant toute prise par un Worker**. Le job porte ses tentatives ; le lien n'est pas réaffecté à un autre job. Les diagnostics terminaux sont immuables par tentative. `unexamined`, `ambiguous` et `failed` ne signifient jamais `no_speech`. Les décisions de sélection restent distinctes des observations du détecteur.

Une Révision de transcription possède un manifeste immuable et une identité indépendante d'un job unique. Chaque segment et mot conserve toutes ses sources : indice réel de piste, job, indice source de segment et, pour un mot, indice source de mot. Les temps sont exprimés dans le repère du clip ; l'adaptation d'un temps relatif à la piste applique son décalage avant de construire le manifeste. Le module valide l'ordre temporel et les sources réellement présentes dans des enfants de transcription réussis. L'ordre est début, fin, indice réel de piste et identité source ; le contenu canonique complet départage les ex æquo restants, indépendamment de l'ordre d'arrivée des enfants.

Le préfactoring stocke des **candidats**, sans les publier. #340 doit vérifier leur complétude, l'aperçu des pertes et sa confirmation, puis publier atomiquement la révision et ses effets sur corrections, voix, indexation et lecteurs. La réussite d'un enfant contourne les effets de publication monopiste (obsolescence des corrections, indexation et projection de résultat) ; le résultat précédent reste visible. Une révision candidate n'est pas une preuve de complétude, même si elle est vide.

Les résultats historiques deviennent des révisions `legacy:<job_id>` sans ASR et sans changement de job ou d'indice de segment. Le résultat HTTP conserve ses anciens champs et gagne `revision`. Les corrections, recherches et exports gardent donc leurs identités et leur comportement. Une piste historique inconnue porte `-1`, réservé à cette projection. Les futurs résultats de l'ancien protocole sont projetés à leur réussite. Une garde optionnelle `expected_revision_id` protège les nouvelles corrections ; les anciens clients restent acceptés.

## Alternatives écartées

- Modifier le choix d'un Plan existant : rendrait les reprises et leurs résultats impossibles à expliquer.
- Un job synthétique de fusion remplaçant les sources : perdrait la provenance multiple et brouillerait les corrections.
- Retranscrire les anciens clips pour fabriquer une révision : coûteux, non déterministe et destructeur pour les identités de correction.
- Publier chaque enfant à sa réussite : exposerait une analyse partielle et invaliderait des corrections avant confirmation.
- Activer Auto avec ces seuls contrats : le diagnostic, l'orchestration et la publication complète appartiennent aux tickets suivants.

## Conséquences

La migration 00111 ajoute les Plans, pistes, liens enfants, observations, révisions et la projection historique. Les identités refusent les mises à jour ; les suppressions explicites d'analyse ou de clip peuvent les supprimer en cascade. Le résultat original du Worker et les corrections restent séparés. Le retour de migration retire uniquement ce nouveau stockage ; les résultats historiques et corrections survivent.

Les quatre chantiers suivants utilisent le même contrat généré et une fixture JSON synthétique partagée. #339 doit adapter sa création de jobs aux transactions du dépôt ; #340 doit migrer ensemble tous les lecteurs et l'identité des corrections pour les révisions multitracks. Aucune API de mutation Auto ni aucun scan réel n'est ajouté ici.
