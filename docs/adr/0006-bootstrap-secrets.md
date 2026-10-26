# Installation initiale et secrets

Statut : accepté dans le plan de phase 0. Amendé par l'[ADR 0013](0013-demarrage-sans-configuration.md) : la clé maître est désormais générée par l'application et conservée dans son volume `secrets`, au lieu d'être fournie par l'opérateur ; la séparation de la clé et de la base reste.

## Contexte et décision

Une transaction verrouillée sérialise la création du premier administrateur. Le code temporaire à usage unique est affiché dans les logs, seule exception à l’expurgation. Les secrets de réglages utilisent AES-GCM avec une clé maître externe.

## Alternatives écartées

Un mot de passe admin par défaut ou des secrets en clair dans une sauvegarde seraient trop risqués.

## Conséquences

Restaurer demande la sauvegarde PostgreSQL ET la clé maître conservée séparément. Le code expire après 15 minutes ; si expiré, redémarrer le serveur pour le renouveler tant que l’installation n’est pas terminée.
