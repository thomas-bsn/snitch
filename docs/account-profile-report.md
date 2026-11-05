# Profil, photos des Personnes et reconnexion des Workers — bilan

## Livré

- Profil bilingue organisé en identité/photo, préférences, connexion/mot de passe et fermeture du compte. Disposition validée par l’utilisateur sur le prototype interactif synthétique.
- Pseudo affiché et identifiant de connexion modifiables ; compte Discord affiché avec son nom global et son vrai `@username`. Métadonnées rafraîchies à la connexion et par le bot ; pseudo choisi conservé.
- Premier mot de passe pour un compte Discord, changement avec mot de passe actuel pour un compte local, conservation de la session courante et révocation des autres. Déliement Discord disponible une fois le mot de passe établi.
- Photo du compte liée à la Personne après approbation, dans l’annuaire, la fiche et le panneau de décision vocale ; lien résolu à chaque lecture sans copie d’image. Initiales sans image/lien/compte actif. Les noms des Personnes restent indépendants.
- Reconnexion des Workers après une réponse HTML/JSON temporaire de proxy, dans la prise de jobs et le renouvellement du lease. Le délai reste plafonné à 10 secondes même après plusieurs heures d’échec.

## Cause reproduite

`PYTHONPATH=worker:worker/snitch_worker/api .venv/bin/python -m unittest discover -s worker/tests -p test_reconnect.py` échouait avant le correctif : `JSONDecodeError` pour une page HTML et `KeyError: 'error'` pour le JSON d’un proxy. Ces exceptions sortaient de la boucle de prise des jobs et du thread de renouvellement. Après correction, une réponse API valide suivant la panne est utilisée sans redémarrer le Worker ; le renouvellement reprend si le lease reste valide. Ce scénario est reproduit avec le vrai client HTTP généré et un transport simulé ; aucun journal de production n’a été capturé.

## Vérifications

- Tests PostgreSQL/pgvector avec détecteur de courses : préférences/identité/mot de passe, liaison approuvée et photo, OAuth Discord et synchronisation d’avatars.
- 36 parcours Playwright : paramètres du compte (11), fermeture (1), annuaire/fiche (6), coque/accessibilité (5), import/photo (8), voix de clip (5). Axe-core en clair et sombre ; mobile à 390 px.
- Worker : 264 tests passent, 9 ignorés (fournisseurs réels facultatifs), dont les trois régressions de reconnexion et de délai.
- Construction et formatage frontend, génération Go/TypeScript/Python du contrat, vérification des messages d’erreur et `go vet`.

## Livraison et limites

Migration additive `00096` ; ADR 0043. La PR comprend le backend, le frontend et le Worker. Les Workers existants doivent recevoir une fois l’image contenant le correctif ; seul le redéploiement de l’application ne met pas à jour leur code. La reconnexion conserve l’adresse et le jeton ; elle ne répare pas une configuration changée ou un jeton révoqué. Un lease expiré/refusé conserve son comportement de rejet du résultat.

Données synthétiques exclusivement. Aucun scan de la vraie bibliothèque, modèle réel, déploiement ou benchmark de phase 7. L’approbation porte sur la disposition du profil ; les validations existantes de liaison Personne et de déliement Discord restent requises dans le produit.
