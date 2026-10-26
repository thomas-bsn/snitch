# Contribuer

Le code du projet est sous licence MIT ; conserver les copyrights et les licences des dépendances. Lire la roadmap, le glossaire et les ADR avant un changement. Le nom du projet est Snitch ; c’est aussi le nom unique de l’interface, sans réglage.

Discuter une proposition dans une issue de `thomas-bsn/snitch`, puis contribuer par une branche et une pull request. Expliquer le comportement modifié et les vérifications effectuées. Voir [développement et tests](docs/development.md). Modifier le contrat avant de régénérer les clients ; inclure tous les artefacts générés et les notices de nouvelles dépendances.

Utiliser uniquement des données synthétiques ou fixtures dédiées. Ne pas inclure de `.env`, clés, tokens, logs de bootstrap, médias privés ni vrais identifiants Discord. Les tests Discord utilisent leur fournisseur simulé. Les fonctionnalités futures suivent une phase validée ; le benchmark de vraie bibliothèque attend une autorisation distincte.
