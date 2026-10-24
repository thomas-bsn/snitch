# Worker de transcription

Le serveur reste propriétaire des clips, des permissions et des résultats. Les transcriptions sont persistées dans le résultat JSON durable du job de transcription; aucun schéma supplémentaire n’est nécessaire pour cette tranche. L’administrateur choisit une piste audio visible dans le détail du clip prêt et planifie une tâche. Le worker annonce `transcription.audio.*`, récupère le média seulement avec son lease actif, mappe l’index global FFmpeg choisi et prépare un WAV PCM 16 kHz mono. Il conserve tout l’axe audio, silences inclus, pour que les temps fournis restent relatifs au clip. Le média original n’est jamais modifié.

## Configurer un fournisseur compatible

Le dépôt ne choisit ni ne télécharge un modèle définitif. Le worker accepte un endpoint compatible avec `POST /v1/audio/transcriptions`, par exemple un serveur local qui implémente ce protocole. Configurer dans l’environnement du conteneur :

- `TRANSCRIPTION_API_URL` : URL de base joignable depuis le worker; le chemin `/v1/audio/transcriptions` est ajouté.
- `TRANSCRIPTION_MODEL` : nom de modèle reconnu par ce serveur. Il est requis; il n’y a pas de valeur par défaut.
- `TRANSCRIPTION_API_KEY` : clé bearer facultative si le fournisseur l’exige. Le worker ne journalise ni la clé ni les détails de réponse du fournisseur.

La requête demande `verbose_json` et les granularités `segment` et `word`. Le fournisseur doit retourner `segments` et des `words` avec texte, `start` et `end` en secondes. Un résultat contenant zéro segment est une transcription réussie sans parole. Une URL ou un modèle absent, une réponse invalide ou une panne du service fait échouer uniquement la tâche en cours; la file applique ses retries bornés et l’administrateur peut reconfigurer le worker puis relancer la tâche.

`RECEIPTS_API_URL` reste une URL HTTPS et `RECEIPTS_WORKER_TOKEN` reste le secret dédié à Receipts. Ces paramètres sont indépendants des identifiants du fournisseur de transcription. Utiliser un fichier d’environnement local non versionné pour les secrets.

Les tests utilisent un transcripteur déterministe et des médias synthétiques multipistes. Ils n’appellent aucun fournisseur externe et ne sélectionnent aucun modèle final.
