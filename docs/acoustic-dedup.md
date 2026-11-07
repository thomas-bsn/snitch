# Déduplication acoustique des paroles (#343)

Décision : [ADR 0053](adr/0053-deduplication-acoustique-des-paroles.md). Version du rapport : `schema_version` 1, adaptateur `1`.

## Méthode et versions

Entrée de l'adaptateur Worker `snitch_worker.acoustic_dedup.run(payload)` : des événements `{id, track_index, job_id, path, clip_offset, start, end}` (PCM16 mono WAV décodé ; `clip_offset` = temps du clip du premier échantillon du fichier, `null` si inconnu ; `start`/`end` = plage de parole dans le repère du clip) et une configuration facultative. Sans liste de paires, toutes les paires de pistes différentes dont les plages se chevauchent sont comparées. La sortie est canonique : l'ordre des événements ne la change pas.

1. **Alignement.** Les plages sont converties en échantillons avec le décalage de chaque piste (positif ou négatif) ; les temps de sortie sont des temps du clip des deux côtés. Un résidu de décalage <= 100 ms est cherché (octets ancres pour `exact_pcm`, corrélation FFT pour `gain`) et rapporté dans `lag_seconds`.
2. **`exact_pcm` (v1, active).** Blocs de 10 ms strictement égaux. Un bloc quasi muet (pic < 32) ou constant ne compte pas comme preuve mais peut prolonger une plage adjacente à du contenu prouvé. Une plage exige >= 0,25 s de contenu informatif, sans bloc différent.
3. **`gain` (v1, active).** Gain médian positif sur les blocs informatifs, écart à 1 d'au moins 1 %, blocs écrêtés exclus, résidu de crête <= 1 LSB par bloc. Nécessite numpy (image Worker) ; sans numpy le cas reste `uncertain_keep`.
4. **Évalué, jamais appliqué.** Corrélation normalisée au meilleur décalage : >= 0,9 (`codec`), entre 0,3 et 0,9 (`mixture`) donnent `uncertain_keep` ; < 0,3 donne `distinct` (étiquette informative, même effet : conserver).
5. **Conservé dans tous les autres cas :** pistes identiques, décalage temporel inconnu, plages sans recouvrement (même enregistrement rejoué plus tard = `distinct`, `timeline_disjoint`), fréquences ou formats différents, fichier illisible, signal absent, événement trop court, événement > 120 s, > 200 paires.

## Limites connues

- Preuves sur fixtures synthétiques (espeak-ng) et un seul encodeur AAC : aucun taux de rappel n'est revendiqué sur de l'audio de jeu. Un AAC à 64 kbit/s a une corrélation de 0,999 et un résidu de 0,14 % de l'énergie, mais reste conservé : aucune règle codec n'est validée.
- Un doublon prouvé seulement sur une partie d'un passage laisse ce passage entier (mots dupliqués possibles à la frontière) plutôt que de couper du texte.
- Un doublon n'est regroupé que si la piste d'indice inférieur a un segment ASR sur la plage correspondante.
- Le rapport n'est pas encore produit par un job Worker ni consommé par l'orchestration ; ce branchement relève de #339/#344 (voir « Points d'intégration »).

## Mesures (machine de développement, Python 3.12, numpy 2.4.6, 1 cœur)

Comparaison d'une paire, signal synthétique mono 16 kHz, fenêtre chargée = durée de l'événement (+ 2 x 100 ms) :

| Événement | copie exacte | copie avec gain | paire sans lien |
| --- | --- | --- | --- |
| 10 s | 18 ms | 25 ms | 21 ms |
| 60 s | 109 ms | 114 ms | 108 ms |
| 300 s | refusé (`budget_exceeded`) en < 1 ms | idem | idem |

Mémoire : pic RSS du processus de mesure 330 Mo (numpy inclus), fenêtre de comparaison 2 octets par échantillon et par côté (3,8 Mo pour 60 s). Le coût est linéaire dans la durée comparée et le nombre de paires, plafonné par `max_compare_seconds` (120), `max_pairs` (200) et `max_lag_seconds` (0,1).

## Points d'intégration

- Worker : `acoustic_dedup.run(payload, cancel=...)`. L'annonce d'une capacité de job et son routage restent à faire (non annoncé, donc les anciens Workers ne reçoivent rien).
- Serveur : `audioplans.ParseEvidence(rapport)` puis `audioplans.MergeAcoustic(segments source, rapport)` -> `Segments` à placer dans `Revision` avant `StageRevision`/`PreviewPublication`/`PublishRevision` (#340). `AudioPlanInput.MergeVersion` = `audioplans.MergeVersion()`.
- `audioplans.EvidenceFixture()` est un rapport réel de l'adaptateur sur les fixtures, partagé par Go et vérifié côté Python.

## Fixtures

`worker/tests/fixtures/acoustic/` : recette `generate.py` (espeak-ng + ffmpeg + numpy, graine fixe), `manifest.json` (SHA-256 des WAV), `generate_evidence.py` (régénère `app/internal/audioplans/fixtures/acoustic-evidence.json`). Texte : « go to the north gate now ». Paires couvertes : copie exacte, recodage FLAC, gains 0,25 / 0,5 / 2,0, AAC 64 k, autre voix mêmes mots, deuxième prise, mélange voix + décor identique à deux voix, deux voix ensemble, copie partielle, décalages ±500 ms et 20 ms, écrêtage, silence, tons périodiques, fréquence ou format différent, fichier corrompu.

## Vérification

- `cd worker && PYTHONPATH=.:snitch_worker/api python -m unittest tests.test_acoustic_dedup` (avec numpy : `SNITCH_REQUIRE_NUMPY=1`, sinon les cas à transformation sont ignorés ; l'image Worker contient numpy).
- `cd app && go test ./internal/audioplans`
- `bash scripts/test.sh ./internal/httpapi -run 'TestFoundation/audio_dedup'`
