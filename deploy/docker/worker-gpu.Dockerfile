# Worker avec fournisseurs vocaux réels (pyannote) et GPU NVIDIA. Client HTTPS autonome ;
# aucun port entrant ni accès PostgreSQL. Aucun modèle n'est inclus ni téléchargé : les
# modèles viennent d'un volume monté (SNITCH_VOICE_*), voir docs/real-voice-worker.md.
# CUDA 12.8 et cuDNN 9 viennent des roues PyPI `nvidia-*` dépendantes de torch 2.8.0 : pas
# d'image de base nvidia/cuda. Le pilote NVIDIA (>= 570 pour CUDA 12.8) et le NVIDIA
# Container Toolkit restent ceux de l'hôte. L'image CPU reste deploy/docker/worker.Dockerfile.
FROM python:3.12.15-slim@sha256:a0f57064cbdcd6285fa45f51237ab0b35ea8ebaed06a1f5973e89e19dfb86704
WORKDIR /worker
ENV NVIDIA_VISIBLE_DEVICES=all \
    NVIDIA_DRIVER_CAPABILITIES=compute,utility \
    HF_HUB_OFFLINE=1 \
    TRANSFORMERS_OFFLINE=1
RUN apt-get update \
 && apt-get install --no-install-recommends -y ffmpeg=7:7.1.5-0+deb13u1 \
 && mkdir -p /usr/share/licenses/snitch/third-party/ffmpeg-debian \
 && cp /usr/share/doc/ffmpeg/copyright /usr/share/licenses/snitch/third-party/ffmpeg-debian/copyright \
 && rm -rf /var/lib/apt/lists/*
RUN ffmpeg -hide_banner -loglevel error -f lavfi -i color=c=black:s=64x64:d=1 -f lavfi -i sine=frequency=440:duration=1 -f lavfi -i sine=frequency=880:duration=1 -map 0:v -map 1:a -map 2:a -c:v mpeg4 -c:a aac -shortest -y /tmp/snitch-multitrack.mp4 \
 && ffmpeg -hide_banner -loglevel error -i /tmp/snitch-multitrack.mp4 -map 0:2 -vn -ac 1 -ar 16000 -c:a pcm_s16le -y /tmp/snitch-selected.wav \
 && test "$(ffprobe -v error -select_streams a:0 -show_entries stream=sample_rate,channels -of csv=p=0 /tmp/snitch-selected.wav)" = "16000,1" \
 && rm /tmp/snitch-multitrack.mp4 /tmp/snitch-selected.wav
COPY worker/voice-runtime.lock worker/build.lock /locks/
# voice-runtime.lock : fermeture complète (uv export de l'extra `voice`) avec empreintes SHA-256.
RUN pip install --no-cache-dir --no-deps -r /locks/build.lock \
 && pip install --no-cache-dir --no-deps --require-hashes -r /locks/voice-runtime.lock
COPY worker/snitch_worker/api /api
RUN pip install --no-cache-dir --no-deps --no-build-isolation /api && pip check
# Contrôle de construction sans GPU ni modèle : appariement torch/CUDA et décodage WAV 16 kHz.
RUN ffmpeg -hide_banner -loglevel error -f lavfi -i sine=frequency=440:duration=1 -ac 1 -ar 16000 -c:a pcm_s16le -y /tmp/probe.wav \
 && python -c "import torch, torchaudio, pyannote.audio; assert torch.__version__.startswith('2.8.0') and torch.version.cuda == '12.8', (torch.__version__, torch.version.cuda); w, sr = torchaudio.load('/tmp/probe.wav'); assert sr == 16000 and w.shape[0] == 1" \
 && rm /tmp/probe.wav
COPY scripts/collect_python_notices.py /collect_python_notices.py
RUN python /collect_python_notices.py
COPY worker/snitch_worker /worker/snitch_worker
COPY worker/scripts/download_models.py /worker/scripts/download_models.py
COPY LICENSE /usr/share/licenses/snitch/LICENSE
USER 10001:10001
CMD ["python", "-m", "snitch_worker.runner"]
