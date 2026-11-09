# Benchmarks IA, #287 (empreinte vocale, corpus de clips)

Cette tranche ajoute le socle backend isolé pour comparer des modèles d'empreinte vocale sur les mêmes clips. Elle ne valide pas l'epic. Voir l'[ADR 0056](adr/0056-benchmarks-ia-isoles.md) pour les décisions.

## Réalisé

- Migration `00130_ai_benchmarks.sql` : jeux de données, voix annotées, runs, tâches de run, résultats. Aucune table lue par l'identification, l'attribution ou la bibliothèque.
- API administrateur (`/api/admin/benchmarks/…`, contrat `contracts/openapi.yaml`) : créer, lister et supprimer un jeu de données ; lancer, lister, lire et annuler un run.
- Vérité terrain figée depuis les validations humaines ; références et tests distincts ; une voix n'est jamais sa propre référence.
- Un run programme des tâches à modèle demandé (ADR 0038), donc des empreintes de benchmark que la production ne lit jamais. Un seul run actif.
- Métriques par modèle : EER et seuil, top-1, vrais/faux positifs, faux négatifs, précision, rappel, scores moyens genuine/impostor, durée écoulée, couverture (`complete`, `partial`, `failed`).
- Résultats calculés à la lecture, de façon idempotente et reprenable.
- Tests : métriques (unitaires) et parcours d'intégration (`TestFoundation/ai_benchmark_fingerprints`) couvrant permissions, règles du jeu de données, annulation, run complet, partiel et en échec, isolation de la production et du modèle actif, historique.

## Transcription (tranche 4)

Étape exécutable : tâche `benchmark.transcription.N` (étape à part, jamais lue comme transcription d'un clip), modèle demandé par la tâche, texte de référence figé à la création du corpus, WER et CER. Voir l'[ADR 0058](adr/0058-benchmark-de-transcription.md). Les modèles sont nommés par l'administrateur (seul un Worker joint le service) ; la découverte par les Workers reste à faire. Un Worker doit être mis à jour pour annoncer `benchmark.transcription.*`.

## Diarisation (tranche 5)

Étape exécutable : tâche `benchmark.diarization.N` (jamais une diarisation de production, qui remplacerait les locuteurs locaux), modèle du catalogue prêt sur un Worker, tours de référence figés à la création du corpus, DER avec association optimale des locuteurs et tolérance de 0,25 s. Voir l'[ADR 0059](adr/0059-benchmark-de-diarisation.md).

## Bout en bout (tranche 6)

Chaînes de deux modèles (diarisation + transcription) composées à partir des tâches des deux étapes et mesurées par le cpWER, avec le WER sans attribution et le DER des mêmes sorties. Voir l'[ADR 0060](adr/0060-benchmark-de-bout-en-bout.md). L'écran Benchmarks compose 2 à 4 chaînes (modèle de diarisation prêt sur un Worker + modèle de transcription, de production ou nommé), sans doublon ; les résultats classent par cpWER, avec WER, DER, couverture et temps écoulé. Tests navigateur : `web/e2e/admin-benchmark.spec.ts` (compositeur, lancement, annulation, détail d'un run semé).

## Activation d'un modèle mesuré (tranche 8)

Depuis le détail d'un résultat de transcription ou de diarisation : « Utiliser en production… », avec aperçu (modèle remplacé, Workers prêts, clips déjà analysés dont ceux portant un travail humain) et confirmation explicite. Elle ne change que le modèle du travail à venir : aucune réanalyse n'est lancée, aucune correction ni validation n'est touchée, et chaque changement est enregistré. Voir l'[ADR 0062](adr/0062-activation-explicite-d-un-modele-mesure.md). L'empreinte garde sa transition (ADR 0044).

## Restant

- Découverte des modèles de transcription par les Workers (aujourd'hui l'administrateur les nomme).
- Un seul run actif à la fois ; RAM et VRAM non remontées par le Worker.
- Tests navigateur : les modèles « prêts sur un Worker » ne sont jamais réellement prêts dans les fixtures.
