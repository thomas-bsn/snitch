# #326 — bilan d’implémentation et audit #357

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

Contrôle des ressources (#354) : chaque Worker déclare son matériel et l'onglet Engines l'affiche avec le disque déjà réservé. Un ordre est refusé avec une cause stable (`platform_incompatible`, `cuda_incompatible`, `insufficient_vram`, `insufficient_ram`, `insufficient_disk`) et sans rien enregistrer ; les ordres ouverts réservent leur budget disque. Une release CUDA est refusée sur un Worker sans CUDA, même si les anciens indicateurs de repli sont présents ; un runtime CPU est une release distincte, installée volontairement. Voir l'ADR 0055.
5 bis. Sur une installation d'engine en échec, l'onglet Engines propose Réessayer, Remplacer (autres versions compatibles du même engine) et Nettoyer ; sur une installation en cours, Annuler. L'annulation est coopérative : le Worker s'arrête à son prochain point sûr et le confirme à sa prochaine connexion si besoin. Le nettoyage ne supprime que les fichiers incomplets de cette installation, jamais un modèle ni un runtime validé.

Aucune clé de confiance, release signée, réinitialisation de Worker ou suppression de modèle existant n'est introduite automatiquement.

## ECAPA-TDNN SpeechBrain (#356)

Engine `speechbrain-ecapa`, famille d'empreintes ECAPA-TDNN (192 dimensions). Adaptateur : `worker/snitch_worker/voice_ecapa.py`. Release reproductible : `worker/engine-releases/speechbrain-ecapa-1.0.3-r1/` (`requirements.in`, `pylock.toml` généré par uv avec URL, taille et SHA-256 de chaque roue, `release.json`, `manifest.json` canonique). Le manifeste épingle 50 roues PyPI (torch 2.8.0 CUDA 12.8 et ses dépendances NVIDIA comprises), conserve le champ historique `cpu_fallback` sans autoriser une exécution CPU de cette release CUDA et déclare les contrats `audio-wav-flac-ogg-8k-48k` en entrée et `fingerprint-192-unit-v1` en sortie.

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

Preuve consignée par #356, non rejouée lors de cet audit : exécution en développement sur CPU (macOS arm64, torch 2.8.0, speechbrain 1.0.3) avec le commit épinglé. **Non exécuté** : l'installation du manifeste sur Linux x86_64 (les roues Linux n'ont pas été installées), CUDA, WSL2, RTX 4070.

### Validation matérielle par l'opérateur

Sur le Worker de production, après import de l'enveloppe, installation de l'engine puis du modèle par ordres explicites, lancer avec le Python du runtime installé (`<racine>/runtimes/speechbrain-ecapa/<identité>/bin/python`) :

```
PYTHONPATH=<répertoire worker> <python du runtime> -m snitch_worker.voice_ecapa_proof <répertoire du modèle> --device auto --release manifest.json
```

et coller la sortie JSON : identité du modèle (`model`), version de la release, `dimensions` (192), `raw_norm`, `repeat_cosine` (attendu proche de 1), `device`, `torch`, `cuda`, `gpu`, `peak_ram_mib`, `peak_vram_mib`, durées de chargement et d'inférence. Ne rien en déduire sur la RTX 4070 tant que cette sortie n'a pas été produite sur la machine.

## TitaNet : publication, preuve d'inférence et preuve matérielle (#355)

Contexte et décisions : [ADR 0055](adr/0055-runtimes-vocaux-approuves.md), section « TitaNet : inférence réelle et release reproductible ».

### Publication bloquée : règle des roues PyPI conservée

La release `worker/engine-releases/nemo-titanet-3.0.0-r1/` ne peut pas encore produire de manifeste signable : `antlr4-python3-runtime==4.9.3`, requis transitivement par NeMo (`nemo-toolkit` → `hydra-core` et `omegaconf`), n'existe sur PyPI que sous forme de source, alors que le registre n'admet que des roues de files.pythonhosted.org et interdit toute compilation. `python3 scripts/engine_release.py build --release worker/engine-releases/nemo-titanet-3.0.0-r1 --out manifest.json` s'arrête en le nommant (message « antlr4-python3-runtime: exactly one wheel is required ») ; les 159 autres paquets ont une roue (12,8 Gio de disque calculés avec `disk_factor` 3, mesure : 3,94 Gio de roues, 7,15 Gio décompressés). Aucune dérogation n'a été introduite. La décision confirmée lors de la reprise est de conserver cette règle et de documenter le blocage. Une roue publiée en amont ou une autre implémentation vérifiée demanderait un travail distinct ; aucune publication n’est déclarée prête.

### Budget disque corrigé pour ECAPA

La mesure consignée par le travail #355 donne 3,71 Gio de roues et 6,43 Gio décompressés pour ECAPA. Le budget antérieur de 8,42 Gio ne couvrait pas leur coexistence et garantissait un refus `disk_full`. La release ECAPA déclare maintenant `disk_factor` 3 ; son manifeste a été régénéré sans téléchargement. Un test compare ce budget à la somme des roues et à la mesure décompressée, arrondie à 6,44 Gio. Cette correction arithmétique ne vaut pas preuve d’installation Linux. TitaNet utilise également le facteur 3.

### Procédure de publication (poste de l'éditeur, jamais sur Snitch)

Même procédure et même outil que pour ECAPA (voir « Publier et signer une release » ci-dessus), avec la release `worker/engine-releases/nemo-titanet-3.0.0-r1` :

1. Régénérer le verrou si les dépendances changent : `cd worker/engine-releases/nemo-titanet-3.0.0-r1 && uv pip compile requirements.in --python-platform x86_64-manylinux_2_28 --python-version 3.12 --exclude-newer AAAA-MM-JJ --only-binary :all: --no-binary antlr4-python3-runtime --format pylock.toml -o pylock.toml` (le `--no-binary` rend seulement visible le paquet sans roue ; il n’autorise aucune publication). Relire le diff, les licences et les roues NVIDIA.
2. `python3 scripts/engine_release.py build --release worker/engine-releases/nemo-titanet-3.0.0-r1 --out worker/engine-releases/nemo-titanet-3.0.0-r1/manifest.json`, puis committer `manifest.json` (`--check` le contrôle ensuite).
3. `python3 scripts/engine_release.py sign --manifest .../manifest.json --key-file /chemin/hors/depot/cle-ed25519 --key-id <id> --out enveloppe.json` (clé en mode 600 fournie par l'opérateur, jamais committée ni envoyée à Snitch ou à un Worker ; la clé publique affichée va dans `SNITCH_ENGINE_TRUST_KEYS`), puis `verify`.
4. Import par le propriétaire dans Administration → Voix, puis les étapes d'exploitation.

### Preuve d'inférence réelle exécutée (CPU, hors production)

Preuve consignée par le travail récupéré de #355, non rejouée lors de cet audit : exécution le 2026-11-08 sur macOS arm64 (CPU), Python 3.12, torch 2.8.0, NeMo 3.0.0, modèle public téléchargé dans un répertoire temporaire hors dépôt : 34 tests de `worker/tests/test_voice_titanet.py` passés, dont les 18 d'inférence réelle (192 dimensions, norme 1, déterminisme identique, égalité exacte avec le chargeur NeMo, chemin de production et sous-processus du pont, refus des archives, configurations, checkpoints et audios incompatibles). Mesures indicatives de cette machine : chargement 3 à 4 s, inférence 0,3 s pour 6 s d'audio, mémoire résidente maximale 1,1 à 1,2 Go. Pour rejouer : `SNITCH_TITANET_TEST_PATH=<répertoire du modèle> <python avec le voice extra> -m unittest tests.test_voice_titanet` depuis `worker/`. Sans cette variable ou sans NeMo, ces tests sont ignorés avec leur raison, jamais simulés.

Limite : l'installation par roues du runtime (Linux x86_64) n'a pas été rejouée ici, et rien n'a été validé sur GPU, WSL2 ou RTX 4070.

### Preuve matérielle à faire par l'utilisateur (NAS, WSL2, RTX 4070)

Sur la machine cible, une fois la release publiée et installée par ordre d'engine, avec les poids installés par ordre de modèle, lancer avec le Python du runtime (remplacer les chemins) :

`<volume>/runtimes/nemo-titanet/<identité>/bin/python -I -c "import sys;sys.path.insert(0,'<répertoire worker>');from snitch_worker import voice_titanet;sys.exit(voice_titanet.main())" <répertoire du modèle> cuda`

puis la même commande avec `cpu`. Le programme n'utilise qu'un audio synthétique et n'écrit rien. Coller en retour la sortie JSON complète : `archive_sha256` (identité du modèle, doit valoir `e838520693f269e7984f55bc8eb3c2d60ccf246bf4b896d4be9bcabe3e4b0fe3` pour TitaNet-L), `dimensions` (192), `norm_raw` et `norm_unit`, `audio_seconds`, `load_ms` et `inference_ms`, `max_rss_bytes` (RAM), `vram_peak_bytes` et `vram_total_bytes` (VRAM, en mode `cuda`), `cuda_device`, `torch`, `torch_cuda` (12.8 attendu), `nemo`, `repeat_max_abs_diff`. Joindre aussi la version de la Release installée et la sortie de `nvidia-smi`. Les valeurs `vram_bytes` et `ram_bytes` de `release.json` sont des estimations à corriger avec ces mesures avant de signer.

## Audit #357 : corrections et preuves

La revue porte sur la branche d’intégration depuis `origin/main` (`cb74b5e64460f279935e074d4b66bdb0e8fbf635`), selon les standards du dépôt et la spec #348, y compris son dernier commentaire. Les changements de #355 ont été récupérés du worktree de relais sans modifier celui-ci. Le conflit de `engine_bridge.py` garde les deux adaptateurs.

| Constat | Gravité | Correction et preuve |
| --- | --- | --- |
| Dispatch resté sur le runtime d’installation après publication d’une mise à jour | Haute | La liaison publiée devient prioritaire dans la clause SQL commune ; régression avec ancien runtime perdu et nouveau prêt. |
| Release CUDA utilisable sur CPU via deux indicateurs historiques | Haute | Refus cohérent dans Go, le plan TypeScript, l’installation et l’inférence Python. Une release CPU distincte est nécessaire. Tests de ressources d’abord rouges puis verts. |
| Télémétrie GPU inconnue conservant `auto` ou `cpu` pour une release CUDA | Haute | Le runtime force CUDA ; l’adaptateur refuse alors un GPU absent. Test de régression rouge puis 23 tests de ressources verts. |
| Portée cochée vide ciblant implicitement le Worker courant ; plan acceptant deux versions du même engine | Moyenne | Une sélection vide reste vide ; le plan réserve un seul ordre par famille et Worker, comme le serveur. Régressions ajoutées aux E2E CI ; contrôle ciblé du plan rouge puis vert. |
| Budget disque ECAPA inférieur aux roues et à leur décompression | Haute | Facteur 3 et manifeste régénéré ; test contre la mesure consignée, sans prétendre avoir installé les roues. |
| Schéma `type: null` incompatible avec OpenAPI 3.0.3 | Haute | `nullable` et référence composée, clients régénérés ; validation des requêtes/réponses des suites engines et runtimes. |
| Manifeste ECAPA contrôlé seulement par le Worker | Moyenne | Test Go important le manifeste exact commité, signé avec une clé éphémère, et validant ses 50 roues et ses contrats. |
| Cinq specs E2E engines absentes du harnais Go de CI | Haute | Enregistrement dans la liste partagée ; test rouge puis vert vérifiant leur présence dans exactement un des cinq lots. Les premiers lots verts ne les couvraient pas. |
| Quatre onglets débordant sur mobile | Moyenne | Retour à la ligne sous 768 px ; mesure sur instance Docker synthétique : largeur document 390 px pour viewport 390 px, contre 464 px avant correction. |

Le test `voice_engine_dispatch` a subi un contrôle par mutation : remplacer `jobs.RuntimeReadyClause` par `TRUE` le fait échouer avec « weights without the runtime opened voice.fingerprint.1 ». La clause a été restaurée, puis les tests ciblés relancés. Les contraintes de causes de la migration 00127 portent l’union des migrations précédentes ; leur test d’intégration vérifie les valeurs admises. Les enums exposées passent par OpenAPI et ses clients générés ; le Worker vérifie aussi ses causes avant transmission.

Vérifications locales de cette reprise : suites Go/DB ciblées sous détecteur de races (registre, ordres, mise à jour/retour arrière, ressources, reprise, dispatch, retrait, annulation, contraintes et permissions HTTP), 66 tests Worker ciblés du cycle de vie et des ressources, 85 tests ciblés des adaptateurs et de la publication dont 43 ignorés faute de modèles ou dépendances ML, build TypeScript, reproductibilité des fichiers générés et contrôle des codes d’erreur. Les preuves CPU réelles ci-dessus proviennent des travaux précédents et ne sont pas assimilées à ces tests locaux. La suite Worker complète et les E2E sont réservés à la CI GitHub.

Audit UI : skill `frontend-audit` indisponible ; remplacement par Impeccable (audit programmatique sans constat statique) et agent-browser sur une instance Docker locale isolée, données synthétiques. Vérification de l’état vide, du Worker hors ligne, des onglets et de la portée d’ajout. Les E2E ne sont pas exécutés localement.

## Livré et écarts restant ouverts

- #349–#354 : installation autonome, inventaire par Worker, dispatch conditionnel, reprise/annulation/nettoyage, mise à jour/retour arrière d’un runtime lié, retrait avec drainage, contrôle du matériel et réservation cumulée du disque.
- #355 : adaptateur TitaNet, verrou et outil de publication, tests de refus et procédure de preuve ; publication bloquée par la dépendance sans roue. #356 : adaptateur ECAPA, manifeste reproductible et commande de preuve.
- Portée UI : « ce Worker » et « Workers cochés ». Aucun groupe persistant de Workers : écart au prototype, sans inventer cette capacité.
- VRAM insuffisante à l’inférence : échec avec `gpu_out_of_memory`, sans mécanisme d’attente. C’est la branche « échoue avec cause stable » du critère #354.
- Annulation reçue par un Worker hors ligne : reste `cancelling` jusqu’à son retour, sans forçage. Un `pip` déjà lancé termine avant le prochain arrêt coopératif.
- Modèles antérieurs à #352 : pas de version précédente connue avant leur première mise à jour. Les liaisons legacy inconnues du serveur sont protégées localement par le Worker ; celui-ci ne connaît pas la version précédente conservée côté serveur.
- Un runtime installé seulement par un ordre de modèle doit encore être inscrit à l’inventaire par un ordre d’engine explicite avant le dispatch d’un moteur non embarqué ; ce dernier ne retélécharge pas un runtime déjà validé.
- Aucune release de production signée ni installation des roues Linux de ces manifestes n’est prouvée par cette reprise. Le budget RAM/VRAM TitaNet reste estimé.
- Restent sans preuve : comparaison ECAPA/TitaNet/WeSpeaker sur parole positive et négative, installation Docker cible, NAS, WSL2, GPU et RTX 4070. Les sons synthétiques ne prouvent pas la discrimination de locuteurs. Seul l’utilisateur réalise la preuve matérielle et fournit les JSON décrits ci-dessus.

L’audit ne clôt matériellement ni #355, ni #357, ni l’epic #326. Aucun scan de bibliothèque réelle n’a été effectué. La PR #347 reste destinée à la relecture, sans fusion automatique.
