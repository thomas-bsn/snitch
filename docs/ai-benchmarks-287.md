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

## Restant

- Écran Admin → AI Models (Models, Benchmarks, Results) : attend #281.
- Diarisation (DER), transcription via le service externe (WER/CER), séparation (SI-SDRi), recherche (Recall@K, MRR).
- Catalogue multi-modèles par Worker : charge à la demande, contraintes RAM/VRAM, suppression d'un modèle inutilisé (#265, #114 couvrent l'existant).
- Activation du modèle de production avec plan de réanalyse : la transition de l'[ADR 0044](adr/0044-changement-du-modele-d-empreinte.md) reste le chemin ; aucun benchmark ne la déclenche.
- E2E UI.

## Écarts et limites

- Le temps écoulé inclut l'attente en file ; ce n'est pas une latence d'inférence. RAM et VRAM ne sont pas remontées par le Worker : `resources` vaut `null`.
- Un seul run actif à la fois.
- Tests faits sur données synthétiques uniquement ; aucun benchmark sur la vraie bibliothèque, qui relève de la phase 7 et de son autorisation.

## Corpus de clips (tranche 2)

Le jeu de voix devient un corpus de clips annotés avec certification « relu en entier » ; les étapes mesurables en sont déduites. Voir l'[ADR 0057](adr/0057-corpus-de-benchmark-et-verite-terrain.md). Routes ajoutées : `GET /api/admin/benchmarks/clips` et `GET /api/admin/benchmarks/models?stage=fingerprint`. Seule l'étape empreinte est exécutable ; transcription, diarisation et bout en bout viennent ensuite. Références de transcription et de diarisation : sortie de production corrigée, donc biaisée en faveur du modèle de production (annoncé dans les résultats).

## Écran Benchmark (tranche 3)

Onglet **Benchmark** de l'administration, réservé aux administrateurs : **Benchmarks** (choisir un corpus, une étape et 2 à 4 modèles, lancer, suivre, annuler) et **Results** (meilleur modèle par étape, historique, détail d'un run avec infobulles « i » sur chaque mesure). Un corpus se crée depuis des clips : rôle test, référence ou exclu, et certification « relu en entier » ; les étapes mesurables sont annoncées en direct. Les étapes pas encore exécutables (transcription, diarisation, bout en bout) s'affichent « Bientôt », jamais comme un bouton factice. Un modèle prêt sur aucun Worker est désactivé avec la raison.

Référence visuelle : `design/ai-models/` (prototype jetable). Tests : `web/e2e/admin-benchmark.spec.ts` (accès réservé, création d'un corpus, résultats d'un run terminé, infobulles au survol et au clavier, accessibilité axe) ; les runs affichés sont semés par `benchmark_seed_integration_test.go`. Le suivi d'un run en cours (bandeau de progression, annulation) n'est couvert que par les tests d'API : le test navigateur demanderait deux modèles prêts sur un Worker.

## Restant

- Bout en bout : chaînes de trois modèles.
- Import de références externes (STM, RTTM) pour lever le biais en faveur du modèle de production.
- Activation du modèle de production avec plan de réanalyse (la transition de l'ADR 0044 reste le chemin).
