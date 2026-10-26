# Démarrage sans configuration

Statut : accepté pour la phase 6 bis. Amende l'[ADR 0006](0006-bootstrap-secrets.md) sur un seul point : la clé maître n'est plus fournie par l'opérateur. La séparation entre la clé et la base, elle, est conservée.

## Contexte

Le déploiement fourni demandait de renseigner à la main trois variables avant de démarrer : le mot de passe de la base, la clé maître de chiffrement et l'adresse publique. C'est exactement là que les installations échouaient : une clé maître mal formée empêche le démarrage, un mot de passe de base faible ou recopié est un secret de plus à ranger, et une adresse publique inexacte fausse l'adresse de retour Discord. Or Snitch tourne dans un déploiement qu'il connaît : il peut produire seul ce qui ne doit pas être choisi par une personne.

La décision transversale de l'ADR 0006 était : « les secrets de réglages utilisent AES-GCM avec une clé maître externe », donc fournie et conservée séparément par l'opérateur. Son intention était qu'une sauvegarde de la base ne suffise pas à lire les secrets. Cette intention ne dépend pas de qui fabrique la clé.

## Décision

**Aucune variable obligatoire.** Le déploiement fourni démarre sans fichier d'environnement. Les variables existantes deviennent facultatives et, si elles sont fournies, prioritaires, ce qui préserve les installations existantes.

**Mot de passe de la base** (ticket 02). Une tâche unique, avant tout autre service, produit un mot de passe aléatoire dans le volume `secrets` ; la base, la migration et l'application le lisent dans un fichier. Une valeur explicite reste prioritaire et est conservée dans le volume.

**Clé maître** (ticket 03). L'application la génère au premier démarrage : 32 octets aléatoires, écrits en base64 (le format de `MASTER_KEY`) dans un fichier privé (mode 0600, propriétaire de l'application) du volume `secrets`. Ce volume n'est ni celui de la base ni un élément de sa sauvegarde ; la clé n'est jamais écrite en base, ni dans les journaux, ni dans une réponse autre que l'export. La résolution se fait dans cet ordre :

1. une `MASTER_KEY` fournie et bien formée est utilisée et **n'est jamais recopiée dans le volume** ; mal formée, elle est refusée au démarrage avec un message clair (c'est une erreur de l'opérateur, pas une perte) ;
2. sinon la clé du volume est lue ;
3. sinon une clé est générée et écrite de façon atomique, sans jamais écraser une clé existante. Un fichier de clé inutilisable (tronqué, mal encodé) est mis de côté sous le suffixe `.unusable`, jamais écrasé, puis remplacé ;
4. si le volume n'est pas inscriptible, l'application refuse de démarrer en le disant : une clé qu'on ne peut pas conserver serait perdue au redémarrage suivant.

**Clé absente ou différente : démarrer, signaler, ressaisir.** Que le volume soit perdu, que la clé fournie diffère de celle qui a chiffré les réglages, ou que le fichier soit abîmé, l'application démarre. Un réglage chiffré que la clé courante ne déchiffre pas est traité comme non configuré (Discord se comporte comme s'il n'était pas configuré : connexion Discord désactivée, comptes locaux inchangés) et l'administration le liste (`unreadable_secrets`) pour qu'il soit ressaisi. Ressaisir chiffre avec la clé courante. Aucune lecture ne renvoie une valeur à moitié déchiffrée.

**Export.** Le Super-administrateur exporte la clé depuis l'administration (jusqu'à l'arrivée du rôle, tout Administrateur, derrière une fonction de contrôle unique que le ticket #82 resserrera). L'export est une requête `POST` (session, origine et jeton CSRF requis), accompagnée du code d'avertissement `master_key_not_replayable` : la clé ne se rejoue pas, elle ne relit pas des secrets chiffrés avec une autre clé et une nouvelle clé ne récupère pas les anciens secrets. L'export est tracé (auteur, origine), jamais la clé.

**Adresse publique** (ticket 04). Déduite de la requête quand aucune valeur n'est fournie, avec les réserves de la spec (mandataire de confiance requis, réglage facultatif qui la fixe). Décrite dans la documentation de ce ticket ; elle n'est pas modifiée ici.

## Alternatives écartées

- **Garder la clé fournie par l'opérateur** : c'est le comportement qui faisait échouer les installations, et il n'apporte rien de plus que la séparation clé/base, qui est conservée.
- **Stocker la clé en base** : une sauvegarde de la base suffirait alors à lire les secrets, ce que l'ADR 0006 voulait précisément empêcher.
- **Recopier une clé fournie dans le volume** (comme pour le mot de passe de la base) : une clé d'essai ou une faute de frappe écraserait définitivement la clé générée, donc des secrets sans retour possible. Le mot de passe de la base n'a pas ce risque, car la base le vérifie et le refuse si elle ne le connaît pas.
- **Refuser de démarrer quand la clé est absente ou différente** : l'administrateur n'aurait alors aucun écran pour réparer ; la perte de la clé coûterait une réinstallation alors qu'elle ne coûte que quelques secrets à ressaisir (aujourd'hui le seul secret chiffré est celui de Discord).
- **Un volume distinct pour la clé seule** : un volume de plus à comprendre sans protéger davantage. Les droits du fichier (0600, propriétaire de l'application) l'isolent de PostgreSQL, qui monte le même volume en lecture seule pour y lire son mot de passe ; la vérification de déploiement et le test de démarrage s'en assurent.
- **Rotation de la clé avec rechiffrement des secrets existants** : hors périmètre de la phase 6 bis.
- **Gestionnaire de secrets externe** : hors périmètre.

## Conséquences

- La sauvegarde comporte **deux éléments distincts** : la base (sauvegarde PostgreSQL) et la clé maître (export depuis l'administration, ou valeur de `MASTER_KEY`). La restauration fonctionne sans la clé, au prix de ressaisir les secrets (guide de sauvegarde mis à jour).
- Une clé conservée sur le même hôte que la base protège d'une sauvegarde de base volée, pas d'un hôte compromis : l'export manuel et la documentation de sauvegarde sont la contrepartie.
- Supprimer le volume `secrets` en gardant la base ne bloque plus les secrets (ils sont à ressaisir), mais bloque toujours la connexion à la base tant que le mot de passe de la base n'est pas reproduit : les deux volumes vont ensemble pour le mot de passe, pas pour la clé.
- Une installation existante qui retire `MASTER_KEY` sans avoir exporté sa clé obtient une clé neuve : ses secrets sont à ressaisir. Pour les conserver, garder la variable, ou fournir la clé exportée.
- Le démarrage dépend d'un volume inscriptible pour l'application : le volume `secrets` n'est donc plus en lecture seule pour elle (il reste en lecture seule pour la migration et la base, et `scripts/check_compose.py` le vérifie).
- Le contrôle « peut exporter la clé » est une fonction unique côté serveur ; le ticket #82 la resserre sans toucher aux autres contrôles d'administration.
- Aucune interface d'export n'est livrée par ce ticket : l'API et son contrat existent, la page d'administration qui les montre arrive avec l'administration de la nouvelle interface.
