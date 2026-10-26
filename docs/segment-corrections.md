# Corriger le texte d'un segment

Périmètre du ticket 49 (phase 6, socle). Le glossaire, l'ADR et la feuille de route sont mis à jour à la clôture de la phase (ticket 57). Spécification : [phase 6](specs/phase-6.md).

## Principe

La transcription du worker (`jobs.result`) n'est jamais modifiée. Une correction est une couche distincte qui recouvre le texte d'un segment. Un segment est identifié par la tâche de transcription visible du clip (`transcription_job_id`) et son rang dans `result.segments` (`index`). Une retranscription crée une nouvelle tâche : les corrections de l'ancienne ne s'appliquent plus (elles ne sont jamais réappliquées silencieusement).

Seul le texte d'un segment est corrigeable ; les débuts, les fins, le nombre de segments et les horodatages des mots d'origine ne changent pas. Le texte corrigé fait foi pour l'affichage sans réaligner les mots. Le worker n'intervient pas et n'accède à aucune de ces données. Aucune donnée vocale n'est concernée.

## Lecture d'un clip

`GET /api/clips/{id}` ajoute à la transcription `transcription_job_id`, et à chaque segment :

| Champ | Sens |
|---|---|
| `index` | rang du segment dans la transcription |
| `original_text` | texte du worker, jamais modifié |
| `text` | texte effectif : le texte corrigé s'il existe, sinon l'original |
| `corrected` | vrai si `text` est une correction |
| `correction_version` | version à renvoyer comme `expected_version` (0 si jamais écrit) |
| `corrected_by`, `corrected_at` | auteur et date de la correction du texte |

La page du clip affiche le texte effectif, signale un segment corrigé (étiquette « Corrigé », auteur, date) et donne accès au texte d'origine.

## Correction (`PUT /api/clips/{id}/segments/{index}/text`)

Corps : `text`, `expected_version` (obligatoire, 0 pour un segment jamais corrigé), `reason` facultatif (500 caractères au plus), `transcription_job_id` facultatif.

- Réservé aux utilisateurs connectés qui voient le clip : anonyme `401`, clip invisible ou inexistant `404 clip_not_found`, segment absent `404 segment_not_found`.
- Texte vide ou réduit à des espaces `400 invalid_text`, de plus de 4000 caractères `400 text_too_long`, motif trop long `400 invalid_reason`, texte identique au texte courant `400 text_unchanged`.
- Contrôle de version optimiste : si la version courante diffère de `expected_version`, `409 segment_conflict` avec l'état courant et son auteur (`current`). Si `transcription_job_id` n'est plus la transcription visible : `409 transcription_changed`.
- Le texte, la version incrémentée et l'entrée d'historique sont écrits dans une seule transaction. La ligne est verrouillée (`FOR UPDATE`) ; deux segments différents ne se bloquent pas.

## Données (migration 00033)

