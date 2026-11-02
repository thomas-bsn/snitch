# 0024 — Adresses de Personnes, de Cas de vérification et de Groupes

Statut : proposé pour l'épopée #121. Précise l'[ADR 0014](0014-refonte-du-front.md) sans la contredire.

## Contexte

Les notifications humaines de #115 (#133) doivent mener au bon cas, et l'interface People n'a aujourd'hui aucune adresse : détail dans une ligne dépliée, état en mémoire. Deux obstacles dans l'API :

- `VoiceGroup.id` est « la plus petite voix visible du groupe ». Il change dès qu'une voix est validée, écartée ou masquée : un lien par id de groupe peut ouvrir un autre groupe, ou aucun.
- Il n'existe pas de lecture d'un Locuteur local seul. La file de vérification ne renvoie que les cas `unknown` ou `ambiguous` : un cas déjà tranché, inexistant ou invisible y est indistinguable.

## Décision

- **Adresses** : `/people/:id` (Personne), `/people/verification?case=<local_speaker_id>` (Cas de vérification), `/people/groups?voice=<local_speaker_id>` (le Groupe qui contient cette voix aujourd'hui). #121 fournit ces cibles ; #133 émet les notifications et ne fait que les consommer.
- **Groupe adressé par une voix, pas par son id.** Si la voix n'est plus dans aucun Groupe, l'écran le dit (« déjà traitée ou plus visible ») au lieu d'ouvrir un autre groupe.
- **Lecture d'un Locuteur local** : `GET /api/local-speakers/{id}` renvoie son niveau de décision, la Personne attribuée le cas échéant et le clip, avec les mêmes règles de visibilité que le reste. Inexistant et invisible répondent le même 404, pour ne pas révéler l'existence d'un clip masqué. Cela permet de distinguer « déjà résolu » de « introuvable ».
- **« Passer » local à la session** : le cas descend en fin de file, sans écriture ni donnée nouvelle. On enchaîne sur le cas suivant après « Confirmer ».

## Alternatives écartées

- Identifiant de Groupe persistant : nouveau modèle de données et règles de fusion/scission de groupes, hors du périmètre « conserver le backend » de #121.
- « Différé » persistant pour « Passer » : nouvelle donnée par utilisateur et par cas, à purger ; le relecteur qui recharge reverrait de toute façon la file.
- Distinguer 403 et 404 pour un cas invisible : révélerait qu'un clip masqué existe.

## Conséquences

- #133 dépend de #137, #138 et #139 pour ses cibles ; son ordre de livraison est à revoir.
- Un lien de groupe conservé dans une notification reste valable tant que la voix visée est non traitée ; après nomination, il affiche l'état « déjà traitée », pas une erreur.
- Un « passer » ne survit pas au rechargement : un cas difficile peut revenir en tête. C'est accepté tant que la file est triée par fréquence ; à revoir si les relecteurs s'en plaignent.
