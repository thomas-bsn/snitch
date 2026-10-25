# Groupes de voix inconnues (phase 5, ticket 36)

Un utilisateur voit des groupes de voix inconnues proches, écoute leurs extraits, exclut ceux qui n'y appartiennent pas et nomme le groupe comme une Personne. Les tests utilisent des empreintes synthétiques (vecteurs dans un plan ou axes orthogonaux).

## Principes

- **Voix inconnue.** Un Locuteur local diarisé sans validation humaine et sans attribution automatique. Une suggestion ou une voix ambiguë reste inconnue (même règle que le filtre par Personne). La piste micro n'est jamais groupée.
- **Singleton par défaut.** Chaque voix inconnue est un groupe à elle seule tant qu'aucun lien de proximité ne la joint à une autre.
- **Liens proposés, jamais confirmés.** `voice_group_links` contient les paires de voix inconnues dont la similarité cosinus atteint le seuil de suggestion des réglages de rapprochement (20 plus proches voisins par voix, via l'index HNSW du ticket 35, dans un même triplet modèle/version/dimensions). Les liens sont reconstruits après chaque rapprochement (validation, correction, empreintes d'un clip, seuils, rematch). Aucun lien ne crée de Personne, d'attribution ni d'échantillon.
- **Groupe calculé, jamais stocké.** Un groupe est l'ensemble connexe des voix reliées par des liens ; il est recalculé en SQL à chaque lecture. Le nombre de voix et de clips distincts vient donc toujours de la base, sur les seuls clips visibles par l'utilisateur. Les groupes sont triés par nombre de clips (ou de voix avec `sort=members`).
- **Visibilité.** Les voix de clips invisibles ne sont ni listées, ni comptées, ni nommées. L'identifiant d'un groupe est sa plus petite voix visible : il peut changer quand les membres changent.
- **Exclusion par voix.** `voice_group_exclusions` retire une voix de son groupe sans modifier les autres : ses propres liens sont ignorés, elle redevient un groupe seul, et un nouveau calcul de proximité ne la rejoint pas. Limite : si la voix exclue était le seul chemin entre deux voix, celles-ci ne restent jointes que si un lien direct existe.
- **Nommer un groupe.** Seule action qui confirme : elle valide chaque voix comme la Personne (attribution humaine, Échantillon vocal validé, historique d'action `group_validate`, auteur et date), dans une transaction, avec les mêmes verrous qu'une validation unitaire. Le client envoie les voix qu'il a écoutées : si le groupe n'est plus exactement cette liste, `409 group_changed` et rien n'est écrit. Une voix validée entre-temps par quelqu'un d'autre est ignorée (`skipped`). Le rapprochement est relancé ensuite.

## API

- `GET /api/voice-groups?sort=clips|members&proposed=true&page&page_size` : groupes et voix (extraits via `GET /api/local-speakers/{id}/excerpt`).
- `GET /api/voice-groups/{id}` : un groupe (`404 voice_group_not_found`).
- `POST /api/voice-groups/{id}/naming` : `{person_id | new_person_name, local_speaker_ids}` ; `409 group_changed`, `404 person_not_found`.
- `PUT` / `DELETE /api/local-speakers/{id}/group-exclusion` : exclure / réintégrer (`409 not_unknown` si la voix est déjà identifiée).

Tout utilisateur connecté, pour les clips visibles. Interface : carte « Voix inconnues à regrouper » (`#voice-groups`).

## Extension prévue (vue de vérification, ticket 37)

`loadVoiceGroups` accepte un filtre et renvoie des groupes complets : la vue de vérification (ticket 37, voir voice-verification.md) en réutilise la définition SQL (voiceGroupsCTE). Limite assumée : le calcul des composantes et des liens est complet à chaque lecture/rapprochement ; à rendre incrémental en phase 7 si les volumes l'exigent.
