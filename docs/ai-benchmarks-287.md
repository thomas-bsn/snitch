# Benchmarks IA, première tranche de #287 (empreinte vocale)

Cette tranche ajoute le socle backend isolé pour comparer des modèles d'empreinte vocale sur les mêmes clips. Elle ne valide pas l'epic. Voir l'[ADR 0056](adr/0056-benchmarks-ia-isoles.md) pour les décisions.

## Réalisé

- Migration `00130_ai_benchmarks.sql` : jeux de données, voix annotées, runs, tâches de run, résultats. Aucune table lue par l'identification, l'attribution ou la bibliothèque.
- API administrateur (`/api/admin/benchmarks/…`, contrat `contracts/openapi.yaml`) : créer, lister et supprimer un jeu de données ; lancer, lister, lire et annuler un run.
- Vérité terrain figée depuis les validations humaines ; références et tests distincts ; une voix n'est jamais sa propre référence.
- Un run programme des tâches à modèle demandé (ADR 0038), donc des empreintes de benchmark que la production ne lit jamais. Un seul run actif.
- Métriques par modèle : EER et seuil, top-1, vrais/faux positifs, faux négatifs, précision, rappel, scores moyens genuine/impostor, durée écoulée, couverture (`complete`, `partial`, `failed`).
- Résultats calculés à la lecture, de façon idempotente et reprenable.
- Tests : métriques (unitaires) et parcours d'intégration (`TestFoundation/ai_benchmark_fingerprints`) couvrant permissions, règles du jeu de données, annulation, run complet, partiel et en échec, isolation de la production et du modèle actif, historique.

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
