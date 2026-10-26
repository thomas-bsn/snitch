# Snitch — Worker vocal réel (complément de la phase 5)

Branche `feature/real-voice-worker`, PR vers `integration/phase-5-voice`. Cette spec n'a pas d'issue GitHub : elle complète la phase 5, dont le bilan signale que seul un fournisseur vocal factice existe.

## Problème

La phase 5 livre toute la mécanique voix (Locuteurs locaux, Personnes, rapprochement, groupes, vérification) mais le worker n'a qu'un fournisseur factice. Sur une vraie vidéo, la diarisation échoue avec `provider_unavailable`. Il n'existe pas d'API standard pour la diarisation ni pour les empreintes vocales : ces modèles doivent être embarqués dans le worker.

## Solution

1. **Diarisation réelle** : un fournisseur `pyannote` dans `worker/snitch_worker/` (modèle de référence `pyannote/speaker-diarization-community-1`), qui remplace le factice quand la configuration du pipeline vocal le demande.
2. **Empreinte vocale réelle** : un fournisseur d'empreintes (modèle d'embedding de locuteur fourni par pyannote, ou SpeechBrain si pyannote ne convient pas ; le choix est motivé dans la doc) qui produit un vecteur par Locuteur local et par tour assez long, avec durée et qualité.
3. **Packaging** : image Docker du worker avec GPU (CUDA) et les bibliothèques, configurée par `.env`, avec un exemple `compose` pour le PC (accès GPU comme un Ollama en Docker, joindre un Whisper/Ollama local via `host.docker.internal` ou un réseau Compose partagé).
4. **Visibilité admin** : le worker déclare les modèles qu'il utilise (transcription, diarisation, empreinte) ; l'application les affiche dans la page des workers pour les administrateurs.

## Décisions

- Aucun téléchargement implicite de modèle : le worker utilise un modèle déjà présent (chemin local ou cache Hugging Face préparé), l'accès doit rester vérifié explicitement par l'administrateur (`access_confirmed`, déjà en phase 5) et le jeton Hugging Face n'est jamais journalisé. Un téléchargement n'a lieu que par une commande explicite et documentée (par exemple un script `download-models`), jamais au démarrage ni pendant une tâche.
- Les importations lourdes (torch, pyannote) sont paresseuses : le worker et ses tests démarrent sans GPU ni bibliothèques ML.
- Les fournisseurs sont injectables ; les tests unitaires utilisent un pipeline simulé. Aucun test ne télécharge de modèle, n'utilise un média réel ni ne requiert un GPU. Le code réel n'est donc pas exécuté de bout en bout ici : le dire dans le bilan.
- Le fournisseur factice reste disponible pour les tests.
- Un modèle ou une version différente produit des empreintes incompatibles : le nom et la version du modèle d'empreinte sont enregistrés avec chaque empreinte (déjà géré par le contrat de phase 5).
- La configuration du worker passe par variables d'environnement. Les secrets restent sur le PC. Pas d'interface sur le PC.
- Contraintes de ressources : 12 Go de VRAM à partager ; les modèles sont chargés une fois et enchaînés, pas en parallèle sans limite.
- Aucun scan, batch ni média réel (AGENTS.md). Les dépendances sont verrouillées (`voice-runtime.lock`, `uv.lock` ; `runtime.lock` supprimé depuis) comme le reste du worker ; les licences des nouvelles dépendances et des modèles sont inscrites dans `deploy/NOTICES.md` / `deploy/licenses` selon le mécanisme existant.

## Hors périmètre

Benchmark et choix définitif de modèle (phase 7), embarquer Whisper dans le worker (il reste appelé par API), embeddings de texte (NAS), nouvelle interface sur le PC, correction de transcription (phase 6).
