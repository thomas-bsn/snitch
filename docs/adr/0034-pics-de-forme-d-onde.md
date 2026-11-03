# 0034 — Pics de forme d'onde calculés par le Worker

Statut : accepté.

## Contexte

La fiche d'un clip et l'écran de traitement des voix affichent une Forme d'onde colorée par voix. Snitch ne produit aujourd'hui aucune donnée d'amplitude. Le NAS (N100, sans GPU, 8 Go partagés) ne doit pas décoder des heures d'audio.

## Décision

- Le Worker calcule les pics d'amplitude (quelques milliers de valeurs par piste) pendant la préparation audio qu'il fait déjà pour la transcription et la diarisation, et les envoie avec son résultat. Le serveur les stocke par clip et par piste audio.
- La couleur vient des tours de parole déjà connus ; les pics ne contiennent aucune information de voix.
- Sans pics (clip analysé avant ce changement), l'interface affiche les pistes de parole sans tracé ; une nouvelle analyse les produit.

## Alternatives écartées

- **Calcul sur le NAS avec ffmpeg** : charge CPU disproportionnée pour le N100.
- **Calcul dans le navigateur** : il faudrait télécharger l'audio entier pour le dessiner.

## Conséquences

- Le contrat du Worker est étendu de façon rétrocompatible : un ancien Worker n'envoie pas de pics.
