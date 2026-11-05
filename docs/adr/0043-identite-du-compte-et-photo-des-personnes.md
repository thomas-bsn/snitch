# 0043 — Identité du compte et photo des Personnes liées

Statut : accepté par l’utilisateur pour la refonte du profil. Complète les ADR 0005, 0018 et 0041.

## Contexte

Les comptes créés avec Discord exposaient leur identifiant de connexion généré (`discord_…`) comme pseudo. Le nom global Discord ne permet pas non plus de reconnaître le vrai `@username`. Aucun formulaire ne permettait de changer son pseudo ou d’établir un mot de passe local. Une Personne liée ne montrait que ses initiales.

## Décision

- Distinguer le **pseudo affiché** (`users.display_name`, facultatif, 1–100 caractères) et l’**identifiant de connexion** (`users.username`, unique sans distinction de casse, 3–64 caractères). Les deux se modifient depuis le profil du compte connecté. Un compte généré par Discord sans pseudo choisi affiche son nom global Discord, sinon son username Discord, plutôt que son identifiant technique. Les comptes locaux gardent leur nom local. Les identifiants et liens existants restent intacts.
- Conserver séparément `discord_username` et `discord_name` : le premier est le vrai username, le second le nom global avec repli sur le username. Les deux sont actualisés à chaque connexion/liaison OAuth et lors de la synchronisation par le bot ; aucune de ces opérations ne remplace un pseudo choisi dans Snitch. Les anciens comptes sont actualisés à leur prochaine connexion ou passe du bot.
- `PUT /api/account/password` établit un premier mot de passe depuis la session d’un compte Discord ; s’il existe déjà, le mot de passe actuel est obligatoire. Conserver Argon2id et les limites existantes (12–256 caractères), la même origine, le CSRF et la limitation du calcul des mots de passe. Verrouiller le compte et revérifier la session avant écriture. Révoquer les autres sessions, conserver celle de la modification. Établir le mot de passe permet ensuite de délier Discord sans perte d’accès.
- `GET /api/persons/{id}/avatar` résout le lien validé vers un compte actif et sert sa photo actuellement choisie, aux utilisateurs connectés. Aucun lien en attente ne suffit. Ne copier aucune image : changement de source, remplacement de photo, retrait ou remplacement du lien et fermeture du compte se reflètent à la lecture suivante. Le cache est privé et revalidé ; son ETag inclut le compte pour éviter un 304 erroné après remplacement de lien. En l’absence d’image, afficher les initiales.
- La Personne garde son propre nom et son Profil vocal ; le lien ne fusionne jamais ces objets avec le compte.

## Alternatives écartées

- Renommer automatiquement les identifiants de connexion existants depuis Discord : casse les habitudes de connexion et introduit des collisions.
- Utiliser le nom global comme vrai username Discord : les deux champs ont des fonctions distinctes.
- Copier la photo à l’approbation : devient obsolète et conserve l’image après déliaison.

## Conséquences

Migration additive `00096`, réversible ; la descente retire seulement les nouveaux noms choisis et métadonnées Discord. Aucun mot de passe ni jeton n’est renvoyé ou journalisé. Les photos restent soumises au choix de source et à la synchronisation de l’ADR 0041. Aucun modèle ni média réel n’est utilisé pour les vérifications.
