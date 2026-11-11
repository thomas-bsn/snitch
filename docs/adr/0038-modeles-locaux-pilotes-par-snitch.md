# 0038 — Modèles locaux pilotés par Snitch : état désiré côté Snitch, Worker exécutant

Statut : accepté pour #265 (spec #303). Révise l'[ADR 0022](0022-modele-vocal-actif-global.md) sur trois points : le catalogue figé dans le code, le « modèle candidat » unique préparé par les Workers eux-mêmes, et « l'ancien modèle reste installé ; rien n'est supprimé automatiquement ». Conserve de 0022 le **modèle vocal actif global** et l'attribution des travaux selon l'état réel du Worker. Ne modifie pas l'[ADR 0027](0027-token-hugging-face-central.md) (jeton central), qu'il réutilise.

## Contexte

Le Worker prépare aujourd'hui lui-même les modèles vocaux : l'installation propose de les pré-télécharger et le démarrage les prépare. Le catalogue est figé dans le code. L'ADR 0022 écartait la suppression automatique pour garder un retour arrière, ce qui laisse s'accumuler des poids inutiles. L'administrateur ne peut pas ordonner à un Worker précis d'installer, de remplacer ou de retirer un modèle, ni savoir ce que chacun porte.

## Décision

- **Invariant** : Snitch possède l'état désiré, le Worker ne possède que l'état réel. Le Worker ne choisit, ne télécharge et ne supprime jamais un modèle de lui-même : ni pré-téléchargement à l'installation, ni préparation au démarrage, ni éviction automatique, ni délai de suppression.
- **Le modèle actif reste global, par type** (diarisation, empreinte, séparation). Un clip donne le même résultat quel que soit le Worker, ce que garantissait 0022. Un Worker ne prend un travail vocal d'un type que s'il a **prêt** le modèle actif de ce type.
- **Le modèle désiré d'un Worker** est la cible de préparation que Snitch lui ordonne, par type. Il peut différer du modèle actif pendant un remplacement (préparation du nouveau). Un modèle prêt mais non actif sur un Worker est mis en réserve : il n'attribue aucun travail, et l'interface le dit. Un administrateur ne peut donc pas obtenir deux résultats différents selon le Worker en choisissant des modèles différents.
- **Benchmark** : un travail peut demander explicitement un modèle précis (« modèle demandé »). Il n'est attribué qu'à un Worker qui l'a prêt, et son résultat reste dans son propre groupe d'empreintes `(modèle, version, dimensions)`, que la production ne compare jamais : aucune identification ni donnée de la bibliothèque n'en dépend. Le modèle actif reste le défaut de tout travail sans modèle demandé. Garantit un résultat identique en production tout en permettant de comparer des modèles.
- **Ordres** : Snitch enregistre des ordres (`install`, `remove`) par Worker et par type, avec clé d'idempotence, auteur et résultat. Le Worker les reçoit sur son tirage sortant habituel, les accuse et rend compte de son état réel. Un ordre est contrôlé à réception (Worker destinataire, identité, type, modèle, révision, idempotence).
- **Remplacement d'un modèle d'un même type** : télécharger et vérifier le nouveau, puis bascule décidée par Snitch quand un Worker compatible est prêt, puis ordre `remove` de l'ancien **après confirmation de la bascule** et seulement quand aucun travail en cours ne l'utilise. L'échec du nouveau, un disque trop petit pour garder les deux, ou un travail encore en cours conservent l'ancien et sont signalés à l'administrateur. Rien n'est effacé en silence.
- **Un modèle maximum par type et par Worker** après un remplacement abouti. Des types différents coexistent.
- **Catalogue en base** : références de modèles (type, dépôt, révision épinglée, adaptateur autorisé, licence, gated, besoins), éditées par le Super-administrateur. Une référence n'est exécutable que si un **adaptateur approuvé** sait la charger et l'inférer ; sinon elle est `unsupported` et non installable. Le moteur n'est jamais choisi par l'utilisateur : Snitch le déduit d'une matrice de compatibilité. Aucun code Hugging Face non approuvé, aucun `trust_remote_code`.
- **Séparation** : le type est catalogué et piloté comme les autres, mais sans adaptateur tant que #268 n'est pas livré : il ne peut pas passer prêt. Révisé par l'[ADR 0065](0065-adaptateur-de-separation-sepformer.md) : un adaptateur existe, un modèle de séparation peut être prêt, et aucun ne peut être actif.
- **Empreintes** : changer le modèle d'empreinte invalide les empreintes existantes. Le versionnage et la transition feront l'objet d'une décision distincte avant le lot qui les implémente, comme l'annonçait 0022.

## Alternatives écartées

