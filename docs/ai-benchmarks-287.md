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

### Séparation des Chevauchements (#402)

Une chaîne peut porter un modèle de séparation du catalogue (`separation_model_id`, API seulement pour l'instant ; le compositeur de l'écran ne le propose pas encore). Quand ses tâches de diarisation et de transcription d'une piste sont terminées, la lecture du run planifie une tâche `benchmark.separation.N` qui demande ce modèle ; le run n'est clos qu'après elle. Le Worker :

- ne sépare que les Chevauchements (tours de la chaîne) où parle un Locuteur local sans aucun mot sur la piste, avec 1 s de contexte, en un seul appel au modèle ; une piste sans tel Chevauchement n'appelle jamais le modèle. Ces locuteurs sont listés par Snitch dans l'entrée de la tâche (`silent_speakers`), avec la règle d'attribution au mot de la production (`segmentparts`, ADR 0065, lissage du mot isolé compris) : le Worker n'a pas de règle propre. La même règle lui donne les temps des mots de chaque Locuteur local (`word_spans`) ;
- transcrit chaque Source séparée avec le modèle de transcription de la chaîne, dans le temps de la piste (début réellement séparé de la fenêtre + temps dans la source) ;
- rapproche chaque source des Locuteurs locaux de la piste par cosinus avec leur empreinte de Parole nette (modèle d'empreinte actif), calculée pour la tâche et jamais conservée ; sous l'écart `ambiguity_margin` des réglages de rapprochement entre les deux meilleurs, la source reste sans locuteur ;
- garde la transcription d'origine telle quelle et liste les mots ajoutés : ceux, dans la zone, d'une source donnée à un Locuteur local qui n'a aucun mot dans cette zone, qu'il en ait ailleurs sur la piste ou non.

Le résultat de la tâche (`silent_speakers`, `zones`, `sources`, `added`) est seulement stocké. Un manque de mémoire GPU fait échouer la tâche avec `gpu_out_of_memory`, sans nouvelle tentative ; la chaîne reste mesurée sans séparation.

**Mesures (#403).** Le bloc `metrics.separation` du résultat d'une chaîne porte sur les pistes certifiées dont la référence contient un Chevauchement (deux locuteurs de référence qui parlent en même temps) et dont la tâche de séparation est terminée :

- cpWER et WER sans puis avec séparation, sur ces mêmes pistes. Les deux hypothèses sont lues mot par mot (chaque mot reçoit le locuteur de la règle au mot de la production, celle qui liste les Locuteurs locaux sans paroles), pour que les mots ajoutés tombent à leur place dans le temps ; celle avec séparation y ajoute les mots de `added` sous le locuteur reçu. Ces chiffres ne se comparent donc pas au cpWER de la ligne de la chaîne, mesuré sur tous les clips et par segment ;
- Locuteurs locaux sans paroles avant (même calcul Go que l'entrée de la tâche) et après (le même décompte sur la transcription fusionnée : les locuteurs des tours qui n'y ont aucun mot, un mot ajouté gardant le locuteur reçu) ;
- mots ajoutés, dont ceux absents des segments de référence qui recouvrent leur zone ; mots dupliqués entre deux sources (même mot, au même moment, dans une source antérieure de la même zone) ; sources restées sans locuteur ;
- `min_clips` (10) : en dessous, l'écran dit que l'effectif est trop faible au lieu de conclure.

**Coût par modèle (#403).** Le Worker ajoute au résultat de ses tâches `benchmark.*` la durée de la tâche (`inference_seconds`, téléchargement et préparation de l'audio exclus ; pour une séparation, la transcription et l'empreinte des sources comprises, ce n'est donc pas la seule inférence du modèle) et, sous CUDA, le pic de mémoire GPU allouée par PyTorch (`gpu_peak_bytes`, `torch.cuda.max_memory_allocated`), le plus haut entre le processus du Worker (remis à zéro au début de la tâche) et les sous-processus d'engine où tournent SepFormer et les modèles à la demande, chacun mesurant le sien ; la transcription, faite par le service externe, n'a pas de pic GPU. `BenchmarkResult.resources` les agrège par modèle du résultat (les deux ou trois modèles d'une chaîne) à la lecture du run : durée cumulée, pic maximal. Une valeur est nulle dès qu'une tâche ne l'a pas remontée (Worker plus ancien, pas de CUDA), jamais zéro ; l'écran affiche « non remonté ». Le biais de la référence de production (ADR 0057) reste annoncé au-dessus des résultats.

## Activation d'un modèle mesuré (tranche 8)

Depuis le détail d'un résultat de transcription ou de diarisation : « Utiliser en production… », avec aperçu (modèle remplacé, Workers prêts, clips déjà analysés dont ceux portant un travail humain) et confirmation explicite. Elle ne change que le modèle du travail à venir : aucune réanalyse n'est lancée, aucune correction ni validation n'est touchée, et chaque changement est enregistré. Voir l'[ADR 0062](adr/0062-activation-explicite-d-un-modele-mesure.md). L'empreinte garde sa transition (ADR 0044).

## Empreinte notée comme la production (#397, ADR 0064)

Le benchmark d'empreinte n'a plus son propre score (centroïde non pondéré des références) : il construit ses Références et note ses tests avec la fonction de score de production (`voice_score.go`), sur des empreintes décidées par la règle d'éligibilité de production.

- **Au lancement**, un run d'empreinte enregistre dans `benchmark_runs.params.scoring` le mode de score actif (`cumulative` ou `best_sample`, avec sa version), la règle d'éligibilité active (version, durées comparable et contributive, écrêtage maximal) et le plafond du poids d'un échantillon (10 s). Il est noté sous ces valeurs, même si elles changent avant la fin du run.
- **Éligibilité** : chaque empreinte de benchmark est décidée, en lecture seule, par l'instruction SQL de la requalification de production sous la règle du run (rien n'est écrit, la décision stockée du pipeline épinglé n'est pas lue). Une empreinte de référence entre dans la Référence si elle est contributive (comparable en mode meilleur échantillon) ; un test n'est noté que s'il est comparable. Les autres sont comptées dans `metrics.ineligible` et ne comptent pas comme mesurées (résultat `partial`).
- **Non-dérive** : pour les mêmes vecteurs et la même Parole nette, le score d'un essai du benchmark est exactement la confiance que donne le rapprochement de production, dans chaque mode (`TestFoundation/ai_benchmark_scoring`). L'égalité est exacte car le même code calcule en Go dans les deux cas (plus de pgvector dans le score) ; les échantillons sont additionnés par Personne puis par Locuteur local, ordre qui ne peut changer qu'au dernier bit avec trois échantillons ou plus d'une même Personne.
- **Affichage** : `BenchmarkRun.scoring` (nul hors empreinte) ; le détail d'un run d'empreinte indique le mode, sa version et la règle. Un run enregistré avant #397 reste lisible avec ses chiffres d'origine et s'affiche « Centroïde (ancien) », sans règle ; ses seuils ne valent pas pour le score actuel. Un run encore ouvert lors de la mise à jour est noté sous le mode et la règle actifs à son règlement, qu'il enregistre alors.
- Un run n'écrit toujours rien dans la bibliothèque : ni échantillon, ni attribution, ni décision d'éligibilité de production (vérifié par `ai_benchmark_scoring` et `voice_benchmark_isolation`).

## Restant

- Découverte des modèles de transcription par les Workers (aujourd'hui l'administrateur les nomme).
- Un seul run actif à la fois ; la RAM n'est pas remontée par le Worker, ni la durée et la mémoire GPU des tâches d'empreinte (étapes `voice.fingerprint.*`).
- Tests navigateur : les modèles « prêts sur un Worker » ne sont jamais réellement prêts dans les fixtures.