`segment_corrections` : état courant, clé `(transcription_job_id, segment_index)`, supprimée en cascade avec la tâche ou le clip. Elle porte le texte corrigé, l'auteur et la date, la `version`, l'override de locuteur (réservé à l'étape locuteur), `stale_at` (corrections remplacées par une retranscription) et le marqueur « à réindexer » (`reindex_pending`) posé par chaque correction de texte.

`segment_correction_events` : historique immuable (ancien et nouveau texte effectifs, ancien et nouveau locuteur, type `text`/`speaker`/`restore`, motif, versions avant et après, auteur par id et nom sans clé étrangère). Aucune modification ni suppression directe (déclencheur `attribution_events_immutable`) ; seule la cascade de la suppression du clip l'efface.

## Point d'extension « texte effectif »

`app/internal/httpapi/segment_text.go` est l'unique lecteur du texte effectif : `visibleTranscriptsCTE` et `effectiveSegmentsCTE` (SQL), `visibleTranscription` et `overlaySegments` (Go), `authorizedSegment` (routes). La recherche et la réindexation doivent les utiliser plutôt que lire `jobs.result` directement.

## Réindexation du seul segment corrigé

Une correction de texte n'attend pas le service d'embeddings : elle est enregistrée avec `reindex_pending=true` (colonnes `reindex_*` de `segment_corrections`), puis un consommateur (`reindexNext`, appelé par la boucle d'indexation existante) recalcule le vecteur de **ce segment seul**.

- Le texte embarqué est le texte effectif (la correction, ou l'original après une restauration). Un seul appel fournisseur par profil, une seule entrée.
- Le vecteur remplace l'ancien sous la même clé `(profil, job de transcription, segment)` : jamais de doublon, les autres lignes de `transcript_embeddings` du clip ne sont pas réécrites. Le profil actif et, pendant une reconstruction, le profil candidat sont mis à jour ; la bascule ne ramène donc pas l'ancien texte.
- Les tâches d'index par transcription (`indexNext`, remplissage, reconstruction) lisent aussi le texte effectif. Un segment corrigé entre la lecture de son texte et l'écriture de la tâche n'est pas écrasé : l'écriture est ignorée pour ce segment, que sa propre réindexation traite.
- Fournisseur indisponible : la correction reste enregistrée (la lecture du clip la montre), le segment reste en attente et il est repris sans doublon, avec un délai croissant (30 s, 60 s, ...), au plus 5 tentatives. Le bail (`reindex_lease_expires_at`, 2 minutes) est repris à son expiration si un consommateur a disparu. Une nouvelle correction du segment remet les tentatives à zéro. Une correction d'une transcription qui n'est plus la transcription visible est ignorée.
- `GET /api/admin/embeddings` expose `reindex_pending` (segments corrigés en attente) et `reindex_failed` (tentatives épuisées) ; l'administration les affiche dans « Index sémantique ». « Indexer les transcriptions existantes » (`POST /api/admin/embeddings/backfill`) redonne leurs tentatives aux segments épuisés.
- Aucune retranscription ni retraitement de la vidéo n'a lieu.

## Retranscription d'un clip corrigé

Périmètre du ticket 55. Retranscrire un clip ne détruit jamais en silence le travail humain.

- **Confirmation.** Tout chemin qui planifie une retranscription d'un clip portant des corrections actives (texte ou locuteur de la transcription visible) répond `409 corrections_would_be_replaced` avec `corrections` (tâche de transcription, nombre de textes et de locuteurs corrigés, auteurs, date de la dernière correction). L'administrateur renvoie la même requête avec `confirm_replace_corrections=true` (paramètre de requête) pour confirmer. Chemins protégés : `POST /api/admin/clips/{id}/transcription`, `PUT` et `DELETE /api/admin/clips/{id}/audio-track` (le changement de piste refusé ne modifie rien), `POST /api/admin/jobs` pour une étape `transcription.audio.*`, et `POST /api/admin/jobs/{id}/retry` d'une tâche de transcription `dead`. La page du clip affiche l'avertissement avec les boutons « Confirmer la retranscription et remplacer les corrections » et « Annuler la retranscription ».
- **Lot.** L'aperçu (`GET /api/admin/transcriptions/batch-preview`) compte `corrected` (clips autrement éligibles qui portent des corrections) et `corrected_segments` ; ces clips sont exclus de `eligible` et du lot. Ils n'y entrent que sur demande explicite : `include_corrected=true` (aperçu) et `include_corrected: true` dans le corps de `batch-schedule`.
- **Retraitement automatique.** Un changement de règle de piste de source (`PUT /api/admin/sources/{id}/audio-track-rule`) ne retranscrit pas les clips corrigés : ils sont ignorés et comptés dans `corrected_clips_skipped`.
- **Succès.** À l'achèvement de la retranscription (`jobs.CompleteWith`, même transaction), les corrections des tâches de transcription précédentes du clip reçoivent `stale_at` : elles restent dans `segment_corrections` et dans l'historique (immuable), sont obsolètes, ne s'appliquent plus (texte effectif, recherche, réindexation filtrent `stale_at IS NULL`) et ne sont jamais réappliquées au nouveau texte, même identique.
- **Échec.** La transcription visible est la dernière tâche réussie : une tâche en attente, en cours ou en échec ne la remplace pas. Le texte actuel et ses corrections restent affichés et actifs.
