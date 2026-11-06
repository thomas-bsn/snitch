# Clips d'une Personne et suppression du profil vocal

Périmètre du ticket 41 (phase 5). Le glossaire, l'ADR et la feuille de route sont mis à jour à la clôture de la phase (ticket 42).

## Lien Personne et compte

Une Personne a au plus un compte, un compte au plus une Personne (`persons.user_id` unique, `409 account_already_linked`). Le lien ne supprime jamais rien.

Un utilisateur demande à lier son compte à une Personne existante (`POST /api/me/person/request`, message facultatif) ; un Administrateur approuve (`POST /api/admin/person-link-requests/{id}/approve`) ou refuse (`…/refuse`). Approuver lie le compte demandeur et **remplace** un éventuel lien existant sur cette Personne : l'ancien compte perd la liaison. La demande la plus récente se consulte (`GET /api/me/person/request`) et s'annule tant qu'elle est en attente (`DELETE /api/me/person/request`). Un compte a au plus une demande en attente, une Personne aussi (`409 request_pending`, `409 person_request_pending`) ; un compte déjà lié doit se délier d'abord. Le déliement se fait par l'Administrateur (`DELETE /api/admin/persons/{id}/account`) ou par le compte lié lui-même (`DELETE /api/me/person`). `GET /api/me/person` renvoie la Personne du compte connecté (`404 no_linked_person` sinon).

Chaque demande, approbation, refus, annulation, liaison et déliaison est consignée dans `person_link_events` (immuable, sans donnée vocale) : action, Personne (id et nom au moment), compte concerné, compte remplacé le cas échéant, auteur et date. Voir l'[ADR 0018](adr/0018-liaison-compte-personne-et-validation.md).

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

Aucun vecteur n'existe hors de `voice_fingerprints` : le résultat d'une tâche d'empreintes est stocké sans vecteur dans `jobs.result` (résumé et condensé seulement), si bien que la suppression d'un profil ne laisse rien dans la file ni dans les réponses de planification ou de reprise.

Le marqueur de retrait ne contient aucune donnée vocale. Tant qu'il est posé :

- le rapprochement ignore les références de la Personne, même insérées hors API ;
- une validation ou correction vers elle est refusée (`409 voice_profile_deleted`) ;
- une règle de piste micro qui la désigne passe à l'état `owner_profile_deleted` : rien ne lui est attribué.

Il n'existe pas encore de levée du retrait ; elle relèverait d'une décision ultérieure. La suppression d'une Personne (fusion, séparation) est le ticket 40.

## Trace

`voice_profile_deletions` (immuable, sans clé étrangère) conserve la Personne, le compte lié, l'auteur et trois compteurs (échantillons, empreintes, attributions) : aucun vecteur, échantillon ou audio. L'historique de chaque voix reçoit une ligne `delete_profile`.

## Suppression d'un clip

La suppression d'un clip efface en cascade ses Locuteurs locaux, empreintes, attributions dérivées et historiques ; le fichier original n'est jamais touché et les Personnes restent.
