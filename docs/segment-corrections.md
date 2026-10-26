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

## Réindexation du seul segment corrigé

Une correction de texte n'attend pas le service d'embeddings : elle est enregistrée avec `reindex_pending=true` (colonnes `reindex_*` de `segment_corrections`), puis un consommateur (`reindexNext`, appelé par la boucle d'indexation existante) recalcule le vecteur de **ce segment seul**.

- Le texte embarqué est le texte effectif (la correction, ou l'original après une restauration). Un seul appel fournisseur par profil, une seule entrée.
- Le vecteur remplace l'ancien sous la même clé `(profil, job de transcription, segment)` : jamais de doublon, les autres lignes de `transcript_embeddings` du clip ne sont pas réécrites. Le profil actif et, pendant une reconstruction, le profil candidat sont mis à jour ; la bascule ne ramène donc pas l'ancien texte.
- Les tâches d'index par transcription (`indexNext`, remplissage, reconstruction) lisent aussi le texte effectif. Un segment corrigé entre la lecture de son texte et l'écriture de la tâche n'est pas écrasé : l'écriture est ignorée pour ce segment, que sa propre réindexation traite.
- Fournisseur indisponible : la correction reste enregistrée (la lecture du clip la montre), le segment reste en attente et il est repris sans doublon, avec un délai croissant (30 s, 60 s, ...), au plus 5 tentatives. Le bail (`reindex_lease_expires_at`, 2 minutes) est repris à son expiration si un consommateur a disparu. Une nouvelle correction du segment remet les tentatives à zéro. Une correction d'une transcription qui n'est plus la transcription visible est ignorée.
- `GET /api/admin/embeddings` expose `reindex_pending` (segments corrigés en attente) et `reindex_failed` (tentatives épuisées) ; l'administration les affiche dans « Index sémantique ». « Indexer les transcriptions existantes » (`POST /api/admin/embeddings/backfill`) redonne leurs tentatives aux segments épuisés.
- Aucune retranscription ni retraitement de la vidéo n'a lieu.

## Corrections simultanées (ticket 51)

Le contrôle de version du `PUT` garantit qu'une correction ne remplace jamais silencieusement une autre : la ligne du segment est verrouillée (`FOR UPDATE`), et pour un segment jamais corrigé l'insertion `ON CONFLICT DO NOTHING` désigne le perdant. De deux corrections parties de la même version, une seule aboutit ; l'autre reçoit `409 segment_conflict` avec la version courante (`current` : texte, version, auteur, date) et rien n'est écrit pour elle. Chaque correction acceptée laisse une entrée d'historique : la correction gagnante n'est donc jamais perdue. Deux segments différents ne se bloquent pas et ne se contredisent pas.

Interface : en cas de conflit, le formulaire reste ouvert avec le brouillon intact et un bloc « Conflit sur le segment N » indique qui a modifié le segment et quand, ainsi que la version actuelle. « Reprendre ma modification » enregistre le brouillon (éventuellement retouché) par-dessus la version actuelle ; « Abandonner ma modification » ferme le formulaire et jette le brouillon. La transcription affichée est actualisée en arrière-plan.

Tests : `segment_concurrency_test.go` (requêtes simultanées par goroutines sur PostgreSQL réel : un gagnant, conflit avec version et auteur, huit écrivains simultanés, segments différents sans conflit) et `web/e2e/segment-conflicts.spec.ts` (deux utilisateurs, drapeau `SNITCH_E2E_SEGMENT_CORRECTIONS=1`).

## Recherche dans le texte corrigé

Les quatre modes de `GET /api/search` lisent le texte effectif (`eff_segments`), jamais `jobs.result` directement :

- **exact** : les mots d'un segment non corrigé sont ses mots d'origine ; ceux d'un segment corrigé sont découpés dans le texte corrigé (séparation sur les blancs, ponctuation de bord retirée), tous datés au début du segment (les mots ne sont pas réalignés). Les mots d'origine d'un segment corrigé ne produisent donc plus aucun résultat ;
- **plein texte** et **approximatif** : la sous-chaîne et la similarité trigramme s'appliquent au texte corrigé et à ses mots ;
- **hybride** : la voie lexicale est la recherche plein texte ci-dessus (rappelée en interne). La voie sémantique lit `transcript_embeddings`, dont le texte et le vecteur datent de l'indexation : le texte affiché est le texte effectif (jointure sur `segment_corrections`), et un segment dont la réindexation est en attente (`reindex_pending`) n'est pas proposé par cette voie, pour ne pas faire remonter l'ancien texte par son ancien vecteur. Il redevient trouvable sémantiquement une fois réindexé (indexation par segment à la charge de la réindexation) ; la voie lexicale le trouve immédiatement.

Chaque résultat porte `corrected` (vrai si `context` est le texte corrigé) ; l'interface affiche alors un badge « Corrigé ». `timestamp` reste le début du mot d'origine (segment non corrigé) ou le début du segment (segment corrigé) : l'ouverture se fait à cet horodatage. Permissions de visibilité, filtres et filtre par Personne s'appliquent avant les comptes et la pagination, comme avant ; le locuteur d'un segment est résolu par `segmentSpeakerJoin`, à qui la recherche passe déjà l'identité du segment (transcription, index).

## Export administrateur (ticket 56)

`GET /api/admin/segment-corrections/export` est réservé aux administrateurs (401 sans session, 403 pour un utilisateur ordinaire). Il prépare la mesure de qualité de la phase 7 : Snitch n'y calcule **aucun taux d'erreur**, il fournit seulement les paires texte d'origine / texte corrigé.

- **Contenu** : un enregistrement par segment dont le texte est actuellement corrigé, sur la transcription visible du clip. Jamais un segment non corrigé, ni un segment restauré au texte d'origine, ni la correction obsolète d'une ancienne transcription. Un administrateur voit tous les clips (publics ou privés), comme pour le reste de l'API.
- **Champs** : `clip_id`, `transcription_job_id`, `audio_stream_index` (la piste), `index` (rang du segment), `start`, `end`, `original_text` (texte du worker, jamais modifié), `corrected_text`, `author` (`id`, `username`), `corrected_at`, `version`. Tri par clip, piste puis rang : l'ordre est stable.
- **Formats** : par défaut une page JSON `{items, page, page_size, total}` (`page` à partir de 1, `page_size` de 1 à 500, 100 par défaut ; 400 `invalid_pagination` sinon). Avec `format=ndjson`, tout l'export est diffusé sans pagination, un objet JSON par ligne, en téléchargement (`Content-Disposition: attachment; filename="segment-corrections.ndjson"`). Tout autre format donne 400 `invalid_format`.
- **Interface** : la carte « Export des corrections » de la page d'administration propose le lien « Télécharger l'export des segments corrigés » (NDJSON).

Suppression : supprimer un clip efface en cascade ses corrections et leur historique (donc ses lignes de l'export) ; le fichier vidéo d'origine n'est jamais touché. Supprimer une Personne ou un profil vocal ne modifie ni les corrections de texte ni leur historique (les corrections de locuteur, elles, retombent sur le locuteur déduit).

Tests : `segment_export_test.go` (sous-test « segment export » de `TestFoundation` : refus 401/403, vide, champs, restauration, pagination, NDJSON, clip privé, suppression de Personne et de profil, suppression du clip, original conservé) et `web/e2e/segment-export.spec.ts` (drapeau `SNITCH_E2E_SEGMENT_EXPORT=1`).

## Corriger le locuteur d'un segment (ticket 54)

`PUT /api/clips/{id}/segments/{index}/speaker` avec `{local_speaker_id, expected_version, reason?, transcription_job_id?}` désigne le Locuteur local d'un segment ; `local_speaker_id: null` revient au locuteur déduit des tours de parole (recouvrement maximal, égalité au plus petit ordinal). Le Locuteur local doit appartenir au même clip et à la même piste audio que la transcription visible, sinon `400 invalid_local_speaker`. Choisir la valeur déjà enregistrée répond `400 speaker_unchanged`. Le contrôle de version est celui du texte (le même compteur `version` couvre texte et locuteur) : `409 segment_conflict` ou `transcription_changed` avec l'état courant.

- Stockage : `segment_corrections.speaker_override_id`, auteur et date ; aucune migration. Un retour à la valeur déduite efface l'override mais conserve la ligne (la version continue). Une correction du texte conserve l'override, et inversement.
- Historique : une entrée `speaker` immuable (ancienne et nouvelle valeur effectives, étiquettes, id NULL pour « déduit », auteur, date, motif). Elle ne se restaure pas par `restore`.
- Aucun effet sur les tours de parole, les empreintes, les attributions Locuteur local → Personne, les échantillons vocaux validés ; aucune diarisation relancée ; `reindex_pending` n'est pas positionné (le locuteur ne change pas le vecteur d'un segment).
- Lecture : `segmentSpeakerJoin` (SQL : recherche et filtre par Personne) et `attachLocalSpeakers` (Go : lecture du clip) appliquent l'override, tant que le Locuteur local existe encore, est du même clip et de la même piste ; sinon la valeur déduite s'applique. La lecture du clip expose `segment.local_speaker_corrected` et `transcription.local_speakers` (choix possibles). Un segment sans locuteur déduit peut recevoir un locuteur désigné.
- Filtre de la bibliothèque : il lit les voix identifiées du clip (niveau clip) ; désigner un autre Locuteur local du même clip ne change donc pas la liste des clips d'une Personne, seulement les segments filtrés en recherche.
- Interface : sélecteur « Locuteur du segment N » et bouton « Enregistrer le locuteur du segment N » (`web/src/segments/SegmentSpeaker.tsx`).
