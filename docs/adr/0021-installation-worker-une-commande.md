# Installation d'un Worker en une commande

Statut : accepté.

## Contexte

Un administrateur doit installer un Worker sur une machine Linux neuve en copiant une seule commande générée par Snitch, sans éditer de fichier, sans cloner le dépôt et sans réparer manuellement les permissions. La commande existe déjà (invitation de 15 minutes, volume nommé, UID 10001, cache Hugging Face inscriptible), mais l'invitation reste rejouable, l'interface ne voit pas les phases de l'installation, le rejeu figé garde une configuration périmée et les erreurs du bootstrap sont masquées.

## Décision

- L'**invitation d'installation** est à usage unique : elle est consommée au premier téléchargement réussi du script. Un bouton « Régénérer la commande » couvre l'échec réseau ou l'expiration.
- Snitch refuse de générer la commande pour une adresse en `http://`, sauf choix explicite de l'administrateur pour un réseau local de confiance. L'intégrité repose sur le HTTPS de l'instance et sur une image épinglée par version, jamais `latest`. Aucune signature supplémentaire.
- Le script envoie à Snitch un événement par phase (`prerequisites`, `image`, `permissions`, `gpu`, `voice-models`, `service`) sous la forme `{phase, statut, code_cause}`, sans texte libre ni secret, avec le jeton du Worker. Avant la création du service, une phase en échec n'est visible que dans le terminal.
- L'interface distingue quatre états : installation en cours, connecté (premier pouls), **voix prête** (inférence de test réussie par le Worker) et erreur (phase et action de reprise). L'absence de nouvelles est affichée comme telle, pas comme un échec. Un Worker créé sans pouls pendant 24 h est « jamais connecté » ; sa suppression est manuelle.
- Rejouer la commande relit la configuration courante depuis Snitch, réutilise volume et modèles, et ne redemande pas le jeton Hugging Face si les modèles sont présents. Le script local n'est qu'un lanceur.
- Un accès Hugging Face refusé n'interrompt pas l'installation : la transcription reste disponible, la voix est signalée indisponible avec sa cause.
- Le bootstrap expose des codes de cause stables (`volume_not_writable`, `hf_cache_not_writable`, `hf_access_denied`, `gpu_unavailable`, `network_unreachable`, `unknown`) à la place de `unexpected error (OSError)`.
- Périmètre : Linux P0. WSL2 (P1) suit dans #125. Sur un système non pris en charge, le script refuse avec un message explicite.

## Précisions décidées pendant le grillage

- L'invitation est consommée dès la demande acceptée de téléchargement du lanceur. La commande initiale ne peut pas servir à nouveau ; si le téléchargement échoue avant que le lanceur soit conservé, l'administrateur génère une nouvelle invitation. Le lanceur local est réutilisable et authentifie les reprises avec le jeton durable du Worker, conservé en mode `600`. Si ce jeton est perdu, il est irrécupérable et une nouvelle invitation crée une nouvelle identité.
- Une invitation utilisée appartient à une seule identité et une seule machine. Une autre machine exige une nouvelle invitation. La suppression d'une identité jamais connectée ne réactive pas son invitation ; après 24 heures sans pouls, l'interface indique « jamais connecté » jusqu'à suppression manuelle.
- Les échecs antérieurs à la création du service sont rapportés dans le terminal. La page d'administration montre le dernier événement reçu et indique séparément l'absence de nouvelles. Le premier pouls signifie « connecté » ; transcription et capacité vocale sont affichées séparément. Une transcription non configurée n'empêche pas l'installation.
- L'installation vocale est facultative. Si elle est demandée et que les modèles manquent, le terminal demande le token HF de façon masquée et la confirmation d'acceptation des conditions. Un refus d'accès ou le choix de ne pas installer la voix laisse la transcription disponible et expose clairement l'indisponibilité vocale. Le token ne figure ni dans la commande ni dans les journaux et est retiré après le bootstrap.
- Une erreur arrête l'étape courante, conserve les données persistantes et permet une reprise idempotente. Les événements hors ligne sont mis en attente localement sous forme de phase, code stable et horodatage, puis transmis à la prochaine connexion authentifiée. La reprise récupère la configuration courante ; le Worker ne l'active qu'après son test local.
- La mise à jour conserve identité, configuration, volume et modèles. Si la nouvelle image échoue, le lanceur tente de restaurer l'image précédente. Linux natif est le périmètre P0 ; WSL2 reste P1, macOS et Docker Desktop ne sont pas promis avant validation. « Voix prête » exige une inférence synthétique réussie, et la démonstration P0 utilise une machine neuve et des tâches synthétiques.
- La suppression d'un Worker révoque immédiatement son jeton, retire ses attributions et remet immédiatement ses travaux en cours dans la file. Les résultats tardifs sont rejetés. L'identité reste révoquée définitivement. Un endpoint limité peut confirmer la révocation sans révéler de configuration ni permettre d'action. La suppression distante ne prétend pas effacer les données locales ; le Worker hors ligne est révoqué côté serveur dès la suppression.
- L'audit de suppression conserve un nom, une date et un motif facultatif, sans secret ni inventaire sensible de la machine. Les événements d'installation détaillés sont conservés tant que le Worker existe ; après suppression, seul cet audit minimal demeure.

