# #326 — bilan de la première tranche

Cette tranche ajoute le registre signé, le provisionnement isolé sur ordre et le branchement des adaptateurs d'empreintes. Elle ne valide pas l'ensemble de l'epic. Voir [ADR 0055](adr/0055-runtimes-vocaux-approuves.md) pour le protocole, la confiance et les limites opérationnelles.

| Défaut / risque | Gravité | Correction ou limite |
| --- | --- | --- |
| TitaNet/ECAPA refusés par la matrice et les contrats de jobs | Haute | Familles explicitement reconnues, enums générés et sélection du moteur d'empreintes indépendante de la diarisation. |
| Collision des dépendances ML dans le Python du Worker | Haute | Environnement isolé par manifeste, sous-processus d'inférence, installation hors index et sans résolution implicite. |
| Exécution de code arbitraire HF / installation non épinglée | Critique | Adaptateurs fixes, métadonnées strictes, configurations filtrées, chargement des poids sécurisé, roues signées et contrôlées. L'éditeur des roues reste une autorité de confiance. |
| Changement de version entre mise en file et exécution | Haute | Manifeste figé dans l'ordre et contrôle de son idempotence. |
| Échec de remplacement détruisant un runtime fonctionnel | Haute | Répertoires distincts, validation avant liaison et conservation des anciennes versions. |
| Modèle marqué Ready avec runtime absent lors d'une préparation | Haute | Refus explicite sans téléchargement pendant l'inférence ou une simple réconciliation. La synchronisation contrôle aussi la présence du runtime des modèles déjà déclarés Ready. |
| Mauvaise version Python dans le contrat initial | Haute | Python 3.12 aligné sur l'image Docker, tests de compatibilité. |
| Révocation sans effet sur les livraisons en attente | Haute | Annulation transactionnelle des ordres non acquittés ; drainage des ordres acquittés. |

## Exploitation

1. Configurer les clés publiques de l'éditeur dans `SNITCH_ENGINE_TRUST_KEYS` côté serveur. Garder les clés privées hors de Snitch.
2. Produire les manifestes avec des roues exactes, incluant toutes les dépendances et un budget disque couvrant téléchargement et décompression. Signer les octets JSON exacts avec Ed25519.
3. Dans Administration → Voix, importer l'enveloppe signée. La liste affiche version, contraintes et licence.
3 bis. Dans l'onglet Engines, choisir un Worker, « Ajouter un engine », cocher une version approuvée et confirmer le plan chiffré : l'ordre installe le runtime seul, sans poids. L'état de chaque Worker (demandé, téléchargement, installation, vérification, prêt, échec) et son historique y sont visibles.
4. Choisir le modèle, les Workers et la version du runtime dans l'installation existante. L'ordre est explicite ; aucun modèle n'est téléchargé par l'import du manifeste.
4 bis. Pour retirer une version : Engines, « Désinstaller » sur la ligne. La fenêtre liste les modèles liés, jobs engagés et transitions d'empreintes ; tant qu'un modèle y est lié, changer ou retirer d'abord ce modèle.
4 ter. Pour un modèle déjà prêt, onglet Engines → « Modèles liés » : choisir une version approuvée et confirmer le plan (disque du runtime, poids réutilisés) pour une mise à jour, ou « Revenir à la version précédente ». Le modèle continue de servir avec l'ancien runtime jusqu'à la réussite de l'inférence de test ; en cas d'échec la liaison précédente est conservée.
5. En cas d'échec, conserver le modèle actif précédent et consulter la cause de l'ordre. Réessayer via les actions existantes. Révoquer une release empêche les nouvelles installations ; cela ne désinstalle pas les environnements utilisés.

