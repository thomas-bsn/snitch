# Personnes, attributions et validation d'un Locuteur local

Phase 5, ticket 33. Voir la [spec](specs/phase-5.md), l'[ADR 0008](adr/0008-identification-progressive.md) et le [glossaire](../GLOSSARY.md). La diarisation qui produit les Locuteurs locaux est décrite dans [diarisation et empreintes](voice-diarization.md).

Un utilisateur connecté écoute un extrait d'un Locuteur local, crée une Personne (ou en choisit une) et valide l'attribution. La première validation fait foi ; une correction explicite reste possible ; chaque changement est historisé. Tous les utilisateurs connectés peuvent créer, valider et corriger ; aucune de ces actions n'exige l'administrateur.

## Modèle

| Table | Rôle |
| --- | --- |
| `persons` | Identité (`display_name`). Jamais un compte : `user_id` est un lien facultatif et unique vers `users`. |
| `speaker_attributions` | Lien Locuteur local - Personne : `person_id`, `decision_level` (`human`, `automatic`, `suggestion`, `ambiguous`, `unknown`), `validation_state` (`validated`, `unvalidated`), `confidence`, `origin`, auteur (id et nom au moment de la décision), `revision`, `decided_at`. |
| `voice_samples` | Échantillon vocal validé : référence d'une Personne. `retired_at`/`retired_reason` retirent un échantillon des références sans l'effacer. Le rapprochement ne lit que les échantillons actifs (`retired_at IS NULL`). |
| `attribution_events` | Historique immuable : action, ancienne et nouvelle Personne (id et nom au moment du changement), anciens et nouveaux niveaux, auteur, date. Aucune donnée vocale. |

Un Locuteur local porte au plus une ligne **validée** et au plus une ligne **machine** (`unvalidated`), chacune garantie par un index unique partiel. L'attribution effective est la ligne validée si elle existe, sinon la ligne machine, sinon « inconnu ». Une attribution automatique n'est jamais un échantillon validé et n'altère pas les références : seules une validation ou une correction humaine créent un `voice_samples`.

## Règles

- **Première validation.** `POST /api/local-speakers/{id}/validation` insère la ligne validée ; l'index unique `speaker_attributions_validated` départage les validations concurrentes. La perdante annule tout (y compris une Personne qu'elle créait dans le même appel) et reçoit `409 already_validated` avec l'attribution existante et son auteur. La validation est possible même quand une confiance automatique existe ; l'ancienne valeur machine est conservée dans l'historique.
- **Création de Personne.** Soit `POST /api/persons`, soit `new_person_name` dans la validation ou la correction (même transaction que l'attribution).
- **Correction.** `POST /api/local-speakers/{id}/correction` exige `expected_person_id` (la Personne vue par l'utilisateur) : si elle a changé, `409 attribution_changed` renvoie l'attribution actuelle. Une correction vers la même Personne est refusée (`same_person`), une correction avant toute validation aussi (`not_validated`). Elle met à jour l'attribution (révision +1, auteur, date), retire l'ancien échantillon des références (`retired_reason = corrected`), en crée un nouveau et ajoute un événement.
- **Historique.** `GET /api/local-speakers/{id}/history`, du plus récent au plus ancien. Les lignes ne peuvent être ni modifiées ni supprimées directement (déclencheur) ; elles disparaissent uniquement avec le Locuteur local, donc avec le clip.
- **Extraits.** `GET /api/local-speakers/{id}/excerpt` sert un WAV mono (16 kHz, dix secondes au plus) du plus long tour du locuteur (`?turn=<indice>` pour un autre), découpé à la demande avec `ffmpeg` (variable `FFMPEG_PATH`, `ffmpeg` par défaut ; présent dans l'image) depuis la piste audio diarisée. Rien n'est stocké ; deux découpes simultanées au plus. Session obligatoire (401 sinon) ; un clip que l'utilisateur ne peut pas voir répond 404, comme un Locuteur local inexistant.
- **Affichage.** `LocalSpeaker` expose `state` (`identified` pour une attribution humaine ou automatique ; une suggestion ou une voix ambiguë reste `unidentified`), `decision_level`, `person` et `attribution`. Chaque segment de transcription porte aussi `local_speaker_person_id`, `local_speaker_person_name` et `local_speaker_decision_level`. Les cinq niveaux se distinguent par libellé, glyphe et style (jamais par la couleur seule). Le nombre d'échantillons validés d'une Personne est affiché ; il n'est pas une garantie de qualité d'identification.

## Re-diarisation d'un clip validé

Une relance (`POST /api/admin/clips/{id}/voice`, onglet Voix du clip) est toujours une action explicite d'un administrateur, avec le modèle actif (même modèle autorisé : nouveau cycle) ; une activation de modèle ne relance rien.

- **Confirmation.** Si la piste porte des locuteurs validés, des échantillons vocaux actifs ou des corrections de locuteur de la transcription, l'API répond `409 voice_rediarization_confirmation_required` avec `stake` (`validated_speakers`, `voice_samples`, `speaker_corrections`) et ne planifie rien ; `confirm_replace_validations: true` confirme.
- **Résultat visible.** L'ancien résultat reste affiché tant que la relance n'a pas réussi (en attente, en cours ou en échec).
- **À la réussite.** Les corrections de locuteur deviennent obsolètes (retirées, avec un événement d'historique, comme les segments corrigés après une retranscription). Les Locuteurs locaux validés ne sont pas supprimés : ils passent en `kind = 'archived'` (migration 00056) avec leur attribution validée, leurs échantillons vocaux et leurs empreintes de locuteur, liés à l'audio, sans tour ni attribution automatique ; ils n'apparaissent plus dans la liste des voix, la transcription ni le rapprochement des voix inconnues, mais l'échantillon reste une référence de la Personne. Les autres locuteurs sont remplacés.
- **Historique.** `GET /api/clips/{id}/voice` liste dans `history` les résultats remplacés (tâche, modèle, version, date, nombre de voix, voix validées conservées).
- Seule une piste micro désignée reste refusée à l'achèvement (`409 diarization_locked`). Un déclencheur PostgreSQL refuse toujours la suppression d'un Locuteur local validé tant que son clip existe (`23001`).

Les attributions automatiques (non validées) sont des données dérivées : elles sont recalculées après une re-diarisation. Le changement de piste sur un clip validé n'est pas prévu.

## Contrat et code

Routes dans `app/internal/httpapi/voice_persons.go` et `voice_excerpt.go` (`registerPersonRoutes`). Écrans web dans `web/src/voice/SpeakerRow.tsx`. Tests : `voice_persons_test.go` (concurrence, historique, permissions des extraits, verrou de re-diarisation, cascade) et `web/e2e/persons.spec.ts`. L'audio de test est synthétique : un faux `ffmpeg` enregistre ses arguments et renvoie des octets fixes.
