# Snitch / Receipts

Receipts (nom d’affichage configurable) est un socle open source MIT pour une bibliothèque de clips de jeu. Il propose les comptes locaux, l’approbation, Discord facultatif, les imports vidéo reprenables, la réconciliation de dossiers autorisés et un worker distant sécurisé. Les étapes IA et la recherche avancée restent à venir ; aucun modèle n’est nécessaire à ce stade.

## Essayer localement

Prérequis : Docker avec Compose v2, Git ; environ 8 Go d’images amont en cache. Les images GHCR ne sont pas encore publiées : construire depuis les sources. Node et Go tournent dans les stages de build, jamais dans le conteneur final.

```sh
git clone https://github.com/thomas-bsn/snitch.git
cd snitch
cp .env.example .env
# Dans .env, choisir POSTGRES_PASSWORD, puis MASTER_KEY avec :
openssl rand -base64 32
# Conserver cette clé séparément et ne jamais committer .env.
docker compose build app migrate
docker compose up -d
# Lire localement le code temporaire dans les logs (ne pas les partager) :
docker compose logs app
```

Ouvrir <http://localhost:8080>, saisir le code d’installation, choisir votre nom et un mot de passe personnel (12–256 caractères), puis créer le premier administrateur. Le code dure 15 minutes et ne sert qu’une fois. Si le code perdu a expiré, redémarrer l’application pour en obtenir un nouveau ; un redémarrage conserve un code encore valide sans le réafficher. Une installation terminée ne se rouvre pas. Les membres peuvent ensuite importer leurs vidéos par blocs, reprendre un transfert interrompu ou l’annuler. La bibliothèque propose pagination, filtres par date/source/propriétaire/état, détails et lecture avec navigation. Les membres voient les clips partagés ; un clip privé reste visible à son propriétaire et aux administrateurs. `MAX_UPLOAD_BYTES`, `UPLOAD_CHUNK_BYTES` et `UPLOAD_SESSION_TTL` règlent la limite (20 GiB), le bloc (4 MiB) et la durée de reprise (168 h). Les fichiers importés persistent dans le volume Docker nommé `media` ; sauvegardez-le séparément de PostgreSQL.

Continuer sans Discord, ou ouvrir « Accès et Discord » pour saisir les identifiants de **votre** application Discord et enregistrer l’URL de retour affichée. Les inscriptions demandent une approbation par défaut. [Guide des comptes, sessions et Discord](docs/access.md).

`docker compose stop` arrête les services ; `docker compose up -d` retrouve les volumes et l’installation. `docker compose down` conserve les données ; ajouter `-v` les détruit. L’override local expose uniquement l’interface sur loopback et PostgreSQL sur loopback 55432 ; `COOKIE_SECURE=false` est réservé à cet essai HTTP. Aucun dossier hôte n’est monté.

Le worker de phase 2 est une image autonome, absente de Compose : l’administrateur crée un worker dans l’interface et copie son jeton une seule fois. Construisez l’image avec `docker build -f deploy/docker/worker.Dockerfile -t snitch-worker .`, puis lancez-la en renseignant `RECEIPTS_API_URL` (URL HTTPS de Receipts), `RECEIPTS_WORKER_TOKEN` et éventuellement `RECEIPTS_WORKER_CAPACITY` (1–4, défaut 1). `RECEIPTS_WORKER_CA_BUNDLE` peut pointer vers une autorité de certification approuvée. La validation TLS reste activée. À chaque pulse, un worker déclare les étapes qu’il sait traiter; le serveur ne lui attribue que celles-ci et garde les autres en attente. Le fournisseur factice de phase 2 déclare uniquement `fixture.*`, retourne des métadonnées déterministes et ne télécharge ni ne lit le média.

## Exploiter ou contribuer

- [Déploiement Forgeyard, variables et migrations](docs/deployment.md)
- [Développement, tests et CI](docs/development.md) · [Contribution](CONTRIBUTING.md)
- [Sauvegarde et restauration](docs/backups.md) · [Licences des distributions](deploy/NOTICES.md)
- [Bilans de phases 0](docs/phase-0-report.md), [1](docs/phase-1-report.md) et [2](docs/phase-2-report.md) · [Roadmap](ROADMAP.md) · [Specs de phase 1](docs/specs/phase-1.md) et [phase 2](docs/specs/phase-2.md) · [Glossaire](GLOSSARY.md)

Les administrateurs peuvent ajouter des dossiers sous les racines configurées. Le scan garde l’historique des emplacements disparus et évite de déclarer des fichiers manquants si un dossier est inaccessible ou si le parcours est incomplet. Une empreinte SHA-256 permet de relier le même contenu à plusieurs emplacements ; remplacer le contenu d’un chemin crée une nouvelle identité de clip. Une disparition d’au moins 20 % des fichiers suivis produit un avertissement de santé. Voir la [décision de réconciliation](docs/adr/0010-identite-et-reconciliation-des-clips.md).

Les vérifications utilisent des fixtures et des répertoires temporaires synthétiques. Aucun déploiement NAS ni scan de bibliothèque réelle n’a été réalisé. La phase 1 est validée ; la phase 2 est implémentée localement et attend sa validation finale avant la phase 3. Le benchmark réel de phase 7 demande une autorisation séparée.
