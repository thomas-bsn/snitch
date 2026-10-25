# Fusion, séparation et suppression de Personnes (phase 5, ticket 40)

Une Personne créée par erreur se corrige sans toucher aux clips : on la fusionne avec une autre, on sépare certaines de ses voix, ou on la supprime. Tout utilisateur connecté peut le faire, comme il peut créer une Personne et valider une voix. Chaque correction est tracée (auteur, date, anciennes valeurs, sans donnée vocale), puis le rapprochement est recalculé en base à partir des empreintes déjà stockées : ni worker, ni vidéo.

## Opérations

| Opération | Route | Effet |
| --- | --- | --- |
| Fusion | `POST /api/persons/{id}/merge` `{target_person_id}` | La Personne du chemin est absorbée par la cible : ses attributions validées et ses Échantillons vocaux (retirés compris, qui sont de l'historique) passent à la cible, les règles de piste micro et surcharges de clip qui la nomment aussi. Elle disparaît. |
| Séparation | `POST /api/persons/{id}/separation` `{local_speaker_ids}` | La validation des voix nommées est retirée : elles redeviennent inconnues et peuvent être validées de nouveau ; leurs Échantillons quittent les références. La Personne reste. |
| Suppression | `DELETE /api/persons/{id}` | Toutes ses validations sont retirées (voix inconnues), ses Échantillons effacés, les règles de piste micro et surcharges qui la nomment supprimées, les attributions machine qui la nommaient recalculées. Clips et empreintes restent. |
| Trace | `GET /api/person-events[?person_id=]` | Corrections de Personnes, les plus récentes d'abord, y compris pour une Personne qui n'existe plus. |

Chaque réponse donne ce qui a changé : `attributions`, `samples`, `microphone_rules`, et `person` (la cible ou la Personne séparée ; `null` après suppression).

## Invariants

- **La première validation fait foi.** Une fusion ne remplace jamais une validation : seule la Personne pointée change (révision +1), l'auteur et la date de la décision d'origine sont conservés, l'auteur de la fusion est dans l'historique. Une voix déjà validée reste refusée à une deuxième validation (`already_validated`).
- **Historique immuable.** Une entrée par voix touchée dans l'historique du Locuteur local (`attribution_events`, actions `merge`, `split`, `delete_person`) avec l'ancienne Personne, l'ancien niveau, l'auteur et la date ; une entrée par correction dans `person_events` (migration 00029), sans clé étrangère, qui survit à la Personne et à l'auteur. Aucune de ces tables n'accepte de modification directe.
- **Références.** Après séparation ou suppression, les voix concernées ne comptent plus parmi les références : le rapprochement ne peut pas reconstituer la Personne depuis des données restantes, et les attributions machine qui la proposaient sont recalculées.
- **Piste micro.** Une règle de source ou une surcharge de clip ne survit pas à la Personne qu'elle nomme : à la fusion elle suit la cible, à la suppression elle est retirée avec une entrée de l'historique de piste micro (`remove`) puis les clips concernés sont réappliqués (le Locuteur local micro disparaît s'il ne porte aucune validation humaine).
- **Comptes.** Une Personne liée à un compte n'est ni absorbée ni supprimée ici (`409 person_linked_account`) : fusionnez l'autre Personne dans elle ; la suppression d'un profil vocal est un autre parcours.
- **Visibilité.** La séparation ne nomme que des voix de clips visibles par l'appelant (sinon `404 local_speaker_not_found`) et refuse tout le lot si une voix n'est pas validée pour la Personne (`409 not_attributed_to_person`, rien n'est changé). Fusion et suppression portent sur la Personne, identité commune à tous.

## Limites

- Le recalcul est synchrone et complet (`matchingChanged(ctx, 0)`), comme pour une validation ; à rendre incrémental en phase 7 si besoin.
- Un échec du recalcul est journalisé sans annuler la correction ; un rapprochement explicite de l'administrateur le répare.

## Interface

La carte « Personnes » liste les Personnes avec leurs références, propose « Fusionner » et « Supprimer » avec confirmation et résultat, et l'historique des corrections. Une voix validée propose « Séparer de cette Personne » (avec confirmation) ; son historique affiche les entrées de fusion, séparation et suppression.

## Tests

Intégration : `TestFoundation/person_corrections` (fusion, séparation, suppression, piste micro, historique, droits) et le test de contrat. Playwright : `web/e2e/person-corrections.spec.ts` (`SNITCH_E2E_PERSON_CORRECTIONS=1`). Données synthétiques uniquement.
