# Receipts

Socle Go et interface française embarquée pour retrouver, dans les prochaines phases, les paroles et les voix de clips de jeu. La phase 0 est en cours de vérification : les tickets #2 et #3 couvrent le démarrage vierge, le premier administrateur et les sessions locales. La configuration facultative de Discord et ses secrets sont couverts par le ticket #5 ; les échanges de connexion Discord seront validés au ticket #6. Aucune bibliothèque ni aucun traitement de média n'est disponible.

## Démarrer le socle en développement

Prérequis : Go 1.27.3, Node 24 avec npm (compilation uniquement), Docker pour PostgreSQL/pgvector et les tests. Le binaire de production ne lance aucun processus Node. Exécuter les commandes depuis la racine du dépôt.

```sh
npm --prefix web ci
npm --prefix web run build
mkdir -p bin
(cd app && go build -o ../bin/receipts ./cmd/receipts)

docker run --name receipts-dev-db --detach \
  --env POSTGRES_USER=receipts --env POSTGRES_DB=receipts \
  --env POSTGRES_PASSWORD=synthetic-development-only \
  --publish 127.0.0.1:5432:5432 \
  pgvector/pgvector:0.8.7-pg18-bookworm@sha256:2358fcba361ed2233a5ed81b5fe4ca779ccb304120ce531a3bf51c0ed7e2bc11
# Attendre que cette commande réussisse avant de migrer :
docker exec receipts-dev-db pg_isready -U receipts -d receipts

export DATABASE_URL=postgres://receipts:synthetic-development-only@localhost:5432/receipts?sslmode=disable
export MASTER_KEY="$(openssl rand -base64 32)"
export PUBLIC_URL=http://localhost:8080
export COOKIE_SECURE=false
export DISPLAY_NAME=Receipts
./bin/receipts migrate
./bin/receipts serve
```

Ouvrir <http://localhost:8080>. L'écran annonce une nouvelle installation et les étapes reportées. Le code d'installation temporaire est la seule exception aux logs expurgés ; il expire après 15 minutes. Saisir ce code, un nom de 3 à 64 caractères et un mot de passe personnel de 12 à 256 caractères dans le wizard, puis créer le premier administrateur. Discord est facultatif : aucun fournisseur externe n’est nécessaire. Aucun compte par défaut n'est créé. Arrêter avec Ctrl+C ou SIGTERM. Les données du conteneur de développement sont jetables ; ne jamais utiliser cet exemple pour des données réelles.

