# 0057 — Corpus de benchmark de clips annotés et vérité terrain

Date : 2026-11-09. Statut : accepté pour #287. Amende l'[ADR 0056](0056-benchmarks-ia-isoles.md) : le jeu de données de voix devient un corpus de clips.

## Contexte

Comparer des modèles de transcription, de diarisation et d'empreinte demande une vérité terrain. Pour l'empreinte, la Personne validée par un humain en est une. Pour la transcription et la diarisation, le texte et les bornes des locuteurs viennent du pipeline de production : seules des corrections humaines s'y ajoutent, et rien n'indique si un segment non corrigé a été relu. Mesurer un autre modèle contre cette référence avantage le modèle de production, qui en est la source.

## Décision

- **Un corpus est un ensemble de clips partagés**, chacun avec un rôle : *test* (ce que l'on mesure) ou *référence* (enrôle les Personnes pour l'empreinte). Un clip n'a qu'un rôle, ce qui empêche une voix d'être sa propre référence. Les clips privés ne sont jamais proposés.
- **Certification « relu en entier ».** L'administrateur atteste, clip de test par clip de test, que le texte et les locuteurs ont été relus en entier. Sans cette attestation, un clip ne sert qu'à l'empreinte. L'attestation est enregistrée dans le corpus (`benchmark_dataset_clips.verified`).
- **Les étapes mesurables sont déduites du contenu**, à la création : empreinte (au moins deux Personnes avec référence, chaque Personne testée enrôlée), transcription (un clip certifié avec transcription), diarisation (un clip certifié avec voix validée), bout en bout (les trois). Un corpus sans aucune étape est refusé.
- **Instantané.** La Personne de chaque voix est copiée à la création. Le texte de référence de la transcription sera lui aussi figé à la tranche qui mesure cette étape.
- **Biais annoncé.** Les résultats de transcription, de diarisation et de bout en bout affichent que la référence vient du pipeline de production corrigé : le modèle de production est avantagé. L'import de références externes (STM, RTTM) lèverait ce biais ; il reste à faire.
- Le catalogue des modèles proposés indique, par modèle, le nombre de Workers où il est prêt : un modèle prêt nulle part laisserait le run en attente.

## Alternatives écartées

- **Seuls les segments corrigés comme référence** : sélection biaisée contre le modèle de production (les segments qu'il a ratés).
- **Un jeu de voix isolées** (ADR 0056) : ne permet ni WER ni DER, et sépare artificiellement les étapes.
- **Annotation dédiée dans Snitch** : fonctionnalité à part entière, hors tranche.

## Conséquences

- Ces étapes seront livrées dans l'ordre : écran Benchmark pour l'empreinte, puis transcription, diarisation, bout en bout. `stages` annonce ce qu'un corpus permet ; un run n'accepte que les étapes déjà exécutables.
- Une transcription ou une diarisation benchmarkée ne doit jamais atteindre les tables de production : elles auront leur propre étape et leur propre stockage.
