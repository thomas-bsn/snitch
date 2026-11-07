# Diagnostics audio — interfaces #338

Décision, paramètres, licence et limites : [ADR 0048](adr/0048-detecteur-de-parole-silero-hors-ligne.md). Un diagnostic est une observation de présence de parole par piste réelle ; il ne choisit rien et n'active pas Auto.

## Inventaire

`MediaTrack` (détails de clip, bibliothèque) expose l'indice ffprobe, le titre, le codec, la langue, les canaux, la disposition et, nouveaux, `duration_seconds` et `start_seconds` **du flux**. `null` signifie inconnu (Matroska n'a souvent pas de durée par flux) ; ce n'est jamais la durée du conteneur. Les clips déjà indexés reçoivent ces champs à leur prochain sondage.

## API

- `POST /api/admin/clips/{id}/audio-plans/{plan_id}/diagnostics` met en file un job `audio.diagnostic.N` par piste retenue du Plan (toutes les pistes inventoriées sélectionnées, quels que soient leur ordre ou titre) et répond l'état du Plan (`AudioPlanState`). Idempotent. Les états `children[].status` (`queued`, `running`, `retry`, `done`, `dead`) et `diagnostics[]` (`unexamined`/`not_started`, `speech`, `no_speech`, `ambiguous`, `failed` avec `cause`, `detector_version`, `detector_config_version`, `intervals`) suffisent à l'affichage ; `queued` sans Worker équipé signifie que la capacité attend un Worker déclarant `audio.diagnostic.*` et ayant le modèle prêt.
- Modèle : le catalogue contient `silero-vad-v6.2.1` (`kind=vad`) ; `POST /api/admin/workers/{id}/voice-model-orders` avec `{"model_id":"silero-vad-v6.2.1","idempotency_key":"..."}` l'installe. L'interface vocale existante ne propose pas ce type (`vad`) à l'installation.
- Worker : `GET /api/worker/jobs/{id}/input` ajoute `audio_diagnostic` (`plan_id`, `track_index`, `input_version` = digest du Plan, versions du détecteur, `start_seconds`, `duration_seconds|null`). Le résultat `{status, cause, intervals[{start,end}], detector_version, detector_config_version}` est rejeté (`400 invalid_audio_diagnostic`) si le statut est `unexamined`, si les intervalles sont incohérents ou si les versions ne sont pas celles du Plan.

## Pour les autres tickets

- #339 : appeler `audioplans.Store.QueueDiagnostics(ctx, planID)` (ou l'endpoint) après `CreatePlan`, puis attendre le diagnostic terminal de chaque piste avant de décider ; un `failed`, `ambiguous` ou `unexamined` ne justifie jamais d'exclure une piste.
- #341 : afficher les statuts/causes ci-dessus et la progression `queued/running/retry/dead` ; le type `vad` apparaît dans le catalogue de modèles (étiquette « Détecteur de parole »).
- #344 : exécuter les tests acoustiques avec l'environnement Torch 2.8.0 et `SNITCH_TEST_VAD_DIRECTORY` pointant vers un dossier contenant `silero_vad.jit` installé explicitement.

## Installation hors ligne

```
python -m snitch_worker.audio_diagnostic install --directory /models/catalog/snakers4__silero-vad/7e30209a3e901f9842f81b225f3e93d8199902b1 --source /chemin/silero_vad.jit
```
`--download` remplace `--source` pour une acquisition en ligne explicite. Vérification de la taille et du SHA-256 avant tout chargement.
