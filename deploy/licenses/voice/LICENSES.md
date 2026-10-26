# Licences de l'image worker GPU (voix)

Ce document complète `deploy/NOTICES.md` pour `deploy/docker/worker-gpu.Dockerfile`. Il est rédigé à partir des fiches officielles lues le 2026-10-26 (PyPI, Hugging Face) ; il n'est pas un avis juridique. Les textes complets des licences des paquets Python installés sont copiés dans l'image sous `/usr/share/licenses/snitch/python` (le build échoue si un paquet n'en fournit aucun).

## Bibliothèques Python (dans l'image)

| Composant | Version | Licence | Remarque |
| --- | --- | --- | --- |
| pyannote.audio | 4.0.7 | MIT (confirmé : le texte `LICENSE` « MIT License, Copyright (c) 2020 CNRS » de la roue est copié dans l'image ; la roue embarque aussi `LICENSE.WeSpeaker`) | Dépend de pyannote-core, pyannote-database, pyannote-metrics, pyannote-pipeline, pyannoteai-sdk, lightning, torchmetrics, matplotlib, etc. |
| torch | 2.8.0 | BSD-3-Clause (métadonnée PyPI) | Roue Linux x86_64 liée à CUDA 12.8. |
| torchaudio, torchcodec | 2.8.0, 0.7.0 | BSD (projets PyTorch) | torchcodec décode avec les bibliothèques FFmpeg du système (Debian, voir NOTICES). |
| soundfile | 0.14.0 | BSD-3-Clause (PyPI) | Backend de `torchaudio.load`, sans lequel le chargement WAV échoue (constaté). |
| primePy | 1.3 | MIT (classifieur PyPI et dépôt amont) | La roue ne contient aucun fichier de licence : le texte amont est dans `deploy/licenses/voice/extra/primePy/LICENSE` et copié dans l'image par `collect_python_notices.py` (répertoire `/licenses-extra`). |
| Autres | voir `worker/voice-runtime.lock` | Variées (MIT, BSD, Apache-2.0, MPL-2.0, PSF...) | Fermeture complète avec empreintes SHA-256. Non relue une par une ici : les textes sont copiés dans l'image. |

## CUDA, cuDNN et bibliothèques NVIDIA (dans l'image)

Les roues `nvidia-*-cu12` (cuda-runtime, cuda-nvrtc, cuda-cupti, cublas, cufft, curand, cusolver, cusparse, cusparselt, cudnn 9.10.2.21, nccl, nvjitlink, nvtx, cufile) et `triton` sont tirées par torch. PyPI les déclare sous une licence propriétaire NVIDIA (`LicenseRef-NVIDIA-Proprietary`, classifieur « Other/Proprietary »). Ce ne sont donc PAS des composants MIT/BSD et l'image ne doit pas être décrite comme telle. Les conditions exactes (CUDA Toolkit EULA, cuDNN SLA, attribution et liste des composants redistribuables) sont portées par le fichier de licence de chaque roue, copié dans l'image par `collect_python_notices.py` ; le texte copié pour cuDNN est le « LICENSE AGREEMENT FOR NVIDIA SOFTWARE DEVELOPMENT KITS » (accord à accepter, réservé aux adultes ou représentants habilités) ; je n'en ai lu que le début, et je n'ai pas pu vérifier la compatibilité avec une redistribution publique de l'image. Avant toute publication de l'image GPU (le workflow ne la publie pas), relire ces textes et le CUDA Toolkit EULA. Triton est sous licence MIT (à confirmer dans le texte copié).

Le pilote NVIDIA et le NVIDIA Container Toolkit ne sont pas dans l'image : ils restent ceux du PC.

## Modèles (hors image, volume monté)

| Modèle | Licence affichée | Accès |
| --- | --- | --- |
| `pyannote/speaker-diarization-community-1` | CC-BY-4.0 | Dépôt « gated » : il faut un compte Hugging Face, accepter les conditions sur la page du modèle (partage d'informations de compte, e-mails occasionnels possibles) et fournir un jeton. |
| `pyannote/wespeaker-voxceleb-resnet34-LM` | CC-BY-4.0 ; la fiche indique que le modèle suit la licence de son jeu de données (VoxCeleb) | À la lecture de la fiche, aucun accès restreint n'est mentionné ; non confirmé de façon authentifiée. `download_models.py` exige quand même `--confirm-terms` pour les deux : vérifier la page au moment du téléchargement et accepter ce qui est demandé. |

CC-BY-4.0 impose l'attribution (pyannote, WeSpeaker). Les poids ne sont ni dans le dépôt ni dans l'image : chaque administrateur les télécharge lui-même après avoir accepté les conditions. Les conditions d'usage de VoxCeleb pour les poids issus de ce jeu de données n'ont pas pu être vérifiées en détail : à relire avant un usage au-delà du personnel.
