# 0066 — Adaptateur de séparation des voix : SepFormer sur la famille SpeechBrain

Date : 2026-11-11. Statut : proposé, à relire par l'utilisateur. Spec : [paroles simultanées](../specs/paroles-simultanees.md), section 2 ; ticket #401. Révise l'[ADR 0038](0038-modeles-locaux-pilotes-par-snitch.md), l'[ADR 0055](0055-runtimes-vocaux-approuves.md) et l'[ADR 0056](0056-engines-a-la-demande.md) sur un point : la séparation n'est plus « sans adaptateur ». Leurs autres règles s'appliquent sans exception.

## Contexte

Le Catalogue de modèles vocaux connaît le type séparation depuis l'ADR 0038, mais une contrainte de la base interdisait qu'un tel modèle soit supporté et le Worker le refusait. La Famille d'engine SpeechBrain s'installait déjà pour ce type sans rien exécuter (ADR 0056). La spec demande de pouvoir installer un modèle de séparation pour le mesurer, avant toute décision de l'utiliser dans la bibliothèque.

## Décision

1. **Un adaptateur de séparation pour la famille SpeechBrain**, limité à l'architecture SepFormer (`speechbrain.lobes.models.dual_path`). Il suit les garde-fous fixes de l'ADR 0056 : aucun code du dépôt, `hyperparams.yaml` lu comme une donnée (jamais HyperPyYAML ni `from_hparams`), poids parcourus puis lus avec `weights_only`, rien n'est téléchargé à l'inférence. Les limites du modèle (fréquence, nombre de voix) et son architecture sont lues dans son propre fichier, contre une liste fermée de classes et de paramètres ; tout le reste est refusé avec une cause stable.
2. **Les règles des autres types s'appliquent.** Ordre explicite de l'Administrateur, contrôle des ressources avant l'ordre (ADR 0055), engine de la famille commandé avec le modèle, inférence de test avant l'état prêt. L'inférence de test sépare un mélange synthétique de deux voix (deux sons harmoniques de hauteurs différentes) et exige autant de sources que le modèle sépare de voix, chacune à la fréquence du modèle et de la longueur du mélange, dont une au moins porte du signal. Elle prouve que le modèle tourne, pas qu'il sépare bien.
3. **Un modèle de séparation supporté déclare ses limites.** Le catalogue porte le nombre de voix séparées et la fréquence d'échantillonnage ; la base refuse un modèle de séparation supporté qui ne les déclare pas. Ces valeurs viennent de la documentation d'origine, consignée ci-dessous, et l'API comme l'interface les affichent.
4. **Être prêt n'ouvre aucun travail de la bibliothèque.** Aucune étape de l'Analyse de clip ne sépare, aucun modèle de séparation ne peut devenir Modèle vocal actif, et le Worker n'a pas d'étape de séparation. Un modèle prêt est seulement utilisable par un travail qui le demande explicitement (le benchmark du ticket #402). Le branchement dans l'analyse reste hors périmètre de la spec.
5. **Un point d'entrée unique côté Worker**, indépendant de l'installation : `voice_separation.separate` (un fichier audio et une liste de fenêtres en entrée, les sources du modèle pour chaque fenêtre en sortie). Un appel ne lance qu'un processus d'inférence, ne charge le modèle qu'une fois et ne décode le fichier qu'une fois, quel que soit le nombre de fenêtres ; une fenêtre invalide fait échouer tout l'appel avant toute séparation. Un modèle absent, jamais prouvé, ou dont l'engine a disparu donne un refus explicite et ne déclenche ni téléchargement ni installation.

## Modèles candidats et faits retenus

Faits relevés le 2026-11-11 sur Hugging Face : métadonnées de l'API (`/api/models/<dépôt>`), fiche du modèle (`README.md`) et `hyperparams.yaml` du commit épinglé. Aucun poids n'a été téléchargé.

| | `speechbrain/sepformer-whamr16k` | `speechbrain/sepformer-wsj03mix` |
|---|---|---|
| Commit épinglé | `21a5b500c6f52fddc387c5d9e5fb13ffd6f039c5` | `21e78b44a378092c34fff32ba8e788a73584f474` |
| Fréquence (`sample_rate` du fichier, confirmée par la fiche) | 16 000 Hz | 8 000 Hz |
| Voix séparées (`num_spks`) | 2 | 3 |
| Données d'entraînement (fiche) | WHAMR! à 16 kHz, « a version of WSJ0-Mix dataset with environmental noise and reverberation » | WSJ0-3Mix, sans autre précision dans la fiche (mélanges artificiels de trois voix du corpus de parole lue WSJ0) |
| Langue déclarée (fiche) | anglais | anglais |
| Résultat annoncé sur le jeu de test (fiche) | 13,5 dB SI-SNRi | 19,8 dB SI-SNRi |
| Licence, accès | Apache-2.0, non restreint | Apache-2.0, non restreint |
| Taille du dépôt | 319 Mo | 320 Mo |

- **WHAMR! 16 kHz est le candidat principal** : parmi les six dépôts SepFormer de séparation examinés (`sepformer-whamr16k`, `-whamr`, `-wsj02mix`, `-wsj03mix`, `-libri2mix`, `-libri3mix`), c'est le seul qui travaille à 16 kHz, la fréquence de la transcription, et l'un des deux entraînés avec bruit et réverbération. Il ne sépare que deux voix.
- **WSJ0-3Mix est le candidat à trois voix**, le cas décrit par la spec (un troisième Locuteur local parle sous les deux autres). Il travaille à 8 kHz : la bande passante des sources est réduite de moitié, ce qui peut coûter à la transcription.
- Les deux sont entraînés sur de la parole lue en anglais, mélangée artificiellement. Un clip de jeu compressé, en français, avec du son d'ambiance en est loin : un résultat médiocre est une issue attendue (spec).
- Un séparateur rend toujours le nombre de voix pour lequel il a été entraîné : sur un passage à une seule voix, les sources en trop contiennent des restes. C'est à l'appelant de cibler les Chevauchements (#402).
- Les dépôts livrent aussi des restes d'entraînement (`optimizer.ckpt`, environ 205 Mo, soit les deux tiers de la taille). L'adaptateur les tolère sans jamais les ouvrir.

Écartés : `sepformer-whamr` (8 kHz, même jeu que le candidat principal, et son dépôt contient un dossier, que l'adaptateur refuse), `sepformer-wsj02mix`, `sepformer-libri2mix` et `sepformer-libri3mix` (8 kHz, lecture propre, sans avantage sur les deux retenus ; la fiche de Libri3Mix annonce deux résultats différents).

## Alternatives écartées

- **Charger par `SepformerSeparation.from_hparams`.** C'est la voie documentée par SpeechBrain, mais elle exécute le YAML comme un programme : contraire aux garde-fous de l'ADR 0056.
- **Accepter tout dépôt SepFormer et lire ses limites à l'ajout.** L'inspection de Snitch ne lit que les métadonnées publiques, qui ne déclarent ni la fréquence ni le nombre de voix. Les limites affichées seraient devinées. La table des modèles de séparation supportés est donc fermée : ajouter une variante demande une ligne, après lecture de sa documentation. C'est un écart assumé au principe « un adaptateur par bibliothèque, aucun code par modèle » de l'ADR 0056 ; l'adaptateur du Worker, lui, reste générique.
- **Réutiliser `max_speakers` pour le nombre de voix.** Ce champ est un maximum toléré par un modèle de diarisation ; un séparateur rend un nombre fixe de sources. Deux champs propres évitent la confusion.
- **Brancher la séparation dans l'Analyse de clip dès maintenant.** Ce serait du code pour un modèle non prouvé (spec, hors périmètre).
- **Normaliser chaque source à son maximum**, comme `separate_file` de SpeechBrain. Une source presque vide serait amplifiée jusqu'à la pleine échelle. Les sources gardent leurs niveaux relatifs : un seul gain commun, appliqué seulement si un échantillon saturerait.

## Conséquences

- La contrainte de la migration 00097 est levée (migration 00151) et remplacée par l'exigence des limites. Les deux candidats sont inscrits au catalogue ; rien n'est installé sans ordre. Une référence de séparation déjà ajoutée pour l'un de ces dépôts devient ce candidat.
- La clause de dispatch (`RuntimeReadyClause`) ne rejette plus la séparation : un modèle prêt, lié à son engine prêt, est éligible pour un travail qui le demande. Aucun travail de production n'existe pour ce type.
- Un seul modèle de séparation par Worker (règle commune). Le remplacement sur place reste refusé pour ce type : on retire, puis on installe.
- Une fenêtre de plus de 60 s est refusée, jamais tronquée : le réseau tient toute la fenêtre en mémoire. Un manque de mémoire GPU pendant l'inférence remonte comme `gpu_out_of_memory`.
- **Non prouvé.** Le chargement a été exécuté avec la vraie bibliothèque (SpeechBrain 1.0.3) sur des poids aléatoires : l'architecture décrite par le fichier publié se construit, ses poids passent le contrôle et se chargent en mode strict, et un mélange ressort en autant de sources que de voix. Aucun poids publié n'a été chargé, aucun GPU n'a été utilisé et la qualité de séparation n'a pas été mesurée : la première installation réelle et le benchmark (#402, #403) en sont la preuve. Les points de contrôle de 2021 pourraient être dans un format que le contrôle des poids refuse ; l'installation échouerait alors avec `runtime_incompatible` et le détail `checkpoint_unsupported_format` dans le journal du Worker.
