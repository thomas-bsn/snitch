# Sondage média avec FFprobe

Statut : accepté pour la phase 1.

## Contexte

La phase 1 doit vérifier qu'un fichier local est lisible comme vidéo et conserver durée, codecs, pistes et métadonnées. Les originaux doivent rester intacts et aucun média réel ne sera parcouru avant le benchmark autorisé.

## Décision

L'application appelle le binaire `ffprobe` directement, sans shell, avec une entrée de fichier local résolue dans une source autorisée, un délai d'exécution borné, une sortie JSON plafonnée et une liste de protocoles limitée à `file,pipe`. Le sondage est en lecture seule et aucun transcodage n'est exécuté.

L'image app contient le paquet Alpine v3.24 `ffmpeg` 8.1.2-r0, qui fournit `ffprobe`. Ce paquet est identifié GPL-2.0-or-later et LGPL-2.1-or-later ; ce composant conserve ses licences propres et ne change pas la licence MIT du code Snitch. Les textes GPL/LGPL et la version/source FFmpeg sont conservés dans les notices. Avant toute publication d'image, fournir les sources correspondantes exactes de l'APK Alpine et les informations de build applicables ; ne pas présenter l'image entière comme MIT uniquement.

## Alternatives écartées

- Parsing d'un seul conteneur en Go : couverture codec/conteneur insuffisante pour les clips hétérogènes attendus.
- Dépendance native liée à libav dans le binaire Go : davantage de complexité de compilation/cross-compilation et une liaison plus étroite.
- Construire immédiatement un FFmpeg LGPL minimal : possible, mais les codecs réellement nécessaires ne sont pas encore établis ; rétrécir la prise en charge avant des fixtures représentatives serait prématuré.

## Conséquences

- Le paquet ajoute ses bibliothèques à l'image app et a une empreinte supérieure ; mesurer cette image après le build.
- Le dépôt conserve le texte de licence upstream. Le lien vers le tag FFmpeg n8.1.2 et le commit Alpine aports de l'APKBUILD identifient les sources ; le processus de publication devra archiver la source et la configuration de build correspondant à chaque binaire distribué.
- Un échec ou dépassement de délai du probe devient un état d'ingestion exploitable ; son stderr et son chemin absolu ne sont pas retournés à l'utilisateur ni journalisés.
- Les tests API peuvent remplacer le binaire au bord du système par un script de fixture ; l'image/smoke exécute le vrai binaire sur une vidéo synthétique.
