# 0058 — Benchmark de transcription (WER/CER)

Date : 2026-11-09. Statut : accepté pour #287. S'appuie sur les [ADR 0056](0056-benchmarks-ia-isoles.md) et [0057](0057-corpus-de-benchmark-et-verite-terrain.md).

## Contexte

La transcription repose sur un service externe (Speaches ou compatible), joint par le Worker, pas par Snitch. Un modèle de transcription n'est pas une entrée du catalogue vocal : c'est un nom que le service connaît. Un résultat de transcription de production est lu partout : texte du clip, index sémantique, corrections. Un benchmark ne doit atteindre aucun de ces lecteurs.

## Décision

- **Une étape de tâche à part : `benchmark.transcription.<piste>`**, jamais `transcription.audio.<piste>`. Les lecteurs de la transcription d'un clip, les embeddings, les corrections (et leur péremption) et la file d'analyse ne la voient pas. Le résultat reste dans `jobs.result`, sans projection.
- **Le modèle vient de la tâche**, pas du Worker (`requested_model_id`, nom de 1 à 64 caractères sans caractère de contrôle) : on compare des modèles sur le même Worker. Le profil de production (langue, préparation, détection de parole) s'applique ; le vocabulaire est vide, pour mesurer un modèle tel qu'il est.
- **Référence figée à la création du corpus** (`benchmark_reference_segments`) : le dernier texte de production de chaque clip certifié, la correction humaine remplaçant le texte de son segment. Une correction ultérieure ne change jamais un corpus.
- **WER et CER** après passage en minuscules et retrait de la ponctuation et des symboles ; les accents comptent. Micro-moyenne : erreurs et tailles de référence sont cumulées sur les clips avant division. Le CER d'un clip trop long (plus de 2,5·10⁸ cellules de comparaison) est laissé de côté et compté (`cer_skipped_clips`), plutôt que de bloquer la requête.
- **Un run partiel dit sa couverture** (pistes mesurées sur pistes totales) ; un modèle partiel n'est jamais désigné meilleur.
- **Les modèles ne sont pas listés par Snitch** : seul un Worker joint le service. Snitch propose le modèle de production et l'administrateur nomme les autres ; un nom inconnu échoue sur le Worker avec le code d'erreur du service. Une découverte par les Workers (liste remontée avec leur état d'atteignabilité) reste à faire.
- **Capacité du Worker** : `benchmark.transcription.*` est annoncé dans ses étapes ; un Worker plus ancien ne prend pas ces tâches, qui attendent.
- Le biais annoncé à l'ADR 0057 s'affiche sur chaque résultat de transcription.

## Alternatives écartées

- **Réutiliser `transcription.audio.*` avec un modèle demandé** : le résultat serait lu comme transcription du clip, indexé et rendrait des corrections obsolètes.
- **Un champ de modèle libre dans la production** : changerait ce que la bibliothèque affiche.
- **Interroger le service depuis Snitch** : il n'est pas joignable de Snitch dans toutes les installations.

## Conséquences

- La file d'attente affiche des tâches d'une étape nouvelle ; elles s'annulent comme les autres.
- Une étape de plus ne coûte ni table ni projection : `jobs.result`, `benchmark_run_jobs` et `benchmark_results` suffisent.