## Alternatives écartées

Une invitation rejouable pendant 15 minutes laisse le jeton du Worker récupérable par toute personne connaissant l'URL, présente dans l'historique shell et les journaux de proxy. Un script figé au premier téléchargement garde une image et une configuration de transcription périmées. Déduire l'état « voix prête » du seul marqueur de modèles confondrait téléchargement et validation. Une signature du script n'apporte rien quand l'origine du script et de l'interface est la même instance.

## Conséquences

Une invitation consommée ne peut pas être reprise après une coupure réseau : il faut en régénérer une. Le script en cache dépend d'un endpoint authentifié par le jeton du Worker. Le jeton du Worker reste stocké en clair dans `worker.env` (mode 600) sur l'hôte. Ajouter une phase ou un code de cause demande de faire évoluer le contrat de l'endpoint d'événements. Les tests s'appuient sur les endpoints Go, un faux `docker` dans le `PATH` et des tests pytest du bootstrap ; aucun test sur machine réelle avant autorisation.

## Mise en œuvre de la suppression (#116)

- `DELETE /api/admin/workers/{id}` (corps facultatif `{reason}`, 200 caractères au plus) révoque dans une seule transaction : le jeton cesse d'authentifier (pouls, claims, entrée, battement, résultat, échec : tous refusés), les tâches en cours repassent en `pending` avec l'événement `worker_revoked` (la tentative interrompue n'est pas comptée), l'inventaire de la machine, la configuration de transcription, les invitations et tous les événements d'installation sont supprimés. Il reste la ligne d'identité : nom, `revoked_at`, `revoke_reason`. Une seconde suppression est sans effet et garde le premier audit ; aucun jeton ne peut être régénéré et une invitation consommée ou supprimée ne se réactive pas.
- `GET /api/worker/revocation` est le seul endroit où un jeton révoqué est accepté. Il répond `{revoked, revoked_at}` et rien d'autre ; un jeton inconnu reçoit `worker_token_invalid`. Une réponse « révoqué » renseigne `revocation_acknowledged_at` : l'interface affiche alors que le lanceur a arrêté le service, sinon « nettoyage local en attente ». Aucun état ne prétend que le disque, le volume ou les modèles sont effacés.
- Le lanceur, sur un 401 de la configuration, interroge cet endpoint ; sur confirmation il fait `docker rm -f` du service, affiche en anglais et en français que volume, modèles et fichiers locaux n'ont pas été effacés, et sort avec succès. Un Worker hors ligne ou dont la machine n'a pas relancé le lanceur reste révoqué côté serveur, nettoyage en attente ; le conteneur déjà lancé continue de recevoir des 401 sans rien pouvoir faire.
- Preuves : `TestFoundation/worker_revocation` (révocation, tâche remise en file puis prise par un autre Worker, résultats tardifs refusés, confirmation étroite, audit, détails supprimés, hors ligne), `TestInstallerStopsServiceOnRevocationConfirmationWithoutErasingLocalData` (faux docker/curl : service arrêté, aucun `volume rm`).
