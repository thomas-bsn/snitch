# Worker de transcription

Le serveur reste propriétaire des clips, des permissions et des résultats. Les transcriptions sont persistées dans le résultat JSON durable du job. L’inspection FFprobe expose pour chaque flux audio son index global, son titre, sa langue, son nombre de canaux et sa disposition. L’administrateur peut définir une règle de sélection par source (titre, langue, canaux, disposition et piste de repli) puis choisir une piste pour un clip particulier. Cette dérogation de clip prévaut sur la règle de source. Si plusieurs pistes correspondent, si les règles se contredisent ou si la sélection ne peut pas être résolue, le clip demande une sélection explicite: Snitch ne choisit jamais la première piste par défaut.

Chaque changement de règle ou de dérogation versionne l’entrée de transcription et ne replanifie que les clips concernés. Un ancien résultat n’est pas présenté comme courant lorsque sa piste ou sa configuration ne correspond plus à la sélection active. Le worker annonce `transcription.audio.*`, récupère le média seulement avec son lease actif, mappe l’index global FFmpeg choisi et prépare un WAV PCM 16 kHz mono. Il conserve tout l’axe audio, silences inclus, pour que les temps fournis restent relatifs au clip. Le média original n’est jamais modifié.

Dans la bibliothèque, l’administrateur peut prévisualiser un lot avec les filtres de visibilité, état, propriétaire, source et dates. Le compteur sépare les clips éligibles, déjà traités pour la version de piste et de vocabulaire courante, déjà en file, indisponibles et en attente d’une sélection de piste. L’action de planification demande une confirmation explicite et ajoute uniquement les tâches éligibles à la file existante. La pause globale, les capacités annoncées par les workers, leur capacité et les leases continuent de contrôler l’acquisition. Aucun lot ne démarre automatiquement ou la nuit.

Le panneau **Vocabulaire de transcription** permet d’ajouter, modifier, activer, désactiver ou supprimer un terme, ses variantes et sa graphie préférée. La version de l’ensemble actif est incluse dans chaque tâche de transcription. Les termes et variantes sont envoyés dans le champ `prompt` du fournisseur compatible lorsqu’il accepte ce champ. Modifier le vocabulaire ne lance pas un scan général : l’administrateur planifie une transcription ciblée depuis le clip concerné. La transcription précédente reste visible pendant l’attente et le traitement, puis est remplacée seulement après le succès de la nouvelle tâche. Les autres étapes et les métadonnées du clip ne sont pas invalidées.

## Configurer un fournisseur compatible

Le dépôt ne choisit ni ne télécharge un modèle définitif. Le worker accepte un endpoint compatible avec `POST /v1/audio/transcriptions`, par exemple un serveur local qui implémente ce protocole. Configurer dans l’environnement du conteneur :

- `TRANSCRIPTION_API_URL` : URL de base joignable depuis le worker; le chemin `/v1/audio/transcriptions` est ajouté.
- `TRANSCRIPTION_MODEL` : nom de modèle reconnu par ce serveur. Il est requis; il n’y a pas de valeur par défaut.
- `TRANSCRIPTION_API_KEY` : clé bearer facultative si le fournisseur l’exige. Le worker ne journalise ni la clé ni les détails de réponse du fournisseur.

La requête demande `verbose_json` et les granularités `segment` et `word`. Le fournisseur doit retourner `segments` et des `words` avec texte, `start` et `end` en secondes. Un résultat contenant zéro segment est une transcription réussie sans parole. Une URL ou un modèle absent, une réponse invalide ou une panne du service fait échouer uniquement la tâche en cours; la file applique ses retries bornés et l’administrateur peut reconfigurer le worker puis relancer la tâche.

`SNITCH_API_URL` reste une URL HTTPS et `SNITCH_WORKER_TOKEN` reste le secret dédié à Snitch (`RECEIPTS_API_URL` et `RECEIPTS_WORKER_TOKEN` sont encore acceptés en alias). Ces paramètres sont indépendants des identifiants du fournisseur de transcription. Utiliser un fichier d’environnement local non versionné pour les secrets.

## Modèles déclarés

