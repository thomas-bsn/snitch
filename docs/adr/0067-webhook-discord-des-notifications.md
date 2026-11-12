# 0067 — Demandes d'accès notifiées et webhook Discord des notifications

Date : 2026-11-12. Statut : accepté. Ticket #283. S'appuie sur les notifications persistantes (#115, #133, #174) : une notification est une action partagée, sa lecture est individuelle, et chaque changement visible publie un événement sur le sujet `user:<id>` du destinataire.

## Contexte

Une Inscription en attente n'était visible que si un administrateur ouvrait « Utilisateurs et accès > Inscriptions ». Le ticket demande de prévenir les administrateurs immédiatement, et de relayer certaines notifications dans un salon Discord. Deux risques encadrent le second point : une notification peut contenir des données privées (noms, clips, causes techniques), et Discord peut être lent, limité (429) ou absent sans que Snitch doive en souffrir.

## Décision

- **Une seule source de vérité : la table `notifications`.** Une Demande d'accès est un nouveau type (`account_request`), levé dans la transaction qui crée le compte en attente (inscription locale ou première connexion Discord). Un index unique sur le compte en fait un seul événement logique : une nouvelle tentative ou un double envoi n'en crée pas un second.
- **Destinataires : les Administrateurs et le Super-administrateur approuvés au moment de la demande.** La règle de visibilité est relue à chaque lecture : un administrateur rétrogradé ne voit plus ces notifications, même anciennes. Un membre n'en reçoit jamais.
- **La résolution suit le compte, pas l'écran.** Un déclencheur sur `users` résout la notification quand le compte est approuvé (par la liste des inscriptions ou par l'édition du membre) ou retiré. Refuser une demande retire le compte en attente (`POST /api/admin/users/{id}/refuse`) : son nom redevient libre et la notification oublie le nom, comme le reste de la trace d'un compte retiré.
- **Le webhook est un canal de livraison, pas un second moteur.** Un déclencheur `AFTER INSERT` sur `notifications` ajoute au plus une ligne par notification dans `notification_deliveries` (boîte d'envoi), si le webhook est activé et la catégorie autorisée. Rien d'autre ne crée de livraison.
- **Aucun appel réseau dans le flux qui lève la notification.** Une boucle de fond (toutes les 5 s) réserve au plus dix livraisons dues par un bail de 2 min validé avant l'appel, envoie un seul message et enregistre le résultat. Aucune transaction n'est ouverte pendant l'appel ; une seconde instance ne prend pas une livraison réservée.
- **Échecs bornés.** 2xx : envoyée. 429 : nouvel essai après l'attente demandée par Discord (`retry_after`, entre 1 s et 10 min). Erreur réseau, délai de 10 s dépassé ou 5xx : nouvel essai avec attente exponentielle depuis 30 s, plafonnée à 30 min, abandon après 6 essais. 401, 403 ou 404 : le webhook n'existe plus ou l'adresse est fausse ; la livraison échoue, le webhook se désactive lui-même (`auto_disabled` dans le journal) au lieu d'échouer indéfiniment. Autre 4xx : échec sans nouvel essai.
- **Résumé expurgé par catégorie.** Catégories : `access_requests` (par défaut), `human_actions`, `worker_offline`, `job_failures`, `job_blocked`. Le message dit seulement quel genre de chose s'est produit, avec un compte de jobs et un lien vers la page de Snitch concernée ; jamais un nom, un clip, une transcription, une cause ou un texte d'erreur, un média ou un jeton. Les mentions Discord sont désactivées. Une décision sur une demande de dossier n'est jamais relayée. Les livraisons dues d'un même lot partent en un message, les résumés identiques regroupés avec leur nombre.
- **L'adresse est un secret.** Elle est chiffrée dans `settings` avec la clé maître, jamais renvoyée par l'API (seuls `configured` et les quatre derniers chiffres de l'identifiant), jamais journalisée : les erreurs gardées sont des codes (`timeout`, `network`, `rate_limited`, `webhook_rejected`, `http_<statut>`), parce qu'une erreur de transport cite l'adresse.
- **Anti-SSRF par liste blanche.** Seule une adresse `https://{discord.com|discordapp.com|canary.discord.com|ptb.discord.com}/api/webhooks/<id>/<jeton>` est acceptée : pas de port, d'identifiants, de requête ni de fragment. Elle est revalidée avant chaque envoi et aucune redirection n'est suivie.
- **Lecture pour les Administrateurs, modification pour le Super-administrateur**, comme les autres réglages Discord. Chaque ajout, rotation, modification, suppression, test et désactivation automatique est consigné dans `notification_webhook_events`, immuable, sans l'adresse.

## Alternatives écartées

- **Appeler Discord dans la requête d'inscription** : bloquerait l'inscription sur un service tiers et l'exposerait à ses lenteurs.
- **Envoyer depuis le code de chaque type de notification** : chaque nouvelle source devrait penser à la livraison ; le déclencheur couvre les sources présentes et futures.
- **Relayer toutes les notifications telles quelles** : exposerait noms de locuteurs, clips et causes à un salon dont les membres ne sont pas ceux de Snitch.
- **Plusieurs webhooks** : aucun besoin exprimé ; une table singleton suffit et l'extension reste possible.
- **Filtrer les adresses par résolution DNS (refus des IP privées)** : la liste blanche d'hôtes Discord rend ce contrôle redondant.

## Conséquences

- Seules les notifications **levées** sont relayées : une notification rafraîchie (plus de voix à vérifier, nouveau job en échec sur la même cause) ou résolue n'envoie rien de nouveau. Une cause résolue puis revenue lève une nouvelle notification, donc un nouveau message.
- Une notification levée pendant que le webhook est désactivé n'est jamais envoyée plus tard ; désactiver le webhook ou retirer une catégorie abandonne les livraisons en attente concernées (`skipped`).
- Les liens des messages utilisent l'adresse publique vue lors du dernier enregistrement des réglages ; sans adresse, le message part sans lien.
- La table `notification_deliveries` grandit avec les notifications relayées ; une livraison disparaît avec sa notification.
- Un administrateur promu après une demande ne la reçoit pas : il la voit dans la liste des inscriptions.
