# 0039 — Synchronisation des photos de profil Discord

Statut : accepté. S'appuie sur le stockage `account_avatars` et sur les secrets chiffrés des réglages (ADR de la clé maître) ; ne change pas la connexion OAuth Discord.

## Contexte

Un compte pouvait importer une photo personnelle, mais la photo de l'identité Discord liée n'était jamais utilisée. Récupérer l'avatar exige un bot Discord (jeton de bot, distinct des identifiants OAuth) et une tâche de fond, car l'avatar change côté Discord sans que l'utilisateur se connecte.

## Décision

- **Source explicite et persistée.** `avatar_preferences.source` vaut `discord`, `custom` ou `none`. Les deux images coexistent dans `account_avatars` (clé `(user_id, source)`) : choisir une source ne détruit jamais l'autre. La table `avatar_preferences` porte aussi une `version` (validateur HTTP) et l'état de synchronisation par compte.
- **Priorité absolue à la photo personnelle.** La synchronisation n'écrit que la ligne `discord`. Elle ne lit ni ne modifie la ligne `custom` ni la source choisie. Un import personnel active `custom` immédiatement. Création de compte par Discord : source `discord`. Liaison : source `discord` seulement si aucune photo personnelle n'existe.
- **Atomicité.** Toute transition (import, retrait, changement de source, liaison, déliaison, écriture de synchronisation) prend d'abord le verrou de ligne `users` du compte (`SELECT … FOR UPDATE`) : elles s'appliquent l'une après l'autre. L'écriture de synchronisation vérifie, sous ce verrou, que le compte est toujours lié à l'identité interrogée.
- **Déliaison.** Elle arrête la synchronisation, supprime l'image Discord téléchargée (donnée issue d'une identité qui n'est plus celle du compte) et son état, garde la photo personnelle. Un compte qui affichait Discord retombe sur sa photo personnelle, sinon sur les initiales. La suppression d'un membre fait de même.
- **Le jeton du bot** est un réglage chiffré (`discord_bot_token`) avec la clé maître, jamais renvoyé ni journalisé. Il est vérifié auprès de Discord (`GET /users/@me`) avant enregistrement. Seul le Super-administrateur le configure, le teste ou le supprime (vérifié côté serveur) ; un Administrateur lit l'état et peut lancer une passe. La suppression coupe la synchronisation et garde les photos.
- **Aucune URL cliente.** Le serveur construit les adresses Discord à partir de la configuration (`DiscordAPIURL`, `DiscordCDNURL`) et d'identifiants validés par expression régulière (identifiant numérique, hash `^(a_)?[0-9a-f]{32}$`). L'image est lue avec une limite de 5 Mo, validée (JPEG/PNG, 16 mégapixels), recadrée et réencodée en PNG 256×256 sans métadonnées, comme une image importée. Aucun jeton OAuth utilisateur n'est conservé.
- **Efficacité.** Un appel profil par compte (le bot ne peut pas grouper) ; l'image n'est téléchargée que si le hash a changé. Les comptes sont traités par ordre de dernière vérification.
- **Planification multi-instances.** L'état de la planification est en base (`discord_bot_state`) : une passe est due si la précédente a commencé il y a plus de 24 h. Une passe prend un bail (`locked_until`, renouvelé à chaque compte) par un `UPDATE … WHERE` atomique : une seule instance l'obtient, un redémarrage ne perd ni ne répète rien, un bail mort expire. « Sync Now » force une passe mais respecte le bail (409 si une passe tourne).
- **Limites et erreurs.** Les erreurs réseau, 5xx et 429 courts (≤ 30 s) sont retentés trois fois avec attente croissante. Un 429 long arrête la passe sans bloquer. Un échec par compte double un délai de reprise (5 min jusqu'à 24 h) et est mémorisé avec un code stable. Un jeton refusé (401) arrête la passe et passe la carte en « Error ». Aucun échec Discord n'empêche une connexion ou une liaison : la synchronisation suivant la création ou la liaison tourne en arrière-plan, après la réponse.
- **Cache navigateur.** `GET /api/account/avatar` répond `Cache-Control: private, no-cache` avec un `ETag` dérivé de la source et de la version : le navigateur revalide à chaque usage (304 sans corps), donc une photo actualisée apparaît partout sans vider le cache.

## Alternatives écartées

- **Utiliser l'avatar renvoyé par l'OAuth (`identify`)** : sans bot ni tâche de fond, il ne se mettrait à jour qu'à la connexion. Le bot est exigé par la demande ; l'avatar OAuth pourrait compléter l'instantané initial dans une évolution.
- **Une seule image par compte, écrasée à chaque changement de source** : perdrait la photo personnelle au retour vers Discord.
- **Verrou consultatif ou file externe pour la planification** : le bail en base suffit et survit aux redémarrages.

## Conséquences

- Migration `00093` : les comptes ayant déjà une photo reçoivent une préférence `custom` ; aucune photo n'est perdue ; la migration est réversible (les images Discord sont alors supprimées).
- Chaque compte lié coûte un appel Discord par jour au plus ; à très grande échelle, le débit sera limité par les quotas Discord (la passe s'arrête sur un 429 long et reprend à la passe suivante).
- Limite connue : un avatar supprimé côté Discord retire l'image Discord, mais un compte Discord supprimé (`404`) garde la dernière photo connue.
