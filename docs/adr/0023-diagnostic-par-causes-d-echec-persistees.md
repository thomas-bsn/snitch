# 0023 — Diagnostic par causes d'échec persistées, logs sans contenu privé

## Contexte

Une vidéo bloquée ou une erreur entre le NAS et le Worker n'est pas diagnostiquable : le serveur journalise sans corrélation (pas de job, worker ni tentative), le Worker n'a ni format ni niveau configurable, et `Jobs.Fail` ramène tout code inconnu à `worker_failed`. Le workflow n'a pas d'entité « run » (#130 est ouvert). L'épopée #122 demande des logs corrélés et un diagnostic visible dans l'interface.

## Décision

- **Corrélation** : `clip_id`, `job_id`, `cycle`, `attempt` et `worker_id` sont le contexte commun des logs du serveur et du Worker, pris du claim. Aucun identifiant de run ou de trace n'est inventé ; un `workflow_run_id` optionnel s'ajoutera avec #130.
- **Causes d'échec** : énumération fermée, déclarée dans le contrat OpenAPI, partagée par le serveur et le Worker. Le Worker ne renvoie qu'un code ; l'action de reprise conseillée est décidée côté serveur. Un code inconnu (ancien Worker) reste `worker_failed`.
- **Persistance** : l'interface lit des colonnes typées de `job_events` (provider, modèle, révision, statut HTTP, durée, étape), ajoutées par migration additive, jamais les logs et sans champ libre.
- **Logs** : JSON de même schéma des deux côtés, `LOG_LEVEL` (défaut INFO), transitions seules en INFO, texte brut des erreurs fournisseur ou SQL jamais journalisé même en DEBUG, filtre central des clés sensibles. Rotation Docker `json-file` bornée.
- **Exception** : le code d'installation à usage unique reste affiché dans les logs (ADR 0006).
- **Visibilité** : la cause technique est réservée aux Administrateurs.

## Alternatives écartées

- Un identifiant de run provisoire : il serait remplacé par celui de #130 et casserait la corrélation historique.
- Lire les fichiers de logs depuis l'interface, ou un panneau d'export : c'est l'entrepôt de logs que l'épopée exclut, et cela expose du texte non maîtrisé.
- Un champ JSON libre de détail : il laisse entrer des secrets en base.
- Loki ou ELK : trop lourd pour un NAS.

## Conséquences

- Les migrations de diagnostic arrivent avec #141 ; #142 se limite à l'affichage et aux permissions.
- Tout nouveau mode d'échec exige un code du catalogue, donc une évolution du contrat.
- Les anciennes lignes `job_events` restent sans détail de diagnostic.
