# 0031 — Voix ignorée, voix Random et refus d'identification

Statut : accepté. Revient sur la décision de la phase 6 bis « aucun état bruit ou à ignorer n'est créé ».

## Contexte

La diarisation produit de fausses voix (bruit, mauvais micro) et des voix réelles qu'on ne veut pas identifier. Elles restent indéfiniment dans les Cas de vérification et les groupes. Par ailleurs, retirer une identification erronée n'empêche pas le rapprochement automatique de la reproposer.

## Décision

- **Disposition d'un Locuteur local**, distincte du niveau de décision : `ignorée` ou `random`. Elle est réversible, journalisée, et demande les mêmes droits que la validation.
  - Ignorée : aucun locuteur affiché pour ses segments ; exclue des statistiques, des groupes, de la vérification et du rapprochement.
  - Random : ses segments sont attribués à « Random » ; exclue des groupes, de la vérification et du rapprochement.
- **Refus d'identification** : « ce n'est pas X » retire l'attribution, la voix redevient `speaker_n` et un refus (voix, Personne) est enregistré. Le rapprochement ne propose plus cette Personne pour cette voix. Si l'attribution était validée, ses échantillons quittent la Référence.
  - Le refus est réversible : « Annuler le refus » d'une Personne refusée pour la voix, avec les mêmes droits que le refus, refusé pendant une analyse active comme lui (`analysis_active`). Refus et annulation sont journalisés (append-only). Après l'annulation, le rapprochement peut de nouveau proposer cette Personne ; l'attribution retirée par le refus n'est pas rétablie.
- Appliquée à un Groupe de voix inconnues, une disposition vaut pour chacune de ses voix.

## Alternatives écartées

- **Nouveaux niveaux de décision** : mélangerait « qui est-ce ? » et « faut-il compter cette voix ? ».
- **Supprimer la voix** : perdrait ses segments, de façon irréversible.

## Conséquences

- Toute lecture des voix (vérification, groupes, À traiter, rapprochement, statistiques, transcription) doit appliquer la disposition.
- Une nouvelle diarisation remplace les Locuteurs locaux d'une piste audio. Dans la transaction qui l'ingère, avant la suppression des anciennes voix, chaque disposition et chaque refus d'une ancienne voix passe à la nouvelle voix dont les tours couvrent la plus grande part de son temps de parole, si cette part atteint la moitié et que le choix est sans ambiguïté. Deux anciennes voix qui aboutissent à la même nouvelle voix avec des dispositions différentes (l'absence de disposition comprise) n'en transmettent aucune ; leurs refus s'additionnent. Le reste est abandonné. Chaque report et chaque abandon est journalisé par le système avec son motif.
- « Supprimer… la diarisation » sans voix de remplacement abandonne dispositions et refus, comptés dans la perte ; la désignation d'une piste microphone aussi, journalisée.
