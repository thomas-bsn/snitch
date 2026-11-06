# 0045 — Boucle de claim supervisée, et distincte de la présence

Statut : accepté. Nuance l'ADR 0029.

## Contexte

Issue #321 : après un redéploiement de Snitch, un Worker reste Online mais ne réclame plus le job `pending` prêt ; seul un `docker restart` du Worker le débloque. La cause exacte sur l'environnement réel (WSL2, NAS, mandataire) n'a pas pu être reproduite : un serveur qui s'arrête, répond une page 502, se bloque ou renvoie un JSON illisible est rattrapé par le Worker. Deux mécanismes produisent en revanche exactement ce symptôme :

- les boucles de claim tournent dans un `ThreadPoolExecutor` : une exception hors de leur liste les tuait sans trace, le `Future` la gardant jusqu'à l'arrêt, alors que la présence, qui attrape toute exception, continuait ;
- tout échec d'une étape avant le claim (pulse, configuration de transcription) annulait le cycle, sans aucun log.

Dans les deux cas, chaque appel encore émis (présence, configuration) met à jour `last_seen_at` : Snitch affiche Online, et la raison d'attente annonçait « Worker occupé ».

## Décision

- **Worker** : aucune erreur récupérable n'arrête une boucle de claim. Elle réessaie avec une attente qui double depuis l'intervalle de sondage, plafonnée à 10 s. Un superviseur relance dans le processus une boucle qui s'est terminée malgré tout, avec le même client et la même identité, sans watchdog externe. Le pouls envoyé avant le claim ne le conditionne plus. Les échecs sont journalisés sans secret : étape logique, catégorie, statut ou code ; la première erreur, puis une ligne par minute au plus, et la reprise.
- **Snitch** : chaque claim est daté (`workers.last_claim_at`), hors de la transaction du claim. Un Worker connecté, compatible et libre (moins de leases vivants que de boucles déclarées) qui n'a pas réclamé depuis 2 minutes est signalé : cause d'attente durable `worker_not_claiming`, incident `job_blocked`, statut « En ligne, ne réclame plus de tâches ». Un Worker qui revient d'une absence repart d'un compteur neuf.

## Alternatives écartées

- **Réinitialiser le client HTTP après des échecs réseau** : httpcore abandonne déjà une connexion après une erreur de transport. Un client fermé pendant qu'un autre thread s'en sert (renouvellement de lease) ajouterait des échecs.
- **Faire déclarer la santé du claim par le Worker dans son pouls** : un thread de claim mort ou bloqué ne se déclare pas lui-même. Snitch observe directement les claims qu'il reçoit.
- **Redémarrer le conteneur (#264)** : complémentaire, mais ne doit pas être nécessaire pour reprendre.

## Conséquences

- Online reste la présence de la machine (ADR 0029). La capacité à prendre des jobs se lit désormais à part, sur la page Workers et dans la raison d'attente.
- Un Worker occupé à pleine capacité par un long traitement ne réclame pas, et n'est jamais signalé.
- Le délai de 2 minutes (`workers.ClaimSilence`) est repris tel quel dans les textes d'interface et la documentation.