Contrôle des ressources (#354) : chaque Worker déclare son matériel et l'onglet Engines l'affiche avec le disque déjà réservé. Un ordre est refusé avec une cause stable (`platform_incompatible`, `cuda_incompatible`, `insufficient_vram`, `insufficient_ram`, `insufficient_disk`) et sans rien enregistrer ; les ordres ouverts réservent leur budget disque. Le repli CPU exige le contrat de la release et la configuration du Worker et n'installe rien ; un runtime CPU est une release distincte. Voir l'ADR 0055.
5 bis. Sur une installation d'engine en échec, l'onglet Engines propose Réessayer, Remplacer (autres versions compatibles du même engine) et Nettoyer ; sur une installation en cours, Annuler. L'annulation est coopérative : le Worker s'arrête à son prochain point sûr et le confirme à sa prochaine connexion si besoin. Le nettoyage ne supprime que les fichiers incomplets de cette installation, jamais un modèle ni un runtime validé.

Aucune clé de confiance, release signée, réinitialisation de Worker ou suppression de modèle existant n'est introduite automatiquement.

## ECAPA-TDNN SpeechBrain (#356)

Engine `speechbrain-ecapa`, famille d'empreintes ECAPA-TDNN (192 dimensions). Adaptateur : `worker/snitch_worker/voice_ecapa.py`. Release reproductible : `worker/engine-releases/speechbrain-ecapa-1.0.3-r1/` (`requirements.in`, `pylock.toml` généré par uv avec URL, taille et SHA-256 de chaque roue, `release.json`, `manifest.json` canonique). Le manifeste épingle 50 roues PyPI (torch 2.8.0 CUDA 12.8 et ses dépendances NVIDIA comprises), accepte le repli CPU explicite (`cpu_fallback`) et déclare les contrats `audio-wav-flac-ogg-8k-48k` en entrée et `fingerprint-192-unit-v1` en sortie.

### Ce qui est accepté, ce qui est refusé

- **Variante de modèle** : uniquement les fichiers du commit `0f99f2d0ebe89ac095bcc5903c4dd8f72b367286` de `speechbrain/spkrec-ecapa-voxceleb` (Apache-2.0), dont le SHA-256 est épinglé dans le code de l'adaptateur. Toute autre révision, tout fichier modifié, manquant, en trop, lien symbolique, sous-dossier (sauf `.cache` de Hugging Face, jamais lu) ou fichier de code (`*.py`, `*.so`...) est refusé avant tout chargement. Ajouter une variante demande une nouvelle version de l'adaptateur.
- **`hyperparams.yaml`** : jamais exécuté. HyperPyYAML n'est pas appelé. Le fichier est analysé comme des données (`yaml.compose`, rien n'est construit) : seules les valeurs YAML simples, `!ref` de noms simples (pas d'expression, pas de clé inconnue) et `!new:`/`!name:`/`!apply:` vers `speechbrain.*` ou `torch.*` sont tolérés, hors liste noire des modules qui exécutent du code ou touchent fichiers et réseau (`torch.load`, `torch.hub`, `torch.jit`, `speechbrain.utils.superpowers`, `speechbrain.utils.fetching`...). Tout autre tag (`!!python/...`, `!include`, `!cmd`, `!apply:os.system`) est refusé. L'architecture lue doit être exactement ECAPA-TDNN (canaux `[1024, 1024, 1024, 1024, 3072]`, 80 mels, 192 neurones, normalisation `sentence`) ; elle est ensuite construite par le code de l'adaptateur, pas par le fichier.
- **Checkpoint** : archive zip PyTorch uniquement ; le flux pickle est parcouru opcode par opcode et seuls `OrderedDict`, les reconstructions de tenseurs et les stockages `torch.*Storage` sont admis, puis `torch.load(weights_only=True)`. Un pickle brut ou piégé est refusé sans être lu ni exécuté.
- **Audio** : conteneurs `.wav`, `.flac`, `.ogg` ; 1 à 8 canaux (moyennés en mono) ; fréquences 8, 16, 22,05, 24, 32, 44,1 et 48 kHz (rééchantillonnées à 16 kHz) ; échantillons finis ; fichier de 2 Gio au plus. Une fenêtre de plus de 120 s est empreinte sur ses 120 premières secondes. Tout le reste est refusé (`unsupported_audio_format`, `unsupported_sample_rate`, `unsupported_channels`, `audio_undecodable`, `audio_not_finite`).
- Un refus de modèle remonte à Snitch avec la cause stable `model_load_failed`, un refus d'audio avec `inference_failed` ; le détail précis (`unsupported_revision`, `hyperparams_tag_refused`, `checkpoint_forbidden_object`...) est porté par l'exception du Worker et affiché par la commande de preuve ci-dessous. Aucun chemin ni texte de bibliothèque ne voyage.

### Contrat de sortie

Une empreinte porte `embedding_model` (dépôt) et `embedding_model_version` (révision), `dimensions` (192), un vecteur unitaire par locuteur et par tour (normalisé par le contrat commun, arrondi à 6 décimales), début et fin (donc la durée) et une qualité de 0 à 1 selon la durée parlée. Elle n'est jamais comparée à une autre famille, version ou dimension (ADR 0044) : le changement de modèle passe par la transition d'empreintes existante.

### Publier et signer une release

1. Régénérer le verrou si les dépendances changent : `cd worker/engine-releases/speechbrain-ecapa-1.0.3-r1 && uv pip compile requirements.in --python-platform x86_64-manylinux_2_28 --python-version 3.12 --only-binary :all: --exclude-newer AAAA-MM-JJ --format pylock.toml -o pylock.toml`. Relire les licences et auditer les roues : la signature approuve du code exécutable.
2. Fabriquer le manifeste, byte à byte reproductible : `python3 scripts/engine_release.py build --release worker/engine-releases/speechbrain-ecapa-1.0.3-r1 --out worker/engine-releases/speechbrain-ecapa-1.0.3-r1/manifest.json` (`--check` échoue si le fichier commité diffère de ses entrées ; un test du Worker le vérifie). Le script refuse toute source, tout autre hôte que files.pythonhosted.org et toute roue sans taille ni SHA-256.
3. Signer, sur la machine de l'éditeur : `python3 scripts/engine_release.py sign --manifest .../manifest.json --key-file /chemin/cle-ed25519 --key-id <id> --out enveloppe.json`. La clé privée (32 octets bruts, hexadécimal, base64 ou PEM PKCS#8 d'`openssl genpkey -algorithm ed25519`) vient d'un fichier fourni par l'opérateur, en mode 600, hors du dépôt ; elle n'est jamais écrite ni transmise à Snitch. Le script affiche la clé publique à placer dans `SNITCH_ENGINE_TRUST_KEYS`. Seul Ed25519 de la bibliothèque standard est utilisé ; il est vérifié contre les vecteurs RFC 8032 et une signature produite a été validée par OpenSSL.
4. Contrôler : `python3 scripts/engine_release.py verify --envelope enveloppe.json --public-key <clé publique base64>`, puis importer l'enveloppe dans Administration → Voix (propriétaire) et suivre « Exploitation ».

