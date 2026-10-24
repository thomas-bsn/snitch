# Snitch / Receipts

Receipts (nom d’affichage configurable) est un socle open source MIT pour une future bibliothèque de clips de jeu. La phase 0 fournit installation, comptes locaux, approbation et Discord facultatif. Ingestion, recherche et worker GPU attendent les phases suivantes. Aucun média ni modèle n’est nécessaire.

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

Ouvrir <http://localhost:8080>, saisir le code d’installation, choisir votre nom et un mot de passe personnel (12–256 caractères), puis créer le premier administrateur. Le code dure 15 minutes et ne sert qu’une fois. Si le code perdu a expiré, redémarrer l’application pour en obtenir un nouveau ; un redémarrage conserve un code encore valide sans le réafficher. Une installation terminée ne se rouvre pas.

Continuer sans Discord, ou ouvrir « Accès et Discord » pour saisir les identifiants de **votre** application Discord et enregistrer l’URL de retour affichée. Les inscriptions demandent une approbation par défaut. [Guide des comptes, sessions et Discord](docs/access.md).

`docker compose stop` arrête les services ; `docker compose up -d` retrouve les volumes et l’installation. `docker compose down` conserve les données ; ajouter `-v` les détruit. L’override local expose uniquement l’interface sur loopback et PostgreSQL sur loopback 55432 ; `COOKIE_SECURE=false` est réservé à cet essai HTTP. Aucun dossier hôte n’est monté.

## Exploiter ou contribuer

- [Déploiement Forgeyard, variables et migrations](docs/deployment.md)
- [Développement, tests et CI](docs/development.md) · [Contribution](CONTRIBUTING.md)
- [Sauvegarde et restauration](docs/backups.md) · [Licences des distributions](deploy/NOTICES.md)
- [Bilan et mesures de phase 0](docs/phase-0-report.md) · [Roadmap](ROADMAP.md) · [Spec](docs/specs/phase-0.md) · [Glossaire](GLOSSARY.md)

Aucune publication d’image, aucun déploiement NAS, aucun scan de bibliothèque réelle n’a été réalisé. La [spec de phase 1](docs/specs/phase-1.md) est publiée dans [l’issue #8](https://github.com/thomas-bsn/snitch/issues/8) ; son implémentation attend la validation utilisateur. Le benchmark réel de phase 7 demande une autorisation séparée.
