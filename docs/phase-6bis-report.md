# Bilan de la phase 6 bis

## Livré

La refonte de l’interface et du wizard prévue par la phase 6 bis est intégrée dans la branche de livraison. Les tickets livrés sont #84, #85, #69, #74, #75, #76, #77, #80, #78 et #79 : navigation du wizard, recherche sémantique, corrections de segments, informations des Workers, file et dossiers, transcription et traitement vocal, membres et exports, fermeture de compte, retrait de l’ancienne interface, puis finition responsive et accessibilité.

L’ancienne application `/legacy/` et ses ressources ont été retirées. Les routes historiques renvoient vers l’application actuelle. Les pages principales ont été contrôlées en thème clair et sombre, ainsi qu’à une largeur mobile de 390 px ; Playwright axe-core contrôle les routes principales.

## Vérifications

- `npm --prefix web run lint` et `npm --prefix web run build` passent. Vite signale un chunk JavaScript de 711 Ko après minification.
- `bash scripts/generate.sh`, `scripts/check_generated.py`, `scripts/check_error_codes.py`, `go vet -tags integration ./...` et `gofmt` passent.
- `SNITCH_E2E=1 bash scripts/test.sh` passe : les tests Go et les parcours Playwright terminent en 349,9 secondes.
- Les contrôles axe-core ciblés passent : 5 contrôles, sans violation, débordement horizontal ni cible tactile visible sous 44 px.

La suite démarre PostgreSQL/pgvector avec Testcontainers, remet la base à l’état de fixtures avant chaque scénario HTTP, puis lance les fichiers Playwright sélectionnés contre l’API locale et la SPA embarquée. Les parcours utilisent des clips, comptes, Workers et fournisseurs synthétiques. Les E2E retirés étaient des doublons qui visaient l’ancienne interface ; leurs parcours sont exercés par `people.spec.ts`, `people-groups.spec.ts`, `clip-corrections.spec.ts` et `admin-voice.spec.ts`. La réindexation, la recherche après correction et l’export gardent leurs propres scénarios ciblés.

## Restant et écarts

Le ticket #81 reste en attente de validation (`needs-triage`) et n’est pas inclus. Le benchmark réel de phase 7 n’a pas été réalisé. La vérification mobile utilise Chromium à 390 px ; aucun appareil physique ni autre moteur de navigateur n’a été testé. Les données de test sont synthétiques.

## Correctif d'installation derrière plusieurs proxies — 29 octobre 2026

Le refus `403 origin_rejected` est reproduit par une requête d'installation HTTPS dont un relais HTTP a remplacé `X-Forwarded-Proto` par `http`. La configuration locale de Forgeyard comporte Caddy devant un Traefik sans confiance configurée pour les en-têtes amont ; les en-têtes effectivement reçus par Snitch en production n'ont pas été capturés.

Snitch retrouve désormais HTTPS depuis les métadonnées `same-origin` du navigateur, derrière un pair TCP de confiance et avec un hôte et port identiques à ceux de la requête. La règle s'applique aux mutations, aux lectures GET/HEAD, aux cookies Secure et aux adresses Discord/Worker. Les origines étrangères, ports différents, pairs non fiables et mutations sans origine restent refusés. `PUBLIC_URL` reste une surcharge facultative. Voir l'[ADR 0016](adr/0016-https-derriere-plusieurs-mandataires.md).

Vérifications : le test de régression reproduisait initialement le 403 ; après correction, les tests des paquets `httpapi` et `config`, `go vet` et le parcours `TestFoundation/public_address_deduced_from_the_request` avec PostgreSQL/pgvector synthétique et détecteur de courses passent. Ce parcours vérifie un code d'installation valide avant création du compte, la connexion, le cookie Secure, les adresses Discord et le fichier de Worker. README et documentation d'accès/déploiement mis à jour.

Restant : publier une image contenant ce correctif, redéployer puis vérifier le parcours sur Forgeyard. Aucun déploiement ni traitement de média réel n'a été lancé. La suite navigateur complète n'a pas été relancée pour ce correctif backend.

