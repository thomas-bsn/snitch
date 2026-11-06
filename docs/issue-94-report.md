# Métadonnées de capture des clips — #94

Le jeu et la date de prise de vue sont désormais extraits du conteneur, puis, en repli, d’un nom DVR reconnu. Le nom original reste conservé et chaque champ garde son origine (`container`, `filename` ou `manual`). Une valeur inconnue reste inconnue plutôt que devinée. La décision est consignée dans l’[ADR 0019](adr/0019-metadonnees-clips.md).

## Champs et origines

- `game` et `capture_time`, nullables, avec `game_origin` et `capture_time_origin`.
- `game_edited` et `capture_time_edited` protègent une correction manuelle : un retraitement ne l’écrase pas, et l’administrateur peut aussi effacer la valeur.
- `recorded_at` reste la date technique utilisée par l’ingestion ; `capture_time` porte la date de prise de vue. Le tri de la bibliothèque utilise `capture_time`, puis la date de découverte en repli.

## Extraction

- Le conteneur est prioritaire : tag `game`, puis `creation_time` (RFC 3339, ou date locale sans fuseau).
- Le nom DVR `Jeu AAAA.MM.JJ - HH.MM.SS.xx.DVR` sert de repli pour le jeu et la date.
- Une date sans fuseau explicite est interprétée dans le fuseau configuré pour l’installation. Sans configuration, ou si l’heure locale est ambiguë ou inexistante (changement d’heure), la date reste inconnue.
- La première découverte d’un contenu fixe ses valeurs : un même contenu réapparu sous un autre nom ne les remplace pas.

## Provenance

`source_files.first_seen_at` et `previous_clip_id` conservent la première observation d’une source et empêchent d’attribuer les métadonnées d’un fichier remplacé à l’ancien clip. La fiche liste les provenances connues (Source ou importateur, dans l’ordre de première observation) ; la carte affiche la première. Un importateur supprimé apparaît comme inconnu. Pour les sources antérieures à la migration, la première observation est approximée par leur dernière stabilisation.

## API et interface

- `GET`/`PUT /api/admin/metadata-location` : pays et fuseau de l’installation.
- `PUT /api/admin/clips/{id}/metadata` : correction ou effacement du jeu et de la date, avec les indicateurs `game_changed` et `recorded_at_changed`.
- `POST /api/admin/clip-metadata/reprocess` : retraitement explicite, un clip par requête avec un curseur `after`. Seules les copies encore disponibles et intactes (taille, date de modification et SHA-256 vérifiés) sont sondées ; une copie modifiée est ignorée et la copie intacte suivante est essayée. Les corrections manuelles sont conservées.
- Filtres de dates calculés sur `capture_time` dans le fuseau configuré, filtre par jeu, tri par date de capture.
- Le wizard propose une étape « localisation » (pays et fuseau) ; Administration › Localisation et métadonnées permet de la modifier et de lancer le retraitement. La fiche clip permet de corriger ou d’effacer les valeurs.

## Migration

`00047_clip_metadata.sql` : table `clip_metadata_settings`, colonnes de métadonnées sur `clips`, colonnes de première observation sur `source_files`, ajout de l’étape `location` au wizard, et reprise de `capture_time` depuis `recorded_at` lorsque la date n’était pas estimée.

## Vérifications

- `gofmt`, `go vet ./...` et `go vet -tags integration ./...`.
- `npm --prefix web run lint`, `npm --prefix web run build`.
- `scripts/check_generated.py` (contrats reproductibles), `check_compose.py`, `check_error_codes.py`.
- Tests unitaires de reconnaissance des noms et des fuseaux (`filename_metadata_test.go`).
- Tests d’intégration avec PostgreSQL et des clips synthétiques : découverte, correction manuelle, retraitement, fichier remplacé à taille et date identiques, seconde copie intacte, filtres de dates locales.
- Aucune bibliothèque réelle n’a été scannée pendant le développement.
