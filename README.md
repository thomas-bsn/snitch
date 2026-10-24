# Snitch / Receipts

Receipts (nom d’affichage configurable) est un socle open source MIT pour une bibliothèque de clips de jeu. Il propose les comptes locaux, l’approbation, Discord facultatif, les imports vidéo reprenables, la réconciliation de dossiers autorisés et un worker distant sécurisé. La phase 3 ajoute la transcription des clips prêts via un worker configurable; aucun modèle final n’est imposé ou inclus dans le dépôt.

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

Le worker autonome (absent de Compose) traite les fixtures et les tâches de transcription. Créez-le dans l’interface pour obtenir un jeton, construisez `deploy/docker/worker.Dockerfile`, puis configurez `RECEIPTS_API_URL` (HTTPS), `RECEIPTS_WORKER_TOKEN`, `TRANSCRIPTION_API_URL` et `TRANSCRIPTION_MODEL`. L’URL du fournisseur doit exposer `/v1/audio/transcriptions` et retourner des segments et mots horodatés; aucun fournisseur ni modèle n’est présélectionné. `TRANSCRIPTION_API_KEY` est facultative et n’est jamais écrite dans les logs. Sans URL ou modèle, la tâche échoue avec un code lisible et peut être relancée après configuration. `RECEIPTS_WORKER_CA_BUNDLE` accepte une autorité de certification personnalisée. La validation TLS de Receipts reste activée. [Configurer le worker et le fournisseur de transcription](docs/transcription-worker.md).

## Exploiter ou contribuer

- [Déploiement Forgeyard, variables et migrations](docs/deployment.md)
- [Développement, tests et CI](docs/development.md) · [Contribution](CONTRIBUTING.md)
- [Sauvegarde et restauration](docs/backups.md) · [Licences des distributions](deploy/NOTICES.md)
- [Bilans de phases 0](docs/phase-0-report.md), [1](docs/phase-1-report.md) et [2](docs/phase-2-report.md) · [Roadmap](ROADMAP.md) · [Specs de phases 1](docs/specs/phase-1.md), [2](docs/specs/phase-2.md) et [3](docs/specs/phase-3.md) · [Glossaire](GLOSSARY.md)

Les administrateurs peuvent ajouter des dossiers sous les racines configurées. Le scan garde l’historique des emplacements disparus et évite de déclarer des fichiers manquants si un dossier est inaccessible ou si le parcours est incomplet. Une empreinte SHA-256 permet de relier le même contenu à plusieurs emplacements ; remplacer le contenu d’un chemin crée une nouvelle identité de clip. Une disparition d’au moins 20 % des fichiers suivis produit un avertissement de santé. Voir la [décision de réconciliation](docs/adr/0010-identite-et-reconciliation-des-clips.md).

Les vérifications utilisent des fixtures et des répertoires temporaires synthétiques. Aucun déploiement NAS ni scan de bibliothèque réelle n’a été réalisé. Les phases 0 à 2 sont validées ; la phase 3 est en cours. Le benchmark réel de phase 7 demande une autorisation séparée.
