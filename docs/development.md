# Développement et vérifications

Go 1.27.3, Node 24.21.0/npm, Python 3.12.15 et Docker Compose v2. Installer `npm --prefix web ci`, créer `.venv` puis `.venv/bin/pip install -r contracts/generators/requirements.lock`. Installer Chromium avec `cd web && npx playwright install chromium` (Linux : `--with-deps`).

```sh
npm --prefix web run lint
npm --prefix web run build
(cd app && go vet ./...)
SNITCH_E2E=1 bash scripts/test.sh
.venv/bin/python scripts/check_generated.py
.venv/bin/python scripts/check_compose.py
docker build -f deploy/docker/app.Dockerfile -t snitch-smoke:local .
docker build -f deploy/docker/worker.Dockerfile -t snitch-worker:local .
docker run --rm snitch-worker:local python -m pip check   # image unique du worker, sans GPU requis pour la construire (~9 Go)
python3 scripts/smoke_compose.py snitch-smoke:local
```

`test -z "$(gofmt -l app)"` vérifie le format Go. Le script lance `go test -race -tags integration ./... -count=1` avec le socket du contexte Docker actif. Les tests utilisent PostgreSQL/pgvector réel via Testcontainers et des données synthétiques ; les parcours navigateur utilisent la vraie API avec Discord simulé, sans compte externe. Aucun scan de bibliothèque n’existe. Ne pas lancer simultanément gros builds et tests sur une VM de 2 Go.

Construire le frontend avec `npm --prefix web run build` avant les tests navigateur : le serveur Go embarque `app/ui/dist` à la compilation et `scripts/test.sh` ne reconstruit pas ces fichiers. Sinon les tests peuvent exercer une ancienne version de la SPA. Le test complet exécute onze suites navigateur.

Le contrat source `contracts/openapi.yaml` génère Go/TypeScript/Python et sqlc avec `bash scripts/generate.sh`. `check_generated.py` compare les octets avant/après. Le build Vite remplace `app/ui/dist` ; préserver son fichier suivi `.gitkeep` avant commit. Les sorties compilées restent ignorées.

Le smoke Compose crée trois projets et volumes uniques puis les supprime, même en échec. Il démarre sans `POSTGRES_PASSWORD` ni `MASTER_KEY` (le fichier `.env` local est ignoré) et vérifie que la clé maître est générée, conservée après un second démarrage, illisible par le service de base, exportable, absente d’un dump de la base et des journaux, puis que sa suppression laisse l’application démarrer en listant les secrets à ressaisir (et que les ressaisir fonctionne) ; il vérifie aussi que le mot de passe est produit, imposé par la base, conservé après un second démarrage et jamais publié ni journalisé, puis qu’un mot de passe explicite contenant des caractères réservés est respecté, redémarre sans perte et survit au retrait de la variable. Il teste aussi l’installation via HTTP, SPA embarquée, absence de modèles, disponibilité sans embed, persistance après redémarrage, SIGTERM et blocage de l’API si migrate échoue. Les logs et identifiants temporaires ne sont jamais imprimés. Un troisième argument écrit les mesures locales sans secrets : `python3 scripts/smoke_compose.py snitch-smoke:local /tmp/footprint.json`.

La CI ajoute aux contrôles de source les builds app/worker avant de permettre la publication. Les dépendances sont verrouillées, les notices complètes sont embarquées ; voir `deploy/NOTICES.md`. La reproductibilité des artefacts générés est distincte de celle des octets d’images, non garantie.