### Preuve d'inférence

Tests : `worker/tests/test_voice_ecapa.py` (sécurité : YAML piégés `!apply:os.system`, `!!python/...`, pickle malveillant, fichiers substitués ; toujours exécutés, PyYAML seul requis) et `worker/tests/test_engine_release.py`. Les tests d'inférence réelle sont ignorés sans `SNITCH_ECAPA_MODEL_DIR` (copie locale du commit épinglé, jamais téléchargée par les tests), torch, torchaudio et speechbrain :

```
SNITCH_ECAPA_MODEL_DIR=/chemin/modele python -m unittest tests.test_voice_ecapa
```

Ils vérifient sur de l'audio synthétique : 192 dimensions finies, déterminisme (distance cosinus < 1e-5 : seul l'ordre des réductions float32 peut varier), identité du commit, vecteur de référence des poids épinglés (distance cosinus < 1e-4 : tolérance choisie au-dessus des écarts float32 attendus entre CPU et BLAS, non mesurés entre machines, et très en dessous de celui d'un autre checkpoint ; à resserrer ou relâcher si la CI Linux la contredit), entrée stéréo 48 kHz, refus des formats, non-appel de HyperPyYAML, copie falsifiée refusée, contrat d'empreinte complet (modèle, version, dimensions, normes à 1e-5, durée, qualité) et passage par `engine_bridge`, le point d'entrée réel d'un runtime installé. Les sons synthétiques ne sont pas de la parole : les tests n'affirment aucune discrimination de locuteurs.

Exécuté en développement sur CPU (macOS arm64, torch 2.8.0, speechbrain 1.0.3) avec le commit épinglé. **Non exécuté** : l'installation du manifeste sur Linux x86_64 (les roues Linux n'ont pas été installées), CUDA, WSL2, RTX 4070.

### Validation matérielle par l'opérateur

