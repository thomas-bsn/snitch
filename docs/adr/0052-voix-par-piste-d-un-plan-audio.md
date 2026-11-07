# 0052 — Voix par piste d'un Plan audio

Statut : accepté pour le ticket #342 du parent #274. Complète l'[ADR 0047](0047-plans-audio-et-revisions-compatibles.md), l'[ADR 0050](0050-publication-atomique-des-revisions-audio.md), l'[ADR 0011](0011-locuteurs-personnes-et-attributions.md) et l'[ADR 0031](0031-voix-ignoree-random-et-refus-d-identification.md).

## Contexte

Les étapes vocales (diarisation, empreintes, rapprochement) existent déjà par piste : leur nom porte l'indice ffprobe réel (`voice.diarization.<i>`, `voice.fingerprint.<i>`) et un Locuteur local appartient à une piste. Ce qui manquait pour un Plan à plusieurs pistes : les lancer sur toutes les pistes retenues, les rattacher au Plan, convertir les temps d'une piste décalée, recenser le travail humain en jeu sur toutes les pistes remplacées, et lire le résultat piste par piste.

## Décision

- **Départ.** `Server.StartPlanVoice` (et `POST /api/admin/clips/{id}/audio-plans/{plan_id}/voice`) ne démarre que si la révision du Plan est la révision visible (`audio_publications`). Chaque piste retenue reçoit une diarisation créée et liée au Plan (`BindChild`, étape `diarization`) dans la même transaction, avant tout claim. Les empreintes sont créées par la projection de la diarisation, dans sa transaction, et liées au même Plan (étape `fingerprint`). Le rapprochement reste celui du clip, déclenché après chaque empreinte ; il ne crée ni Personne ni Échantillon vocal validé.
- **Piste micro.** La désignation du clip est réappliquée avant le départ ; la piste désignée n'est jamais diarisée (`not_applicable`, cause `microphone_direct_attribution`). L'attribution directe ne touche aucune autre piste.
- **Reprise des seuls travaux manquants.** Un lancement répété ne duplique rien. Une diarisation déjà faite avec le même pipeline et encore courante pour la piste est conservée (`reused`). Une étape en échec, morte ou annulée est reprise sur place : même job, nouveau cycle. Les modèles sont épinglés au premier lancement (`audio_plan_voice_starts.pipeline_version`) : toutes les pistes d'un Plan utilisent la même version.
- **Temps.** Un Worker mesure une piste depuis son premier échantillon ; la révision d'un Plan est en temps du clip. Le décalage de la piste (inventaire du Plan) est ajouté une fois, au stockage des tours et des empreintes, et retranché une fois quand les tours sont rendus à un Worker. Le Worker ignore le décalage. Un décalage inconnu vaut zéro.
- **Provenance.** Chaque Locuteur local expose `audio_stream_index` ; la lecture de la voix d'un clip dont la transcription visible est une révision de Plan couvre toutes les pistes retenues (`plan_id`, `tracks`). Les segments ne prennent que les voix de leur piste ; la couverture est calculée par piste, y compris pour une piste retenue sans segment : une voix détectée sans parole reste visible (#273).
- **Protection du travail humain.** Le départ calcule, sur toutes les pistes à remplacer, les voix validées, échantillons vocaux (Références), corrections de locuteur et Personnes touchées. Sans confirmation : 409 `voice_rediarization_confirmation_required` avec ce recensement, rien n'est planifié. Les anciennes voix restent visibles et inchangées jusqu'à une diarisation réussie ; une validation ou correction faite après l'instant confirmé (`stake_at`) verrouille les voix de la piste (`diarization_locked`, ancien résultat intact). Un travail déjà en cours est couvert par la confirmation de son lancement.
- **Absence de Worker, annulation, suppression.** Une étape en attente sans Worker vocal prêt (fournisseur local) porte la cause `voice_worker_unavailable` ; la transcription publiée n'est jamais retirée. L'annulation d'un job lié est propre à sa piste. La suppression par étapes existante supprime les jobs, donc leurs liens (cascade) ; le Plan revient à `not_started`.
- **Production.** Rien n'est déclenché par l'ancien chemin : l'appel appartient à l'orchestration finale (#344) ou à l'action explicite de l'administrateur. Les voix des clips historiques gardent leur lecture monopiste (`tracks` absent).

## Conséquences

La complétion d'un enfant `diarization` ou `fingerprint` exécute désormais sa projection (seuls les enfants `diagnostic` et `transcription` en sont dispensés). #344 appelle `StartPlanVoice` après publication ; les analyses historiques gardent `scheduleVoice`.
