# Empreintes et reconnaissance sur voix brèves

Première des deux specs issues de [#268](https://github.com/thomas-bsn/snitch/issues/268) (Pipeline V2). Elle achève aussi [#360](https://github.com/thomas-bsn/snitch/issues/360) (recalcul des empreintes historiques). La seconde spec (séparation des voix simultanées, attribution des paroles au mot) viendra après. Décision : [ADR 0064](../adr/0064-reference-cumulative-et-parole-nette.md). Issue : [#387](https://github.com/thomas-bsn/snitch/issues/387).

## Problème

Les clips durent 15 à 30 secondes et chaque Locuteur local n'y parle que quelques secondes. C'est le cas courant de la bibliothèque, pas un cas limite. Trois règles actuelles rendent alors les validations humaines presque sans effet :

- **Une empreinte n'est utilisable qu'avec environ 5 s de parole cumulée dans le clip.** La qualité vaut `min(1, durée / 10 s)` et le seuil est 0,5 (`worker/snitch_worker/voice.py`, `voice_result.go`). Mesure du 8 novembre 2026 : 32 empreintes utilisables sur 94, et 25 références exploitables pour 85 attributions validées.
- **Les validations ne se cumulent pas.** Le score d'une Personne est le meilleur cosinus contre chacune de ses références prises isolément (`voice_matching.go`). Trois extraits de 2 s de la même Personne ne valent pas un extrait de 6 s : ils sont tous les trois rejetés.
- **Les Chevauchements entrent dans l'empreinte.** Un tour pendant lequel un autre Locuteur local parle est inclus tel quel.

S'y ajoutent deux défauts de mesure :

- Les seuils de rapprochement (0,75 et 0,90) sont globaux et n'ont jamais été réglés sur des voix réelles. Ils survivraient à tort à un changement de modèle d'empreinte.
- Le benchmark d'empreinte note par centroïde (ADR 0056) alors que la production note par maximum. Un seuil lu dans l'un ne vaut pas pour l'autre.

## Objectif

Qu'une Personne validée sur quelques clips courts soit reconnue dans les clips suivants, y compris ceux déjà analysés et en attente, pour qu'un traitement de nuit de plusieurs milliers de clips ne laisse à vérifier que les voix nouvelles ou douteuses. Sans nouveau modèle et sans jamais baisser un seuil à l'aveugle.

## Solution

### 1. Parole nette

La Parole nette d'un Locuteur local est sa parole hors Chevauchement : ses tours, moins les intervalles où un autre Locuteur local de la même piste parle. Elle se déduit des tours de la diarisation, sans modèle supplémentaire.

- Le Worker calcule l'empreinte sur les intervalles de Parole nette d'au moins 0,5 s.
- Un Locuteur local sans aucun intervalle net suffisant garde une empreinte calculée sur ses tours entiers, marquée comme chevauchée. Elle n'est jamais éligible comme référence.
- Limite connue : un Locuteur local qui est une fausse détection retire de la Parole nette à ses voisins. L'aperçu chiffré (point 6) mesure la part de parole ainsi perdue.

### 2. Mesures de qualité

Chaque empreinte porte ses mesures, et non plus un seul nombre dérivé de la durée :

| Mesure | Origine |
| --- | --- |
| Durée de Parole nette | géométrie des tours |
| Part de Chevauchement | géométrie des tours |
| Part d'échantillons écrêtés, niveau moyen | audio, côté Worker |
| Cohérence : accord entre les vecteurs des tours d'un même Locuteur local (absente avec un seul tour) | vecteurs, côté Worker |
| Version des mesures | contrat |

Aucune estimation de rapport signal sur bruit : elle n'est pas fiable sur une piste où le jeu est mixé avec les voix. La cohérence est stockée et affichée ; elle ne devient un critère qu'après mesure, car son échelle dépend du modèle.

### 3. Éligibilité à deux usages, décidée par Snitch

Aujourd'hui `usable` est figé à l'enregistrement : changer un seuil ne change aucune ligne. Désormais Snitch garde les mesures et applique une règle versionnée. Changer la règle requalifie les empreintes par une opération en base, sans Worker ni audio.

- **Comparable** : assez de Parole nette pour être rapprochée seule d'une Référence.
- **Contributive** : assez de Parole nette pour entrer dans une Référence une fois validée par un humain. Ce seuil est plus bas que le premier.

Les raisons de rejet restent lisibles et gagnent en précision (`too_short`, `overlap_only`, `clipped`). Les deux durées minimales ne sont pas fixées par cette spec : elles partent de valeurs provisoires et sont arrêtées par l'administrateur après l'aperçu chiffré.

### 4. Référence cumulative

La Référence d'une Personne, pour un modèle d'empreinte donné, cumule toutes ses empreintes contributives validées : moyenne des vecteurs pondérée par la durée de Parole nette, avec un plafond par échantillon pour qu'un seul clip long ne domine pas. Le score d'une voix est son cosinus avec cette Référence.

- Seules les validations humaines actives y entrent. Une Attribution automatique n'y entre jamais (ADR 0008, ADR 0011).
- Elle est recalculée à chaque ajout, retrait ou correction d'un échantillon, à chaque suppression de Profil vocal, et par groupe `(modèle, version, dimensions)` sans aucun mélange (ADR 0044).
- Le mode de score (Référence cumulative ou meilleur échantillon) est un réglage versionné. La Précision mesurée affiche les deux sur les mêmes données ; le cumulatif est le mode par défaut.

L'étiquette de qualité d'une Référence dans l'annuaire reflète ce que le rapprochement utilise réellement : nombre d'échantillons contributifs, clips distincts et Parole nette cumulée, au lieu du nombre de validations.

### 5. Seuils mesurés, par modèle d'empreinte

La [Précision mesurée](../voice-automation.md) rejoue déjà le rapprochement de production sur les échantillons validés, références et tests sur des clips distincts. Elle est étendue plutôt que doublée :

- distribution des scores « même Personne » et « autre Personne » ;
- balayage des seuils : pour chaque valeur, attributions automatiques, erreurs, voix laissées à vérifier ;
- **inconnus simulés** : les tests d'une Personne sont rejoués sans sa Référence et doivent être refusés. Le taux de fausses reconnaissances d'une voix inconnue est ainsi mesuré sans annotation supplémentaire ;
- proposition de seuils (suggestion, automatique, marge d'ambiguïté), appliquée seulement par l'administrateur.

Les seuils sont rattachés au groupe `(modèle, version, dimensions)`. Un groupe sans seuils appliqués ne produit que des suggestions : après un changement de modèle d'empreinte, l'Attribution automatique reste suspendue jusqu'à une nouvelle mesure. Un score n'est jamais présenté comme une probabilité.

Le benchmark d'empreinte (ADR 0056) utilise la même fonction de score que la production et enregistre le mode dans les paramètres du run.

### 6. Aperçu chiffré et recalcul des empreintes existantes (#360)

Une action d'administrateur, en trois temps :

1. **Aperçu, sans effet.** Par Personne et par modèle : empreintes présentes, comparables, contributives, rejetées par raison, sources audio absentes. Pour plusieurs durées minimales (1 s, 1,5 s, 2 s) : empreintes récupérées et Personnes qui obtiennent une Référence.
2. **Requalification.** Les mesures géométriques (Parole nette, Chevauchement) sont calculées en base depuis les tours déjà stockés, et la règle est appliquée. Aucun Worker, aucune lecture audio. Les vecteurs restent ceux d'origine et sont marqués comme tels.
3. **Recalcul.** Des jobs d'empreinte recalculent les vecteurs sur la Parole nette à partir des diarisations existantes. Rien n'est rediarisé. L'ancienne empreinte reste en service jusqu'au succès de la nouvelle. Une source audio absente laisse l'empreinte requalifiée et le signale.

Progression, reprise, annulation et rapport final (récupérées, toujours rejetées, erreurs, sources absentes). Aucune Personne, validation ni Échantillon vocal validé n'est modifié ou supprimé.

### 7. Rapprochement après validation

Valider une Personne relance déjà le rapprochement de toute la bibliothèque. Il s'exécute aujourd'hui dans la requête de validation, sous un verrou global, limité à 5 minutes.

- La validation répond dès qu'elle est enregistrée ; le rapprochement suit en arrière-plan.
- Plusieurs validations rapprochées déclenchent un seul rapprochement.
- L'interface indique qu'un rapprochement est en cours et quand il s'est terminé.
- Sa durée est mesurée sur un volume synthétique de l'ordre de la bibliothèque visée (30 000 Locuteurs locaux, 200 Personnes) et consignée dans le bilan.

## Hors périmètre

- Séparation des voix simultanées, attribution des paroles au mot, détecteur de Chevauchement dédié : seconde spec.
- Changement de modèle de diarisation ou d'empreinte.
- Toute réutilisation d'une Attribution automatique comme référence.
- Diagnostic des jobs d'empreinte morts en `provider_unavailable` relevés dans #360 : incident distinct, à traiter avant de lancer le recalcul sur l'instance.

## Données

Développement et tests sur empreintes et audio synthétiques uniquement. L'aperçu chiffré, la Précision mesurée et le recalcul sont lancés par l'administrateur sur son instance ; aucun agent ne les lance sur la vraie bibliothèque.

## Critères d'acceptation

- Un Locuteur local avec 2 s de Parole nette obtient une empreinte comparable sous une règle à 1,5 s, sans qu'aucun seuil de score n'ait été abaissé (test Worker et API).
- Trois échantillons validés de 2 s d'une même Personne forment une Référence qui reconnaît un quatrième extrait ; aucun des trois n'y suffisait seul sous la règle précédente (test API sur vecteurs synthétiques).
- Un intervalle en Chevauchement n'entre pas dans l'empreinte ; un Locuteur local entièrement chevauché est marqué et n'entre dans aucune Référence.
- Changer la règle d'éligibilité requalifie les empreintes sans job Worker.
- Retirer un échantillon, corriger une validation ou supprimer un Profil vocal recalcule la Référence ; deux groupes `(modèle, version, dimensions)` ne sont jamais mélangés.
- Une Attribution automatique ne modifie jamais une Référence.
- La Précision mesurée affiche distributions, balayage et taux de fausses reconnaissances d'inconnus simulés ; les seuils ne changent que sur action de l'administrateur ; un groupe sans seuils appliqués ne produit aucune Attribution automatique.
- Le benchmark d'empreinte et la production donnent le même score pour les mêmes vecteurs (test de non-dérive).
- L'aperçu ne modifie rien ; le recalcul est idempotent, reprenable et annulable ; une source absente ou un échec ne retire ni Personne ni validation.
- La validation d'une Personne répond sans attendre le rapprochement ; deux validations simultanées ne perdent aucune mise à jour.
- L'étiquette de Référence de l'annuaire suit les échantillons contributifs.

## Tickets

Une seule PR, dans cet ordre.

1. **Parole nette et mesures de qualité d'une empreinte** (#389). Aucun blocage.
2. **Éligibilité décidée par Snitch, à deux usages** (#390). Bloqué par #389.
3. **Référence cumulative et mode de score** (#391). Bloqué par #390.
4. **Rapprochement en arrière-plan après une validation** (#392). Bloqué par #391.
5. **Précision mesurée étendue et seuils par modèle d'empreinte** (#393). Bloqué par #391.
6. **Aperçu chiffré et requalification des empreintes existantes** (#394, première moitié de #360). Bloqué par #390.
7. **Recalcul des empreintes existantes sur la Parole nette** (#395, seconde moitié de #360). Bloqué par #389 et #394.
8. **Étiquette de Référence de l'annuaire** (#396). Bloqué par #391.
9. **Benchmark d'empreinte aligné sur le score de production** (#397). Bloqué par #391.
