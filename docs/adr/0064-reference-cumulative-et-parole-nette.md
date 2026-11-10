# 0064 — Référence cumulative, Parole nette et éligibilité décidée par Snitch

Date : 2026-11-11. Statut : proposé, à relire par l'utilisateur. Complète l'[ADR 0008](0008-identification-progressive.md), l'[ADR 0011](0011-locuteurs-personnes-et-attributions.md) et l'[ADR 0044](0044-changement-du-modele-d-empreinte.md). Spec : [voix brèves](../specs/voix-breves.md). Tickets #268 et #360.

## Contexte

Les clips durent 15 à 30 secondes et chaque Locuteur local y parle quelques secondes. Trois choix de la phase 5 rendent ces voix inexploitables :

- la qualité d'une empreinte dépend de la seule durée cumulée dans le clip et exige environ 5 s ;
- le score d'une Personne est le meilleur cosinus contre chacune de ses références prises une à une, donc des extraits courts ne se cumulent jamais ;
- `usable` est figé à l'enregistrement de l'empreinte, donc tout changement de règle demande un recalcul.

Par ailleurs les tours en Chevauchement entrent dans l'empreinte, et les seuils de rapprochement sont globaux alors que l'échelle des scores dépend du modèle.

## Décision

1. **La Référence d'une Personne est cumulative.** Pour un groupe `(modèle, version, dimensions)`, c'est la moyenne des vecteurs de ses échantillons validés contributifs, pondérée par leur durée de Parole nette, plafonnée par échantillon. Le score d'une voix est son cosinus avec cette Référence.
2. **L'empreinte est calculée sur la Parole nette** : les tours du Locuteur local, moins les intervalles où un autre Locuteur local de la même piste parle. Le Chevauchement se déduit des tours ; aucun modèle n'est ajouté.
3. **Deux usages, deux seuils.** Une empreinte est *comparable* (rapprochée seule d'une Référence) ou seulement *contributive* (elle entre dans une Référence après validation humaine). Le second seuil est plus bas.
4. **Snitch décide de l'éligibilité.** Le Worker rapporte des mesures (Parole nette, part de Chevauchement, écrêtage, niveau, cohérence des tours) ; Snitch applique une règle versionnée. Changer la règle est une opération en base.
5. **Les seuils de rapprochement appartiennent au groupe d'empreintes.** Un groupe sans seuils appliqués par l'administrateur ne produit que des suggestions.
6. **Le mode de score reste un réglage mesurable** (Référence cumulative ou meilleur échantillon), pour que le choix soit vérifié sur les données de l'installation et réversible.

## Alternatives écartées

- **Baisser `min_quality`.** Rend comparables des extraits de 1 s pris isolément, sans cumul : plus de faux positifs pour un gain faible.
- **Garder le meilleur échantillon comme seul score.** Il exige que chaque échantillon soit comparable seul, ce qui écarte les voix brèves. Il fait aussi monter le score avec le nombre de références, par simple chance, donc le seuil dérive quand la base grossit.
- **Séparer les voix avant de calculer l'empreinte.** Un séparateur laisse des artefacts dans le signal ; écarter la zone chevauchée ne coûte rien et ne dégrade pas le reste. À réévaluer seulement sur preuve par benchmark.
- **Estimer un rapport signal sur bruit.** Non fiable sur une piste où le jeu est mixé avec les voix.
- **Enrichir la Référence avec les Attributions automatiques.** Une erreur se propagerait à toute la bibliothèque (ADR 0008).
- **Détecteur de Chevauchement dédié.** Aucun besoin démontré tant que les tours de la diarisation ne sont pas pris en défaut.

## Conséquences

- Une moyenne efface les variations d'une même voix (autre micro, autre contexte) que le meilleur échantillon tolérait. C'est la raison du réglage du point 6 et de la mesure comparée.
- Un Locuteur local qui est une fausse détection retire de la Parole nette à ses voisins. La part perdue est mesurée par l'aperçu chiffré.
- La Référence devient une donnée dérivée à maintenir : elle est recalculée à chaque changement d'échantillon et reconstruite après un Changement du modèle d'empreinte, dont les règles restent celles de l'ADR 0044.
- Les empreintes existantes gardent leur vecteur d'origine tant qu'un recalcul explicite ne les a pas remplacées ; leur éligibilité, elle, est requalifiée sans Worker.
- Les valeurs par défaut des seuils cessent d'être des constantes du code : elles viennent d'une mesure appliquée par l'administrateur.