## Racine vidéo unique, scan demandé et photo de profil — 30 octobre 2026

### Livré

La racine vidéo est désormais unique et configurable dans l’installation guidée et Administration › Dossiers. L’enregistrer ne lance aucun scan ; l’interface affiche sa santé, son dernier scan et la présence d’un Worker. Un changement de racine est refusé tant qu’un envoi attend sa publication, puis les fichiers de l’ancienne racine sont marqués manquants sans perdre leurs clips, transcriptions, corrections ni attributions ; la nouvelle racine reste non indexée.

Depuis la bibliothèque, on choisit un fichier, plusieurs fichiers, un sous-dossier récursif ou toute la racine. La sélection est prévisualisée puis confirmée, et les fichiers demandés restent en attente sans Worker. La file expose `scan_requested` et les codes d’erreur : elle distingue « en attente d’un Worker », « en attente de configuration », « indexé », « manquant » et « à vérifier », au lieu d’afficher un fichier `ready` comme indexé. Des sous-dossiers se créent depuis la bibliothèque et servent de destination d’import.

Un upload nommé peut viser un sous-dossier ; sa destination et son état de publication sont persistés et relistés après rechargement. Un nom déjà présent demande confirmation avant écrasement ; le nouveau contenu remplace le fichier et devient un nouveau clip, et les résultats de l’ancien contenu ne sont supprimés que si aucune autre copie ne les utilise. Une publication interrompue est reprise de façon idempotente (contenu déjà installé reconnu par son empreinte, staging republié), une boucle serveur la reprend, et l’interface permet de réessayer ou d’annuler un envoi complet non publié. Annuler ou expirer un envoi libère son intention pour ne plus bloquer un changement de racine.

La photo de profil (JPEG/PNG, 5 Mo, 16 mégapixels, recadrage carré 256 px, sans métadonnées) s’ajoute, se remplace et se retire depuis le profil, avec les textes dans les deux catalogues. L’écran d’administration des dossiers ne montre plus que les sources historiques supplémentaires ; la racine gérée n’y est plus dupliquée.

Voir l’[ADR 0017](adr/0017-racine-video-unique-et-publication-nommee.md) et la [documentation utilisateur](video-root.md).

### Vérifications

- Tests unitaires : `go test ./internal/httpapi ./internal/library ./internal/uploads ./internal/config`.
- Tests d’intégration ciblés : `TestFoundation/resumable_uploads` couvre la reprise après interruption avant et après renommage, le changement de destination pendant le transfert, l’échec de permission puis la reprise, deux uploads concurrents vers le même chemin, la conservation des copies distinctes et le changement de racine bloqué puis appliqué. `account_preferences` et `folder_sources` restent verts.
- Parcours Playwright `video-storage.spec.ts` : configuration sans scan automatique, prévisualisation et attente sans Worker, création de sous-dossier, import dans un sous-dossier, écrasement confirmé, changement de racine et photo de profil. Les parcours `admin-folders`, `import`, `account-settings` et `shell-accessibility` restent verts ; axe-core ne signale aucune violation et les cibles tactiles visibles font au moins 44 px.
- `npm --prefix web run lint`, `npm --prefix web run build`, `scripts/check_generated.py`, `scripts/check_error_codes.py`, `gofmt` et `go vet` passent.
- Les fixtures sont synthétiques ; aucun scan ni traitement de la vraie bibliothèque n’a été lancé.

### Restant et écarts

- La phase 7 (benchmark réel) garde son autorisation propre et n’a pas été lancée.
- Le parcours « Worker créé puis battement de cœur » n’est pas simulé de bout en bout : le test vérifie l’attente sans Worker et l’état de la file ; la reprise réelle après connexion d’un Worker reste à valider sur le PC de jeu.
- L’inspection visuelle a porté sur les captures Playwright et les contrôles axe-core, pas sur un appareil physique ni un autre moteur de navigateur.