`migrate` n'exige que `DATABASE_URL`. La commande est rejouable et ne réapplique pas les migrations réussies ; un échec rend un code de sortie non nul. `serve` ne lance aucune migration et refuse une base vierge non préparée. Dans le déploiement orchestré, le job de migration devra réussir avant le démarrage de l'API (ticket #7).

## Installation et sessions locales

Le code d’installation apparaît dans le log JSON `one-time installation code` (champ `expires_in_minutes: 15`) au démarrage de `serve`. Il est utilisable une fois pendant 15 minutes ; ni la page ni l’API publique ne le renvoient. Aucun administrateur ni mot de passe par défaut n’est précréé. Une seule création réussit même si deux demandes arrivent ensemble. Après réussite, l’installation reste fermée après redémarrage ; utiliser ensuite « Se connecter » avec les identifiants personnels et « Se déconnecter » pour révoquer la session courante.

Si le navigateur ou le serveur s’interrompt **avant** la création, retrouver le code dans les logs du premier démarrage : un redémarrage conserve tout code encore valide et ne le réaffiche pas. Si ce code a été perdu ou a expiré, attendre son expiration puis redémarrer `serve` : un nouveau code est émis tant que l’installation n’est pas terminée. Le précédent est alors refusé. Une interruption **après** création ne rouvre jamais le wizard ; se reconnecter avec le compte créé. Aucun changement SQL manuel n’est nécessaire.

Les mots de passe utilisent Argon2id version 19 : mémoire 32 MiB, trois itérations, parallélisme 1, sel aléatoire de 16 octets et résultat de 32 octets. Les limites de saisie comptent les caractères Unicode côté serveur. Les jetons opaques de session contiennent 32 octets aléatoires ; seul leur SHA-256 est persisté. Une session expire après sept jours fixes, sans prolongation automatique. Déconnexion et expiration rendent le jeton inutilisable, y compris s’il est renvoyé directement à l’API.

Le cookie `receipts_session` utilise `Path=/`, `HttpOnly`, `SameSite=Lax` et une durée maximale de 604800 secondes. `Secure` est activé par défaut ; conserver HTTPS et `COOKIE_SECURE=true` en production. Chaque mutation doit transmettre `Origin` égal à `PUBLIC_URL`. Les mutations authentifiées exigent aussi `X-CSRF-Token`, obtenu avec `GET /api/auth/me`. Les fonctions privées exigent une session approuvée et les routes `/api/admin/…` le rôle administrateur ; masquer un écran n’est jamais la protection d’accès.

L’installation, la connexion et l’inscription partagent une limite par adresse réseau distante : rafale de dix demandes, puis une demande toutes les six secondes ; `429` annonce `Retry-After: 6`. Deux traitements de mots de passe au plus s’exécutent simultanément par processus ; un excès renvoie `429 busy`. Les en-têtes de proxy transmis par le client sont ignorés : derrière un proxy, les clients partageant son adresse partagent actuellement cette limite. Les refus de connexion sont identiques pour compte inexistant, mot de passe incorrect et compte en attente. Les secrets saisis ne sont ni journalisés ni renvoyés.

## Discord facultatif et sauvegarde des secrets

Après la création de l’administrateur, « Accès et Discord » permet de continuer sans Discord et de conserver une installation entièrement locale. Seuls les administrateurs peuvent consulter et modifier ces réglages. Pour préparer la connexion :

1. Créer votre propre application dans le [Discord Developer Portal](https://discord.com/developers/applications). Aucun identifiant du propriétaire de ce dépôt n’est nécessaire.
2. Dans les réglages OAuth2 de cette application, récupérer le **Client ID** et le **Client Secret**, puis ajouter l’URL de retour affichée par Receipts aux redirections autorisées : `PUBLIC_URL/api/auth/discord/callback`, avec l’origine publique exacte de votre installation.
3. Enregistrer ces deux valeurs dans Receipts. Le premier enregistrement exige un secret ; un identifiant seul ou des champs constitués uniquement d’espaces sont refusés. Le statut configuré confirme la présence des deux valeurs, sans vérifier leur validité auprès de Discord.
4. Pour une modification ultérieure, laisser le secret vide conserve sa valeur. Un nouveau secret le remplace. « Désactiver Discord » (ou enregistrer un identifiant vide) efface les deux valeurs ; les comptes locaux restent disponibles. Réactiver demandera de saisir à nouveau le secret.

Ces étapes suivent la [documentation OAuth2 officielle de Discord](https://docs.discord.com/developers/topics/oauth2). Enregistrer les réglages ne déclenche aucun appel fournisseur. La connexion ou la liaison ouvre ensuite Discord dans le navigateur ; le serveur échange le code avec Discord puis lit l’identité publique via la seule portée `identify`, sans accès aux messages ni à l’adresse e-mail. Ces échanges et leur validation sont le périmètre du ticket #6.

Le secret n’est jamais renvoyé au navigateur après saisie, ni écrit dans les logs. L’interface n’en indique que la présence. Les valeurs sont chiffrées en PostgreSQL par AES-256-GCM avec nonce aléatoire et nom du réglage authentifié ; toute altération ou mauvaise clé empêche leur lecture. Conserver `MASTER_KEY` entre redémarrages et la sauvegarder **séparément** de PostgreSQL dans un emplacement sûr. Une restauration utilisable demande la base **et la même clé** ; une nouvelle clé ne déchiffre pas les anciens secrets. Le test complet de restauration reste prévu en phase 7.

## Configuration et santé

| Variable | Usage |
| --- | --- |
| `DATABASE_URL` | URL PostgreSQL requise ; conserver les identifiants hors du dépôt. |
| `MASTER_KEY` | 32 octets aléatoires en base64, requis pour `serve` ; sauvegarder séparément de la base et conserver la même clé entre redémarrages. |
| `PUBLIC_URL` | Origine HTTP(S) publique sans chemin ni identifiants ; `http://localhost:8080` par défaut. |
| `COOKIE_SECURE` | `true` par défaut, exige HTTPS ; `false` uniquement pour le développement HTTP. |
| `LISTEN_ADDR` | Adresse et port d'écoute, `:8080` par défaut. `healthcheck` utilise la même variable. |
| `DISPLAY_NAME` | Nom affiché dans le navigateur, `Receipts` par défaut. |
| `EMBEDDINGS_URL` | URL HTTP(S) facultative du point d'accès compatible embeddings ; pas d'identifiants ni query string. |
| `EMBEDDINGS_MODEL` | Modèle déjà disponible chez le fournisseur ; aucun téléchargement implicite. Vide : pas d'appel au fournisseur. |

L'exemple [.env.example](.env.example) prépare les variables du futur déploiement Compose ; ses références d'images sont des exemples, pas une publication annoncée. Pour ce démarrage local, les embeddings sont facultatifs et peuvent rester absents. Aucun chemin vers la bibliothèque de l'hôte n'est nécessaire.

- `GET /api/health/live` : processus vivant, HTTP 200, `{"status":"ok"}`.
- `GET /api/health/ready` : schéma accessible, HTTP 200 ; `degraded` si embeddings absents ou injoignables, `ok` si leur sonde réussit. Une base inaccessible donne HTTP 503 avec `database_unavailable`.
- `GET /api/setup` : état d'installation, nom d'affichage et phases reportées ; aucun secret d'installation.
- `./bin/receipts healthcheck` : sortie 0 si l'API est prête, y compris en mode dégradé.

Les logs de l'application sont du JSON avec erreurs génériques, sans URL PostgreSQL, clé maître ni corps de requête. Le pool est limité à six connexions. SIGTERM laisse jusqu'à dix secondes aux requêtes en cours avant la fermeture du pool.

## Vérifier et contribuer

```sh
npm --prefix web run build
bash scripts/test.sh
RECEIPTS_E2E=1 bash scripts/test.sh
npm --prefix web run lint
```

Les tests utilisent uniquement une base Testcontainers jetable avec pgvector et des données synthétiques. Le scénario de l'exécutable vérifie migrations séparées/rejouées, SPA embarquée, port configurable, santé pendant une panne PostgreSQL, logs et arrêt. Les tests API couvrent aussi l'échec d'une migration invalide. `RECEIPTS_E2E=1` lance les scénarios navigateur sur la vraie API, dont installation sans Discord, code invalide, rechargement, connexion/déconnexion et configuration/conservation/désactivation du secret Discord ; `RECEIPTS_E2E_FOUNDATION=1` permet de limiter le navigateur au socle du ticket #2. Installer Chromium une fois avec `cd web && npx playwright install chromium`. Le script détecte le contexte Docker actif ; les surcharges Testcontainers peuvent être fournies par l'environnement.

Le contrat source est [contracts/openapi.yaml](contracts/openapi.yaml). Les opérations d’installation et de session locale sont vérifiées avec leurs entrées, réponses et refus. Les réglages Discord sont vérifiés avec conservation/remplacement du secret, stockage chiffré, refus d’altération et permissions. Les échanges Discord présents dans le brouillon restent à vérifier au ticket #6. Pour régénérer les types Go, TypeScript, Python et les requêtes sqlc :

```sh
python3 -m venv .venv
.venv/bin/pip install -r contracts/generators/requirements.lock
bash scripts/generate.sh
python3 scripts/check_generated.py
```

Les versions des générateurs sont épinglées. Le dernier contrôle compare tous les octets générés avant et après régénération et échoue si les artefacts divergent. Le worker reste un squelette de contrat, sans traitement GPU.

La [roadmap](ROADMAP.md), la [spec de phase 0](docs/specs/phase-0.md), le [glossaire](GLOSSARY.md) et les [ADR](docs/adr/) décrivent le périmètre. Les tests du brouillon qui dépassent les tickets #2 et #3 ne constituent pas la validation des tickets suivants. Le déploiement, la CI et le bilan complet de phase restent à finaliser avant validation du passage à la phase 1.
