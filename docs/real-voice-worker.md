# Worker vocal réel : fournisseurs pyannote (diarisation et empreintes)

Complément de la phase 5 ([spec](specs/phase-5-real-voice-worker.md)), points 1 et 2. Le point 3 (image GPU, verrouillage, exemple Compose) est décrit dans [Installer sur le PC gamer](#installer-sur-le-pc-gamer) ; ce document décrit aussi le code du worker, sa configuration et ses limites.

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

## Dépendances Python (verrouillées)

Elles sont dans l'extra `voice` de `worker/pyproject.toml` (absent de l'image CPU `worker.Dockerfile` et de `runtime.lock`, qui restent inchangés) : `pyannote.audio==4.0.7` (série 4, requise par `community-1` ; PyPI : `torch>=2.8.0`, `torchaudio>=2.8.0`, `torchcodec>=0.7.0`), `torch==2.8.0`, `torchaudio==2.8.0`, `soundfile==0.14.0` (sans lui, `torchaudio.load` de torchaudio 2.8 n'a aucun backend dans l'image : constaté au premier build), `torchcodec==0.7.0` (couple torchcodec 0.7 / torch 2.8 de la matrice de compatibilité de torchcodec ; à reconfirmer). `worker/uv.lock` les résout ; `worker/voice-runtime.lock` en est l'export complet avec empreintes SHA-256 (`cd worker && uv export --extra voice --no-dev --no-emit-project --no-emit-package snitch-client --format requirements-txt -o voice-runtime.lock`), installé avec `--require-hashes` par `deploy/docker/worker-gpu.Dockerfile`. Les roues PyPI de torch 2.8.0 pour Linux x86_64 sont liées à CUDA 12.8 et tirent cuDNN 9.10 et les bibliothèques CUDA par des paquets `nvidia-*-cu12` : l'image reste une base `python:3.12-slim` épinglée par condensé, sans image `nvidia/cuda`. Le pilote de l'hôte doit gérer CUDA 12.8 (série 570 ou plus récente d'après la documentation NVIDIA ; à confirmer avec `nvidia-smi`). `huggingface_hub` (résolu en 2.3.0) n'est utilisé que par `download_models.py`.

