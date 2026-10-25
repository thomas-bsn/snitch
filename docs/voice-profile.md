# Clips d'une Personne et suppression du profil vocal

Périmètre du ticket 41 (phase 5). Le glossaire, l'ADR et la feuille de route sont mis à jour à la clôture de la phase (ticket 42).

## Lien Personne et compte

Un administrateur lie une Personne à un compte par son nom (`PUT /api/admin/persons/{id}/account`) ou supprime le lien (`DELETE`). Une Personne a au plus un compte, un compte au plus une Personne (`409 account_already_linked`). Le lien ne supprime jamais rien. `GET /api/me/person` renvoie la Personne du compte connecté (`404 no_linked_person` sinon).

## Voir ses clips

L'écran « Ma voix » liste les clips où la Personne est identifiée en réutilisant le filtre `person_id` de la bibliothèque : la visibilité du compte s'applique avant la liste, un clip privé d'un autre n'apparaît jamais. Une identification est une validation humaine ou une attribution automatique.

## Ce que constitue le profil vocal

Le profil vocal d'une Personne regroupe :

- ses Échantillons vocaux validés (références actives et retirées) ;
- les empreintes des Locuteurs locaux dont l'attribution effective (validation humaine d'abord, sinon rapprochement ou piste micro) est cette Personne ;
- toutes les attributions à son nom, qui disparaissent : les voix redeviennent inconnues ; elle n'est plus candidate des voix ambiguës.

Les clips, la Personne et les données des autres Personnes ne sont pas touchés.

## Suppression (`DELETE /api/persons/{id}/voice-profile`)

Autorisée au compte lié à la Personne et aux administrateurs (`403 forbidden` sinon). Une seule transaction, sous le verrou du rapprochement, efface le profil, pose le marqueur de retrait `persons.voice_profile_deleted_at` et écrit la trace. Ensuite le rapprochement est relancé et les pistes micro qui désignent la Personne sont réévaluées. Répéter la suppression est sans effet et n'est pas retracée.

## Pas de reconstitution

Le marqueur de retrait ne contient aucune donnée vocale. Tant qu'il est posé :

- le rapprochement ignore les références de la Personne, même insérées hors API ;
- une validation ou correction vers elle est refusée (`409 voice_profile_deleted`) ;
- une règle de piste micro qui la désigne passe à l'état `owner_profile_deleted` : rien ne lui est attribué.

Il n'existe pas encore de levée du retrait ; elle relèverait d'une décision ultérieure. La suppression d'une Personne (fusion, séparation) est le ticket 40.

## Trace

`voice_profile_deletions` (immuable, sans clé étrangère) conserve la Personne, le compte lié, l'auteur et trois compteurs (échantillons, empreintes, attributions) : aucun vecteur, échantillon ou audio. L'historique de chaque voix reçoit une ligne `delete_profile`.

## Suppression d'un clip

La suppression d'un clip efface en cascade ses Locuteurs locaux, empreintes, attributions dérivées et historiques ; le fichier original n'est jamais touché et les Personnes restent.
