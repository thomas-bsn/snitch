# Sauvegarde et restauration

Dans Forgeyard, PostgreSQL porte `x-forgeyard.backup: {type: postgres}`. Dans « Volumes et sauvegardes », configurer le stockage S3 et sa rétention, planifier les sauvegardes PostgreSQL (`pg_dump`) et vérifier un résultat réussi. Le bucket, ses identifiants, son chiffrement et les droits d’accès appartiennent à l’exploitant ; ils ne sont pas inclus dans le dépôt. Vérifier périodiquement que les archives sont lisibles et stockées hors du NAS.

Conserver **séparément** la valeur exacte de `MASTER_KEY`, par exemple dans un gestionnaire de secrets sauvegardé hors du NAS. Exporter aussi les paramètres d’exploitation non secrets, références d’images/digests et version du schéma. Ne pas ranger la clé dans la même archive ou le même accès S3 que la base. Une sauvegarde PostgreSQL seule ne permet pas de déchiffrer les réglages Discord ; générer une nouvelle clé ne récupère pas les anciens secrets.

Procédure de restauration préparée (exercice complet réservé à la phase 7) :

1. Arrêter l’application et les jobs ; conserver l’instance endommagée et ses volumes pour retour arrière. Choisir une sauvegarde saine et relever sa version applicative.
2. Recréer une instance PostgreSQL compatible avec la même extension pgvector, sur un nouveau volume nommé. Restaurer via la fonction de restauration Forgeyard dans le service PostgreSQL concerné. Pour une archive manuelle, choisir `pg_restore` pour format custom ou `psql` pour SQL, selon le format effectivement produit ; ne pas deviner le format ni écraser un volume sans contrôle.
3. Rétablir **la même MASTER_KEY** depuis sa sauvegarde séparée dans les variables chiffrées Forgeyard. Le mot de passe de la base n’est pas à rétablir (le nouveau déploiement produit le sien dans son volume `secrets` ; l’ancien n’est pas nécessaire pour importer un `pg_dump`). Rétablir l’origine HTTPS, cookies Secure et images compatibles. Restaurer les identifiants registry/S3 depuis leur gestionnaire séparé si nécessaire.
4. Exécuter les migrations de la version choisie et attendre leur réussite avant de démarrer app. Ne pas créer un nouvel administrateur sur une base restaurée déjà installée.
5. Vérifier santé, fermeture du wizard, connexion admin/local, comptes et approbations, puis déchiffrement des réglages Discord. Vérifier les logs sans les partager ; garder un chemin de retour vers l’ancienne instance jusqu’à validation.

Si la clé est perdue, les secrets chiffrés sont irrécupérables : une restauration complète n’est pas validée. Conserver une session locale d’administration et préparer une procédure contrôlée de reconfiguration avant toute modification manuelle. Ni Snitch ni ce plan ne sauvegardent les vidéos ; un NAS sans miroir n’est pas une protection des originaux. L’exercice de restauration, sa durée et ses garanties seront mesurés en phase 7.
