# Liaison compte ↔ Personne : demande et validation par un administrateur

Statut : accepté pour le lot #96. Précise l'[ADR 0011](0011-locuteurs-personnes-et-attributions.md) sans la contredire.

## Contexte

L'ADR 0011 pose qu'une Personne n'est jamais un compte, un lien optionnel et unique pouvant la relier à un compte. La phase 5 a livré ce lien (`persons.user_id`, unique), mais seule la voie administrateur existe (`PUT`/`DELETE /api/admin/persons/{id}/account`), et aucune interface ne l'expose. Rien ne permet donc à un utilisateur de demander à retrouver la Personne à laquelle il correspond, afin de voir les clips où elle est identifiée et de gérer son profil vocal.

Un utilisateur ne doit pas pouvoir revendiquer silencieusement l'identité d'une autre Personne : le lien ouvre des droits (voir ses clips, supprimer son profil vocal). Il faut une décision explicite et traçable.

## Décision

**Demande et validation.** Un utilisateur connecté envoie une demande de liaison à une Personne déjà créée, avec un message facultatif. Un Administrateur approuve ou refuse la demande. La demande et la décision sont visibles des deux côtés.

**Unicité.** Le lien reste unique dans les deux sens : une Personne a au plus un compte, un compte a au plus une Personne (contrainte `persons.user_id` unique). Un compte a au plus une demande en attente, et une Personne aussi.

**Personne déjà liée.** Une demande peut viser une Personne déjà liée à un autre compte. À l'approbation, l'Administrateur remplace alors le lien existant — l'ancien compte perd la liaison — ou refuse la demande. C'est une décision explicite, jamais un effet de bord, et le remplacement est consigné.

**Compte déjà lié.** Un compte déjà lié à une Personne ne peut pas demander une autre Personne : il doit d'abord se délier.

**Déliement.** L'Administrateur peut délier une association, et le compte lié peut retirer sa propre liaison. Le déliement ne supprime ni la Personne, ni ses attributions, ni ses clips : il retire seulement l'association et, avec elle, l'accès de l'onglet « Ma voix » et le droit de supprimer le profil vocal.

**Trace.** Toute demande, approbation, refus, annulation, liaison (y compris un remplacement) et déliaison est consignée dans un journal immuable, sans donnée vocale : action, Personne (id et nom au moment), compte concerné (id et nom), compte remplacé le cas échéant, auteur et date.

**Nature de la preuve.** La validation est humaine et administrative. Aucune preuve biométrique de la Personne n'est exigée.

## Alternatives écartées

- Preuve contrôlée par la Personne (défi vocal) : intrusive, non fiable comme preuve d'identité, disproportionnée pour une fonction de confort.
- Auto-liaison par une preuve sans Administrateur : laisse revendiquer silencieusement l'identité d'une autre Personne.
- Refus automatique de toute demande visant une Personne déjà liée : bloquerait le vrai titulaire tant qu'un Administrateur n'a pas délié un lien erroné.
- Journaliser dans `person_events` : ce journal porte des compteurs vocaux et cible une autre Personne ; il ne décrit pas un compte.
- Unifier les deux voies (administrateur et demande) en supprimant la liaison directe : l'Administrateur doit pouvoir corriger lui-même sans passer par une demande.

## Conséquences

- La liaison directe par l'Administrateur reste possible et écrit désormais la même trace que la voie par demande.
- Un compte fermé garde sa ligne (suppression douce) et donc son éventuel lien ; il ne bloque aucune nouvelle demande, puisqu'une demande peut viser une Personne déjà liée et que l'approbation remplace le lien.
- Le droit de supprimer un profil vocal reste réservé au compte lié et aux Administrateurs ; il se perd au déliement, comme le prévoit l'ADR 0011 côté « compte lié ».
- Les tests doivent couvrir : demande, refus, approbation, remplacement d'un lien existant, annulation, déliement par le titulaire et par un Administrateur, interdiction du compte déjà lié, et absence de secret dans la trace.
