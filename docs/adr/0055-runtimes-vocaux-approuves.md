# 0055 — Runtimes vocaux approuvés

Date : 2026-11-08. Statut : accepté pour la première tranche de #326.

Le catalogue Hugging Face décrit des poids, pas une autorisation d'exécuter du code. Un registre distinct contient des manifestes de runtimes signés Ed25519. Les clés publiques sont fixées par l'opérateur (`SNITCH_ENGINE_TRUST_KEYS`, objet JSON associant un identifiant à une clé publique brute encodée base64). Seul le propriétaire peut importer ou révoquer une version ; les administrateurs peuvent consulter et sélectionner les versions approuvées.

Un manifeste schema 1 contient exactement `schema`, `engine`, `version`, `adapter_version`, `python`, `platform`, `cuda`, `license`, `kind`, `architecture`, `disk_bytes`, `vram_bytes`, `wheels`. Les roues contiennent `name`, `url`, `sha256`, `size`. Python 3.12 et Linux x86_64 correspondent à l'image Worker. CUDA vaut `cpu` ou une version majeure.mineure. Les seules URL acceptées sont HTTPS sur files.pythonhosted.org, sans identifiants ni paramètres. Toutes les dépendances transitives doivent être fournies sous forme de roues avec taille et SHA-256 ; aucune résolution réseau ou compilation de sources n'est permise.

Le document signé est transmis comme enveloppe JSON `key_id`, `payload` (octets JSON exacts en base64), `signature` (signature de ces octets en base64). L'import conserve ces preuves et refuse toute réutilisation d'une version. La signature approuve du code exécutable : son éditeur doit auditer les roues et leurs licences. Un environnement virtuel isole les dépendances ; il ne constitue pas une sandbox contre un éditeur malveillant.

L'ordre d'installation du modèle fige le manifeste choisi. Le Worker crée un environnement par empreinte canonique sous le volume des modèles, contrôle tailles, empreintes, disque, imports, dépendances et CUDA, puis exécute l'adaptateur livré par Snitch dans un sous-processus. Les tokens Hugging Face et les hooks Python/pip ne sont pas transmis. Un verrou interprocessus protège l'installation. Le marqueur de runtime n'est écrit qu'après validation ; la liaison du modèle n'est remplacée qu'après une inférence synthétique réussie. Les anciens environnements sont conservés. Ni démarrage, ni reconnexion, ni inférence ne déclenchent l'installation d'un runtime.

Les familles reconnues sont pyannote, Sortformer, TitaNet et ECAPA-TDNN. Les nouveaux chargeurs interdisent les scripts et configurations exécutables des dépôts. La séparation reste désactivée. Les chemins legacy et les moteurs embarqués continuent de fonctionner sans inscription au registre. Le modèle actif global et la transition d'empreintes ADR 0044 restent inchangés.

La révocation interdit les nouveaux ordres et annule les livraisons non acquittées. Les installations acquittées peuvent terminer et les jobs existants ne sont pas interrompus. Ce n'est pas une suppression distante immédiate de code déjà installé.

## Limites de cette tranche

Le provisionnement est associé aux ordres de modèles existants. Le cycle de vie autonome des engines (inventaire par Worker, suppression après drainage, quotas cumulés, nettoyage et annulation coopérative du provisionnement) reste à livrer. Une mise à jour de runtime pour des poids déjà Ready n'a pas encore d'action dédiée. La synchronisation retire la déclaration Ready si la preuve locale du runtime disparaît ; cela ne constitue pas une nouvelle inférence de validation.

Aucun bundle de roues de production n'est publié par ce changement. TitaNet/ECAPA nécessitent encore des releases signées testées avec de vrais poids, et une validation NAS/WSL2/RTX 4070 ainsi que les mesures RAM/VRAM et latence. Les tests synthétiques ne remplacent pas ces preuves. L'epic #326 reste ouvert.
