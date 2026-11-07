# Plans audio et révisions — interfaces #337

Le [contrat OpenAPI](../contracts/openapi.yaml) fait référence pour les trois clients générés. Le module Go `snitch/internal/audioplans` est l'interface persistante commune. Voir l'[ADR 0047](adr/0047-plans-audio-et-revisions-compatibles.md). Aucun diagnostic réel, nouvel ordonnancement Auto, ni publication multitracks n'est activé par ce ticket.

## Contrat partagé

La fixture [audio-plan-input.json](../app/internal/audioplans/fixtures/audio-plan-input.json) est synthétique et lisible par Go, Python et TypeScript ; `audioplans.ContractFixture()` en renvoie une copie embarquée. Les pistes réelles sont **2 et 7**, pas 0 et 1. La piste 7 a une durée inconnue et un décalage de 1,25 s ; ne pas lui attribuer la durée du conteneur. Les bornes temporelles des diagnostics et manifestes sont dans le repère du clip. Les diagnostics exprimés dans le repère d'une piste doivent être convertis avant enregistrement.

Schémas : `AudioPlanInput`, `AudioPlan`, `AudioPlanState`, `AudioTrackInventory`, `AudioTrackDiagnostic`, `AudioPlanChild`, `AudioTrackDecision`, `AudioLossPreview`, `TranscriptRevision`, `TranscriptManifestSegment`, `TranscriptManifestWord`, `TranscriptSource`. Le manifeste contient les temps et les identités sources ; le texte original reste dans le résultat du job. Les nouvelles révisions interdisent les indices de piste négatifs ; seul l'historique peut porter `-1` quand l'information manque.

Lecture administrateur : `GET /api/admin/clips/{id}/audio-plans/{plan_id}`. Les permissions du clip sont appliquées ; un Plan appartenant à un autre clip répond 404. Le champ facultatif `transcription.revision` s'ajoute aux détails de clip. Les entrées de correction de texte, restauration et locuteur acceptent `expected_revision_id` ; une révision différente répond 409 `transcription_changed`, comme la garde historique de job.

## Écritures et responsabilités

| Ticket | Écritures autorisées / interface | Idempotence et obligations |
| --- | --- | --- |
| #338 diagnostic | `RecordDiagnostic(ctx, AudioTrackDiagnostic) error` | Observation terminale par `(plan, piste, tentative)` ; répétition identique acceptée, contenu différent refusé. Job lié à l'étape diagnostic, tentative réelle >= 1. Détecteur/configuration et digest du Plan doivent correspondre. `speech` exige des intervalles, `no_speech` les interdit. `failed` et `ambiguous` conservent leur cause. |
| #339 orchestration | `CreatePlan(ctx, runID, AudioPlanInput) (AudioPlan,error)`, `BindChild(ctx, AudioPlanChild) error` | Plan unique par Run + digest canonique. Une nouvelle sélection crée un nouveau Plan. Un lien par Plan/piste/étape, un job ne peut être lié deux fois. Répéter le même lien est accepté. Création et liaison du job dans une même transaction, avant claim. |
| #340 publication | `CanonicalRevision(TranscriptRevision)`, `StageRevision(ctx, TranscriptRevision)`, `GetRevision(ctx,id)` | Identité `plan:<id>:<digest>` du contenu canonique, répétition identique acceptée. Sources vérifiées contre des enfants transcription réussis ; temps finis, positifs et mots contenus dans leurs segments. Stockage d'un candidat uniquement. Publication, complétude, aperçu/confirmation des pertes et invalidations atomiques appartiennent à #340. |
| #341 sélection UI | Contrats générés et `GetPlan(ctx,id)` / GET administrateur | Lecture sans mutation. Les décisions portent auteur, date, motif et `forced`. L'aperçu des pertes doit être rattaché à la révision attendue et recalculé/validé transactionnellement avant publication par #340. Le Plan lu ne devient pas un Plan actif implicitement. |

`New(db)` accepte un pool **ou une transaction pgx** ; le propriétaire de la transaction gère commit/rollback. `ErrConflict` signale une contradiction d'identité ou de provenance ; `pgx.ErrNoRows` une absence. Les données invalides de création peuvent aussi renvoyer une erreur de validation. `CanonicalInput` et `CanonicalRevision` prennent une copie profonde et ne modifient pas leur entrée. Une liste de sélection est triée et dédoublonnée ; l'inventaire et les décisions sont triés par indice. Toutes les versions et l'inventaire complet participent au digest, pas seulement les pistes incluses. Les tableaux obligatoires sont émis comme tableaux vides, jamais `null`.

### Créer et lier avant de rendre le job visible

```go
tx, err := pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback(ctx)
store := audioplans.New(tx)
plan, err := store.CreatePlan(ctx, runID, input)
if err != nil { return err }
// Créer ou retrouver le job pending dans tx (pas via un service ouvrant sa
// propre transaction), avec sa vraie version d'entrée compatible Worker.
err = store.BindChild(ctx, audioplans.Child{
    PlanId: plan.Id, TrackIndex: 2, Stage: "transcription",
    JobId: jobID, InputVersion: jobInputVersion,
})
if err != nil { return err }
return tx.Commit(ctx)
```