- **Un modèle actif par Worker** : refusé comme en 0022. Les résultats dépendraient du Worker qui traite le clip.
- **Le Worker réconcilie seul vers ce qu'il juge utile** (« modèle manquant, donc télécharger X ») : état désiré éclaté entre les machines, téléchargements imprévus, impossibilité d'auditer.
- **Garder l'ancien modèle indéfiniment** (0022) : accumulation de poids, sans règle de libération. Remplacé par un retrait ordonné après bascule.
- **Éviction LRU ou délai de suppression** : suppression que ni l'administrateur ni Snitch n'ont décidée, risque de couper un travail en cours.
- **Choix libre du moteur ou liste « Autre/plugin »** : exécuterait des dépôts inconnus. Refusé.

## Conséquences

- Le travail vocal porte un modèle demandé facultatif ; sans lui, c'est le modèle actif. Les références des Personnes n'existent que dans l'espace du modèle actif : un modèle d'empreinte non actif ne peut donc servir qu'au benchmark.
- Il faut une table d'ordres et un état par Worker et par type plus riche que l'actuel (`requested`, `switching`, `removing` s'ajoutent), avec migration sans perte des trois références historiques.
- Le Worker perd sa logique de préparation automatique : il devient un exécutant d'ordres, et son état survit à un redémarrage sans rien télécharger.
- Un Worker neuf n'a aucun modèle vocal tant que l'administrateur n'a rien ordonné : ses travaux vocaux attendent, la transcription Speaches n'est pas touchée.
- La suppression d'un modèle libère de l'espace mais retire le retour arrière local : revenir à l'ancien impose un nouvel ordre d'installation, donc un nouveau téléchargement.
- L'inspection d'un dépôt Hugging Face fait un appel sortant depuis Snitch vers une adresse fournie par un utilisateur : l'hôte doit être restreint à Hugging Face, avec délai et taille de réponse bornés.
- Le glossaire gagne les termes d'ordre de modèle et de modèle désiré, et redéfinit catalogue, modèle actif et modèle candidat.

### Mise en oeuvre de l'inspection (#309)

- Hôte exact `huggingface.co` en HTTPS, sans identifiant ni port ; aucune redirection hors hôte, au plus deux dedans ; adresses internes, privées et loopback refusées à la connexion (après résolution DNS) ; délai de 8 s, réponse limitée à 1 Mio ; aucun jeton envoyé (métadonnées publiques seulement).
- Matrice d'adaptateurs, par ordre de priorité quand plusieurs conviennent (le premier gagne, jamais l'utilisateur) : 1. NeMo (diarisation ; bibliothèque ou étiquette `nemo`, ou fichier `.nemo`), 2. pyannote (diarisation et empreinte ; bibliothèque ou étiquette `pyannote-audio`). Transformers est dans la matrice cible mais sans adaptateur Worker : un dépôt qui ne convient qu'à lui reste `unsupported` ; la séparation aussi (avant #268). Un dépôt avec code distant (`custom_code`, `auto_map`) n'est jamais pris en charge.
- Le type n'est proposé que si les étiquettes en déclarent un seul (`speaker-diarization`, `speaker-embedding`, `audio-source-separation`) ; le nom du dépôt et le `pipeline_tag` ne prouvent rien. Un type déclaré de façon fiable ne peut pas être contredit à l'ajout.
- La révision épinglée est le commit lu à l'inspection ; l'ajout relit ce commit exact et refuse toute réponse qui en porte un autre.

### Mise en oeuvre du remplacement (#308)

- Un remplacement est un ordre `install` du nouveau modèle portant `replaces_model_id` (le seul cas où deux modèles d'un même type coexistent sur un Worker, le refus `voice_model_type_occupied` restant la règle sinon). Snitch tient un état de remplacement par Worker et par type : `preparing` (le nouveau est installé et vérifié, l'ancien reste actif et en service), `switching` (le nouveau est prêt), `removing` (l'ordre `remove` de l'ancien est parti).
- La bascule est la création d'une nouvelle version du pipeline vocal avec le nouveau modèle, décidée par Snitch dès qu'un Worker connecté l'a prêt et seulement si l'ancien est le modèle actif. Les travaux déjà épinglés à l'ancienne version la gardent.
- L'ordre `remove` n'est créé que si aucun travail non terminé (en attente, en cours, en reprise) n'est épinglé à l'ancien modèle ou ne le demande : il est évalué à chaque tirage du Worker et après chaque rapport. Les travaux en attente comptent, sinon ils attendraient un modèle que plus personne ne détient.
- Échec du nouveau (téléchargement, test, `disk_full` détecté par le Worker avant de télécharger d'après la taille du catalogue), retrait en échec : l'ancien est conservé, la cause est affichée, Retry recommande l'installation ou le retrait. « Annuler le remplacement » n'est possible que tant que le nouveau n'est pas en service et qu'aucun téléchargement n'est en cours. Rien n'est jamais effacé sans ordre explicite : le Worker ne retire que sur un ordre `remove`.
- Limite : seule la diarisation est basculée. Remplacer le modèle d'empreinte est refusé (`voice_replacement_kind_unsupported`) en attendant la transition versionnée (#311).
- La désinstallation explicite d'un modèle est aussi un ordre `remove` par Worker qui le porte, refusée tant qu'un travail l'utilise ; la liste de retrait que le Worker exécutait seule a disparu.
