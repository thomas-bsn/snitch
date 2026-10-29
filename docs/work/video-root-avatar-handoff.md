# Reprise — racine vidéo, scan manuel, photo de profil

État au 30 octobre 2026 : **travail en cours, non livré, ne pas fusionner en l'état**. Branche `integration/phase-6bis-ui-refonte`. Le dernier changement livré sur main reste `5b7b772` (correction HTTPS). L'utilisateur a autorisé commit et push sur main **quand le travail sera terminé**. Il a signalé ne disposer que de 10 % de quota ; ce checkpoint préserve les modifications pour reprise.

## Demande validée

- Une seule racine vidéo, renseignée dans le wizard et la configuration, par exemple `/videos`, nécessairement montée et accessible dans le conteneur.
- Sous-dossiers créables depuis la bibliothèque ; uploads vers la racine ou un sous-dossier.
- Collision de nom : prévenir, obtenir confirmation, écraser et repartir du nouveau fichier ; anciens résultats/corrections invalidés et nouveau scan.
- **Dernière correction utilisateur : aucune découverte automatique.** Un bouton permet de choisir un fichier, plusieurs fichiers, un sous-dossier ou toute la racine. Prévisualisation puis confirmation ; les dossiers sont récursifs. Les demandes restent en attente sans Worker.
- Ajouter/remplacer/supprimer une photo de profil.
- Aucun scan/test sur de vrais médias ; les restrictions de phase 7 restent en vigueur.

## Modifications présentes

- Migration 41 : `account_avatars` (PNG en base). API GET/PUT/DELETE `/api/account/avatar`, JPEG/PNG <=5 Mo, <=16 MP, recadrage carré 256 px, suppression des métadonnées. UI profil et menu du compte.
- Migration 42 : `clip_sources.managed_root` unique et `source_files.scan_requested`. Nouveau module `library/managed.go`. Les sources existantes conservent leur ancien comportement ; la racine gérée n'est pas parcourue par le watcher. `scanManaged` inspecte uniquement les chemins demandés et attend un Worker connecté. Callback serveur `scheduleResolvedTranscription` après indexation locale.
- API `/api/admin/video-root` (GET authentifié, PUT admin), `/browse` (GET authentifié), `/folders` et `/scan` (POST admin). `VideoStorage.tsx` dans bibliothèque, wizard import et administration dossiers : configuration, navigation, sélection, aperçu, file et création de dossiers.
- Migration 43 : `upload_destinations`, intention persistante de destination/version à remplacer. Création d'upload avec `destination` relatif et `overwrite_version` renvoyée par le 409. Confirmation native du navigateur ; champ de destination dans Import.
- `publishUpload` copie depuis le staging vers un fichier temporaire dans la racine, vérifie la version, renomme, lie le nouveau clip au chemin et détache l'ancien. Suppression des résultats de l'ancien clip seulement s'il n'a plus d'autres références. Le staging du nouveau fichier est supprimé après commit. Publication idempotente par hash en cas d'interruption.
- Le résolveur média vérifie taille/mtime des fichiers de racine gérée avant streaming, pour ne pas servir un nouveau contenu avec les anciennes métadonnées.
- Dernier ajout : goroutine de reprise toutes les 30 s des publications non terminées, liée au contexte d'arrêt. **Cet ajout n'a pas encore été testé en scénario d'interruption.**
- Contrat OpenAPI et clients Go/TS/Python générés. Le générateur Python avertit qu'il ne prend pas en charge les types MIME image de l'avatar (PUT absent, réponse GET non décodée) : à résoudre/documenter proprement.

## Vérifications effectuées

