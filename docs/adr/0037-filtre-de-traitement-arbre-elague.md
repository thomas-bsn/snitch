# 0037 — Filtre de traitement de la Bibliothèque : arbre élagué

Statut : accepté. Amende la ligne « Filtres » de l'[ADR 0032](0032-explorateur-du-disque-et-indexation-a-la-demande.md) pour le seul filtre de traitement ; les autres filtres restent à plat.

## Contexte

L'ADR 0032 rend les résultats filtrés « à plat, avec leur dossier » : les dossiers disparaissent de la vue. Pour trouver, par exemple, les vidéos transcrites, il faut alors perdre l'arborescence. L'état proposé (`transcribed`) ne répondait pas non plus à « transcrite » : il exclut les vidéos dont les voix sont traitées (`analyzed`).

## Décision

- **Paramètre `stage`** de `GET /api/library/explorer`, à côté de `state` (qui garde ses six valeurs, dont les échecs et le « en cours »). Étapes :
  - `indexed` : le clip existe (au sens du glossaire, une Vidéo non indexée n'a pas de clip). Ce n'est pas l'index sémantique ;
  - `transcribed` : une transcription exploitable existe (travail de transcription terminé, ou étape `transcription` terminée) ;
  - `voices` : étapes `diarization`, `fingerprints` et `matching` terminées ;
  - `complete` : `transcription`, `indexing`, `diarization`, `fingerprints` et `matching` terminées.
- Les étapes se lisent dans `clip_analysis_steps`, source commune de la Task Queue et de la fiche clip : pour chaque étape, la plus récente exécution non annulée où elle s'applique. Une exécution de portée « voix » ne rend donc jamais une analyse « complète ».
- **Arbre élagué** : sans `recursive`, un filtre actif répond pour le dossier demandé avec ses vidéos qui correspondent et, comme dossiers, les seuls sous-dossiers qui mènent à une correspondance, avec `match_count`. Un dossier sans correspondance, même dans ses descendants, disparaît ; aucun fichier ni enregistrement n'est touché. Avec `recursive`, la réponse reste plate (analyse d'un dossier).
- **Interface** : seule l'étape de traitement demande l'arbre ; les autres filtres (nom, jeu, dates, visibilité, propriétaire, Personne, voix à traiter, état) gardent la vue plate. Les combiner avec une étape donne l'arbre.
- **Droits** : les compteurs ne comptent que ce que l'appelant peut lire. Les Dossiers privés restent exclus du parcours pour qui ne peut pas les ouvrir (ADR 0026).

## Alternatives écartées

- **Remplacer `state`** : ses valeurs « en cours » et « échec » n'ont pas d'équivalent dans les étapes.
- **Arbre pour tous les filtres** : change le comportement et les tests existants des filtres de métadonnées sans que le besoin soit établi.
- **Index sémantique pour « Indexées »** : aucun marqueur par clip n'existe côté liste ; « indexée » suit le glossaire.

## Conséquences

- Le parcours récursif est borné par `explorerMaxEntries` : au-delà, `video_selection_too_large`, comme pour la vue plate.
- Le champ `state` d'une vidéo garde sa sémantique actuelle ; l'étape et l'état peuvent se contredire en apparence (`analyzed` est aussi `transcribed`), c'est voulu.
- Non couvert ici : mise à jour en direct des résultats sans rechargement (#270) et retour au dossier après une fiche clip (#276).
