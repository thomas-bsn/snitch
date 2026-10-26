# Worker vocal réel : fournisseurs pyannote (diarisation et empreintes)

Complément de la phase 5 ([spec](specs/phase-5-real-voice-worker.md)), points 1 et 2. Le packaging Docker et le verrouillage des dépendances sont traités à part ; ce document décrit le code du worker, sa configuration et ses limites.

## Sélection du fournisseur

Le fournisseur est celui du snapshot du pipeline vocal (`provider` : `pyannote` ou `fake`, voir [diarisation](voice-diarization.md)). `fake` reste disponible et ne touche jamais au média. Tout autre nom échoue avec le code expurgé `provider_unavailable`.

- Code : `worker/snitch_worker/voice.py` (sélection, correspondance vers le format des résultats, limites de l'API) et `worker/snitch_worker/voice_pyannote.py` (fournisseur réel).
- Audio d'entrée : pour `pyannote`, le worker télécharge le média de la tâche (`media_url`, jeton de lease) et le prépare comme la transcription (ffmpeg, mono 16 kHz, piste de l'étape), dans un répertoire temporaire supprimé ensuite. Une pré-vérification (chemins de modèles) a lieu avant tout téléchargement.

## Configuration (variables d'environnement)

| Variable | Rôle |
| --- | --- |
| `SNITCH_VOICE_DIARIZATION_MODEL_PATH` | Répertoire local du pipeline de diarisation (contient `config.yaml`) ou chemin du `config.yaml`. Obligatoire. |
| `SNITCH_VOICE_EMBEDDING_MODEL_PATH` | Chemin local du modèle d'empreinte. Obligatoire. |
| `SNITCH_VOICE_EMBEDDING_MODEL_VERSION` | Version enregistrée avec chaque empreinte (par exemple la révision Hugging Face téléchargée). Obligatoire : deux versions produisent des empreintes incomparables. |
| `SNITCH_VOICE_EMBEDDING_MODEL_NAME` | Nom enregistré. Défaut `pyannote/wespeaker-voxceleb-resnet34-LM`. |
| `SNITCH_VOICE_DEVICE` | `auto` (défaut : GPU si disponible, sinon CPU), `cuda` (échoue avec `cuda_unavailable` sans GPU), `cpu`. |

Le nom et la version du modèle de diarisation viennent du snapshot du pipeline et sont recopiés dans le résultat (l'API les compare au snapshot).

## Aucun téléchargement implicite

- Le worker force `HF_HUB_OFFLINE=1` et `TRANSFORMERS_OFFLINE=1` avant de charger une bibliothèque et ne lit jamais `HF_TOKEN`.
- Chemin absent, bibliothèque ML absente, modèle illisible, mémoire GPU insuffisante : la tâche échoue avec `provider_unavailable` (la file ne conserve que ce code ; tout autre code devient `worker_failed`). Le message est constant, la cause d'origine est supprimée (aucun chemin, jeton ni message de bibliothèque), et seul un code de raison stable est journalisé : `model_path_missing`, `embedding_version_missing`, `device_invalid`, `cuda_unavailable`, `ml_libraries_missing`, `model_load_failed`, `inference_failed`, `result_too_large`, `invalid_embedding`, `audio_missing`, `unknown_provider`.
- Le seul téléchargement est la commande manuelle `worker/scripts/download_models.py` : elle exige `--confirm-terms` (conditions acceptées sur Hugging Face) et `HF_TOKEN` dans l'environnement, n'affiche jamais le jeton ni les erreurs de bibliothèque, et n'est appelée par aucun autre code. Épingler `--revision` pour que la version enregistrée corresponde au modèle réellement présent.

## Choix du modèle d'empreinte : pyannote plutôt que SpeechBrain

Le modèle retenu est l'embedding de locuteur WeSpeaker ResNet34 distribué par pyannote (`pyannote/wespeaker-voxceleb-resnet34-LM`), celui que le pipeline `community-1` utilise lui-même.

- Aucune dépendance de plus : `pyannote.audio` est déjà requis pour la diarisation et fournit `Model` et `Inference` (fenêtre unique) ; SpeechBrain (ECAPA-TDNN) ajouterait une seconde pile, sa propre gestion de modèles et un risque de conflit de versions `torch`/`torchaudio` dans l'image.
- Cohérence : les empreintes sont calculées par le même type de modèle que celui qui a séparé les voix, sur les mêmes tours.
- Même mécanisme hors ligne (chemin local, aucun cache implicite) et même processus de téléchargement explicite.
- Réserve : le choix définitif et la comparaison de qualité relèvent du benchmark de la phase 7. L'interface est un simple chargeur injectable (`load_embedder`) ; un fournisseur SpeechBrain pourrait s'ajouter sans changer le contrat, en changeant `embedding_model` et `embedding_model_version`.

## Résultats produits

Mêmes champs que le fournisseur factice, validés par l'API :

- Diarisation : locuteurs renommés `SPEAKER_00`… dans l'ordre de première apparition, tours triés, tours vides ignorés. Plus de 32 locuteurs ou 2000 tours échouent côté worker (`result_too_large`) plutôt que d'être rejetés par l'API.
- Empreintes : un vecteur normalisé (norme 1) par Locuteur local (portée `speaker` : moyenne des vecteurs de tours pondérée par la durée, `start`/`end` = premier début et durée de parole embarquée) et un par tour d'au moins 0,5 s (portée `turn`, `turn_index` = index fourni par l'API). Les tours plus courts ne sont pas embarqués (la fenêtre du modèle est trop courte) ; un locuteur sans tour embarquable n'a pas d'empreinte. Au-delà de 400 empreintes, seuls les tours les plus longs sont conservés.
- Qualité : heuristique non calibrée, `durée / 3 s` par tour et `durée de parole / 10 s` par locuteur, bornée à 1. L'API calcule `usable` elle-même (durée minimale et qualité minimale du snapshot) ; la calibration réelle viendra du benchmark.

