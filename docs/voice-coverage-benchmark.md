# Écart entre locuteurs détectés et paroles transcrites (#273)

Entrée de #268 (séparation vocale, nouveaux modèles) : elle ne la remplace pas. Cette page décrit comment établir pourquoi une voix détectée par la diarisation n'apparaît pas dans la transcription, avant de changer de modèle.

## Cause dans le code Snitch

Un segment de transcription n'a qu'un Locuteur local : celui dont les tours le recouvrent le plus (égalité : le premier). Une voix toujours dominée, ou qui parle là où Whisper n'a produit aucun segment, n'apparaît donc sur aucun segment. Avant #273 rien ne le signalait. Désormais `speaker_coverage` (voir [voice-diarization.md](voice-diarization.md)) garde chaque voix détectée visible, avec son statut.

## Établir le diagnostic d'un clip

1. `GET /api/admin/clips/{id}/voice-trace` : sans texte, donc joignable à un rapport. Il fournit les versions de modèles, les tours bruts, les segments avec leurs voix candidates, la voix retenue et la couverture.
2. Lire le statut de la voix manquante :

| Statut | Lecture | Cause probable (ticket) |
| --- | --- | --- |
| `words` | Des paroles lui sont attribuées. | Aucun écart. |
| `speech_without_words` | Elle parle sous des segments attribués à d'autres. | (b) mots transcrits mais attribués à la voix dominante ; ou (c) fausse voix si ses tours sont très courts. Comparer `overlapped_seconds` et les candidats des segments. |
| `no_transcript` | Aucun segment ne la recouvre, au milieu de la transcription. | (a) voix détectée mais mots non transcrits (voix faible, ASR muet). |
| `outside_transcript` | Aucun segment ne la recouvre, avant ou après toute la transcription. | (d) mauvaise piste ou (e) décalage de temps / transcription tronquée. |

3. Écarter d'abord une erreur de Snitch : piste sélectionnée (`audio_stream_index` de la trace), bornes des segments, corrections manuelles (`corrected`).
4. Seulement ensuite comparer les modèles. Les cas où l'ASR est muet (`no_transcript`, `speech_without_words` avec voix faible) sont ceux qu'une séparation vocale pourrait aider ; les autres ne le seront pas.

## Scénarios de référence

`voice_coverage_reference_test.go` encode, avec une vérité terrain synthétique (aucun média, aucun clip réel), un scénario par cause : quatre voix transcrites, parole simultanée, voix faible sous une voix forte, trou de l'ASR, décalage de 30 s, transcription vide d'une piste, pistes microphone et PC indépendantes. `voice_coverage_integration_test.go` rejoue le cas « trois voix détectées, deux dans la transcription » à travers l'API réelle.

## Limites

- Aucun clip réel n'a été mesuré (aucun scan de la vraie bibliothèque avant l'autorisation de la phase 7). Les taux d'écart sur des clips FR/EN réels, avec voix faibles et parole simultanée, restent à mesurer avec cette méthode sur des clips explicitement choisis.
- `speech_without_words` ne distingue pas une vraie parole simultanée d'une fausse voix créée par la diarisation : seule l'écoute ou une vérité terrain humaine le tranche.
- La trace ne couvre que la transcription visible (piste sélectionnée).
