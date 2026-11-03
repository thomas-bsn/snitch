# 0030 — Suppression d'analyse par étapes, avec cascade

Statut : accepté. Complète l'[ADR 0024](0024-analyse-de-clip-durable.md).

## Contexte

Seule la suppression définitive du clip existe. L'utilisateur veut supprimer toute l'analyse d'un clip, ou seulement certaines étapes, depuis sa fiche. Des verrous protègent le travail humain : le trigger `local_speakers_keep_validated` refuse de supprimer un Locuteur local validé tant que le clip existe, et les journaux immuables n'acceptent qu'une suppression en cascade.

## Décision

- **Super-administrateur seulement**, depuis la fiche, après une confirmation qui liste ce qui sera perdu : segments et corrections, voix et identifications, voix validées et Personnes dont la Référence perd des échantillons.
- **Cascade par dépendances** : supprimer la transcription supprime l'indexation et les corrections ; supprimer la diarisation supprime les empreintes, le rapprochement et les identifications ; supprimer les empreintes supprime le rapprochement. Les voix ne dépendent pas de la transcription. Tout supprimer retire aussi l'historique des analyses, les essais de diarisation et les notifications du clip.
- **Refus pendant une analyse active** (409) : il faut l'annuler d'abord.
- **Levée contrôlée du verrou** : le trigger accepte la suppression d'un Locuteur local validé uniquement dans la transaction de suppression d'analyse, signalée par un réglage local à cette transaction. Aucune autre voie ne peut le contourner.
- **Trace** : un événement d'audit au niveau du clip (auteur, étapes, date) survit à la suppression. Les Personnes ne sont jamais supprimées.

## Alternatives écartées

- **Supprimer le clip et le réimporter** : perd le fichier, les métadonnées et l'identité du clip.
- **Archiver au lieu de supprimer** : l'utilisateur veut un retrait réel ; l'archivage reste réservé à la rediarisation.
- **Désactiver le trigger globalement** : retire la protection partout.

## Conséquences

- Une analyse `voice` exige toujours une transcription valide (ADR 0024) : après suppression de la seule transcription, les voix restent, mais leur réanalyse impose de retranscrire.