La version d'entrée du job peut différer du digest du Plan : elle décrit son étape et sa compatibilité Worker. Le diagnostic rapporte en revanche `input_version` du **Plan**, pour empêcher le mélange d'observations entre Plans. Les noms d'étapes sont `audio.diagnostic.<index>`, `transcription.audio.<index>`, `voice.diarization.<index>` et `voice.fingerprint.<index>`. La liaison exige un job `pending`, du même clip et de la bonne étape/version ; elle refuse d'adopter un ancien job déjà publié. Les reprises utilisent les tentatives du même job, pas une réaffectation de lien.

Les états enfants sont calculés depuis le job : pending→queued, leased→running, retry_wait→retry, failed→dead, done/cancelled inchangés. Les diagnostics non encore observés sont `unexamined`, cause `not_started`, tentative 0. Les observations restent stockées ; la lecture prend la tentative la plus élevée par piste. `loss_preview` reste absent tant que son propriétaire ne l'a pas calculé : aucun faux aperçu vide n'est fourni.

## Compatibilité et suite

La migration 00111 matérialise les manifestes historiques `legacy:<job_id>` et installe une capture des réussites de l'ancien protocole. `audio_visible_legacy_transcripts` exclut les enfants liés et conserve le dernier résultat historique réussi. Lecture, recherche lexicale/sémantique, corrections et export continuent à utiliser les jobs et indices historiques ; aucun ASR n'est déclenché. La complétion des enfants conserve l'événement et son résultat mais ne publie ni correction obsolète, ni tâche d'indexation, ni projection monopiste.

#340 devra remplacer cette projection par une lecture de la révision publiée, traiter les identités de correction et d'indexation d'une fusion, protéger la publication contre les corrections concurrentes et vérifier toutes les pistes requises avant publication (y compris une révision vide). Un candidat persisté n'autorise aucune de ces mutations. Le modèle ne promet pas de transférer automatiquement une correction d'une source vers une fusion différente. La suppression explicite d'un Run/clip supprime ses identités en cascade, conformément aux règles existantes ; pas de conservation hors de la durée de vie du clip.

## Vérification

- `cd app && go test ./internal/audioplans` : identité canonique, provenance multiple, ordre temporel et ex æquo.
- `bash scripts/test.sh ./internal/httpapi -run 'TestAudioPlansMigration|TestFoundation/(audio_plan|segment_|search_corrections|HTTP_contract)'` : PostgreSQL réel et données synthétiques, migration/rejeu/retour, contrats HTTP, compatibilité lecture/recherche/correction/export, diagnostic par tentative et complétion d'enfant via Worker.
- `python3 scripts/check_generated.py` : reproductibilité des clients et du modèle SQL générés.

## Sélection depuis la préparation et la réanalyse (#341)

`GET/POST /api/admin/clips/{id}/audio-selection[/preview]` valident le choix (Auto ou indices réels), les décisions et le forçage, renvoient un aperçu signé puis créent le Plan dans la transaction du démarrage. Le moteur durable est injecté par `Server.AudioSelection` (`AudioSelectionEngine`) ; absent, la production reste mono-piste. Voir l'[ADR 0051](adr/0051-selection-des-pistes-audio-et-apercu-des-pertes.md).

## Publication (#340)

`Store.PreviewPublication(ctx, planID)` puis `Store.PublishRevision(ctx, revisionID, aperçu, confirmé)` publient atomiquement une révision complète. Erreurs : `ErrIncomplete`, `ErrPublicationChanged` (nouvel aperçu requis), `ErrConfirmationRequired`. Les lecteurs utilisent les vues `audio_visible_*`. Voir l'[ADR 0050](adr/0050-publication-atomique-des-revisions-audio.md).

## Voix par piste (#342)

Après la publication, `Server.StartPlanVoice(ctx, planID, clipID, auteur, confirmer)` (ou `POST /api/admin/clips/{id}/audio-plans/{plan_id}/voice`, corps `{"confirm_replace_validations": bool}`) crée, pour chaque piste retenue sauf la piste micro, une diarisation liée au Plan (`BindChild`, étape `diarization`) ; la projection du résultat crée l'étape `fingerprint` liée au même Plan. `GET .../voice` rend `AudioPlanVoice` : état et cause par piste (`not_started`, `queued`, `running`, `done`, `failed`, `cancelled`, `not_applicable`), travaux conservés (`reused`) et travail humain en jeu (`stake`). Erreurs : `audio_plan_not_published`, `audio_plan_superseded`, `voice_step_in_progress`, `voice_rediarization_confirmation_required`.

- Les Workers font les travaux usuels (`voice.diarization.<i>`, `voice.fingerprint.<i>`) ; le décalage de la piste est géré côté serveur, une seule fois (stockage des tours en temps du clip, restitution en temps de la piste au Worker).
- `GET /api/clips/{id}/voice` ajoute `plan_id` et `tracks` quand la transcription visible est une révision de Plan, et liste les voix de toutes les pistes retenues ; chaque voix porte `audio_stream_index`. Sans révision de Plan, la lecture monopiste est inchangée.
- Les transcriptions d'une révision de Plan exposent `speaker_coverage_by_track` : une voix détectée sans parole reste visible, même sur une piste sans segment.
- Raccord avec #344 : appeler `StartPlanVoice` une fois la révision publiée ; relire `PlanVoiceState` pour l'état d'avancement. Voir l'[ADR 0052](adr/0052-voix-par-piste-d-un-plan-audio.md).
