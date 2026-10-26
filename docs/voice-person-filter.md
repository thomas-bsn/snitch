# Filtre par Personne (recherche et bibliothèque)

Périmètre du ticket #39 (phase 5). Le filtre ne s'appuie que sur les attributions tracées (table `speaker_attributions`) : aucune voix n'est reconnue en dehors d'elles.

## Règles

- **Attribution effective** d'un Locuteur local : la ligne validée (humaine) si elle existe, sinon la ligne machine, sinon aucune.
- **Personne identifiée** : niveau de décision `human` ou `automatic` avec une Personne. Une suggestion, une ambiguïté ou l'absence d'attribution forment l'état **inconnu** ; le niveau tracé reste affiché sur le résultat.
- **Segment de recherche** : attribué au Locuteur local dont les tours recouvrent le plus le segment (même règle que l'affichage du clip, égalité au plus petit rang). Un segment sans recouvrement est inconnu.
- **Clip de la bibliothèque** : il correspond à une Personne s'il contient une voix identifiée comme elle ; il correspond à « inconnu » s'il contient au moins une voix non identifiée ou s'il n'a aucun Locuteur local (clip jamais diarisé : ses segments sont inconnus en recherche). Un clip dont toutes les voix sont identifiées n'est pas inconnu. La bibliothèque et la recherche appliquent ainsi la même règle : sans voix identifiée, c'est inconnu.

## API

- `GET /api/search` et `GET /api/clips` acceptent `person_id` (entier) ou `person_unknown=true` (exclusifs ; sinon `400 invalid_filter`).
- Les trois modes de recherche (exact, plein texte avec approximatif, hybride) appliquent le filtre au segment. En hybride, les candidats sémantiques sont filtrés avant leur coupe à 100 ; les candidats sémantiques n'exigeant pas de correspondance de mots, ils peuvent être d'autres segments de la Personne.
- Visibilité, autres filtres et filtre Personne s'appliquent avant le total et la pagination ; aucun extrait de clip privé n'est exposé.
- Chaque résultat de recherche porte `person` (ou `null`) et `decision_level` (`unknown` sans attribution). Chaque clip porte `persons` (Personnes identifiées, triées par nom, niveau humain prioritaire).
- La planification par lot de transcriptions (administrateur) n'utilise pas ce filtre : elle précède toute attribution vocale.

## Interface

La recherche et la bibliothèque proposent un sélecteur « Personne » (toutes, voix inconnues, ou une Personne). Chaque résultat indique la Personne ou « inconnue » et s'ouvre au timecode du segment ; les cartes de clip et le détail listent les Personnes identifiées avec leur niveau.
