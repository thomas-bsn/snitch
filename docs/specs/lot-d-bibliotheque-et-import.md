# Lot D — Bibliothèque et import

Fait partie des évolutions des lots A à D (voir [ROADMAP](../../ROADMAP.md)). Le parcours de l'arborescence réelle de la Racine vidéo est autorisé par l'utilisateur pour l'affichage. Aucune transcription n'est lancée sans action, et les tests utilisent des fixtures. Référence visuelle : [`prototype/fiche-video`](https://github.com/thomas-bsn/snitch/tree/prototype/fiche-video) (`/prototype/library?variant=D`, `/prototype/import?variant=A`).

## Problème

- **Faille** : le parcours de la Racine vidéo est ouvert à tout utilisateur connecté, Dossiers privés compris.
- **Bibliothèque** : elle ne montre que les clips indexés, avec un panneau de dossiers limité au premier niveau et un bloc de scan d'administration.
- **Import** : il ne permet ni de choisir des vidéos déjà sur le serveur, ni de demander la création d'un dossier.

## Solution

Explorateur sur la Racine vidéo réelle avec indexation à la demande ([ADR 0032](../adr/0032-explorateur-du-disque-et-indexation-a-la-demande.md)) ; import en deux choix ; Demandes de création de dossier ([ADR 0033](../adr/0033-demandes-de-creation-de-dossier.md)).

## Comportements

- **Accès** : le parcours exige une session et applique la règle des Dossiers privés.
- **Explorateur** :
  - barre d'outils : dossier parent, fil d'Ariane, recherche dans le dossier, Filtres, bascule Liste / Icônes ;
  - arbre des dossiers à gauche ;
  - vue liste : nom, état, durée, propriétaire, ajout, taille, voix à traiter ;
  - vue icônes : grandes icônes avec l'état ;
  - filtres actuels (Jeu, Du, Au, Visibilité, Propriétaire, Personne, avec « Voix à traiter ») et État (Non indexée, Non analysée, En cours, Transcrite, Analysée, Échec). Le filtre Source disparaît ;
  - avec un filtre actif : résultats à plat du dossier et de ses sous-dossiers, avec une colonne Dossier ;
  - menu d'un dossier : « Analyser les vidéos non analysées » (Administrateur), « Rendre privé / partagé » (Super-administrateur, règle existante) ;
  - ouvrir une Vidéo non indexée l'indexe sans la transcrire et ouvre sa fiche.
- **Retraits** : le panneau « Dossiers » et le bloc « scanner les vidéos et gérer les dossiers » quittent la Bibliothèque ; la Racine vidéo reste réglable dans Administration › Dossiers.
- **Import** : deux choix.
  - « Déjà sur le serveur » : sélecteur dans la Racine vidéo (fichiers, plusieurs, dossiers récursifs), total affiché, confirmation, puis indexation et analyse selon le réglage du lot B. Ouvert à tous, en respectant les Dossiers privés. Pas de création de dossier ici.
  - « Depuis mon ordinateur » : téléversement existant avec choix du dossier de destination.
- **Demande de création de dossier** : au choix de la destination, « Nouveau dossier ». Pour un Administrateur, le dossier est créé ; pour un utilisateur, la demande est en attente et on ne peut pas y téléverser. Les Administrateurs l'approuvent ou la refusent à côté des inscriptions en attente ; le demandeur est notifié.

## Critères d'acceptation

- Un utilisateur sans droit ne liste ni ne voit le contenu d'un Dossier privé, dans l'explorateur comme dans le sélecteur d'import (test API, premier ticket livré).
- Un dossier de fixtures mélangeant fichiers indexés et non indexés affiche les bons états ; ouvrir un fichier non indexé crée un seul clip, sans job de transcription si l'utilisateur n'en lance pas.
- Non-régression de #195 : un seul clip importé dans un sous-dossier qui contient d'autres fichiers non importés.
- Sélectionner un dossier dans l'import compte les vidéos des sous-dossiers et lance l'indexation.
- Demande : en attente, approuvée (le dossier existe), refusée ; notification ; téléversement refusé vers une demande en attente.

## Dépendances

Lot B ticket 1 (déclenchement de l'analyse) ; fiche vidéo ticket 1 (fiche d'un clip jamais analysé).

## Tickets

1. **Sécurité : parcours de la Racine vidéo réservé et Dossiers privés appliqués**. Aucun blocage.
2. **#195 — Explorateur de la Bibliothèque** (issue existante, portée élargie) : API de parcours croisée avec la base, indexation à la demande, interface liste et icônes, filtres, menu de dossier, retraits. Bloqué par 1.
3. **Import en deux choix et sélecteur de vidéos du serveur**. Bloqué par 1, lot B ticket 1.
4. **Demandes de création de dossier**. Bloqué par 3.
