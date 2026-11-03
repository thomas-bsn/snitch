# 0029 — Présence du Worker indépendante des jobs, perte de lease traitée tout de suite

Statut : accepté.

## Contexte

Le pulse de présence n'est envoyé qu'entre deux jobs. Pendant une transcription ou une diarisation longue, seul le renouvellement de lease part, et il ne met pas à jour `last_seen_at` : le Worker passe Offline après 30 s alors qu'il travaille. Par ailleurs, un renouvellement de lease en échec arrête son thread en silence et n'est constaté qu'à la fin du job ; le lease expire au bout de 2 minutes et un autre Worker peut reprendre le même job.

## Décision

- **Présence** : un thread dédié envoie le pulse toutes les 10 s, indépendamment des jobs et du renouvellement des leases. Le seuil Online reste 30 s (trois pulses manqués). En cas d'erreur réseau, il réessaie avec un délai croissant plafonné à 10 s et se reconnecte seul. Le test de joignabilité du serveur de transcription (toutes les 60 s) tourne de façon asynchrone à côté et ne retarde jamais le pulse ; le pulse transmet le dernier résultat connu.
- **Lease** : une erreur réseau fait réessayer le renouvellement jusqu'à l'expiration du lease. Un refus du serveur (job réattribué, annulé, Worker révoqué) annule le job immédiatement, sans envoyer de résultat. À la reconnexion, un résultat dont le lease a expiré est jeté.

## Alternatives écartées

- **Allonger le seuil Offline** : masque aussi les vraies pannes.
- **Faire valoir le renouvellement de lease comme présence** : un Worker sans job resterait sans présence, et les deux rythmes diffèrent.

## Conséquences

- Le statut Online reflète la machine, pas l'activité ; l'activité se lit sur les jobs.
- Un même job ne peut plus être traité deux fois par perte silencieuse de lease.
- Limite réelle : un appel C ou CUDA unique n'est pas interruptible depuis Python ; le Worker l'attend, écarte son résultat, et un autre Worker peut refaire le job entre-temps.
