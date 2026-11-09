# 0063 — Journal en direct et pause à distance d'un Worker

Date : 2026-11-09. Statut : accepté. S'appuie sur l'[ADR 0056](0056-engines-a-la-demande.md) (installation d'un engine par pip), dont il corrige un angle mort, et sur le principe posé depuis le début : un Worker appelle Snitch, Snitch n'appelle jamais un Worker.

## Contexte

Un engine à la demande échouait à l'installation avec la seule cause `runtime_incompatible` et un journal qui s'arrêtait à `engine_order_started` : la sortie de `venv`, de `pip`, de l'import et de la sonde CUDA partait dans `DEVNULL`, et rien n'était journalisé à l'échec. L'administrateur ne pouvait ni comprendre la panne ni la corriger sans aller lire les logs du conteneur sur la machine, souvent distante.

Il fallait voir le journal du Worker depuis Snitch, et pouvoir le mettre à l'arrêt sans se connecter à sa machine. Le **redémarrage géré** existe déjà : un superviseur hôte facultatif (service `systemd` utilisateur, [guide du Worker](../transcription-worker.md#redémarrage-géré-facultatif)) exécute `docker stop` puis `docker start` sur le conteneur fixe du Worker, avec motif et historique. Il reste le seul moyen d'agir sur le conteneur.

## Décision

- **Le journal remonte par poussée.** Dès sa première ligne, le Worker envoie son journal par lots (au plus 200 lignes, toutes les 2 s) à `POST /api/worker/logs`. Ce sont les lignes JSON déjà expurgées par `log.py` (jetons, URLs signées, chemins de clips) : aucune ligne n'est écrite autrement pour être envoyée. Une **session** est un processus du Worker ; ses lignes sont numérotées à partir de 0, de sorte qu'un lot renvoyé après une réponse perdue n'est stocké qu'une fois.
- **Snitch garde une fenêtre bornée** : les 5000 dernières lignes de chaque Worker. Aucune rétention illimitée, aucune table qui grossit avec un Worker bavard. L'interface montre la session en cours, depuis le démarrage du Worker (dans la limite de la fenêtre), avec un suivi en direct (case à cocher) qui interroge Snitch toutes les 2 s tant qu'elle est cochée et que l'onglet est visible.
- **Hors de portée de Snitch, le Worker ne bloque jamais.** Les lignes attendent dans un tampon borné (2000), les plus anciennes sautent en premier, un échec d'envoi n'est pas journalisé (il s'alimenterait lui-même) et l'attente entre deux essais double jusqu'à 30 s.
- **La pause est un état, pas une commande.** L'administrateur fixe l'état voulu (en pause ou non) ; le Worker le lit à son relevé (`GET /api/worker/control`, toutes les 5 s, et une fois avant son premier job) et confirme ce qu'il applique (`POST /api/worker/control/ack`). Une consigne donnée pendant que le Worker était hors ligne s'applique donc une fois, à son retour ; un Worker en pause qui redémarre reste en pause.
  - **Pause** : le Worker ne prend plus de nouveau job ; ceux en cours se terminent ; présence, ordres d'engines, journal et relevé continuent.
  - **Reprise** : lève la pause.
  - L'interface distingue l'état voulu (`paused`) de l'état confirmé (`applied_paused`) : « mise en pause… » tant qu'ils diffèrent.
- **Redémarrer, arrêter, démarrer le conteneur : le redémarrage géré, pas ce mécanisme.** Un Worker qui quitterait de lui-même pour être relancé par `restart: unless-stopped` doublerait le superviseur avec moins de garanties (pas de motif, pas d'historique, politique de redémarrage non vérifiée, boucle possible si l'accusé se perd). Un arrêt complet ou un démarrage à distance exige un processus qui survit au conteneur ; c'est le rôle du superviseur, pas du Worker. Le panneau de contrôle y renvoie.
- **Réservé aux Administrateurs**, comme le reste de la gestion des Workers, y compris la lecture du journal. Chaque demande de pause ou de reprise est enregistrée avec son auteur (colonne `requested_by`, ligne du journal du serveur).
- **L'échec d'une installation d'engine dit enfin ce qui a cassé.** Chaque étape (`venv`, `pip install`, `pip check`, import, sonde CUDA) journalise son échec avec son nom, son code de sortie et la fin de sa sortie d'erreur, expurgée comme toute ligne. Cette fin de sortie reste dans le journal du Worker : Snitch n'en reçoit toujours que la cause stable (ADR 0056), c'est la fenêtre du journal qui la rend lisible.

## Alternatives écartées

- **Snitch appelle le Worker** (webhook, SSH, WebSocket entrant) : demande une adresse joignable pour chaque Worker, souvent derrière un réseau domestique, et inverse un principe de sécurité tenu partout ailleurs.
- **Un flux (SSE, WebSocket) du Worker vers Snitch puis vers le navigateur** : plus fluide, mais ajoute un état de connexion à maintenir des deux côtés pour un gain invisible sous la seconde. L'interrogation avec curseur est reprise, rejouable et sans état.
- **Une file de commandes** : une commande périmée s'exécuterait à retardement sur un Worker qui revient hors ligne, et deux commandes peuvent se contredire. Un état voulu converge vers la dernière volonté.
- **Un redémarrage par sortie du processus** : voir plus haut. Il a été réalisé puis retiré avant publication, en découvrant que le redémarrage géré existait.
- **Le socket Docker dans le Worker pour tout piloter** : accès root à la machine pour une commodité.
- **Conserver tout le journal** : le journal d'un Worker qui boucle remplirait la base. Pour un historique long, il reste les logs du conteneur.

## Conséquences

- **« Arrêter » à distance se limite à la pause.** Un Worker en pause reste un processus vivant qui pulse : il est joignable et visible comme tel, il ne consomme pas de GPU pour de nouveaux jobs. Libérer la machine (arrêter le conteneur) n'est possible que par le redémarrage géré, qui fait un `stop` suivi d'un `start`, sans arrêt prolongé. Un arrêt prolongé et un démarrage séparés demanderaient d'étendre le protocole du superviseur ; c'est une suite possible, non décidée ici.
- **Chaque relevé de contrôle compte comme présence** (comme tout appel authentifié du Worker).
- **Les lignes du journal sont stockées en clair dans la base de Snitch**, expurgées de ce que `log.py` sait reconnaître. Ce qu'une dépendance écrit dans sa sortie d'erreur sans passer par un motif connu (un chemin non standard, un nom d'utilisateur) peut s'y trouver : la fenêtre de 5000 lignes et le droit réservé aux Administrateurs bornent l'exposition, ils ne la suppriment pas.
- **Les lignes de plus de 4000 octets sont tronquées** avant l'envoi ; une ligne coupée n'est donc plus du JSON valide et s'affiche en texte brut.
- **Un Worker plus ancien que ce changement n'envoie rien et ne lit pas la pause** : son journal est vide dans l'interface, sans erreur, et une pause demandée reste « en cours » (voulue, jamais confirmée) tant qu'il n'est pas mis à jour. Snitch ne peut pas distinguer ce cas d'un Worker lent à répondre.