- `go test ./internal/httpapi ./internal/library ./internal/uploads` passe (avant les derniers ajustements de reprise).
- `DOCKER_HOST=unix:///Users/thomasboisson/.colima/default/docker.sock TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock go test -race -tags integration ./internal/httpapi -run '^TestFoundation/(account_preferences|folder_sources|resumable_uploads)$' -count=1` passe (11,8 s) : photo persistante et supprimable, racine sans scan automatique, aperçu sans mutation, attente sans Worker, traversal/symlink refusés, upload nommé, répétition de completion, confirmation d'écrasement, suppression de l'ancien clip, refus d'ancienne version.
- Build web passe (724 Ko de JS, avertissement Vite connu).
- `scripts/check_error_codes.py` passe : 147 codes bilingues.
- `go vet ./internal/httpapi ./internal/library ./internal/uploads` et `git diff --check` passent sur le checkpoint.

## À terminer avant main

1. Examiner et éprouver la publication/reprise : interruption avant/après rename et commit, fichier modifié pendant transfert, concurrence de deux uploads vers le même chemin, permissions, conservation des copies distinctes et anciennes corrections. Vérifier la persistance/reprise des destinations après reload de l'interface (destination stockée côté serveur ; la liste des uploads ne remonte pas encore ce champ). Réexaminer les cas de staging orphelin et erreurs de publication restant invisibles dans la liste des uploads terminés.
2. Tester de bout en bout un scan demandé, attente sans Worker puis reprise avec Worker, planification de transcription et réglages manquants. La file affiche actuellement `ready` comme « Indexé » même si `scan_requested` attend encore une configuration de pipeline ; exposer clairement cet état et les erreurs. `source_files.scan_requested` n'est pas encore dans la réponse de la file.
3. Les changements de racine après la première configuration sont actuellement refusés par `SetManagedRoot` (même chemin accepté). Définir et implémenter le comportement de reconfiguration sans perdre/réattribuer silencieusement les fichiers ; ce verrou est une limite provisoire, pas une décision utilisateur.
4. Finition UI : choix du sous-dossier d'import (champ texte actuellement), états d'erreur, placement, labels et traductions centralisées (plusieurs textes FR/EN sont encore inline), accès mobile/clavier. Le panneau d'administration conserve aussi la vieille UI multi-sources : adapter au modèle d'une racine sans casser les installations existantes.
5. Vérifications Playwright ciblées et inspection visuelle des nouvelles vues. Pas encore exécutées. Skill impeccable déjà lu ; context déjà exécuté ; le détecteur devra être lancé sur les fichiers modifiés. Harness `SNITCH_E2E=1` + `SNITCH_E2E_ACCOUNT=1`, `SNITCH_E2E_SOURCES=1`, `SNITCH_E2E_UPLOADS=1` ou `SNITCH_E2E_WIZARD=1`; regarder `integration_test.go` pour le nom du sous-test Playwright. Les fixtures sont synthétiques.
6. Compléter le contrat et les vérifications de génération (`scripts/check_generated.py`), lint, tests d'intégration appropriés. L'inventaire de couverture OpenAPI exclut explicitement les routes avatar/racine car couvertes par leurs scénarios dédiés ; ne pas laisser de route sans test utile.
7. Rédiger spec/ADR et docs utilisateur, README et bilan. L'écrasement consenti amende l'invariant historique « ne jamais modifier les originaux » ; l'utilisateur l'a explicitement demandé. Les autres copies d'un contenu dédupliqué doivent garder leurs résultats. Expliquer montage Docker et sauvegarde de la racine, sans inventer la syntaxe Forgeyard.
8. Après validation technique, commit final puis avancer main et push origin main (déjà autorisé). Vérifier lancement CI ; ne pas déployer ni scanner la vraie bibliothèque.

## Environnement

Repo `/Users/thomasboisson/dev/snitch`, module Go dans `app/`, npm dans `web/`, `.venv` à la racine. Docker via Colima socket ci-dessus. `scripts/generate.sh` régénère sqlc, Go/TS/Python. Le build web supprime `app/ui/dist/.gitkeep` : restaurer le fichier vide avant commit. `.claude/` était déjà non suivi au début : ne pas l'ajouter ni le modifier.