À chaque pouls (environ toutes les 45 secondes), le worker déclare à l’application les modèles qu’il utilise ; les administrateurs les voient par worker dans la page Workers (« Transcription : …, Diarisation : …, Empreinte vocale : … », ou « Modèles : non déclaré »). Seuls des identifiants non secrets sont envoyés : fournisseur, nom du modèle, version et, pour la transcription, l’hôte et le port de `TRANSCRIPTION_API_URL` (`hote:port`). Le schéma, le chemin, les paramètres, d’éventuels identifiants dans l’URL, `TRANSCRIPTION_API_KEY` et `SNITCH_WORKER_TOKEN` ne sont jamais lus dans la déclaration. Le worker écarte lui-même toute valeur que l’API refuserait (caractères non sûrs, longueur excessive, forme de clé ou de jeton) ; l’API la refuse de son côté avec `invalid_worker_models`. La transcription est déclarée dès que `TRANSCRIPTION_MODEL` est défini (fournisseur `openai-compatible`). Un rôle non configuré est simplement « non déclaré » ; la déclaration est informative et ne change pas l’identité des tâches.

Les tests utilisent un transcripteur déterministe et des médias synthétiques multipistes. Ils n’appellent aucun fournisseur externe et ne sélectionnent aucun modèle final.


## Profil de pipeline et VAD

L’administration expose le profil actif, son identifiant immuable et le mode de détection d’activité vocale. Les versions de piste, vocabulaire, préparation, modèle logique, langue et paramètres VAD forment l’identité des tâches. Le VAD énergie (ou désactivé) renvoie des intervalles relatifs au WAV source ; le worker décale les timestamps des mots et segments avec les mêmes offsets avant enregistrement. Le modèle concret et ses secrets restent configurés sur le worker ; le profil `model` doit correspondre à `TRANSCRIPTION_MODEL`.

## Compose généré à la création d’un worker

Quand un administrateur crée un worker (page Workers), la réponse de `POST /api/admin/workers` contient, en plus du jeton, un objet `bundle` avec le `compose.yaml` et le `.env` du worker GPU. L’interface les affiche avec des boutons Copier et Télécharger. Comme le jeton, ils ne sont générés qu’à la création, jamais stockés ni relus : le `.env` contient le jeton, à ne pas versionner ni partager. Le texte vient d’un gabarit unique côté serveur (`app/internal/workers/bundle.go`, testé) ; le compose est statique et reste cohérent avec `deploy/worker/compose.yaml` (GPU NVIDIA, `extra_hosts`, `pull_policy: always`), mais utilise un volume nommé `snitch-models` en écriture, car les modèles se téléchargent au premier démarrage.

Le `.env` généré contient :

- `SNITCH_API_URL` : le `PUBLIC_URL` de l’instance. S’il n’est pas en HTTPS, l’interface (champ `insecure_api_url`) et un commentaire du `.env` avertissent que le worker refuse le HTTP.
- `SNITCH_WORKER_TOKEN` : le jeton qui vient d’être créé, seul secret généré.
- `SNITCH_WORKER_IMAGE` : la variable `WORKER_IMAGE` de l’application. Par défaut `ghcr.io/thomas-bsn/snitch-worker-gpu:latest` : l’application ne connaît pas de version de publication. Épingler `WORKER_IMAGE` sur un tag ou un digest publié pour maîtriser les mises à jour.
- `TRANSCRIPTION_API_URL`, `TRANSCRIPTION_MODEL` et `HF_TOKEN` : valeurs `A_RENSEIGNER` à fournir. `HF_TOKEN` ne sert qu’au premier téléchargement des modèles et n’est jamais lu par le worker.
- `SNITCH_VOICE_DOWNLOAD_ON_START=1` et `SNITCH_VOICE_CONFIRM_TERMS=1` : téléchargement des modèles au premier démarrage.

Les valeurs du `.env` sont échappées pour Docker Compose (guillemets doubles, `\`, `\"`, `\$` et retours à la ligne si un caractère n’est pas sûr). Sur le PC : placer les deux fichiers dans un dossier, renseigner les `A_RENSEIGNER`, puis `docker compose up -d`. Un navigateur peut renommer `.env` au téléchargement ; le bouton propose `snitch-worker.env` (à renommer, ou `docker compose --env-file snitch-worker.env up -d`).
