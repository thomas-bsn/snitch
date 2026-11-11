# Worker vocal réel : fournisseurs pyannote (diarisation et empreintes)

Complément de la phase 5 ([spec](specs/phase-5-real-voice-worker.md)), points 1 et 2. Le point 3 (image worker, verrouillage, exemple Compose) est décrit dans [Installer sur le PC gamer](#installer-sur-le-pc-gamer) ; ce document décrit aussi le code du worker, sa configuration et ses limites.

## Sélection du fournisseur

Le fournisseur est celui du snapshot du pipeline vocal (`provider` : `pyannote` ou `fake`, voir [diarisation](voice-diarization.md)). `fake` reste disponible et ne touche jamais au média. Tout autre nom échoue avec le code expurgé `provider_unavailable`.

- Code : `worker/snitch_worker/voice.py` (sélection, correspondance vers le format des résultats, limites de l'API) et `worker/snitch_worker/voice_pyannote.py` (fournisseur réel).
- Audio d'entrée : pour `pyannote`, le worker télécharge le média de la tâche (`media_url`, jeton de lease) et le prépare comme la transcription (ffmpeg, mono 16 kHz, piste de l'étape), dans un répertoire temporaire supprimé ensuite. Une pré-vérification (chemins de modèles) a lieu avant tout téléchargement.

## Configuration (variables d'environnement)

| Variable | Rôle |
| --- | --- |
| `SNITCH_VOICE_DIARIZATION_MODEL_PATH` | Répertoire local du pipeline de diarisation (contient `config.yaml`) ou chemin du `config.yaml`. Obligatoire. |
| `SNITCH_VOICE_EMBEDDING_MODEL_PATH` | Chemin local du modèle d'empreinte. Obligatoire. |
| `SNITCH_VOICE_EMBEDDING_MODEL_VERSION` | Version enregistrée avec chaque empreinte (par exemple la révision Hugging Face téléchargée). Obligatoire, mais à défaut de la variable le worker la lit dans le marqueur `.snitch-voice-models.json` écrit par le bootstrap à côté des modèles (le dossier parent du chemin d'empreinte) : deux versions produisent des empreintes incomparables. |
| `SNITCH_VOICE_EMBEDDING_MODEL_NAME` | Nom enregistré. Défaut `pyannote/wespeaker-voxceleb-resnet34-LM`. |
| `SNITCH_VOICE_DEVICE` | `auto` (défaut : GPU si disponible, sinon CPU), `cuda` (échoue avec `cuda_unavailable` sans GPU), `cpu`. |

Le nom et la version du modèle de diarisation viennent du snapshot du pipeline et sont recopiés dans le résultat (l'API les compare au snapshot).

## Aucun téléchargement implicite

- Le worker force `HF_HUB_OFFLINE=1` et `TRANSFORMERS_OFFLINE=1` avant de charger une bibliothèque et ne lit jamais `HF_TOKEN`.
- Chemin absent, bibliothèque ML absente, modèle illisible, mémoire GPU insuffisante : la tâche échoue avec `provider_unavailable` (la file ne conserve que ce code ; tout autre code devient `worker_failed`). Le message est constant, la cause d'origine est supprimée (aucun chemin, jeton ni message de bibliothèque), et seul un code de raison stable est journalisé : `model_path_missing`, `embedding_version_missing`, `device_invalid`, `cuda_unavailable`, `ml_libraries_missing`, `model_load_failed`, `inference_failed`, `result_too_large`, `invalid_embedding`, `audio_missing`, `unknown_provider`.
- Un seul chemin de téléchargement : l'ordre d'installation que Snitch donne à un Worker (ADR 0038, voir [« Un Worker neuf n'a aucun modèle »](#un-worker-neuf-na-aucun-modèle)). Ni l'installation, ni le démarrage, ni une reconnexion, ni la liste des modèles affichée dans Snitch ne téléchargent quoi que ce soit ; le bootstrap de l'image (`python -m snitch_worker.bootstrap`) ne télécharge plus rien et le script manuel `download_models.py` n'existe plus. Le processus du worker ne reçoit jamais `HF_TOKEN`.

## Choix du modèle d'empreinte : pyannote plutôt que SpeechBrain

Le modèle retenu est l'embedding de locuteur WeSpeaker ResNet34 distribué par pyannote (`pyannote/wespeaker-voxceleb-resnet34-LM`), celui que le pipeline `community-1` utilise lui-même.

- Aucune dépendance de plus : `pyannote.audio` est déjà requis pour la diarisation et fournit `Model` et `Inference` (fenêtre unique) ; SpeechBrain (ECAPA-TDNN) ajouterait une seconde pile, sa propre gestion de modèles et un risque de conflit de versions `torch`/`torchaudio` dans l'image.
- Cohérence : les empreintes sont calculées par le même type de modèle que celui qui a séparé les voix, sur les mêmes tours.
- Même mécanisme hors ligne (chemin local, aucun cache implicite) et même processus de téléchargement explicite.
- Réserve : le choix définitif et la comparaison de qualité relèvent du benchmark de la phase 7. L'interface est un simple chargeur injectable (`load_embedder`) ; un fournisseur SpeechBrain pourrait s'ajouter sans changer le contrat, en changeant `embedding_model` et `embedding_model_version`.

## Résultats produits

Mêmes champs que le fournisseur factice, validés par l'API :

- Diarisation : locuteurs renommés `SPEAKER_00`… dans l'ordre de première apparition, tours triés, tours vides ignorés. Plus de 32 locuteurs ou 2000 tours échouent côté worker (`result_too_large`) plutôt que d'être rejetés par l'API.
- Empreintes sur la Parole nette (ADR 0064) : chaque tour est embarqué sur ses intervalles hors Chevauchement (le tour moins les moments où un autre Locuteur local de la piste parle) d'au moins 0,5 s ; son vecteur est la moyenne de ces intervalles pondérée par leur durée. Un vecteur normalisé (norme 1) par Locuteur local (portée `speaker` : moyenne de tous ses intervalles embarqués pondérée par la durée, `start`/`end` = premier début et durée de Parole nette embarquée) et un par tour qui en a (portée `turn`, `start`/`end` = bornes du tour, `turn_index` = index fourni par l'API). Un Locuteur local sans aucun intervalle net d'au moins 0,5 s garde une empreinte calculée sur ses tours entiers d'au moins 0,5 s, marquée `overlapped`. Un locuteur sans rien d'embarquable n'a pas d'empreinte. Au-delà de 400 empreintes, seuls les tours les plus longs sont conservés.
- Mesures (`measures`, version 1, `snitch_worker/voice_quality.py`) jointes à chaque empreinte : Parole nette embarquée (0 pour une empreinte chevauchée), part de Chevauchement du tour ou du locuteur (géométrie des tours), marque `overlapped`, part d'échantillons écrêtés et niveau RMS moyen en dBFS de l'audio embarqué (lus dans le WAV 16 bits préparé ; `null` s'il est illisible), cohérence (cosinus moyen entre les vecteurs des tours du locuteur, `null` avec un seul tour et pour une empreinte de tour). Aucune estimation de rapport signal sur bruit. Format : [voice-diarization.md](voice-diarization.md#mesures-dune-empreinte).
- Qualité : heuristique non calibrée, `durée / 3 s` par tour et `durée de parole / 10 s` par locuteur, bornée à 1, sur la Parole nette embarquée. L'éligibilité ne dépend plus de la qualité : Snitch la décide d'après les mesures et sa règle d'éligibilité ([rapprochement](voice-matching.md#éligibilité-des-empreintes), #390). Durée et qualité minimales du snapshot ne s'appliquent plus qu'aux empreintes de benchmark.

## Ressources

Les modèles sont chargés une fois par processus (mémoire) et l'inférence est sérialisée par un verrou : avec une capacité supérieure à 1, les tâches vocales ne se disputent pas les 12 Go de VRAM.

## Dépendances Python (verrouillées)

Elles sont dans l'extra `voice` de `worker/pyproject.toml` (l'ancienne image CPU et `runtime.lock` ont été supprimés : l'image `worker.Dockerfile` est l'unique image du worker ; son nom n'indique plus GPU, mais elle exige un GPU NVIDIA pour des performances raisonnables, `SNITCH_VOICE_DEVICE=cpu` restant possible mais lent et non mesuré) : `pyannote.audio==4.0.7` (série 4, requise par `community-1` ; PyPI : `torch>=2.8.0`, `torchaudio>=2.8.0`, `torchcodec>=0.7.0`), `torch==2.8.0`, `torchaudio==2.8.0`, `soundfile==0.14.0` (sans lui, `torchaudio.load` de torchaudio 2.8 n'a aucun backend dans l'image : constaté au premier build), `torchcodec==0.7.0` (couple torchcodec 0.7 / torch 2.8 de la matrice de compatibilité de torchcodec ; à reconfirmer). `worker/uv.lock` les résout ; `worker/voice-runtime.lock` en est l'export complet avec empreintes SHA-256 (`cd worker && uv export --extra voice --no-dev --no-emit-project --no-emit-package snitch-client --format requirements-txt -o voice-runtime.lock`), installé avec `--require-hashes` par `deploy/docker/worker.Dockerfile`. Les roues PyPI de torch 2.8.0 pour Linux x86_64 sont liées à CUDA 12.8 et tirent cuDNN 9.10 et les bibliothèques CUDA par des paquets `nvidia-*-cu12` : l'image reste une base `python:3.12-slim` épinglée par condensé, sans image `nvidia/cuda`. Le pilote de l'hôte doit gérer CUDA 12.8 (série 570 ou plus récente d'après la documentation NVIDIA ; à confirmer avec `nvidia-smi`). `huggingface_hub` (résolu en 2.3.0) n'est utilisé que par le sous-processus de téléchargement d'un ordre d'installation (`snitch_worker.voice_download`).

Licences : voir `deploy/NOTICES.md` et `deploy/licenses/voice/LICENSES.md` (les roues NVIDIA sont sous licence propriétaire NVIDIA ; les modèles sont en CC-BY-4.0 avec conditions d'accès Hugging Face).

## Un Worker neuf n'a aucun modèle

Un Worker s'installe (commande unique, `compose.yaml` généré ou `install.py`, Linux ou WSL2) sans télécharger aucun poids et sans poser de question sur les modèles vocaux : plus de « Download the voice models? », plus de conditions à confirmer ni de jeton à saisir à l'installation, plus de `SNITCH_VOICE_DOWNLOAD_ON_START`, `SNITCH_VOICE_CONFIRM_TERMS` ni `HF_TOKEN` dans les fichiers générés (ADR 0038). La transcription via Speaches fonctionne tout de suite ; la diarisation, l'empreinte et la séparation attendent un modèle installé.

Les modèles s'installent depuis Snitch (Administration, Voix) : Snitch enregistre un ordre, le Worker le tire sur sa requête sortante habituelle, le contrôle (destinataire, identité, type, moteur, dépôt, révision épinglée, idempotence), l'accuse, puis télécharge, charge et fait tourner une inférence synthétique avant de déclarer le modèle prêt. L'état des ordres (`orders.json`) et la preuve de vérification (`.snitch-ready`) vivent dans le volume : un redémarrage ou une reconnexion ne retélécharge rien et ne réexécute que les ordres que Snitch tient encore ouverts, déjà validés. Tant que le modèle actif d'un type n'est pas prêt sur un Worker, Snitch ne lui attribue aucun travail vocal local de ce type.

Mises à jour : `pull_policy: always` vérifie l'image à chaque `docker compose up`. Pour une installation faite par la commande Snitch, la version se choisit dans **Admin > Mises à jour** (ADR 0036) et l'agent local applique le changement dans les 10 minutes. L'image par défaut est `ghcr.io/thomas-bsn/snitch-worker:latest` ; pour rester compatible avec l'application, épingler `SNITCH_WORKER_IMAGE=ghcr.io/thomas-bsn/snitch-worker:vX.Y.Z` (version de Snitch déployée) et la changer avec l'application. Deux versions de modèle produisent des empreintes incomparables.

Le volume de modèles est inscriptible : c'est là que les ordres d'installation téléchargent et conservent les modèles (`catalog/`). Non vérifié ici : le téléchargement réel (aucun jeton ni réseau Hugging Face) et le droit d'écriture de l'utilisateur 10001 sur un montage hôte.

## Jeton Hugging Face : installer d'autres modèles depuis Snitch

Un ordre d'installation fait télécharger le modèle par le Worker. Le téléchargement lit le jeton dans un fichier privé du volume des modèles, `/models/.secrets/hf_token` (dossier `0700`, fichier `0600`), que seul le sous-processus de téléchargement ouvre : le jeton ne passe jamais par Snitch, ni par l'API, ni par l'environnement du processus d'inférence. Le fichier vit dans le volume : supprimer le Worker dans Snitch le vide avec le reste.

- **Depuis Snitch** : le jeton central (Administration, Voix) est transmis avec l'ordre d'un téléchargement qui en a besoin, et seulement alors ; le Worker l'écrit dans ce fichier, sans jamais remplacer un jeton local.
- **Compose ou `.env`** : `HF_TOKEN` n'est plus lu par personne ; ne le mettez pas dans le `.env`.
- **Worker déjà installé, sans fichier** : l'écran affiche « Jeton Hugging Face absent sur ce Worker ». Enregistrez-le une fois, depuis la machine du Worker (bash), en le passant sur l'entrée standard (jamais en argument ni en variable). Le nom de l'image s'obtient avec `docker inspect -f '{{.Config.Image}}' snitch-worker-<id du Worker>` :

```bash
read -rs HF && printf '%s' "$HF" | docker run -i --rm --user 10001:10001 \
  -v snitch-models-<id du Worker>:/models --entrypoint python <image du Worker> \
  -m snitch_worker.bootstrap store-token; unset HF
```

La commande affiche `stored`, ou `cause=invalid_token` si la valeur n'a pas la forme d'un jeton (`hf_…`), sans jamais le répéter. Un jeton refusé ne remplace pas celui déjà gardé. Le jeton doit avoir accès aux dépôts du modèle choisi (conditions acceptées sur Hugging Face).

## Installer sur le PC gamer

Cible : un PC Linux (ou Windows avec WSL2 et Docker) avec une carte NVIDIA, par exemple une RTX 4070 (12 Go). Le worker ne publie aucun port et appelle l'API Snitch en HTTPS sortant ; les secrets restent sur ce PC. Les fichiers sont dans `deploy/worker/` (`compose.yaml`, `.env.example`).

### Prérequis communs

1. **Pilote NVIDIA récent** (`nvidia-smi` fonctionne, CUDA 12.8 ou plus), Docker avec Compose v2 et le [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) configuré pour Docker. Vérifier l'accès GPU : `docker run --rm --gpus all ubuntu nvidia-smi`.
2. **Créer le worker dans l'interface Snitch** (page des workers, administrateur) pour obtenir le jeton, affiché une seule fois. Snitch doit être joignable en HTTPS depuis le PC.

### Sur Linux (chemin recommandé)

3. **Préparer `.env`.** `cd deploy/worker && cp .env.example .env`, puis renseigner `SNITCH_API_URL`, `SNITCH_WORKER_TOKEN`, `TRANSCRIPTION_API_URL` (et `TRANSCRIPTION_API_KEY` si besoin), `SNITCH_MODELS_DIR` (dossier du PC qui recevra les modèles) (laisser `SNITCH_VOICE_EMBEDDING_MODEL_VERSION` vide). Ne jamais versionner `.env`.
4. **Rendre le dossier des modèles inscriptible par l'utilisateur 10001 du conteneur** :
   ```sh
   mkdir -p "$SNITCH_MODELS_DIR"
   sudo chown 10001:10001 "$SNITCH_MODELS_DIR"
   ```
   (ou `chmod 777 "$SNITCH_MODELS_DIR"` si vous préférez éviter `sudo`).
5. **Lancer.** `docker compose up -d` dans `deploy/worker`, puis `docker compose logs -f worker`. Le worker démarre sans aucun modèle vocal et ne télécharge rien : installer ensuite les modèles depuis Snitch (Administration, Voix).

### Sur Windows / WSL2

Le montage d'un dossier hôte vers le conteneur sous WSL2 ne propage pas toujours les permissions Linux attendues par l'utilisateur 10001 du conteneur, et un ordre d'installation échouerait alors avec `models_volume_unwritable`.

3. **Préparer `.env`.** `cd deploy/worker && cp .env.example .env`, puis renseigner `SNITCH_API_URL`, `SNITCH_WORKER_TOKEN`, `TRANSCRIPTION_API_URL` (et `TRANSCRIPTION_API_KEY` si besoin), `SNITCH_MODELS_DIR` (chemin WSL2 sous `/home/...`, ex : `/home/user/snitch-models`, pas `/mnt/c/...`). Rien à régler pour les modèles : un Worker neuf n'en télécharge aucun, y compris sous WSL2.
4. **Rendre le dossier inscriptible par l'utilisateur 10001, dans le système de fichiers Linux de WSL2** :
   ```sh
   mkdir -p "$SNITCH_MODELS_DIR"
   sudo chown 10001:10001 "$SNITCH_MODELS_DIR"
   ```
5. **Lancer.** `docker compose up -d` dans `deploy/worker`, puis `docker compose logs -f worker`. Le worker démarre sans modèle vocal, hors ligne et sans `HF_TOKEN`. Les modèles s'installent ensuite depuis Snitch (Administration, Voix), qui donne la cause stable d'un échec (dossier non inscriptible, réseau, accès Hugging Face).

### Étapes communes (Linux et Windows/WSL2)

7. **Vérifier dans Snitch.** Le worker apparaît actif sur la page des workers avec ses capacités (`transcription.*`, `voice.*`). La fiche indique la machine déclarée ; pour remplacer le nom du conteneur par celui du PC, renseigner `SNITCH_WORKER_MACHINE_NAME` dans `.env`. Le système affiché est celui du conteneur. Un échec `provider_unavailable` renvoie au code de raison stable des journaux (`model_path_missing`, `cuda_unavailable`, `model_load_failed`...).
8. **Brancher la transcription (Whisper).** Un serveur compatible `POST /v1/audio/transcriptions` : sur le PC hors Docker, `TRANSCRIPTION_API_URL=http://host.docker.internal:<port>` (le compose ajoute `host.docker.internal:host-gateway`) ; dans un autre projet Compose, activer l'option de réseau partagé du compose et viser le nom du service. Le profil de bibliothèque définit le modèle et la langue. Si le serveur Whisper utilise un autre nom pour le même modèle, renseigner ce nom pour le Worker dans Administration › Transcription ; les résultats restent étiquetés avec le nom du profil.
9. **Embeddings de texte (Ollama).** Ils restent côté NAS (`EMBEDDINGS_URL` et `EMBEDDINGS_MODEL` de l'application, voir [déploiement](deployment.md)) et ne passent pas par le worker. Un Ollama sur le PC peut servir de fournisseur si le NAS peut le joindre (réseau privé de votre choix).
10. **Configurer le pipeline vocal** dans l'interface (fournisseur `pyannote`, modèle `pyannote/speaker-diarization-community-1`, une fois installé sur le Worker) (le modèle se choisit dans le catalogue, sans case de confirmation) : l'état réel du Worker remplace la confirmation de l'administrateur que les conditions ont été acceptées.
6. **Vérifier dans Snitch.** Le worker apparaît actif sur la page des workers avec ses capacités (`transcription.*`, `voice.*`). La fiche indique la machine déclarée ; pour remplacer le nom du conteneur par celui du PC, renseigner `SNITCH_WORKER_MACHINE_NAME` dans `.env`. Le système affiché est celui du conteneur. Un échec `provider_unavailable` renvoie au code de raison stable des journaux (`model_path_missing`, `cuda_unavailable`, `model_load_failed`...).
7. **Brancher la transcription (Whisper).** Un serveur compatible `POST /v1/audio/transcriptions` : sur le PC hors Docker, `TRANSCRIPTION_API_URL=http://host.docker.internal:<port>` (le compose ajoute `host.docker.internal:host-gateway`) ; dans un autre projet Compose, activer l'option de réseau partagé du compose et viser le nom du service. Le profil de bibliothèque définit le modèle et la langue. Si le serveur Whisper utilise un autre nom pour le même modèle, renseigner ce nom pour le Worker dans Administration › Transcription ; les résultats restent étiquetés avec le nom du profil.
8. **Embeddings de texte (Ollama).** Ils restent côté NAS (`EMBEDDINGS_URL` et `EMBEDDINGS_MODEL` de l'application, voir [déploiement](deployment.md)) et ne passent pas par le worker. Un Ollama sur le PC peut servir de fournisseur si le NAS peut le joindre (réseau privé de votre choix).
9. **Configurer le pipeline vocal** dans l'interface (fournisseur `pyannote`, modèle `pyannote/speaker-diarization-community-1`, une fois installé sur le Worker) (le modèle se choisit dans le catalogue, sans case de confirmation) : l'état réel du Worker remplace la confirmation de l'administrateur que les conditions ont été acceptées.

### VRAM : 12 Go partagés

Un Ollama chargé avec un gros modèle peut occuper l'essentiel des 12 Go. Garder `SNITCH_WORKER_CAPACITY=1`, ne pas faire tourner en même temps une transcription Whisper sur GPU, la diarisation et un grand modèle Ollama (`OLLAMA_KEEP_ALIVE` court, ou arrêter Ollama pendant les traitements vocaux). Une mémoire insuffisante fait échouer la tâche avec `provider_unavailable` (relançable). `SNITCH_VOICE_DEVICE=cpu` est un repli très lent. La consommation réelle n'a pas été mesurée (aucun GPU ici).

### Test avec 10 vidéos

Importer 10 vidéos choisies par vous par l'interface d'import de Snitch (aucun scan de dossier), les transcrire, puis lancer « Diariser les voix » clip par clip. AGENTS.md interdit tout média réel avant l'autorisation du benchmark de la phase 7 : cette autorisation relève de l'utilisateur, jamais d'un agent ni de la CI. Relever durée, VRAM (`nvidia-smi`) et erreurs ; ce n'est pas le benchmark de la phase 7.

### Vérifié et non vérifié

Construit et contrôlé sans GPU (voir le Dockerfile) : installation avec empreintes, `pip check`, import de torch/torchaudio/pyannote.audio, `torch.version.cuda == 12.8`, décodage WAV 16 kHz par torchaudio. Non vérifié : tout accès GPU réel, `nvidia-smi` dans le conteneur, chargement des modèles, le téléchargement d'un ordre réel, la compatibilité de `huggingface_hub` 2.3.0 avec pyannote.audio 4.0.7 au chargement local, l'hôte Windows/WSL2.

## Limite : jamais exécuté de bout en bout ici

Le code réel (`_load_pipeline`, `_load_embedder`, chargements de `pyannote.audio`, `torchaudio.load`, `Inference.crop`) n'a pas été exécuté : pas de GPU, pas de modèle, pas de jeton Hugging Face, aucun média réel dans cet environnement. Les tests (`worker/tests/test_voice_real.py`) injectent un pipeline et un embedder simulés et vérifient la sélection, la correspondance vers le format des résultats, la qualité, les tours courts, la limite de 400, les erreurs expurgées, le chargement unique, le mode hors ligne forcé et l'absence d'import de `torch`/`pyannote` au démarrage. Les points suivants reposent sur la documentation pyannote et sont à vérifier à la première exécution réelle : chargement d'un pipeline depuis un répertoire local (`Pipeline.from_pretrained(<config.yaml>)` avec des chemins relatifs), sortie `DiarizeOutput.speaker_diarization` (4.x) ou `Annotation` (3.x), chargement d'un `Model` depuis un chemin local, `Inference(window="whole").crop` sur un dictionnaire `waveform`/`sample_rate`.

## Reconnexion après un redéploiement de Snitch

Le Worker conserve son jeton et réessaie automatiquement quand Snitch ou son mandataire redémarre, y compris si le mandataire répond temporairement avec une page HTML « Bad Gateway » ou un JSON qui n’est pas celui de l’API. La prise des nouveaux jobs reprend au prochain sondage ; la présence et le renouvellement du lease restent indépendants, avec un délai de reprise plafonné à 10 secondes. Un lease expiré ou refusé ne permet jamais de publier un résultat périmé.

Depuis #321, aucune erreur récupérable n’arrête la boucle qui réclame les jobs : socket réinitialisé, délai dépassé, page 502/503, JSON illisible par le client généré ou erreur inattendue. Elle réessaie avec une attente qui double depuis l’intervalle de sondage, plafonnée à 10 secondes, sans redémarrage du conteneur ni nouvelle identité. Si un thread de claim s’arrête malgré tout, le Worker l’écrit (`claim_loop_stopped`) et le relance lui-même, sans watchdog externe. Un pouls refusé n’empêche plus de réclamer : la présence garde son propre thread et ses propres logs.

Les échecs sont visibles dans les logs du Worker, sans jeton ni URL : `claim_poll_failed` donne l’étape logique (`pulse`, `transcription_config`, `claim`, `job`), la catégorie (`network`, `timeout`, `http_status`, `invalid_response`, `api_error`, `unauthorized`, `internal`), le statut HTTP ou le code d’erreur de Snitch s’il est connu, le nombre d’essais et le temps écoulé depuis le dernier succès. La première erreur est écrite, puis au plus une ligne par minute ; `claim_poll_recovered` signale la reprise et sa durée.

Côté Snitch, être en ligne ne veut pas dire réclamer des jobs. Chaque claim est daté (`workers.last_claim_at`). Un Worker en ligne et libre qui n’a rien réclamé depuis 2 minutes s’affiche « En ligne, ne réclame plus de tâches » sur la page Workers. Les tâches qu’il pourrait prendre affichent la raison `worker_not_claiming`, au lieu de « Worker occupé » ou « Démarrer un Worker », et un incident administrateur `job_blocked` s’ouvre une fois le délai de grâce passé. Un Worker occupé à pleine capacité par un long traitement n’est jamais signalé, ni un Worker qui revient d’une absence avant son premier claim. Le diagnostic de la page du Worker s’ouvre alors aussi, avec les commandes pour lire ses journaux. Décision : [ADR 0045](adr/0045-boucle-de-claim-supervisee-et-distincte-de-la-presence.md).

Ce correctif doit être présent dans **l’image du Worker**, pas seulement dans l’application. Mettre à jour une fois les Workers existants pour en bénéficier ; les redéploiements suivants de Snitch ne nécessitent plus leur redémarrage pour ce cas. Un changement de jeton révoqué ou d’adresse du serveur reste une modification de configuration distincte.
