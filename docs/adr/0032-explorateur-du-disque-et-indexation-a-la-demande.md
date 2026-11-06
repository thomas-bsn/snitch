# 0032 — Explorateur sur la Racine vidéo réelle, indexation à la demande

Statut : accepté. Élargit #195, qui se limitait aux chemins déjà indexés. Reste dans la règle de l'[ADR 0017](0017-racine-video-unique-et-publication-nommee.md) : aucune découverte automatique, aucune analyse sans action. Amendé par l'[ADR 0038](0038-indexation-de-toute-la-racine.md) : l'indexation de masse reste exclue en arrière-plan, mais un administrateur peut la demander explicitement.

## Contexte

L'utilisateur veut parcourir la Bibliothèque comme un explorateur de fichiers et voir aussi les vidéos jamais traitées. Le parcours existe (`GET /api/admin/video-root/browse`), mais il est enregistré `protected(false)` : tout utilisateur connecté peut lister la Racine vidéo, Dossiers privés compris.

## Décision

- **L'explorateur liste le disque** (dossiers et fichiers vidéo de la Racine vidéo) et le croise avec la base. Chaque fichier porte un état : non indexée, non analysée, en cours, transcrite, analysée, échec. Lister ne lit pas le contenu des fichiers.
- **Indexation à la demande** : ouvrir une Vidéo non indexée la sonde et calcule son empreinte (sans transcription), crée le clip et ouvre sa fiche. Aucune indexation de masse.
- **Accès** : le parcours exige une session et applique la règle des Dossiers privés de l'[ADR 0026](0026-dossier-prive.md) : seulement le nom et le cadenas pour qui ne peut pas l'ouvrir. Cette correction précède toute autre évolution de la Bibliothèque.
- **Filtres** : avec un filtre actif, les résultats couvrent le dossier courant et ses sous-dossiers, à plat, avec leur dossier. Amendé par l'[ADR 0037](0037-filtre-de-traitement-arbre-elague.md) pour le filtre de traitement (arbre élagué).

## Alternatives écartées

- **Chemins indexés seulement** (#195 initial) : les vidéos jamais importées resteraient invisibles.
- **Indexer toute la racine en arrière-plan** : lecture complète de gros fichiers sur le NAS, contraire à l'ADR 0017.

## Conséquences

- Parcourir un dossier volumineux coûte une lecture de répertoire par affichage ; la pagination côté serveur est requise.
- Le panneau « Dossiers » et le bloc de scan quittent la Bibliothèque. La Racine vidéo reste réglable dans Administration › Dossiers, et rendre un dossier privé passe dans le menu du dossier.
