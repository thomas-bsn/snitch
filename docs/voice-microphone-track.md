# Piste micro dédiée

Phase 5, ticket 34. Certaines captures enregistrent le micro d'un joueur sur une piste audio séparée. Cette piste a un propriétaire connu : inutile de la diariser, elle reçoit directement sa Personne.

## Désigner la piste d'une source

Un administrateur ouvre « Piste micro pour cette source » dans l'écran des sources. L'écran liste les pistes audio réellement détectées sur les clips prêts de la source (numéro global FFmpeg, titre, langue, nombre de clips). Aucun titre n'est supposé : on choisit une piste détectée, on indique si elle est reconnue par son numéro, son titre exact (sans tenir compte de la casse) ou les deux, puis on choisit ou on crée la Personne propriétaire.

À l'enregistrement, la désignation est appliquée à tous les clips prêts de la source, sans worker, sans média et sans tâche de file. L'écran annonce le résultat par état (par exemple « 1 clip désigné, 2 clips sans piste correspondante »). Enregistrer une règle identique la réapplique sans créer de version.

Les clips importés après l'enregistrement de la règle sont couverts à la prochaine application : nouvel enregistrement de la règle, ou planification d'une étape vocale sur le clip.

## Ce que reçoit la piste

La piste désignée devient un Locuteur local de type `microphone` (libellé « Piste micro »), sans tâche de diarisation ni empreinte, avec un seul tour couvrant le clip. Son Attribution est automatique, non validée, d'origine `microphone_track`, de confiance 1, signée par l'administrateur qui a fixé la règle. Elle n'est jamais un Échantillon vocal validé : elle ne contribue pas aux références (ADR 0008). Tout utilisateur connecté peut la valider ou la corriger comme un autre Locuteur local ; la validation humaine l'emporte et n'est jamais perdue.

Les autres pistes audio d'un clip restent traitées normalement, y compris la diarisation. La planification d'une étape vocale sur la piste micro est refusée (`microphone_track_not_diarized`), de même qu'un résultat de diarisation tardif sur cette piste.

## Aucune piste abandonnée en silence

`GET /api/clips/{id}/microphone-track` indique le rôle de chaque piste (`microphone`, `candidate`, `other`) et un état explicite :

| État | Sens |
| --- | --- |
| `none` | aucune règle ne concerne le clip |
| `designated` | piste micro résolue et attribuée |
| `disabled` | le clip a été déclaré sans piste micro |
| `track_not_found` | la règle ne correspond à aucune piste du clip ; rien n'est attribué |
| `ambiguous_track` | plusieurs pistes correspondent ; les candidates sont listées |
| `conflicting_sources` | les règles de plusieurs sources divergent |
| `blocked_by_validation` | la piste désignée porte des voix diarisées déjà validées par un humain ; elles ne sont pas remplacées |

Une désignation qui ne s'applique pas retire l'attribution machine précédente de la piste micro, mais conserve toute validation humaine.

## Remplacer pour un clip

Dans les détails du clip, un administrateur peut choisir une autre piste et un autre propriétaire, déclarer que le clip n'a pas de piste micro, ou revenir à la règle de la source. Chaque changement (règle de la source ou clip) est historisé de façon immuable : ancienne et nouvelle désignation avec le nom de la Personne à l'époque, auteur, date. L'historique du clip est lu dans la même réponse ; les attributions de la piste micro figurent aussi dans l'historique du Locuteur local (`microphone_assign`, `microphone_unassign`). Il disparaît avec le clip.

## API

- `GET|PUT|DELETE /api/admin/sources/{id}/microphone-track` (administrateurs).
- `GET /api/clips/{id}/microphone-track` (utilisateurs connectés, clips visibles).
- `PUT|DELETE /api/admin/clips/{id}/microphone-track` (administrateurs).

## Limites

- Une désignation remplace une diarisation non validée de la même piste ; elle ne remplace jamais des voix validées.
- Pas de désignation automatique à l'import : le déclenchement reste explicite.
- Le rapprochement vocal (ticket 35) doit ignorer les Locuteurs locaux de type `microphone` et leur ligne `microphone_track`.
- Supprimer une Personne exigera de retirer d'abord les règles et remplacements qui la nomment (clé étrangère sans cascade).
