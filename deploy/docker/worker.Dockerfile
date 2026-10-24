# Squelette de contrat phase 0 ; CUDA et exécuteur ajoutés en phase 2.
FROM python:3.12.15-slim@sha256:a0f57064cbdcd6285fa45f51237ab0b35ea8ebaed06a1f5973e89e19dfb86704
WORKDIR /worker
COPY worker/runtime.lock worker/build.lock /locks/
RUN pip install --no-cache-dir --no-deps -r /locks/build.lock -r /locks/runtime.lock
COPY worker/receipts_worker/api /api
RUN pip install --no-cache-dir --no-deps --no-build-isolation /api && pip check
COPY scripts/collect_python_notices.py /collect_python_notices.py
RUN python /collect_python_notices.py
COPY worker/receipts_worker /worker/receipts_worker
COPY LICENSE /usr/share/licenses/receipts/LICENSE
USER 10001:10001
CMD ["python", "-c", "print('Receipts worker: contrat HTTP prêt ; exécution disponible en phase 2')"]
