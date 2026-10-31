# Liaison d'un compte à une Personne par demande — #96

Décision : l'utilisateur demande, un Administrateur valide ou refuse ; liens et déliements sont tracés ([ADR 0018](adr/0018-liaison-compte-personne-et-validation.md)). Les trois points du ticket sont tranchés : approbation par un Administrateur (sans preuve biométrique) ; une demande peut viser une Personne déjà liée, l'approbation **remplace** alors le lien existant ; le déliement est ouvert à l'Administrateur comme au compte lié. Un compte a au plus une demande en attente, une Personne aussi ; un compte déjà lié doit se délier d'abord.

## Réalisé

- Migration `00045` : `person_link_requests` (au plus une en attente par compte et par Personne) et `person_link_events` (journal immuable, sans donnée vocale).
- Compte : demander (`POST /api/me/person/request`), lire la plus récente (`GET`), annuler (`DELETE /api/me/person/request`), se délier (`DELETE /api/me/person`).
- Administration : file (`GET /api/admin/person-link-requests`), approuver (`POST …/{id}/approve`, remplacement inclus), refuser (`POST …/{id}/refuse`).
- La liaison directe de l'Administrateur (`PUT`/`DELETE /api/admin/persons/{id}/account`) est désormais transactionnelle et écrit la même trace.
- Interface : onglet « Ma voix » (demande avec recherche de Personne et message, état en attente, annulation, refus, déliement) et sous-onglet « Demandes de liaison » dans Administration › Utilisateurs et accès.
- Contrat OpenAPI, clients générés (Go, TypeScript, Python) et textes d'erreur en anglais et en français.
- Tests d'intégration : demande, conflits, message borné, file réservée aux Administrateurs, refus, approbation, remplacement, déliement par le titulaire et par l'Administrateur, liaison directe tracée, immutabilité et absence de donnée vocale dans la trace.

## Restant et écarts

- Aucune preuve contrôlée par la Personne : écartée par décision.
- Un déliement ne modifie ni la Personne, ni ses clips, ni ses attributions, ni son profil vocal (voulu).
- Un compte fermé garde sa ligne et son éventuel lien ; il ne bloque aucune nouvelle demande, puisque l'approbation remplace le lien.
- L'écran du compte ne se rafraîchit pas tout seul après la décision d'un Administrateur : la Personne apparaît au prochain chargement.

## Vérifications

- `gofmt -l app` vide ; `go vet ./...` et `go vet -tags integration ./...` verts.
- `npm --prefix web run lint` et `npm --prefix web run build` verts.
- `scripts/check_error_codes.py` et `scripts/check_generated.py` verts.
- Tests d'intégration ciblés (`person_link_requests`, Personnes, profil vocal) et tests unitaires verts.
