# Racine vidéo, sous-dossiers et écrasement

Snitch peut travailler avec **un seul dossier vidéo** — par exemple `/videos` — monté dans le conteneur de l'application. C'est le dossier dans lequel la bibliothèque de jeu peut déposer ses fichiers et dans lequel Snitch peut écrire les vidéos importées. Les anciennes sources en lecture seule (`CLIPS_ROOTS`) restent disponibles séparément.

## Monter le dossier

Le chemin saisi dans l'interface est celui **vu par le conteneur**, pas celui de la machine hôte. Montez le dossier dans le service `app` de votre déploiement, par exemple :

```yaml
services:
  app:
    volumes:
      - /chemin/hote/videos:/videos
```

Le dossier doit être lisible et inscriptible par l'utilisateur de l'application (UID 10001). Snitch vérifie qu'il peut y créer un fichier avant d'enregistrer la racine ; sinon il répond `video_root_unavailable`.

Sur Forgeyard, ajoutez le volume dans la configuration du service avant de redéployer. La syntaxe exacte dépend de la plateforme : ne la devinez pas, reprenez celle de votre projet.

## Configurer la racine

Dans **Administration › Dossiers** (ou dans l'étape « Import » de l'installation guidée), saisissez le chemin du conteneur, par exemple `/videos`, puis **Enregistrer le dossier**.

- Enregistrer un chemin **ne lance aucun scan**.
- La santé de la racine et la date du dernier scan sont affichées.
- L'interface indique si un Worker est connecté. Sans Worker, les fichiers demandés restent en attente.

### Changer de racine

Un changement de racine est refusé tant qu'un envoi attend sa publication (`video_root_busy`). Terminez ou annulez ces envois, puis confirmez le changement.

Après le changement :

- les fichiers indexés sous l'ancienne racine sont marqués **manquants** ;
- leurs clips, transcriptions, corrections et attributions sont conservés tant qu'une copie du contenu existe ;
- la nouvelle racine n'est pas indexée : choisissez ce que vous voulez scanner.

## Demander un scan

Dans **Administration › Dossiers**, ouvrez **Scanner des vidéos et gérer les dossiers**, naviguez dans la racine et :

1. cochez un ou plusieurs fichiers ou sous-dossiers, ou utilisez **Scanner toute la racine** ;
2. prévisualisez la liste des fichiers vidéo trouvés ;
3. confirmez avec **Ajouter à la file**.

Un sous-dossier sélectionné inclut ses sous-dossiers. Rien n'est analysé avant la confirmation. La file affiche chaque fichier avec son état :

- **Indexé** : le fichier est prêt et plus rien n'attend ;
- **En attente d'un Worker** : la demande est enregistrée, mais aucun Worker n'est connecté ;
- **En attente de configuration** : un Worker est là, mais un réglage de transcription manque ;
- **Manquant** : le fichier n'existe plus à cet emplacement (par exemple après un changement de racine) ;
- **À vérifier** : une erreur est affichée avec son code traduit.

Un fichier n'est jamais présenté comme indexé tant que sa demande n'est pas réellement traitée.

## Explorer la racine

La **Bibliothèque** parcourt la racine comme un explorateur (ADR 0032) : un arbre des dossiers à gauche, une barre d'outils (dossier parent, fil d'Ariane, recherche dans le dossier, Filtres, vue Liste ou Icônes), et pour chaque vidéo son état : **Non indexée**, **En cours**, **Non analysée**, **Transcrite**, **Analysée** ou **Échec**.

- Lister lit seulement le répertoire ; aucune vidéo n'est lue ni indexée par la navigation, et rien n'est scanné en arrière-plan.
- Ouvrir une vidéo **non indexée** l'indexe sans la transcrire, puis ouvre sa fiche ; l'analyse se lance ensuite depuis la fiche. Ce parcours n'exige pas de Worker.
- Avec un filtre actif (recherche, État, Jeu, dates, visibilité, propriétaire), les résultats sont à plat, avec leur dossier, pour le dossier courant et ses sous-dossiers.
- Le menu d'un dossier propose **Analyser les vidéos non analysées** (administrateurs) et **Rendre privé / partagé** (Super-administrateur, voir ci-dessous).
- Les clips sans fichier dans la racine (envois non publiés dans la racine, anciennes sources) restent visibles dans « Tous les clips ».

## Dossiers système du NAS

Les dossiers nommés exactement `#recycle` (corbeille UGREEN) et `@eaDir` (métadonnées NAS) sont exclus automatiquement, à tous les niveaux de l'arborescence. Snitch ne les ouvre pas, même s'ils sont inaccessibles au conteneur. La même règle s'applique à l'estimation et au lancement de l'Indexation de la Racine, au scan manuel et au sélecteur d'import, à l'explorateur (avec ou sans filtres) et à la découverte des vidéos des sources historiques. Une sélection directe de ces dossiers ou de leur contenu est refusée.

