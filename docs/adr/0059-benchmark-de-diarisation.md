# 0059 — Benchmark de diarisation (DER)

Date : 2026-11-09. Statut : accepté pour #287. S'appuie sur les [ADR 0056](0056-benchmarks-ia-isoles.md), [0057](0057-corpus-de-benchmark-et-verite-terrain.md) et [0058](0058-benchmark-de-transcription.md).

## Contexte

Une diarisation de production remplace les locuteurs locaux d'un clip (ADR 0022) : lancer un autre modèle de diarisation avec les mêmes tâches détruirait des validations humaines. L'ADR 0038 n'autorise donc un modèle demandé que pour une étape d'empreinte.

## Décision

- **Une étape de tâche à part : `benchmark.diarization.<piste>`**, jamais `voice.diarization.<piste>`. Le résultat reste dans `jobs.result` : aucun locuteur local, aucun tour, aucune empreinte n'en dépend. Le Worker exécute le code de production sous le nom de l'étape de voix qu'il imite, sans calculer de pics de forme d'onde.
- **Le modèle est une entrée du catalogue** (catégorie diarisation, activée et prise en charge), et le Worker doit l'avoir prêt (même règle que pour une empreinte). La configuration du pipeline de production s'applique, avec le moteur, le dépôt et la révision du modèle demandé ; `access_confirmed` n'est jamais forcé : tant que l'accès au modèle n'est pas confirmé pour la production, il ne tourne pas.
- **Référence figée à la création du corpus** (`benchmark_reference_turns`) : les tours de parole des clips certifiés, chaque locuteur étant la Personne validée, ou, faute de validation, la voix locale elle-même. Les temps sont remis sur l'horloge du flux audio, comme un Worker les mesure (le décalage d'un plan multipiste est retiré).
- **DER** : parole de référence manquée + fausse alarme + confusion de locuteurs, rapportées à la parole de référence, cumulées sur les clips. Les locuteurs du modèle sont anonymes : ils sont associés à ceux de la référence par l'affectation qui maximise leur temps commun (algorithme hongrois). Tolérance de 0,25 s autour de chaque borne de référence ; résolution de 10 ms ; la parole superposée compte une erreur par locuteur manquant ou en trop. Un clip de plus de dix heures est laissé de côté et compté.
- **Le biais annoncé à l'ADR 0057 s'affiche** : les bornes de référence sont celles du pipeline de production, dont seuls les noms ont été validés.
- Si la diarisation de production n'est pas dans le catalogue (diarisation factice, modèle retiré), aucune comparaison avec « la production » n'est affirmée.

## Alternatives écartées

- **Un modèle demandé sur `voice.diarization`** : ce résultat est projeté dans les locuteurs locaux.
- **Un DER sans tolérance** : une borne décalée de quelques dizaines de millisecondes compterait comme une erreur alors qu'aucune annotation n'est précise à ce point.

## Conséquences

- Les modèles de diarisation sont comparés avec les mêmes tâches, Workers et règles de disponibilité que ceux d'empreinte.
- Une étape de plus ne coûte qu'une table de référence : `jobs.result`, `benchmark_run_jobs` et `benchmark_results` suffisent.
