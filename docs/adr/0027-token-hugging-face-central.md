# 0027 — Token Hugging Face central, chiffré et distribué aux Workers

Statut : accepté pour #194. Révise l'[ADR 0022](0022-modele-vocal-actif-global.md) (« le jeton reste sur la machine du Worker et ne transite jamais par Snitch ») et la décision de la phase 6 bis qui écrivait le jeton dans le fichier d'environnement côté navigateur seulement.

## Contexte

Un Worker recréé avec un nouveau volume perd son jeton et ne peut plus télécharger de modèle candidat (`hf_token_missing`) sans une commande `store-token` sur sa machine. L'utilisateur veut saisir le jeton une fois dans Snitch.

## Décision

- **Stockage** : le jeton est rangé dans la table `settings` existante, chiffrée en AES-256-GCM avec la clé maître (hors base et hors sauvegardes de la base). Aucune variable d'environnement, aucun nouveau mécanisme de chiffrement. Il n'est jamais renvoyé par une lecture ; l'interface affiche seulement « configuré / absent / erreur ».
- **Saisie** : réservée au Super-administrateur, dans Administration › Voice Processing et dans une étape facultative du wizard. Enregistrer, remplacer et supprimer sont journalisés sans la valeur.
- **Distribution versionnée** : le pulse du Worker ne transporte que le numéro de version du jeton central. Quand il change, le Worker le récupère sur un point d'accès dédié, réservé aux Workers authentifiés et non révoqués, l'écrit dans son fichier secret existant, puis renvoie un accusé de réception. Une suppression centrale fait purger la copie centrale à la version suivante. Un Worker hors ligne se synchronise à sa reconnexion ; l'interface affiche « non synchronisé ».
- **Provenance retenue par le Worker** : le Worker sait si son jeton vient de Snitch ou d'un dépôt local (`store-token`, installation). Le jeton central ne remplace qu'un jeton central. Un jeton local reste prioritaire tant qu'il n'est pas supprimé explicitement ; l'interface affiche la source par Worker.
- Le jeton reste absent du processus d'inférence, des arguments, des fichiers de déploiement et des logs.

## Alternatives écartées

- **Jeton en variable d'environnement** : lisible par `docker inspect`, figé au démarrage, non modifiable depuis l'interface.
- **Le central écrase toujours le local** : remplacement destructif silencieux, refusé par #194.
- **Le jeton envoyé à chaque pulse** : expose le secret à chaque requête sans besoin.

## Conséquences

- La surface de confiance s'élargit : un serveur compromis livre le jeton. Le chiffrement au repos ne protège que les sauvegardes de la base.
- Une clé maître perdue rend le jeton illisible : il est signalé parmi les secrets illisibles et doit être ressaisi. La rotation de la clé passe par une nouvelle saisie.
- L'installation initiale (#203) doit fonctionner sans jeton central.
