# Corriger le texte d'un segment

Périmètre des tickets 49 (phase 6, socle) et 50 (historique et restauration). Le glossaire, l'ADR et la feuille de route sont mis à jour à la clôture de la phase (ticket 57). Spécification : [phase 6](specs/phase-6.md).

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

## Historique et restauration (ticket 50)

`GET /api/clips/{id}/segments/{index}/history` : réservé aux utilisateurs connectés qui voient le clip (anonyme `401`, clip invisible `404 clip_not_found`, segment absent `404 segment_not_found`). Il renvoie l'état courant (`current`, comme une réponse de correction), le texte d'origine et les entrées de la transcription visible du clip, de la plus récente à la plus ancienne. Chaque entrée porte son `id`, son type (`text`, `speaker`, `restore`), l'ancien et le nouveau texte effectifs (`old_text`, `new_text`, `old_corrected`, `new_corrected`), les libellés de locuteur avant et après, le motif, les versions avant et après, l'auteur (nom au moment de l'écriture) et la date. Les entrées d'une transcription remplacée par une retranscription ne sont plus exposées : seul le segment de la transcription visible est lu.

`POST /api/clips/{id}/segments/{index}/restore` : corps `expected_version` (obligatoire), exactement l'un de `original: true` (texte d'origine du worker) ou `event_id` (texte produit par une entrée antérieure du même segment), `reason` et `transcription_job_id` facultatifs.

- Une restauration est une correction : même transaction, même verrou (`FOR UPDATE`), même contrôle de version (`409 segment_conflict` avec l'état courant, ou `409 transcription_changed`), version incrémentée, segment marqué à réindexer, et une nouvelle entrée de type `restore` (`restored_from_event_id`, ou `restored_to_original`). Aucune entrée passée n'est réécrite ni supprimée.
- Restaurer l'original efface la correction du texte (le segment n'est plus « corrigé », son auteur et sa date sont effacés) mais conserve la ligne et sa version. Restaurer une entrée dont le nouveau texte était l'original revient au même.
- Refus : `400 invalid_request` (ni `original` ni `event_id`, ou les deux), `400 invalid_reason`, `400 text_unchanged` (le texte effectif est déjà celui demandé), `400 event_not_restorable` (entrée de locuteur), `404 event_not_found` (entrée absente ou d'un autre segment).

Dans l'interface, le bouton « Historique du segment N » de la page du clip déroule les entrées ; « Restaurer la version V du segment N » et « Restaurer le texte d'origine du segment N » restaurent, avec un motif facultatif.

## Données (migration 00033)

`segment_corrections` : état courant, clé `(transcription_job_id, segment_index)`, supprimée en cascade avec la tâche ou le clip. Elle porte le texte corrigé, l'auteur et la date, la `version`, l'override de locuteur (réservé à l'étape locuteur), `stale_at` (corrections remplacées par une retranscription) et le marqueur « à réindexer » (`reindex_pending`) posé par chaque correction de texte.

`segment_correction_events` : historique immuable (ancien et nouveau texte effectifs, ancien et nouveau locuteur, type `text`/`speaker`/`restore`, motif, versions avant et après, auteur par id et nom sans clé étrangère). Aucune modification ni suppression directe (déclencheur `attribution_events_immutable`) ; seule la cascade de la suppression du clip l'efface.

## Point d'extension « texte effectif »

`app/internal/httpapi/segment_text.go` est l'unique lecteur du texte effectif : `visibleTranscriptsCTE` et `effectiveSegmentsCTE` (SQL), `visibleTranscription` et `overlaySegments` (Go), `authorizedSegment` (routes). La recherche et la réindexation doivent les utiliser plutôt que lire `jobs.result` directement.