## Ressources

Les modèles sont chargés une fois par processus (mémoire) et l'inférence est sérialisée par un verrou : avec une capacité supérieure à 1, les tâches vocales ne se disputent pas les 12 Go de VRAM.

## Dépendances Python requises (à verrouiller par le packaging)

Non ajoutées à `pyproject.toml`, `uv.lock` ni `runtime.lock` par ce changement. Versions à vérifier au verrouillage (compatibilité de `torch` avec CUDA de l'image à confirmer) :

- `pyannote.audio` : 4.x (le pipeline `community-1` requiert la série 4 ; version testée à ma connaissance : 4.0.x). Elle tire notamment `torch`, `torchaudio`, `pyannote.core`, `huggingface_hub`, `lightning`.
- `torch` et `torchaudio` : versions appariées, compatibles avec `pyannote.audio` 4.x (série 2.8.x à ma connaissance) et la version CUDA de l'image.
- `huggingface_hub` (déjà une dépendance de pyannote ; utilisée seulement par `download_models.py`, à ne pas lancer dans l'image de production si elle n'a pas de réseau).
- `ffmpeg` : déjà requis par la transcription.
- SpeechBrain : non requis.

Licences à inscrire dans `deploy/NOTICES.md` / `deploy/licenses` : `pyannote.audio` (MIT), pyannote `community-1` et `wespeaker-voxceleb-resnet34-LM` (conditions d'accès Hugging Face, licences des modèles à relire), torch (BSD-3), WeSpeaker/VoxCeleb pour les poids (vérifier).

## Limite : jamais exécuté de bout en bout ici

Le code réel (`_load_pipeline`, `_load_embedder`, chargements de `pyannote.audio`, `torchaudio.load`, `Inference.crop`) n'a pas été exécuté : pas de GPU, pas de modèle, pas de jeton Hugging Face, aucun média réel dans cet environnement. Les tests (`worker/tests/test_voice_real.py`) injectent un pipeline et un embedder simulés et vérifient la sélection, la correspondance vers le format des résultats, la qualité, les tours courts, la limite de 400, les erreurs expurgées, le chargement unique, le mode hors ligne forcé et l'absence d'import de `torch`/`pyannote` au démarrage. Les points suivants reposent sur la documentation pyannote et sont à vérifier à la première exécution réelle : chargement d'un pipeline depuis un répertoire local (`Pipeline.from_pretrained(<config.yaml>)` avec des chemins relatifs), sortie `DiarizeOutput.speaker_diarization` (4.x) ou `Annotation` (3.x), chargement d'un `Model` depuis un chemin local, `Inference(window="whole").crop` sur un dictionnaire `waveform`/`sample_rate`.
