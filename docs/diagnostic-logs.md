# Diagnostiquer un clip bloqué : logs et causes d'échec

Cette page explique comment retrouver pourquoi un clip est bloqué entre le NAS (serveur Snitch) et un Worker WSL2/GPU. Elle applique l'[ADR 0023](adr/0023-diagnostic-par-causes-d-echec-persistees.md) et le terme « Cause d'échec » du [glossaire](../GLOSSARY.md). Tous les exemples sont synthétiques.

## Ce qui est journalisé

Le serveur et le Worker écrivent une ligne JSON par événement, avec le même schéma : `timestamp`, `level`, `service` (`snitch` ou `worker`), `event`, puis le contexte de corrélation et, selon le cas, `step`, `duration_ms`, `error_code`, `model` et `revision`.

| Champ | Sens |
| --- | --- |
| `clip_id` | le clip analysé |
| `job_id` | la tâche (une étape d'un clip) |
| `cycle` | le lancement de l'analyse ; un « Relancer » ouvre un nouveau cycle |
| `attempt` | la tentative du cycle (1, 2, 3…) ; un retry est une autre tentative |
| `worker_id` | le Worker qui a pris la tâche |
| `step` | l'étape (`transcription.audio.2`, `voice.diarization.2`, `voice.fingerprint.2`…) |
| `error_code` | la Cause d'échec, issue du catalogue ci-dessous |

Les identifiants viennent du claim côté serveur et sont repris tels quels par le Worker : une même tâche se retrouve donc avec les mêmes valeurs des deux côtés. Il n'existe pas d'identifiant de run ; `workflow_run_id` viendra avec #130.

En INFO, seules les transitions sont écrites : `job_claimed`, `job_started`, `job_completed`, `job_failed` (WARN), `job_retry_scheduled` (avec `delay_ms`), `job_lease_expired`, `job_dead` (WARN), `job_cancelled` (avec l'auteur), `worker_revoked`, `voice_matching_summary`, et les jalons de modèle (`voice_model_download_resumed`, `voice_model_validated`, `voice_model_activated`, `voice_model_failed`) ainsi que `diarization_model_loaded` et `diarization_inference_done` (processeur CPU ou GPU, durées, nombre de locuteurs et de segments), et `voice_fingerprint_inference_done` (vecteurs produits, tours embarqués, tours écartés ou raccourcis parce qu'ils débordent de la fin de l'audio). Un échec d'empreinte écrit `voice_fingerprint_failed` (WARN) avant `job_failed` : étape (`model_load`, `audio_decode`, `segment_extraction`, `embedding_inference`, `post_processing`), classe de l'exception (`exc_class`), extrait expurgé de son message (`error_summary`, 240 caractères au plus), nombre et durées des segments, segment en cours, fréquence d'échantillonnage, canaux, durée de l'audio et forme de la fenêtre. Le serveur ne reçoit que le code d'erreur (`error_code`, par exemple `inference_failed`). Les heartbeats, les claims vides (`claim_empty`) et la progression de téléchargement (par paliers) n'apparaissent qu'en DEBUG.

## Régler le niveau

`LOG_LEVEL` vaut `ERROR`, `WARN`, `INFO` (défaut) ou `DEBUG`, séparément pour chaque service.

- NAS : `LOG_LEVEL` dans le `.env` de Snitch, puis `docker compose up -d`.
- Worker : `LOG_LEVEL` dans le `.env` du dossier `deploy/worker/` (ou celui généré à la création du Worker), puis `docker compose up -d`.

Activez DEBUG volontairement, le temps d'une investigation, puis revenez à INFO. DEBUG ajoute du volume, jamais de contenu privé (voir plus bas).

## Exemple : un clip bloqué entre le NAS et le Worker WSL2/GPU

Situation : l'administrateur voit le clip 12 en « diarisation » depuis longtemps. Sur la page du clip, la cause affichée est « GPU saturé ». Il confirme dans les logs.

1. Sur le NAS, retrouver la tâche du clip :

   ```
   docker compose logs app | grep '"clip_id":12'
   ```

   ```json
   {"timestamp":"2026-01-10T20:14:02Z","level":"INFO","service":"snitch","event":"job_claimed","clip_id":12,"job_id":88,"step":"voice.diarization.2","cycle":0,"attempt":1,"worker_id":"wk_example"}
   {"timestamp":"2026-01-10T20:14:41Z","level":"WARN","service":"snitch","event":"job_failed","clip_id":12,"job_id":88,"step":"voice.diarization.2","cycle":0,"attempt":1,"worker_id":"wk_example","error_code":"gpu_out_of_memory"}
   {"timestamp":"2026-01-10T20:14:41Z","level":"INFO","service":"snitch","event":"job_retry_scheduled","clip_id":12,"job_id":88,"attempt":1,"worker_id":"wk_example","error_code":"gpu_out_of_memory","delay_ms":30000}
   ```

   Le serveur a donné la tâche au Worker `wk_example`, qui a échoué avec `gpu_out_of_memory` et un retry est programmé.

2. Sur le PC gamer (WSL2), suivre la même tâche avec le même `job_id` :

   ```
   docker logs snitch-worker 2>&1 | grep '"job_id":88'
   ```

   ```json
   {"timestamp":"2026-01-10T20:14:03Z","level":"INFO","service":"worker","event":"job_started","clip_id":12,"job_id":88,"step":"voice.diarization.2","cycle":0,"attempt":1,"worker_id":"wk_example"}
   {"timestamp":"2026-01-10T20:14:40Z","level":"WARN","service":"worker","event":"job_failed","clip_id":12,"job_id":88,"step":"voice.diarization.2","cycle":0,"attempt":1,"worker_id":"wk_example","error_code":"gpu_out_of_memory","duration_ms":36500}
   ```

   Ajouter `"event":"diarization_model_loaded"` au filtre pour voir le processeur utilisé et le temps de chargement.

3. Appliquer l'action conseillée du catalogue (`gpu_out_of_memory` : vérifier le GPU). Dans WSL2, `nvidia-smi` montre la mémoire occupée ; fermer ce qui la consomme (jeu, autre modèle), puis laisser le retry reprendre ou relancer l'analyse.

4. Un retry est une nouvelle ligne `job_claimed` avec `attempt` incrémenté. Si le bail a expiré parce que le Worker ne répondait plus, le serveur écrit `job_lease_expired` (avec le `worker_id` qui l'avait) puis la tâche repart ; après le nombre maximal de tentatives, `job_dead` avec `max_attempts_exhausted`.

Filtrer plusieurs critères : `docker compose logs app | grep '"job_id":88' | grep -v '"level":"INFO"'` ne garde que les alertes de la tâche. Pour un Worker précis, filtrer sur `"worker_id":"wk_example"`.

## Catalogue des causes d'échec

Le catalogue est fermé et déclaré dans le contrat (`FailureCode`). Le Worker n'envoie qu'un code ; l'action conseillée est décidée par le serveur et affichée aux seuls Administrateurs. Un code inconnu (ancien Worker) est enregistré comme `worker_failed`.

| Code | Famille | Action conseillée |
| --- | --- | --- |
| `worker_failed` | général | consulter les logs du Worker |
| `provider_unavailable` | général | vérifier le service de transcription |
| `invalid_input` | général | vérifier le média du clip |
| `audio_preparation_failed` | général | vérifier le média du clip |
| `lease_expired` | général | vérifier la connexion du Worker au serveur |
| `max_attempts_exhausted` | général | relancer la tâche |
| `cancelled_by_admin` | général | relancer la tâche si besoin |
| `input_version_superseded` | général | aucune (remplacée par une version plus récente) |
| `worker_revoked` | général | vérifier la connexion du Worker, recréer un Worker si besoin |
| `whisper_network` | Whisper | vérifier le service de transcription et le réseau |
| `whisper_timeout` | Whisper | vérifier la charge du service de transcription |
| `whisper_http_error` | Whisper | vérifier le service (le statut HTTP est enregistré) |
| `whisper_model_missing` | Whisper | installer le modèle de transcription |
| `whisper_invalid_response` | Whisper | vérifier que le service est compatible `/v1/audio/transcriptions` |
| `pyannote_model_missing` | diarisation | installer le modèle vocal sur le Worker |
| `pyannote_access_denied` | diarisation | vérifier le jeton Hugging Face et les conditions du modèle |
| `gpu_unavailable` | diarisation | vérifier le GPU (pilote, WSL2, passthrough Docker) |
| `gpu_out_of_memory` | diarisation | libérer la mémoire du GPU |
| `diarization_inference_failed` | diarisation | consulter les logs du Worker |
| `hf_auth_required` | Hugging Face | vérifier le jeton Hugging Face (401/403) |
| `hf_not_found` | Hugging Face | vérifier le dépôt du modèle (404) |
| `hf_network` | Hugging Face | vérifier le réseau du Worker |
| `models_volume_unwritable` | Hugging Face | vérifier les droits du volume `/models` |
| `hf_cache_error` | Hugging Face | vérifier le volume `/models` ; vider le cache corrompu |

Un nouveau mode d'échec exige un nouveau code, donc une évolution du contrat.

Dans la Task Queue (`GET /api/admin/analysis/queue`, Administrateurs), chaque étape en échec porte `failure_code` et `recommended_action`, lus dans le dernier événement d'échec persisté de la tâche qu'elle suit ; jamais de texte brut. Une étape qui attend un Worker (`awaiting_worker`, `awaiting_compatible_worker`) porte l'action `start_worker`, sauf pour une reprise différée. Les tâches mortes ou en échec qu'aucune analyse ne suit restent listées avec les mêmes champs. Le résumé (`summary`) donne les tâches en cours (baux non expirés, comme l'ordonnanceur), les clips en attente, le plafond de concurrence et la pause.

## Rotation des logs

Les trois fichiers compose (`compose.yaml`, `compose.override.yaml`, `deploy/worker/compose.yaml`) appliquent à chaque service le pilote Docker `json-file` avec 10 Mo par fichier et 5 fichiers, soit 50 Mo au plus par service. Aucun Loki ni ELK n'est nécessaire. `scripts/check_compose.py` vérifie cette configuration.

## Ce qui n'est jamais journalisé

À aucun niveau, DEBUG compris : jetons (API, bail, Hugging Face `HF_TOKEN`), cookies, en-têtes `Authorization`, URLs signées, chemins sensibles, audio, transcriptions, empreintes et vecteurs vocaux, noms de Personnes et scores individuels de matching. Un filtre central masque les clés interdites côté Go et côté Python, et le texte brut des erreurs de fournisseur ou de base de données est remplacé par un code et une catégorie (`error_code`, `error_category`) ; seul `voice_fingerprint_failed` y ajoute `error_summary`, extrait de 240 caractères au plus d'où sont retirés jetons, en-têtes d'autorisation, URLs, chemins et chaînes opaques longues (ADR 0028). Le matching vocal ne journalise que des comptes par niveau de décision et des raisons fermées (`no_reference`, `below_threshold`, `tied_candidates`, `micro_track_direct`).

## Exception : le code d'installation

Le code d'installation à usage unique du premier démarrage reste affiché dans les logs du serveur (`docker compose logs app`) : c'est ainsi que le super-administrateur le lit pour terminer l'installation ([ADR 0006](adr/0006-bootstrap-secrets.md)). Il est la seule exception. Ne partagez pas ces logs tant que l'installation n'est pas terminée.

## Ce que l'interface montre

Les Administrateurs voient, sur le clip et dans la Task Queue, l'étape, l'heure, la Cause d'échec et l'action conseillée. Ces données viennent de l'historique persisté des tâches, pas des logs. L'utilisateur ordinaire ne voit que l'état de son clip.

## Limites connues

- L'attente d'un Worker capable (aucun Worker ne peut prendre une tâche en attente) n'est pas journalisée avec sa durée ; elle se lit dans l'interface (travaux en attente) et par l'absence de `job_claimed`.
- Pas de panneau de logs dans l'interface ni d'export expurgé (hors périmètre de #122).
- Deux vocabulaires de causes coexistent pour Hugging Face. Les échecs de tâches et d'état des modèles vocaux emploient le catalogue ci-dessus (`hf_auth_required`, `hf_not_found`, `hf_network`…). L'installation du Worker (`bootstrap`) garde ses propres codes (`hf_access_denied`, `network_unreachable`, `volume_not_writable`…), qui passent par le contrat des événements d'installation et fusionnent 401, 403 et 404 en `hf_access_denied`. Ils ne sont pas unifiés : ce serait une évolution de ce contrat, hors de #122.

## Vérification automatisée

Le scénario d'échec simulé (données synthétiques, faux fournisseurs, aucun vrai clip) vérifie que la trace complète est propre : côté serveur, `testSimulatedFailureTraceIsClean` (GPU saturé, retry, bail expiré, reprise, empreintes) ; côté Worker, le test de diarisation en échec de `worker/tests/test_voice_real.py` et les tests Whisper de `worker/tests/test_runner.py`. Aucun ne doit contenir transcription, jeton, URL signée ni vecteur vocal.
