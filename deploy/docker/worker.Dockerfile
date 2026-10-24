# Client HTTPS autonome ; aucun port entrant ni accès PostgreSQL.
FROM python:3.12.15-slim@sha256:a0f57064cbdcd6285fa45f51237ab0b35ea8ebaed06a1f5973e89e19dfb86704
WORKDIR /worker
RUN apt-get update \
 && apt-get install --no-install-recommends -y ffmpeg=7:7.1.5-0+deb13u1 \
 && mkdir -p /usr/share/licenses/receipts/third-party/ffmpeg-debian \
 && cp /usr/share/doc/ffmpeg/copyright /usr/share/licenses/receipts/third-party/ffmpeg-debian/copyright \
 && rm -rf /var/lib/apt/lists/*
RUN ffmpeg -hide_banner -loglevel error -f lavfi -i color=c=black:s=64x64:d=1 -f lavfi -i sine=frequency=440:duration=1 -f lavfi -i sine=frequency=880:duration=1 -map 0:v -map 1:a -map 2:a -c:v mpeg4 -c:a aac -shortest -y /tmp/receipts-multitrack.mp4 \
 && ffmpeg -hide_banner -loglevel error -i /tmp/receipts-multitrack.mp4 -map 0:2 -vn -ac 1 -ar 16000 -c:a pcm_s16le -y /tmp/receipts-selected.wav \
 && test "$(ffprobe -v error -select_streams a:0 -show_entries stream=sample_rate,channels -of csv=p=0 /tmp/receipts-selected.wav)" = "16000,1" \
 && rm /tmp/receipts-multitrack.mp4 /tmp/receipts-selected.wav
COPY worker/runtime.lock worker/build.lock /locks/
RUN pip install --no-cache-dir --no-deps -r /locks/build.lock -r /locks/runtime.lock
COPY worker/receipts_worker/api /api
RUN pip install --no-cache-dir --no-deps --no-build-isolation /api && pip check
COPY scripts/collect_python_notices.py /collect_python_notices.py
RUN python /collect_python_notices.py
COPY worker/receipts_worker /worker/receipts_worker
COPY LICENSE /usr/share/licenses/receipts/LICENSE
USER 10001:10001
CMD ["python", "-m", "receipts_worker.runner"]
