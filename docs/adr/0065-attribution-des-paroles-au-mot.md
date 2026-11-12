# 0065 — Attribution des paroles au mot, appliquée à la lecture

Date : 2026-11-11. Statut : proposé, à relire par l'utilisateur. Complète l'[ADR 0011](0011-locuteurs-personnes-et-attributions.md), l'[ADR 0012](0012-corrections-de-segments.md) et l'[ADR 0052](0052-voix-par-piste-d-un-plan-audio.md). Spec : [paroles simultanées](../specs/paroles-simultanees.md), section 1. Tickets #388 et #398.

## Contexte

Un segment transcrit recevait en entier le Locuteur local qui y parle le plus. Quand deux personnes se répondent dans le même segment, les mots de la seconde étaient donnés à la première, alors que la transcription fournit déjà des mots horodatés et la diarisation des tours.

Trois lectures dépendent de cette attribution : la fiche d'un clip, la Couverture d'un Locuteur local et le filtre de recherche par Personne. Le benchmark de bout en bout (ADR 0060) doit en plus mesurer la même règle que la production. Les Corrections (ADR 0012) sont rattachées à un segment par son identité : rien ne doit la changer.

## Décision

1. **Une règle versionnée, une fonction pure.** `segmentparts.Attribute(règle, segment, mots, tours)` ne reçoit que des valeurs : les bornes du segment, les horodatages de ses mots, les tours de sa piste. Elle rend les parts du segment. La version 1 (`RuleSegment`) est la règle d'avant ; la version 2 (`RuleWord`) est la règle au mot. Une version ne change jamais de comportement ; un nouveau comportement est une nouvelle version. La fiche d'un clip, la Couverture, la trace vocale et le benchmark appellent cette fonction.
2. **La règle au mot.** Chaque mot prend le Locuteur local dont les tours le recouvrent le plus. À égalité, il garde celui du mot précédent, sinon le premier dans l'ordre des tours. Un mot qu'aucun tour ne recouvre prend le locuteur du mot couvert précédent, sinon du suivant. Une voix qui ne tient qu'un mot entre deux mots d'une même autre voix lui est rattachée, de gauche à droite. Si aucun mot n'est couvert, ou si le segment n'a pas de mots horodatés, la règle par segment s'applique ; sans aucun tour, le segment reste sans locuteur.
3. **Des parts, pas de nouveaux segments.** Un segment est rendu avec `parts` : des suites de mots consécutifs, chacune avec son locuteur. Le nombre de segments, leurs bornes, leur index et leur tâche d'origine ne changent pas. Le locuteur du segment (`local_speaker_id`) garde son sens : la voix qui y parle le plus, ou celle d'une Correction.
4. **Les Corrections priment, au segment entier.** Une Correction de locuteur donne une seule part, à la voix désignée. Un segment au texte corrigé n'a plus de mots horodatés fiables (ADR 0012) : une seule part, au locuteur de la règle par segment.
5. **Appliquée à la lecture.** Rien n'est stocké : ni parts, ni locuteur par mot. La règle active est un réglage d'administrateur (`speaker_attribution_rules`, historique en ajout seul, la version la plus haute est active ; `GET` et `PUT /api/admin/speaker-attribution-rule`). La règle au mot est le défaut. Revenir à la règle par segment prend effet à la lecture suivante et ne réécrit aucune donnée.
6. **La Couverture suit les parts.** Une voix qui tient une part d'un segment n'y est pas comptée comme parlant sous les paroles des autres ; `attributed_segments` compte les segments dont elle tient une part. Avec la règle par segment, le calcul est celui d'avant.
7. **Le filtre par Personne suit les parts, en SQL.** La recherche filtre, compte et pagine en base. Avec la règle au mot, elle appelle la fonction SQL `segment_part_speakers`, jumelle de la fonction Go : un segment est retenu dès qu'une de ses parts est attribuée à la Personne, et le résultat affiche alors cette Personne. Avec la règle par segment, la requête d'avant est conservée telle quelle. Un test de parité compare les deux implémentations sur des cas écrits à la main et des cas générés (égalités, mots sans tour, mots isolés, voix d'une autre piste).
8. **Une piste à la fois.** La fonction ne reçoit que les tours de la piste du segment (ADR 0052) ; la fonction SQL filtre sur la piste.

## Alternatives écartées

- **Redécouper les segments aux changements de voix.** Change leur nombre et leur identité : les Corrections et leur historique ne s'y rattacheraient plus (ADR 0012).
- **Stocker le locuteur de chaque mot, ou les parts.** Une donnée dérivée de plus à maintenir à chaque rediarisation, Correction ou changement de règle ; revenir en arrière demanderait une réécriture.
- **Filtrer par Personne en Go, après la requête.** Le total et la pagination ne seraient plus calculés en base ; le filtre « voix inconnues » obligerait à charger tous les résultats d'une recherche.
- **Une seule implémentation, en SQL.** Le benchmark et la fiche d'un clip auraient besoin de la base pour attribuer des mots qu'ils ont déjà en mémoire, et la règle serait plus difficile à tester.
- **Garder le mot isolé d'une autre voix.** Les horodatages de mots sont approximatifs : un mot seul entre deux mots d'une même voix est plus souvent une erreur d'alignement qu'une réponse.
- **Un modèle d'alignement de mots dédié.** Aucun besoin démontré (hors périmètre de la spec).

## Conséquences

- La règle existe en deux langages. Le test de parité est le garde-fou : toute évolution de la règle au mot touche la fonction Go, la fonction SQL (nouvelle migration) et ce test.
- Une réponse d'un seul mot entre deux mots d'une autre voix n'apparaît pas comme une part. C'est le prix du lissage.
- Le locuteur du segment peut, dans de rares cas, ne tenir aucune part (il parle entre les mots). Les parts font foi pour « qui a dit quoi » ; le locuteur du segment reste celui que désigne ou remplace une Correction.
- L'historique d'une Correction de locuteur continue de nommer le locuteur déduit par la règle par segment.
- La fiche d'un clip transporte les mots deux fois (dans le segment et dans ses parts). Les clips durent quelques dizaines de secondes.
- Le gain réel de la règle au mot sera mesuré par le cpWER des deux règles dans le benchmark de bout en bout (#400).
