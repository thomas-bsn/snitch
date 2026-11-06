# 0033 — Demandes de création de dossier

Statut : accepté.

## Contexte

À l'import, un utilisateur peut vouloir ranger une vidéo dans un dossier qui n'existe pas. Créer un dossier dans la Racine vidéo modifie le disque partagé ; seuls les Administrateurs le font aujourd'hui.

## Décision

- Un utilisateur crée une **Demande de création de dossier** (dossier parent, nom). Elle est en attente, approuvée ou refusée, avec un journal immuable, sur le modèle des demandes de liaison Personne. Un Administrateur ou le Super-administrateur crée le dossier directement.
- Les demandes se traitent à côté des inscriptions en attente ; le demandeur est notifié de la décision.
- On ne téléverse pas vers un dossier en attente. La création n'est proposée qu'au choix de la destination d'un téléversement, jamais dans la sélection de vidéos déjà sur le serveur.

## Alternatives écartées

- **Création libre** : n'importe quel utilisateur modifierait l'arborescence partagée.
- **Téléverser vers le dossier en attente puis publier à l'approbation** : il faudrait gérer des fichiers en suspens en cas de refus.

## Conséquences

- Une nouvelle table de demandes et un nouveau type de notification.
