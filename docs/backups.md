# Sauvegarde et restauration

Dans Forgeyard, PostgreSQL porte `x-forgeyard.backup: {type: postgres}`. Dans « Volumes et sauvegardes », configurer le stockage S3 et sa rétention, planifier les sauvegardes PostgreSQL (`pg_dump`) et vérifier un résultat réussi. Le bucket, ses identifiants, son chiffrement et les droits d’accès appartiennent à l’exploitant ; ils ne sont pas inclus dans le dépôt. Vérifier périodiquement que les archives sont lisibles et stockées hors du NAS.

Une sauvegarde de Snitch comporte **deux éléments distincts**, à ne jamais ranger au même endroit :

1. **la base**, sauvegardée par Forgeyard (`pg_dump`), qui ne contient aucune clé ;
2. **la clé maître**, que l’application a générée seule (le déploiement n’en demande aucune) et conserve dans le volume `secrets`, hors de la base. Le Super-administrateur l’exporte depuis l’administration (en attendant le rôle, tout Administrateur) : l’export s’accompagne de l’avertissement qu’elle ne se rejoue pas (l’écran d’administration correspondant arrive avec la refonte de l’interface ; l’opération existe déjà dans l’API, `POST /api/admin/master-key/export`). Conserver l’export dans un gestionnaire de secrets sauvegardé hors du NAS, jamais dans la même archive ni le même accès S3 que la base. Si `MASTER_KEY` est fournie explicitement, conserver cette valeur de la même façon.

Exporter aussi les paramètres d’exploitation non secrets, références d’images/digests et version du schéma. Une sauvegarde PostgreSQL seule ne permet pas de déchiffrer les réglages Discord ; générer une nouvelle clé ne récupère pas les anciens secrets, mais **ne bloque pas le démarrage** : voir « Restauration sans la clé » ci-dessous. Exporter la clé une fois après l’installation suffit tant qu’elle ne change pas (elle ne change que si le volume `secrets` est supprimé ou si une autre `MASTER_KEY` est fournie).

Procédure de restauration préparée (exercice complet réservé à la phase 7) :

1. Arrêter l’application et les jobs ; conserver l’instance endommagée et ses volumes pour retour arrière. Choisir une sauvegarde saine et relever sa version applicative.
2. Recréer une instance PostgreSQL compatible avec la même extension pgvector, sur un nouveau volume nommé. Restaurer via la fonction de restauration Forgeyard dans le service PostgreSQL concerné. Pour une archive manuelle, choisir `pg_restore` pour format custom ou `psql` pour SQL, selon le format effectivement produit ; ne pas deviner le format ni écraser un volume sans contrôle.
3. Rétablir **la même clé maître** depuis sa sauvegarde séparée : la fournir dans la variable chiffrée Forgeyard `MASTER_KEY` (facultative, prioritaire sur toute clé générée) ; sans elle, voir « Restauration sans la clé ». Le mot de passe de la base n’est pas à rétablir (le nouveau déploiement produit le sien dans son volume `secrets` ; l’ancien n’est pas nécessaire pour importer un `pg_dump`). Rétablir l’origine HTTPS, cookies Secure et images compatibles. Restaurer les identifiants registry/S3 depuis leur gestionnaire séparé si nécessaire.
4. Exécuter les migrations de la version choisie et attendre leur réussite avant de démarrer app. Ne pas créer un nouvel administrateur sur une base restaurée déjà installée.
5. Vérifier santé, fermeture du wizard, connexion admin/local, comptes et approbations, puis déchiffrement des réglages Discord. Vérifier les logs sans les partager ; garder un chemin de retour vers l’ancienne instance jusqu’à validation.

### Restauration sans la clé

Si la clé est perdue ou si le volume `secrets` a été recréé, les secrets chiffrés sont irrécupérables, mais **la restauration reste possible** : l’application démarre avec une clé neuve, traite les secrets illisibles comme non configurés (la connexion Discord est désactivée, les comptes locaux fonctionnent) et l’administration les liste. Les ressaisir (par exemple l’identifiant et le secret Discord depuis le portail Discord) chiffre à nouveau avec la clé courante. Pensez à exporter alors la nouvelle clé. Rien d’autre n’est perdu : la base ne dépend pas de la clé.

Une restauration complète, secrets compris, n’est validée que si la clé d’origine est disponible. Conserver une session locale d’administration (un compte local, pas seulement Discord) avant toute modification manuelle, pour pouvoir ressaisir les secrets. Ni Snitch ni ce plan ne sauvegardent les vidéos ; un NAS sans miroir n’est pas une protection des originaux. L’exercice de restauration, sa durée et ses garanties seront mesurés en phase 7.
