# 0036 — Mise à jour automatique du Worker depuis Snitch

Statut : accepté. Complète l'[ADR 0021](0021-installation-worker-une-commande.md) (installation et mise à jour du Worker) ; ne modifie ni la révocation ni le nettoyage local.

## Contexte

L'image du Worker est publiée sous `ghcr.io/thomas-bsn/snitch-worker`. Le défaut et le compose d'exemple suivent `latest`, réécrivable à chaque publication, et le lanceur installé fige l'image de l'invitation. Un Worker ne reçoit donc jamais une nouvelle version sans intervention sur sa machine. Le Worker tourne en UID 10001 sans accès Docker et ne doit pas en avoir : une mise à jour exige un agent séparé, sur l'hôte, qui dispose de Docker.

## Décision

- **Versionnage** : une version est une étiquette d'image `vX.Y.Z` de `snitch-worker`. Chaque publication du Worker depuis `main` en reçoit une automatiquement, dans le workflow existant, par un seul build publié sous `latest`, `sha-<commit>` et `vX.Y.Z` (même image). La source de vérité est l'ensemble des tags Git : `worker/vX.Y.Z` (automatiques) et `v*` (releases manuelles, qui publient aussi le Worker sous leur tag). La première version est `v0.1.0`, puis PATCH + 1 sur le plus haut tag connu des deux familles. Le tag `worker/vX.Y.Z` est créé sur le commit publié par l'API des références, qui refuse un tag existant : ce refus sert de verrou entre publications concurrentes ; une relance du même commit réutilise son tag. Le préfixe `worker/` évite le déclencheur `v*` et les tags créés avec `GITHUB_TOKEN` ne lancent pas de workflow : aucune boucle. Une release manuelle qui viserait une version déjà prise par une publication automatique pour un autre commit échoue. Une étiquette publiée n'est jamais réécrite ; une correction publie une nouvelle version. La version du Worker est indépendante de celle de l'application (`snitch` n'est versionnée que par les tags `v*` manuels) : l'avertissement de compatibilité ci-dessous reste donc indicatif. Le défaut `latest` du lanceur et du compose d'exemple reste en place dans cette évolution : une mise à jour choisie dans Snitch l'emporte sur lui, mais le défaut lui-même doit être changé dans une évolution dédiée.
- **Version cible** : un réglage unique, dans Snitch, contient la version cible du parc de Workers. Vide, aucune mise à jour automatique. Valeur choisie dans le catalogue, ou saisie manuelle, validée par `^v\d+\.\d+\.\d+$`. Un retour à une version antérieure est permis (rollback volontaire).
- **Image construite par le serveur** : Snitch n'accepte jamais une référence d'image libre. L'image renvoyée aux Workers est `<dépôt configuré>:<version cible>`, le dépôt étant celui de `WORKER_IMAGE` sans l'étiquette.
- **Catalogue** : les versions disponibles viennent de la liste des étiquettes du dépôt GHCR, filtrée par la même regex, triée en semver et mise en cache quelques minutes. Si GHCR est injoignable, l'administrateur peut saisir une version manuellement.
- **Agent local** : un script généré par le lanceur (`updater.sh`) est planifié sur l'hôte (cron) toutes les 10 minutes. Il lit la cible avec le jeton durable du Worker, compare à l'image du conteneur en cours, tire la nouvelle image, puis recrée le conteneur avec les mêmes arguments que le lanceur (GPU, `--add-host`, fichier d'environnement, volume). Il ne monte pas le socket Docker dans le Worker.
- **Épinglage** : l'agent enregistre le digest de l'image tirée (`RepoDigests`) et le renvoie avec le statut. Il ne relance jamais une image sans son digest connu.
- **Rollback** : après le redémarrage, l'agent attend 45 secondes puis vérifie que le conteneur tourne et que son compteur de redémarrages n'a pas augmenté. En cas d'échec, il relance l'image précédente (digest mémorisé) et signale `rolled_back`.
- **Statut** : l'agent envoie à Snitch un état (`idle`, `pulling`, `restarting`, `updated`, `rolled_back`, `failed`), la version, le digest et un code de cause stable, sans texte libre. Snitch affiche le dernier statut par Worker.
- **Compatibilité** : Snitch affiche un avertissement quand la version cible diffère de la version de l'application ; il ne bloque pas. Une garde de compatibilité bloquante exigera une décision distincte.

## Alternatives écartées

- **Socket Docker dans le Worker** : donnerait à un processus non privilégié le contrôle de l'hôte. Refusé par l'ADR 0021.
- **Watchtower ou service tiers sur l'hôte** : dépendance supplémentaire, pas de statut ni de rollback piloté par Snitch.
- **Suivre `latest`** : une publication non choisie se propage sans contrôle.
- **Image libre saisie par l'administrateur** : une cible arbitraire exécutée en root sur chaque machine. Refusé.
- **Vérification de santé uniquement par le pouls Snitch** : fiable, mais une panne du réseau ferait croire à une mise à jour ratée. Le compteur de redémarrages est retenu comme vérification locale ; le pouls reste visible dans l'interface.

## Conséquences

- La confiance repose sur le registre GHCR et sur le HTTPS de l'instance. Une étiquette réécrite ou un compte GHCR compromis change l'image servie ; l'épinglage par digest limite les dégâts à la première installation d'une version, pas aux relances.
- L'agent tourne en root (accès Docker) sur la machine. Son script est écrit par le lanceur en mode `700`, son jeton en `600`.
- Un Worker hors ligne ne se met pas à jour ; il rattrape la cible à son retour.
- Le lanceur lit la cible à chaque reprise, donc rejouer la commande met aussi à jour.
- Le catalogue est vide tant qu'aucune publication du Worker n'a eu lieu depuis cette évolution ; les images déjà publiées (`latest`, `sha-*`) n'ont pas de version rétroactive.
- Une version réservée dont le build échoue laisse un trou dans la numérotation, sans image associée : le catalogue lit GHCR, il ne l'affiche pas.
- Chaque `main` qui touche le Worker crée une version : le catalogue grossit vite et l'administrateur choisit parmi des versions peu différenciées. Un élagage des anciennes versions relève d'une évolution dédiée.
- Rejouer un ancien commit depuis l'interface GitHub republie `latest` sur cette ancienne image (comportement préexistant, inchangé).
- Une révocation ne retire pas la tâche cron de l'hôte : l'agent reçoit un 401 et se tait. Le nettoyage de la planification relève de l'administrateur (ligne `updater.sh` dans `crontab -l`) ou d'une nouvelle exécution du lanceur. À traiter dans une évolution.
- Sans `crontab` sur l'hôte (certaines distributions WSL2 minimales), le lanceur l'indique et les mises à jour exigent de relancer la commande.
- Sans cible, la commande d'installation garde l'image de l'invitation, qui reste par défaut `ghcr.io/thomas-bsn/snitch-worker:latest` tant qu'aucune version n'est choisie. Le défaut n'est pas changé ici.
- Vérifications : tests Python de la réservation (`scripts/test_reserve_worker_version.py` : première version, incrément, relance, concurrence, jamais d'écrasement, garde des releases manuelles), tests Go du catalogue, de l'API (`TestFoundation/worker_update`), du lanceur (cible, dépôt étranger refusé, planification idempotente) et de l'agent (cible absente, mise à jour, rollback, échec de tirage, empreinte inconnue, dépôt étranger).

## Addendum (#358) : auto-mise à jour de l'updater et images par digest

- Un Worker lancé par `repo@sha256:…` est reconnu comme du même dépôt que `repo:vX.Y.Z` ; un dépôt étranger reste refusé (`target_mismatch`).
- `GET /api/worker/update` annonce `updater_sha256`. Si le hash du `updater.sh` local diffère, l'updater télécharge `GET /api/worker/updater` (jeton du Worker), n'installe le script que si son hash correspond à l'annonce et qu'il passe `bash -n`, le remplace de façon atomique et poursuit avec la nouvelle version. Échec : cause `updater_update_failed`, la mise à jour de l'image n'est pas bloquée. Jeton, `worker.env`, volume et conteneur ne sont pas touchés.
- Limite : un `updater.sh` antérieur à cet addendum ne sait pas se mettre à jour et ne lit que le JSON de la cible ; le serveur ne peut pas le réparer seul. Une seule exécution de la commande d'installation (nouvelle invitation) suffit : elle télécharge désormais toujours le lanceur courant (un lanceur conservé réinstallait l'ancien updater) et conserve identité, jeton et volume `/models`. L'écran Mises à jour l'indique sur `target_mismatch`.
- Upgrade du backend : tout updater déjà conforme se met à jour au prochain sondage (10 min), sans intervention.
- Confiance : le hash et le script viennent du même serveur authentifié. Le hash protège contre la corruption, pas contre un serveur compromis ; il n'y a pas de signature. C'est cohérent avec le modèle existant, car ce serveur choisit déjà l'image que le Worker exécute. Une signature hors-ligne reste possible plus tard.
- L'updater se rafraîchit même sans cible de version. L'échec du rafraîchissement n'est signalé que lorsqu'une mise à jour est en attente, pour ne pas figer un Worker à jour sur « Failed ».
