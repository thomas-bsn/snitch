# 0048 — Détecteur de parole Silero hors ligne et diagnostics par piste

Statut : accepté pour le ticket #338 du parent #274. Complète l'[ADR 0047](0047-plans-audio-et-revisions-compatibles.md) (Plans et diagnostics immuables) et l'[ADR 0038](0038-modeles-locaux-pilotes-par-snitch.md) pour la commande explicite de modèles. Interfaces : [Diagnostics audio](../audio-diagnostics.md).

## Contexte

Avant d'envoyer une piste à la transcription, Snitch doit savoir si elle contient de la parole. Le VAD par énergie existant (`detect_voice_spans`) découpe du bruit comme de la voix ; il ne suffit pas. Une piste « jeu » peut porter du chat ou du dialogue et une piste « micro » peut être vide : ni le titre, ni l'ordre, ni les métadonnées ne décident. Une erreur, un modèle absent ou une faible certitude ne sont jamais une absence fiable de parole.

## Décision

**Détecteur.** Silero VAD 6.2.1, graphe TorchScript CPU `silero_vad.jit` (dépôt `snakers4/silero-vad`, commit `7e30209a3e901f9842f81b225f3e93d8199902b1`, 2 272 526 octets, SHA-256 `e1122837f4154c511485fe0b9c64455f7b929c96fbb8d79fbdb336383ebd3720`). Licence MIT, aucun compte, aucun service payant. La version du détecteur est `silero-v6.2.1-<commit>` ; la version de configuration est `channels-16k-512-hi050-lo020-min096-v1`. Un adaptateur Snitch charge le fichier avec `torch.jit.load` ; ni `torch.hub`, ni le paquet `silero-vad`, ni Hugging Face ne sont utilisés. Dépendance : le `torch==2.8.0` déjà épinglé de l'image vocale complète (extra `acoustic` du paquet worker pour une image dédiée) ; aucune dépendance de plus dans l'image de base. WebRTC VAD et ONNX Runtime ont été étudiés : le premier ne donne pas de score incertain exploitable, le second n'enlève pas Torch de l'image actuelle.

**Paramètres.** Audio décodé par ffmpeg sur l'indice ffprobe réel (`-map 0:N`), PCM 16 kHz sur **chaque canal indépendamment** (une piste stéréo en opposition de phase reste détectable ; l'état récurrent est réinitialisé par canal). Fenêtres de 512 échantillons (32 ms), dernière fenêtre complétée par du silence. Parole : score >= 0,50 sur au moins 3 fenêtres (96 ms) consécutives ; les scores entre 0,20 et 0,50 et les rafales plus courtes sont de l'incertitude, jamais de l'absence. Les intervalles des canaux sont fusionnés et exprimés dans le repère du clip : le décalage du flux (`start_time` ffprobe de l'inventaire du Plan) est ajouté exactement une fois.

**Verdicts.** `speech` (intervalles non vides), `no_speech` (tous les canaux décodés en entier, tous les scores bas, signal ni quasi muet ni saturé, durée décodée cohérente avec la durée propre du flux, décalage connu), `ambiguous` (preuves insuffisantes : score intermédiaire, rafale courte, signal trop faible ou écrêté, durée incohérente, décalage inconnu), `failed` (cause stable : `model_missing`, `model_checksum_mismatch`, `runtime_incompatible`, `decode_failed`, `unsupported_audio`, `inference_failed`, `invalid_track_index`). `unexamined` est l'état d'une piste sans observation et n'est jamais envoyé par un Worker. Les scores sont des sorties de modèle, pas des probabilités calibrées ; **aucune qualité générale n'est revendiquée**, seulement le comportement testé sur des fixtures synthétiques.

**Installation explicite.** Le modèle est une entrée du catalogue (`kind=vad`, `engine=silero-jit`, `silero-vad-v6.2.1`, migration 00114) installée par une commande d'administrateur (`POST /api/admin/workers/{id}/voice-model-orders`) comme les modèles vocaux. Seul l'artefact épinglé est accepté. L'adaptateur le récupère depuis l'URL figée du commit (aucun jeton Hugging Face), vérifie taille et SHA-256 **avant** de charger l'objet TorchScript (exécutable), exécute une vraie inférence CPU, puis écrit la marque de disponibilité. Une copie manuelle hors ligne est possible : `python -m snitch_worker.audio_diagnostic install --source silero_vad.jit` ; `--download` est l'acquisition explicite en ligne de commande. Un diagnostic ne télécharge jamais : modèle absent, corrompu ou incompatible donne `failed` avec sa cause. Les causes d'une commande échouée réutilisent le vocabulaire fermé existant (`hf_network` pour un échec réseau d'acquisition, `runtime_incompatible` pour somme de contrôle ou chargement, `disk_full`, `models_volume_unwritable`, `inference_failed`).

**Job distinct, idempotent, compatible.** `audio.diagnostic.<indice réel>` est un job par piste, lié au Plan (`audio_plan_children`, version d'entrée `audio-diagnostic-<plan>-<digest>`). La mise en file est une seule instruction SQL idempotente ; rejouer ne crée rien et ne relance pas un job mort (relance = commande d'administrateur existante). Les tentatives, le bail, l'annulation et la reprise sont ceux des jobs existants ; l'observation terminale est enregistrée dans la transaction de complétion, par tentative réelle, de façon immuable (ADR 0047). Un Worker ancien ne déclare pas `audio.diagnostic.*` et ne prend jamais ce format. Un Worker récent ne le déclare que si Torch est installé, et Snitch ne lui attribue le job que si **ce Worker a rapporté le modèle épinglé prêt** (`worker_voice_models`) : sinon le job reste en attente (capacité expliquée par l'état en file du Plan et par l'absence de modèle prêt).

**Coût.** Stockage 2,3 Mo ; CPU uniquement, un thread, fenêtres bornées en mémoire (le décodage temporaire est un WAV 16 kHz) ; aucun GPU requis. Pas de mesure de performance sur une bibliothèque réelle (benchmark de phase 7 non autorisé).

## Alternatives écartées

- Énergie seule : ne distingue pas la musique ou les explosions de la parole.
- WebRTC VAD comme repli automatique : changerait l'identité du détecteur sans le dire et ne donne que des votes booléens.
- ONNX Runtime : un second moteur à épingler sans retirer Torch de l'image.
- `torch.hub.load` / paquet `silero-vad` : acquisition implicite et état global Torch.
- Déduire l'absence de parole du titre « jeu » ou de l'ordre des pistes : explicitement interdit.

## Conséquences

- Les tests acoustiques exécutent le vrai modèle (variable `SNITCH_TEST_VAD_DIRECTORY` ; ignorés sinon) sur une voix française synthétique (espeak-ng) et des signaux synthétiques ; les tests d'orchestration utilisent un détecteur injectable et ne prouvent rien sur la qualité.
- Le résultat n'est qu'une observation : le choix d'une piste et sa publication restent aux tickets #339 à #341.
- Changer de modèle ou de paramètres change `detector_version` / `detector_config_version`, donc l'identité du Plan.
