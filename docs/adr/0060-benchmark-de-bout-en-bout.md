# 0060 — Benchmark de bout en bout : chaînes et cpWER

Date : 2026-11-09. Statut : accepté pour #287. S'appuie sur les [ADR 0058](0058-benchmark-de-transcription.md) et [0059](0059-benchmark-de-diarisation.md).

## Contexte

Le meilleur modèle de chaque étape n'est pas forcément la meilleure combinaison. Mesurer « qui a dit quoi » demande de composer une diarisation et une transcription. Y ajouter l'empreinte demanderait une variante de l'empreinte lisant une diarisation de benchmark, donc un changement du Worker.

## Décision

- **Une chaîne est une paire** : un modèle de diarisation du catalogue et un modèle de transcription (nom du service). Un run compare 1 à 4 chaînes, distinctes.
- **Aucune étape de tâche nouvelle** : une chaîne est calculée à partir des tâches `benchmark.diarization.N` et `benchmark.transcription.N`. Deux chaînes qui partagent un modèle partagent ses tâches : un modèle ne tourne jamais deux fois sur le même clip.
- **cpWER** (« concatenated minimum-permutation WER ») : chaque segment transcrit prend le locuteur dont les tours le recouvrent le plus (règle par segment ; depuis #400, un second cpWER applique la règle au mot, toutes deux par `segmentparts.Attribute`, ADR 0065) ; les mots de chaque locuteur sont concaténés dans l'ordre du temps ; les locuteurs du modèle sont associés à ceux de la référence par l'affectation qui minimise les erreurs de mots (algorithme hongrois) ; un locuteur absent d'un côté coûte tous les mots de l'autre. Le WER sans attribution et le DER des mêmes sorties sont rapportés : le cpWER voit ce que le WER ne voit pas (un bon texte donné au mauvais locuteur).
- **Le locuteur de chaque segment de référence est figé avec lui** (celui dont les tours de référence le recouvrent le plus).
- **Étape mesurable** quand le corpus a un clip certifié avec transcription et voix validée. Elle est retirée si l'horloge d'un flux multipiste est décalée par rapport à la transcription (décalage de plan non nul) : les temps ne se composeraient pas.
- **L'identification d'une Personne reste une étape à part** (empreinte) : elle n'est pas dans la chaîne.
- Le biais de l'ADR 0057 s'applique (référence issue de la production corrigée).

## Alternatives écartées

- **Chaîner l'empreinte** : demande une entrée de Worker pour une diarisation de benchmark.
- **Un WER seul** : ignore l'attribution, qui est la raison d'être d'un bout en bout.

## Conséquences

- Aucun changement de Worker pour cette étape.
- Un corpus multipiste décalé n'a pas d'étape bout en bout ; il garde les deux autres.