Licences : voir `deploy/NOTICES.md` et `deploy/licenses/voice/LICENSES.md` (les roues NVIDIA sont sous licence propriétaire NVIDIA ; les modèles sont en CC-BY-4.0 avec conditions d'accès Hugging Face).

## Installer sur le PC gamer

Cible : un PC Linux (ou Windows avec WSL2 et Docker) avec une carte NVIDIA, par exemple une RTX 4070 (12 Go). Le worker ne publie aucun port et appelle l'API Snitch en HTTPS sortant ; les secrets restent sur ce PC. Les fichiers sont dans `deploy/worker/` (`compose.yaml`, `.env.example`).

1. **Prérequis.** Pilote NVIDIA récent (`nvidia-smi` fonctionne, CUDA 12.8 ou plus), Docker avec Compose v2 et le [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) configuré pour Docker. Vérifier l'accès GPU : `docker run --rm --gpus all ubuntu nvidia-smi`.
2. **Créer le worker dans l'interface Snitch** (page des workers, administrateur) pour obtenir le jeton, affiché une seule fois. Snitch doit être joignable en HTTPS depuis le PC.
3. **Préparer `.env`.** `cd deploy/worker && cp .env.example .env`, puis renseigner `SNITCH_API_URL`, `SNITCH_WORKER_TOKEN`, `TRANSCRIPTION_API_URL` et `TRANSCRIPTION_MODEL` (et `TRANSCRIPTION_API_KEY` si besoin), `SNITCH_MODELS_DIR` (dossier du PC qui recevra les modèles) et, après l'étape suivante, `SNITCH_VOICE_EMBEDDING_MODEL_VERSION`. Ne jamais versionner `.env`.
4. **Télécharger les modèles (une seule fois, explicite).** Créer un compte Hugging Face gratuit, ouvrir les pages de `pyannote/speaker-diarization-community-1` (accès « gated » : accepter les conditions) et de `pyannote/wespeaker-voxceleb-resnet34-LM` (à la lecture de la fiche, aucune restriction affichée ; à vérifier, et accepter les conditions si elles sont demandées), créer un jeton de lecture, puis :
   ```sh
   docker compose build
   mkdir -p "$SNITCH_MODELS_DIR"
   docker run --rm --user "$(id -u):$(id -g)" -e HF_TOKEN -e HF_HUB_OFFLINE=0 \
     -v "$SNITCH_MODELS_DIR:/models" --entrypoint python snitch-worker-gpu:local \
     /worker/scripts/download_models.py --confirm-terms --revision <révision> \
     --diarization-dir /models/diarization --embedding-dir /models/embedding
   ```
   avec `HF_TOKEN` exporté dans le shell (jamais écrit dans `.env`). L'image force `HF_HUB_OFFLINE=1` ; `-e HF_HUB_OFFLINE=0` n'est donné qu'à cette commande ponctuelle. Noter la révision utilisée : c'est la valeur de `SNITCH_VOICE_EMBEDDING_MODEL_VERSION`. Le conteneur du worker monte ensuite les modèles en lecture seule et n'a jamais besoin du jeton.
5. **Lancer.** `docker compose up -d --build` dans `deploy/worker`, puis `docker compose logs -f worker`.
6. **Vérifier dans Snitch.** Le worker apparaît actif sur la page des workers avec ses capacités (`transcription.*`, `voice.*`). Un échec `provider_unavailable` renvoie au code de raison stable des journaux (`model_path_missing`, `cuda_unavailable`, `model_load_failed`...).
7. **Brancher la transcription (Whisper).** Un serveur compatible `POST /v1/audio/transcriptions` : sur le PC hors Docker, `TRANSCRIPTION_API_URL=http://host.docker.internal:<port>` (le compose ajoute `host.docker.internal:host-gateway`) ; dans un autre projet Compose, activer l'option de réseau partagé du compose et viser le nom du service. Le profil de pipeline de l'interface doit utiliser le même nom de modèle que `TRANSCRIPTION_MODEL`.
8. **Embeddings de texte (Ollama).** Ils restent côté NAS (`EMBEDDINGS_URL` et `EMBEDDINGS_MODEL` de l'application, voir [déploiement](deployment.md)) et ne passent pas par le worker. Un Ollama sur le PC peut servir de fournisseur si le NAS peut le joindre (réseau privé de votre choix).
9. **Configurer le pipeline vocal** dans l'interface (fournisseur `pyannote`, modèle `pyannote/speaker-diarization-community-1`, version téléchargée) et confirmer `access_confirmed` : c'est la confirmation explicite par l'administrateur que les conditions ont été acceptées.

### VRAM : 12 Go partagés

Un Ollama chargé avec un gros modèle peut occuper l'essentiel des 12 Go. Garder `SNITCH_WORKER_CAPACITY=1`, ne pas faire tourner en même temps une transcription Whisper sur GPU, la diarisation et un grand modèle Ollama (`OLLAMA_KEEP_ALIVE` court, ou arrêter Ollama pendant les traitements vocaux). Une mémoire insuffisante fait échouer la tâche avec `provider_unavailable` (relançable). `SNITCH_VOICE_DEVICE=cpu` est un repli très lent. La consommation réelle n'a pas été mesurée (aucun GPU ici).

### Test avec 10 vidéos

Importer 10 vidéos choisies par vous par l'interface d'import de Snitch (aucun scan de dossier), les transcrire, puis lancer « Diariser les voix » clip par clip. AGENTS.md interdit tout média réel avant l'autorisation du benchmark de la phase 7 : cette autorisation relève de l'utilisateur, jamais d'un agent ni de la CI. Relever durée, VRAM (`nvidia-smi`) et erreurs ; ce n'est pas le benchmark de la phase 7.

### Vérifié et non vérifié

Construit et contrôlé sans GPU (voir le Dockerfile) : installation avec empreintes, `pip check`, import de torch/torchaudio/pyannote.audio, `torch.version.cuda == 12.8`, décodage WAV 16 kHz par torchaudio. Non vérifié : tout accès GPU réel, `nvidia-smi` dans le conteneur, chargement des modèles, la commande de téléchargement ci-dessus, la compatibilité de `huggingface_hub` 2.3.0 avec pyannote.audio 4.0.7 au chargement local, l'hôte Windows/WSL2.

## Limite : jamais exécuté de bout en bout ici

Le code réel (`_load_pipeline`, `_load_embedder`, chargements de `pyannote.audio`, `torchaudio.load`, `Inference.crop`) n'a pas été exécuté : pas de GPU, pas de modèle, pas de jeton Hugging Face, aucun média réel dans cet environnement. Les tests (`worker/tests/test_voice_real.py`) injectent un pipeline et un embedder simulés et vérifient la sélection, la correspondance vers le format des résultats, la qualité, les tours courts, la limite de 400, les erreurs expurgées, le chargement unique, le mode hors ligne forcé et l'absence d'import de `torch`/`pyannote` au démarrage. Les points suivants reposent sur la documentation pyannote et sont à vérifier à la première exécution réelle : chargement d'un pipeline depuis un répertoire local (`Pipeline.from_pretrained(<config.yaml>)` avec des chemins relatifs), sortie `DiarizeOutput.speaker_diarization` (4.x) ou `Annotation` (3.x), chargement d'un `Model` depuis un chemin local, `Inference(window="whole").crop` sur un dictionnaire `waveform`/`sample_rate`.
