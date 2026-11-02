# 0026 — Dossier privé : une règle d'accès unique, par provenance

Statut : accepté pour #93.

## Contexte

Le Super-administrateur veut rendre un dossier privé : les autres utilisateurs voient son nom avec un cadenas, mais ne peuvent ni le parcourir ni ouvrir ses clips. La règle d'accès à un clip (« partagé, ou propriétaire, ou administrateur ») était recopiée dans une quinzaine de requêtes : listes, recherche, compteurs, notifications, voix, fichiers. Un clip est dédupliqué par son contenu ; il peut donc avoir plusieurs provenances (plusieurs fichiers, un dépôt manuel).

## Décision

- **Granularité** : un dossier privé est un dossier de premier niveau d'une Source, celui que la bibliothèque propose déjà en navigation. Il est enregistré à part (`private_folders`) et ne modifie pas la visibilité des clips.
- **Une seule règle**, en SQL : `clip_readable(clip, propriétaire, visibilité, utilisateur, administrateur)`. Toute lecture d'un clip passe par elle. Le propriétaire et le Super-administrateur lisent toujours ; les autres ont besoin de la visibilité propre au clip **et** d'un dossier ouvert.
- **Les administrateurs ne passent pas** : seul le Super-administrateur ouvre un dossier privé.
- **Une provenance visible suffit** : un clip présent dans un dossier privé reste accessible s'il a aussi un fichier présent hors de tout dossier privé, ou un dépôt manuel qui n'est pas destiné à un dossier privé. Une copie manquante ne compte pas : déplacer un fichier dans un dossier privé le masque.
- **Filtrer par un dossier privé ne renvoie rien** à qui ne peut pas l'ouvrir, même pour un clip lisible par une autre provenance : on ne révèle pas ce qu'il contient.
- **Contrôle d'accès** : seul le Super-administrateur peut modifier le réglage (`PUT /api/admin/library/folders/privacy`).

## Alternatives écartées

- Réutiliser `clip_sources.visibility` : elle est recopiée sur le clip à l'indexation, donc figée. Elle masquerait une Source entière, pas un dossier.
- Recopier la règle dans chaque requête en ajoutant une condition : quinze copies, chacune pouvant être oubliée lors d'un ajout.
- Laisser le dossier privé l'emporter sur toute autre provenance : contredit #93 (une source privée ne change pas la visibilité d'un clip venant aussi d'une source autorisée).

## Conséquences

- Les nouvelles requêtes qui lisent un clip doivent appeler `clip_readable` ; un `JOIN clips` sans elle est un défaut d'accès.
- Les écrans d'exploitation des administrateurs (file de tâches, Workers, sources) ne sont pas filtrés : ils montrent des tâches, pas le contenu des clips.
- La règle évalue des sous-requêtes par clip ; c'est négligeable à l'échelle d'une bibliothèque personnelle. Si une liste devient lente, il faudra matérialiser l'état « ouvert » du clip.
