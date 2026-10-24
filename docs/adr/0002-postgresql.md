# PostgreSQL, SQL explicite et migrations séparées

Statut : accepté dans le plan de phase 0.

## Contexte et décision

PostgreSQL avec pgvector conserve données métier, sessions et futures tâches. pgx et sqlc encadrent le SQL ; goose applique les migrations dans une commande distincte.

## Alternatives écartées

SQLite ne couvre pas les tests PostgreSQL et une base vectorielle séparée alourdit le NAS.

## Conséquences

Une migration échouée bloque le démarrage orchestré ; démarrer directement serve ne migre jamais la base.