Il n'est pas nécessaire de changer les permissions de ces dossiers système. Les refus de permission sur les dossiers ordinaires restent signalés : un scan manuel ou une estimation incomplète échoue ; la réconciliation historique affiche un avertissement et conserve les états précédents. Les Dossiers privés accessibles au Super-administrateur restent parcourus et indexés pour lui.

## Dossiers privés

Le Super-administrateur peut rendre privé un dossier de premier niveau de la racine (ADR 0026). Le dossier reste visible avec un cadenas pour les autres utilisateurs, mais son contenu n'est ni listé, ni prévisualisé, ni ajouté à la file, ni créé dessous, y compris pour les administrateurs. Dans la Bibliothèque, son nom reste dans l'arbre avec un cadenas, et ses vidéos n'apparaissent ni dans la liste, ni dans les résultats filtrés, ni dans la recherche. Un clip présent aussi dans un autre dossier reste accessible par ce dossier-là.

## Créer un sous-dossier et y importer

Dans le même panneau, **Nouveau sous-dossier** crée un dossier dans le dossier courant. Le nom ne peut pas contenir de barre oblique ni commencer par un point.

À l'import, **Choisir un dossier** permet de sélectionner un sous-dossier existant comme destination. Sans choix, le fichier va à la racine. Le fichier est d'abord envoyé dans un staging, puis publié dans la racine une fois complet.

## Importer : deux choix

L'écran **Importer** propose deux cartes.

- **Déjà sur le serveur** : tout utilisateur connecté choisit des vidéos déjà présentes dans la racine (fichiers, plusieurs à la fois, dossiers entiers, sous-dossiers compris). Le total est affiché, puis la sélection est ajoutée à la file après confirmation ; elle est ensuite indexée puis analysée selon le réglage de la bibliothèque. Ce sélecteur ne crée aucun dossier. Il applique la règle des Dossiers privés : un dossier privé apparaît avec son cadenas, ne peut pas être sélectionné, et son contenu n'est ni compté ni ajouté, sauf pour le Super-administrateur (`POST /api/library/import/scan`, même décompte qu'un scan).
- **Depuis mon ordinateur** : téléversement par blocs, avec le dossier de destination (racine par défaut une fois choisie). Un dossier privé n'est pas une destination pour un utilisateur, et un dossier en attente de décision n'en est pas une non plus.

## Demander un dossier

Au choix de la destination d'un téléversement, **Nouveau dossier** (ADR 0033) :

- un Administrateur crée le dossier aussitôt (`POST /api/folder-requests`, réponse `created`) ;
- un utilisateur envoie une **demande de création de dossier**, en attente : rien ne peut être téléversé dans ce dossier (`folder_request_pending`) tant qu'elle n'est pas tranchée.

Les demandes apparaissent dans **Administration › Inscriptions**, à côté des inscriptions en attente, avec **Approuver** (le dossier est créé) ou **Refuser** (rien n'est créé). Le demandeur reçoit une notification de la décision. Une demande sous un dossier privé est réservée au Super-administrateur. Le journal des demandes et des décisions est immuable (`folder_request_events`) ; une demande refusée peut être redemandée.

## Écraser un fichier existant

Si un fichier porte déjà ce nom, Snitch demande confirmation et affiche le chemin. Après confirmation :

- l'ancien fichier est remplacé par le nouveau contenu ;
- un nouveau clip est créé pour ce contenu ;
- les résultats de l'ancien contenu ne sont supprimés que si aucune autre copie ne les utilise. Si le même contenu existe ailleurs dans la racine, cette autre copie garde ses transcriptions et corrections.

Deux envois simultanés vers le même chemin ne s'écrasent pas : un seul est publié, l'autre est signalé `destination_changed` et peut être réessayé ou annulé.

## Reprendre ou annuler une publication

Un envoi complet dont la publication a échoué (permission, destination modifiée, coupure) reste visible dans **Importer** avec son état **Non publiée**. **Réessayer la publication** relance la publication sans renvoyer le fichier ; **Annuler** libère l'intention et supprime le staging. Une boucle serveur reprend aussi automatiquement les publications en attente.

## Photo de profil

Dans **Compte › Profil**, **Choisir une photo** accepte un JPEG ou PNG de 5 Mo et 16 mégapixels au plus. L'image est recadrée au centre en carré et ses métadonnées sont retirées. **Retirer la photo** revient aux initiales. La photo apparaît aussi dans le menu du compte.

## Sauvegarde

La racine vidéo contient des originaux : sauvegardez-la séparément de PostgreSQL et du volume `media`, avec une rétention adaptée. Une sauvegarde de la base ne contient ni les vidéos ni leurs fichiers. Voir [sauvegarde et restauration](backups.md).