Sur le Worker de production, après import de l'enveloppe, installation de l'engine puis du modèle par ordres explicites, lancer avec le Python du runtime installé (`<racine>/runtimes/speechbrain-ecapa/<identité>/bin/python`) :

```
PYTHONPATH=<répertoire worker> <python du runtime> -m snitch_worker.voice_ecapa_proof <répertoire du modèle> --device auto --release manifest.json
```

et coller la sortie JSON : identité du modèle (`model`), version de la release, `dimensions` (192), `raw_norm`, `repeat_cosine` (attendu proche de 1), `device`, `torch`, `cuda`, `gpu`, `peak_ram_mib`, `peak_vram_mib`, durées de chargement et d'inférence. Ne rien en déduire sur la RTX 4070 tant que cette sortie n'a pas été produite sur la machine.

## Validation et restant

Les tests hors réseau couvrent signatures, permissions, révocation, persistance des manifestes, versions concurrentes, intégrité, budgets, environnement sans secrets, absence de téléchargement implicite et compatibilité legacy. Validation locale : tests Go/DB ciblés avec détecteur de races, 352 tests Worker (21 ignorés), build TypeScript, lint et reproductibilité des fichiers générés réussis. La suite Go complète a été lancée puis interrompue après plus de dix minutes, pendant les calculs Argon2 du test existant `delete_clip` ; elle n'est pas déclarée verte. La CI doit confirmer la suite complète. Les E2E locaux sont volontairement omis à la demande de l'utilisateur et laissés à la CI GitHub.

Livré par #349 : ordre d'installation d'un engine indépendant des poids, inventaire par Worker (états, progression, causes, historique, générations) et onglet Administration → Voix → Engines. Livré par #350 : dispatch de production, de benchmark et de transition d'empreintes conditionné au modèle, à sa révision et à son runtime prêts sur le même Worker (tests Docker synchronisation, plusieurs Workers, poids sans runtime, révision périmée, runtime perdu, release révoquée). Limite : un runtime installé par un ancien ordre de modèle, absent de l'inventaire, n'ouvre le dispatch d'un moteur non embarqué qu'après un ordre d'engine explicite. Livré par #351 : réessayer, remplacer depuis un échec, annulation coopérative durable et nettoyage borné des fichiers incomplets. Livré par #353 : retrait explicite d'une release sur un Worker, refusé tant qu'un modèle y est lié (modèles, jobs engagés et transitions d'empreintes affichés), livré après drainage, idempotent, reprenable et tracé avec son auteur ; le Worker ne retire que la release désignée et jamais un runtime encore lié à un modèle. Limites : le lien modèle-release se lit dans l'ordre d'installation du modèle (un modèle installé avant les runtimes approuvés n'a pas de lien côté Snitch ; seul le contrôle du Worker le protège) ; la garde d'un nouvel ordre de modèle pendant un retrait n'a pas de test d'intégration dédié. Livré par #352 : mise à jour du runtime d'un modèle Ready et retour à la version précédente (ordres explicites, poids réutilisés sans téléchargement, installation à côté de l'ancienne, liaison publiée après inférence de test, échec sans effet sur la liaison, aucune cible révoquée, historique, section « Modèles liés » de l'onglet Engines). Livré par #354 : matériel déclaré par chaque Worker, contrôle de plateforme, CUDA, VRAM, RAM et disque cumulé avant tout ordre d'engine, de modèle ou de mise à jour de runtime, repli CPU explicite. Contraintes de base : les migrations 00123, 00125 et 00126 réécrivent chacune la contrainte des causes ; la migration 00127 porte l'union de tous les états et causes (test d'intégration « voice engine constraints »). Limites : le garde-fou « version précédente d'un modèle » du retrait ne couvre que les modèles suivis depuis #352 ; le Worker ne connaît pas la version de retour arrière. Restent nécessaires pour fermer #326 : retour arrière des engines, nettoyage avec drainage, mises à jour des runtimes de modèles Ready, releases publiées et vraie inférence TitaNet (ECAPA : inférence CPU réelle prouvée en développement par #356, GPU à valider par l'opérateur), tests Docker/WSL2/GPU et benchmark autorisé. Aucun scan de bibliothèque réelle n'a été effectué.
