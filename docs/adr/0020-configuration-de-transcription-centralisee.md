# Configuration de transcription centralisée et activation par Worker

Statut : accepté.

## Contexte et décision

Snitch est la source de vérité de l'adresse du serveur de transcription, du modèle de bibliothèque et de la langue. La configuration peut être enregistrée avant le premier Worker, avec un statut « non testé ». Un mode « transcription désactivée » explicite est distinct d'une configuration incomplète. Une adresse commune s'applique par défaut ; un Worker peut avoir sa propre adresse lorsque son réseau l'exige. Un changement de l'adresse commune préserve ces adresses propres. Le modèle et la langue de bibliothèque restent communs et leur changement concerne tous les Workers.

Enregistrer définit la version souhaitée même si un Worker est hors ligne ou si le test échoue. Le test est réalisé depuis chaque Worker disponible ; un résultat obtenu depuis le NAS ne valide pas le Worker. Un Worker teste depuis sa machine la configuration demandée avant de l'activer pour de nouveaux jobs. Ses jobs en cours gardent la version avec laquelle ils ont commencé. Si le test échoue ou si le Worker est hors ligne, il garde sa dernière version active et peut continuer à traiter des jobs avec elle ; l'interface montre l'écart entre version demandée et version active. Sans version active validée, aucun job de transcription ne lui est attribué. L'installation d'un Worker reste possible sans configuration de transcription ; son attente doit être explicite.

Après mise à jour d'un Worker existant, les variables historiques `TRANSCRIPTION_API_URL` et `TRANSCRIPTION_MODEL` ne servent qu'à l'amorçage et à la compatibilité initiale. Une configuration reçue et validée depuis Snitch prend priorité. L'interface distingue les anciens Workers incapables de recevoir les changements.

Si la configuration souhaitée d'un ancien Worker incompatible diffère de celle qu'il utilise, aucun nouveau job de transcription ne lui est attribué ; ses jobs en cours peuvent se terminer et l'interface demande sa mise à jour. Un Worker récent qui signale une ancienne version active validée peut continuer à traiter avec cette version si la nouvelle échoue au test.

Un test réussi depuis le Worker établit que le serveur répond et que le modèle demandé est disponible. Les échecs de réseau et de modèle sont distincts. Le test ne transcrit aucun vrai clip de la bibliothèque ; seules des données synthétiques peuvent être utilisées si une requête de transcription est nécessaire.

## Alternatives écartées

Exiger une configuration avant toute installation empêcherait les déploiements préparatoires ou volontairement sans transcription. Activer une valeur dès son enregistrement risquerait d'envoyer des jobs vers une adresse inaccessible depuis le Worker. Faire primer indéfiniment les variables du `.env` empêcherait les changements centralisés après installation. Laisser un ancien Worker incompatible traiter après un changement central créerait une divergence silencieuse.

## Conséquences

Le profil implicite actuel `default/fr` ne suffit plus à représenter une configuration explicitement choisie. La synchronisation doit exposer les versions demandée, reçue, testée et active ainsi que les échecs de validation, sans faire passer une configuration enregistrée sans Worker pour une connexion testée.
