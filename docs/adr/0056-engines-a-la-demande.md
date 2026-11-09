# 0056 — Engines à la demande

Date : 2026-11-09. Statut : accepté. Spec parente : #368 ; ticket : #369. Complète l'ADR 0055 sans le supprimer.

## Contexte

L'ADR 0055 n'autorise l'exécution d'un moteur vocal non embarqué qu'à travers une Release d'engine signée : un éditeur de confiance signe un manifeste de roues avec son empreinte, le Super-administrateur l'importe, puis un Administrateur l'installe. Chaque adaptateur est écrit pour une Variante de modèle épinglée octet pour octet. Le registre est vide par défaut : l'onglet Engines affiche « No approved release matches » et un ordre de modèle échoue avec un message qui ne dit pas qu'un engine manque.

L'utilisateur est l'opérateur unique de sa propre installation. Il veut essayer les modèles qu'il découvre (Nemotron-3, TitaNet, ECAPA-TDNN, SepFormer, ResNet293-LM…) sans produire, signer et importer une enveloppe pour chacun, et sans qu'un modèle nouveau exige du code. La chaîne de signature protège contre un éditeur tiers qui n'existe pas ici : son coût est mal placé.

## Contradiction explicite avec l'ADR 0055

L'ADR 0055 pose qu'aucun code d'engine n'est exécuté sans avoir été approuvé par signature et figé par empreinte (« jamais d'exécution non validée »). Cette décision-ci la contredit pour le parcours par défaut : un engine à la demande est résolu par pip sur le Worker, sans signature, sans manifeste figé et sans empreinte connue à l'avance. L'ADR 0055 n'est pas abrogé : le parcours signé reste implémenté et utilisable, mais il passe sous un bloc « Avancé » réservé au propriétaire. Les termes Runtime approuvé, Release d'engine, Enveloppe de release et Variante de modèle épinglée restent valables pour lui. Le parcours par défaut n'est plus le parcours signé.

## Décision

- **Un engine est une famille de bibliothèque et une version.** Les familles supportées sont NeMo, SpeechBrain et WeSpeaker. Une table côté Snitch associe chaque famille à une bibliothèque, une version par défaut, les tâches couvertes (diarisation, empreinte, séparation) et ses contraintes de matériel. Un modèle du catalogue est rattaché à une famille.
- **Installation par pip dans un environnement virtuel isolé** par famille et par version, sous le volume des modèles. Plusieurs versions coexistent côte à côte sur un Worker, afin que deux modèles aux exigences différentes aient chacun la leur.
- **Le `pip freeze` de l'installation terminée est conservé** : il documente l'état exact des paquets résolus et sert de référence pour diagnostiquer ou revenir en arrière.
- **Un adaptateur par bibliothèque, pas par modèle.** Chaque adaptateur charge tout modèle de sa famille et respecte les contrats de sortie existants (tours de parole, vecteur normalisé). Ajouter une famille se limite à un adaptateur et une entrée de la table.
- **Garde-fous fixes, hors table et non désactivables par modèle** : aucun code de dépôt de modèle n'est exécuté (pas de code distant) ; les poids ne sont lus que dans un format sûr (pas de pickle exécutable) ; aucune configuration n'est lue comme du code ; aucun jeton Hugging Face ni hook de paquet n'est transmis aux installations de paquets. Un modèle qui exige du code distant est refusé avec une cause stable.
- **L'installation est proposée avec le modèle, après confirmation.** Choisir un modèle dont l'engine manque affiche, dans la même confirmation, l'engine, sa version, le disque et la durée estimés. Une fois confirmés, l'ordre d'engine puis l'ordre de modèle sont créés ensemble ; le modèle attend son engine. Un engine prêt ou en cours est réutilisé. Rien ne s'installe sans confirmation, et une reconnexion n'installe jamais rien.
- **Une famille non supportée est refusée avec un message distinct**, avant tout ordre, pour ne pas laisser croire à un problème d'environnement. Les causes d'échec d'installation (engine absent, paquet introuvable, résolution impossible, compilation échouée) sont aussi distinguées.
- **Séparation : l'engine est installable, la fonction reste désactivée.** L'engine SpeechBrain s'installe pour préparer la suite ; aucun adaptateur de séparation n'est livré et l'interface dit « engine installé, séparation pas encore disponible ». La décision de l'ADR 0055 sur la séparation inactive est inchangée.
- **Le reste du mécanisme est réutilisé** : contrôle du matériel avant tout ordre, ordres idempotents avec génération, inventaire et historique, annulation, nettoyage, désinstallation avec dépendances, mise à jour et retour arrière, inférence de test avant publication de la liaison, dispatch conditionné à l'inventaire. Les identités signée et canonique locale ne sont jamais confondues avec celle d'un engine à la demande.

## Alternatives écartées

- **Un adaptateur par modèle (variantes épinglées étendues).** C'est le modèle de l'ADR 0055 : sûr, mais tout modèle nouveau demande du code et une révision connue. Il contredit le but, essayer ce qu'on découvre, et multiplie le code à maintenir par un facteur égal au nombre de modèles.
- **Un registre signé embarqué dans Snitch.** Livrer des manifestes déjà signés supprimerait l'étape de signature pour l'opérateur, mais Snitch deviendrait l'éditeur : il faudrait auditer, signer et republier à chaque version de bibliothèque, et la clé serait celle du projet, non celle de l'opérateur. On échange une charge d'opérateur contre une charge de mainteneur, sans gain réel de confiance pour un opérateur unique.
- **Supprimer le parcours signé.** Rejeté : il reste utile à qui veut un chemin strict et reproductible, et il garde la valeur des Releases déjà importées.

## Conséquences

- **Reproductibilité perdue, assumée.** La version est résolue par pip au moment de l'installation : deux installations de la même famille et version peuvent résoudre des dépendances transitives différentes. Le `pip freeze` permet de diagnostiquer et de revenir en arrière, pas de garantir l'identique. Le parcours signé reste la voie reproductible.
- **Périmètre de confiance : l'opérateur seul.** Le choix de la famille, de la version et la confirmation reviennent à l'opérateur de sa propre installation. Ce modèle ne convient pas à une installation où des administrateurs peu fiables ordonnent des installations ; dans ce cas, le parcours signé reste la réponse.
- **Un environnement virtuel n'est pas une sandbox.** Il isole les dépendances, pas un paquet malveillant : celui-ci s'exécute avec les droits du Worker. Les garde-fous fixes réduisent la surface (code distant, pickle, jeton) mais ne la suppriment pas.
- **Dépendance au réseau et à l'index de paquets** à l'installation : disponibilité, compilation de sources et retraits de versions deviennent des causes d'échec visibles, non plus exclues par construction.
- **Deux parcours coexistent**, ce qui élargit la surface de test et de documentation ; l'interface doit rendre le parcours par défaut évident et reléguer l'import d'enveloppe sous « Avancé ».
- **Rattachement d'un modèle à une famille (#372).** Le catalogue porte une colonne `family`, vide par défaut. Vide : le contrat de l'ADR 0055 ne change pas (adaptateur embarqué sans engine, adaptateur signé avec Release). Renseignée : le modèle demande l'engine à la demande de cette famille, quel que soit le nom de son adaptateur. C'est ce qui distingue Sortformer (adaptateur `nemo` embarqué, famille vide) d'un NeMo plus récent (adaptateur `nemo`, famille `nemo`). Une famille absente de la table de Snitch, ou qui ne couvre pas la tâche du modèle, est refusée (`unsupported_engine_family`) avant toute écriture.
- **Un engine est une famille, une version et une tâche.** Une famille qui couvre plusieurs tâches (NeMo : diarisation et empreinte) donne une identité, une ligne de registre et un ordre par tâche ; l'ordre par famille accepte `kind`, la première tâche de la famille par défaut. Le dispatch d'un modèle de famille exige l'engine à la demande prêt de sa famille et de sa tâche.
- Les moteurs embarqués (pyannote, NeMo embarqué, Silero), le modèle actif global et la transition d'empreintes de l'ADR 0044 ne changent pas.
