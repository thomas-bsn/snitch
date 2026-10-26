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

## Corrections simultanées (ticket 51)

Le contrôle de version du `PUT` garantit qu'une correction ne remplace jamais silencieusement une autre : la ligne du segment est verrouillée (`FOR UPDATE`), et pour un segment jamais corrigé l'insertion `ON CONFLICT DO NOTHING` désigne le perdant. De deux corrections parties de la même version, une seule aboutit ; l'autre reçoit `409 segment_conflict` avec la version courante (`current` : texte, version, auteur, date) et rien n'est écrit pour elle. Chaque correction acceptée laisse une entrée d'historique : la correction gagnante n'est donc jamais perdue. Deux segments différents ne se bloquent pas et ne se contredisent pas.

Interface : en cas de conflit, le formulaire reste ouvert avec le brouillon intact et un bloc « Conflit sur le segment N » indique qui a modifié le segment et quand, ainsi que la version actuelle. « Reprendre ma modification » enregistre le brouillon (éventuellement retouché) par-dessus la version actuelle ; « Abandonner ma modification » ferme le formulaire et jette le brouillon. La transcription affichée est actualisée en arrière-plan.

Tests : `segment_concurrency_test.go` (requêtes simultanées par goroutines sur PostgreSQL réel : un gagnant, conflit avec version et auteur, huit écrivains simultanés, segments différents sans conflit) et `web/e2e/segment-conflicts.spec.ts` (deux utilisateurs, drapeau `SNITCH_E2E_SEGMENT_CORRECTIONS=1`).
