# Racine vidéo unique, scan demandé et publication nommée

Statut : accepté pour la fin de la phase 6 bis. Amende l'invariant historique « les originaux ne sont jamais modifiés » pour les fichiers déposés dans la racine vidéo, sur demande explicite de l'utilisateur. Les sources historiques en lecture seule restent inchangées.

## Contexte

L'import actuel écrit chaque vidéo dans le volume `media` sous un nom généré : l'utilisateur ne retrouve pas ses fichiers dans une arborescence, et le dossier de la bibliothèque de jeu ne peut pas être réutilisé directement. La phase 6 bis a introduit une racine vidéo gérée (`clip_sources.managed_root`), mais plusieurs limites restaient :

- les changements de racine étaient refusés après la première configuration ;
- une sélection de fichiers ou de dossiers ne pouvait pas être demandée explicitement, et une file `ready` s'affichait comme indexée alors qu'elle attendait encore un Worker ou un réglage ;
- l'upload nommé ne remontait ni la destination ni l'état de publication après un rechargement, et un échec de publication restait invisible ;
- le champ de destination était un texte libre, sans choix d'un sous-dossier existant ;
- la photo de profil n'existait que dans un écran secondaire, avec des textes non traduits.

Les contraintes demeurent : aucune découverte automatique de la bibliothèque, aucune analyse sans confirmation, les fichiers en attente restent en attente sans Worker, et les tests n'utilisent que des fixtures synthétiques jusqu'à l'autorisation de la phase 7.

## Décision

**Une seule racine vidéo, configurable, jamais parcourue d'elle-même.** `PUT /api/admin/video-root` enregistre un chemin déjà monté et inscriptible. Enregistrer un chemin ne lance aucun scan. `GET` renvoie la racine, sa santé, son dernier scan et `worker_available`. La racine est le seul dossier dans lequel Snitch peut écrire des vidéos ; les anciennes sources `clip_sources` restent lisibles en lecture seule et ne sont pas fusionnées.

**Changement de racine sûr.** `SetManagedRoot` refuse le changement tant qu'une publication d'upload est en attente (`video_root_busy`, 409). Après confirmation explicite dans l'interface, il met à jour la racine, marque les `source_files` de l'ancienne racine `missing` et conserve leurs clips, transcriptions, corrections et attributions. La nouvelle racine démarre non indexée : l'utilisateur choisit ce qu'il veut scanner. Aucun fichier n'est relié par ressemblance de chemin.

**Scan demandé, prévisualisé, confirmé.** `POST /api/admin/video-root/scan` avec `confirm: false` liste les fichiers vidéo de la sélection (fichier, fichiers, sous-dossier récursif ou racine entière) sans rien écrire. Avec `confirm: true`, il crée les `source_files` avec `scan_requested`. La file expose `scan_requested` et `error_code`. Sans Worker connecté, les fichiers restent en attente ; l'interface distingue « en attente d'un Worker », « en attente de configuration », « indexé », « manquant » et « à vérifier ». Un fichier `ready` dont `scan_requested` est vrai n'est jamais présenté comme indexé.

**Upload nommé, destination persistante et publication idempotente.** Un upload peut viser un sous-dossier relatif de la racine. L'intention (destination, version attendue) est stockée dans `upload_destinations` ; `attempts` et `last_error` suivent les reprises. La publication copie le staging dans un fichier temporaire de la racine, vérifie que la version attendue n'a pas changé, renomme, puis lie le nouveau clip au chemin et détache l'ancien. Elle est idempotente : après une interruption, le contenu déjà installé est reconnu par son empreinte, et un staging encore présent est republié. Une boucle de reprise reprend les publications non terminées ; l'interface liste aussi les uploads complets non publiés, permet de réessayer (`POST /complete` est idempotent) ou d'annuler. Annuler ou expirer un upload libère son intention pour ne pas bloquer un changement de racine.

**Écrasement confirmé, copies préservées.** Un nom déjà présent renvoie `409` avec la version courante ; l'interface demande confirmation puis renvoie cette version. Le nouveau contenu remplace le fichier et devient un nouveau clip. Les résultats de l'ancien clip ne sont supprimés que s'il ne reste aucune autre copie (`source_files`) du même contenu ; une copie distincte garde donc ses transcriptions et corrections. Une course entre deux uploads vers le même chemin est arbitrée par la version attendue : un seul gagne, l'autre est signalé `destination_changed`.

**Photo de profil.** `PUT /api/account/avatar` accepte un JPEG ou PNG de 5 Mo et 16 mégapixels au plus, recadre le carré central en 256 px et stocke uniquement les pixels réencodés, sans métadonnées. `GET` sert le PNG, `DELETE` le retire. Le menu du compte et le profil utilisent la même source ; les textes sont dans les deux catalogues.

## Alternatives écartées

- **Scanner la racine automatiquement** : contredit la demande et exposerait des fichiers non choisis.
- **Fusionner les anciennes sources dans la racine** : un chemin identique ne prouve pas que le contenu est le même ; marquer `missing` est réversible et explicite.
- **Refuser tout changement de racine** : empêcherait de corriger un montage sans réinstaller.
- **Écraser sans confirmation** : perte silencieuse d'un original.
- **Supprimer l'ancien clip à chaque remplacement** : ferait perdre les résultats d'une autre copie dédupliquée.
- **Garder la destination uniquement dans le navigateur** : un rechargement perdrait l'intention et laisserait un staging orphelin.
- **Un endpoint de reprise dédié** : `POST /complete` est déjà idempotent et reprend la publication ; un second chemin aurait dupliqué les contrôles.
- **Stocker la photo dans le volume `media`** : la base la sauvegarde avec le compte et évite un fichier orphelin.

## Conséquences

- L'invariant « ne jamais modifier les originaux » vaut désormais pour les sources historiques en lecture seule ; les fichiers de la racine vidéo peuvent être remplacés après confirmation. La documentation utilisateur le dit explicitement.
- Un changement de racine laisse des `source_files` `missing` et leurs clips lisibles tant qu'une copie existe. La sauvegarde doit couvrir la racine vidéo en plus de PostgreSQL et du volume `media`.
- La file et l'import exposent des codes stables (`video_root_busy`, `video_root_permission`, `destination_changed`, `publication_failed`) traduits dans les deux langues ; `scripts/check_error_codes.py` les vérifie.
- Les tests d'intégration couvrent l'interruption avant et après renommage, le changement de destination pendant le transfert, l'échec de permission puis la reprise, deux uploads concurrents, la conservation des copies distinctes et le changement de racine bloqué puis appliqué. Le parcours Playwright `video-storage.spec.ts` couvre la configuration sans scan, la prévisualisation, l'attente sans Worker, le sous-dossier, l'écrasement confirmé, le changement de racine et la photo de profil.
- Aucun scan de la vraie bibliothèque n'a été lancé ; les fixtures sont synthétiques.
