# 0062 — Activer un modèle mesuré : explicite, jamais une réanalyse

Date : 2026-11-09. Statut : accepté pour #287. S'appuie sur l'[ADR 0044](0044-changement-du-modele-d-empreinte.md) (transition d'empreinte) et sur les ADR 0056 à 0061.

## Contexte

Un benchmark désigne parfois un meilleur modèle que celui de la production. Le mettre en production change ce que produiront les analyses à venir, et peut tenter de refaire les clips déjà analysés, dont certains portent le travail d'un humain (corrections de texte, validations de locuteurs).

## Décision

- **Un changement de modèle n'est jamais automatique** : ni à la fin d'un benchmark, ni parce qu'un modèle arrive premier. Il se demande depuis le détail d'un résultat (transcription, diarisation), avec un aperçu puis une confirmation.
- **Il ne change que le modèle du travail à venir.** Rien n'est planifié, aucune correction ni validation n'est touchée, les clips déjà analysés gardent leur résultat. La réanalyse reste une action séparée des écrans qui la portent déjà (l'aperçu et la confirmation de la transcription en lot, qui excluent par défaut les clips corrigés), et l'aperçu d'activation y renvoie.
- **L'aperçu dit ce qui est en jeu** : le modèle remplacé, le nouveau, les Workers qui le détiennent (diarisation), le nombre de clips déjà analysés et celui des clips portant un travail humain.
- **La confirmation nomme le modèle remplacé** (`expected_current`) : un changement survenu depuis l'aperçu est refusé (409), pas écrasé. Une case à cocher exprime que le changement ne vaut que pour le travail à venir. Un modèle déjà en production ne peut pas être « changé ».
- **Diarisation** : le modèle doit être détenu par un Worker (même règle que l'activation existante) ; les seuils du pipeline restent. **Transcription** : seul le modèle change ; la langue, la préparation et la détection de parole restent.
- **L'empreinte** change par sa transition (ADR 0044), qui recalcule les références avant de basculer ; une chaîne n'est pas un modèle. Les deux sont refusés ici (`benchmark_activation_unsupported`).
- **Chaque changement est enregistré** avec son auteur, son ancien et son nouveau modèle et le run qui l'a motivé (`benchmark_activations`).

## Alternatives écartées

- **Activer à la fin d'un run** ou proposer un bouton sans confirmation : une mesure sur un petit corpus, biaisée en faveur de la production (ADR 0057), ne doit pas décider seule.
- **Lancer la réanalyse avec l'activation** : mettrait en jeu le travail humain sans que l'administrateur l'ait choisi.

## Conséquences

- Pour faire profiter les clips existants du nouveau modèle, l'administrateur passe ensuite par la réanalyse, qui a ses propres garde-fous.
- La transition d'empreinte et cette activation sont deux chemins distincts, chacun avec le contrôle qui convient à son modèle.
