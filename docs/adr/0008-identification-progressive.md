# Identification vocale progressive à partir d'un lot initial

Statut : objectif confirmé par l'utilisateur ; critères d'activation à mesurer et à détailler dans la spec de phase 5.

## Contexte

La bibliothèque représente environ 1 To de clips courts. Confirmer chaque personne dans chaque clip rendrait le traitement initial impraticable. L'utilisateur souhaite renseigner les voix sur un lot de départ puis réduire progressivement son intervention.

## Décision

Constituer des références à partir d'échantillons vocaux validés pour chaque personne, puis évaluer leur capacité à reconnaître cette personne sur d'autres clips. Après atteinte des critères mesurés et activation par l'administrateur, attribuer automatiquement les occurrences suffisamment fiables ; garder les occurrences inconnues ou ambiguës à vérifier.

Les validations et corrections humaines enrichissent les références. L'amélioration attendue vient de ces références et du rapprochement vocal, sans exiger de réentraîner le modèle. Distinguer les attributions automatiques des validations humaines ; une prédiction automatique ne constitue pas à elle seule un échantillon validé.

Cette décision précise la « confirmation initiale » du plan : elle n'impose pas de confirmer chaque occurrence de chaque clip pendant tout le batch. Elle conserve l'évaluation préalable, le contrôle administrateur de l'automatisation, la traçabilité et la priorité des corrections humaines.

## Alternatives écartées

- Validation manuelle exhaustive de toutes les occurrences : charge incompatible avec le volume visé.
- Attribution automatique dès le départ sans références évaluées : qualité inconnue et risque de propagation d'erreurs.
- Réentraînement systématique à chaque validation : aucune nécessité démontrée ; les empreintes déjà extraites doivent permettre de réexaminer les attributions sans retraiter les vidéos.

## Conséquences

La phase 5 devra prévoir une constitution pratique du lot initial et une interface de vérification centrée sur les incertitudes. La phase 7 évaluera les références sur des clips distincts de ceux ayant servi à les constituer, la précision par personne, la couverture automatique et le volume de vérification restant.

Nombre d'échantillons, seuils et garanties de qualité restent ouverts. L'ajout de références peut aider sans garantir une amélioration systématique. Le benchmark et son autorisation préalable restent nécessaires avant le batch de la vraie bibliothèque.
