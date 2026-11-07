# 0046 — Menu contextuel, logo et propriétés de la Bibliothèque

Statut : accepté. Complète l'[ADR 0032](0032-explorateur-du-disque-et-indexation-a-la-demande.md) et reprend la règle des [Dossiers privés](0026-dossier-prive.md).

## Contexte

L'explorateur de la Bibliothèque n'offrait ses actions que par un menu à trois points, dans la seule vue liste, et seulement sur les dossiers des administrateurs. La vue icônes n'avait aucun menu. Les colonnes Durée, Propriétaire et Ajouté le venaient du clip : elles restaient vides pour une vidéo jamais indexée, et la taille d'un dossier n'était pas calculée. L'utilisateur veut les mêmes actions dans les deux vues, un logo propre à chaque dossier et des propriétés pour les dossiers comme pour les vidéos (#323).

## Décision

- **Un seul modèle d'actions** alimente le menu à trois points de la vue liste et le menu contextuel (clic droit ou appui long) des lignes et des tuiles. Un dossier propose l'analyse des vidéos non analysées (Administrateur), le passage privé ou partagé (Super-administrateur, dossier de premier niveau), le changement de logo (Administrateur) et les propriétés. Une vidéo propose ses propriétés. Un dossier privé verrouillé n'a aucune action : son contenu n'est pas atteignable.
- **Le logo vient de l'instance** : un Administrateur importe un PNG ou un JPEG de 2 Mo et 16 mégapixels au plus. L'image est vérifiée par décodage complet puis stockée telle quelle, sans ré-encodage. Aucun appel à un service distant n'est fait pour l'afficher ; la jaquette Steam automatique reste le repli.
- **L'identité du logo est le chemin** : un logo appartient à une Source et à un chemin relatif, jamais à un nom de dossier. Renommer ou déplacer un dossier ne fait donc pas hériter du logo d'un autre.
- **Quatre versions au plus** : le logo actif et les trois précédents sont conservés. Restaurer une ancienne image la copie dans une nouvelle version, comme le fait Discord ; revenir à la jaquette automatique conserve l'historique. La liste porte le numéro de version active, ce qui casse le cache du navigateur quand le logo change.
- **Un dossier privé reste au Super-administrateur** : lire ou modifier son logo, demander ses propriétés ou y écrire suit la règle de l'ADR 0026. Pour les autres, le dossier est verrouillé et aucune de ces routes ne révèle son contenu.
- **Les propriétés sont calculées côté serveur** : taille sur le disque, nombre de vidéos et de sous-dossiers, vidéos indexées, durée cumulée des clips indexés, propriétaires et dates. Un dossier agrège tout son sous-arbre, borné à 10 000 entrées comme la liste, et ne compte que les clips que l'appelant peut lire. Une vidéo répond pour son fichier, y compris quand il n'a pas encore de clip.
- **La vue liste garde le parcours clavier** : les trois points y restent le moyen d'ouvrir le menu au clavier. La vue icônes ajoute le clic droit et l'appui long sur la tuile, avec les mêmes actions.

## Alternatives écartées

- **Deux menus écrits séparément pour la liste et les icônes** : les actions auraient divergé à la première évolution.
- **Identifier le logo par le nom du dossier** : un renommage ou un dossier homonyme aurait volé le logo d'un autre.
- **Ré-encoder les images** : cela dégraderait une jaquette déjà petite et changerait son contenu sans raison.
- **Choisir une URL distante ou une autre jaquette Steam** : l'utilisateur a demandé l'import d'une image et le retour à l'automatique, pas un choix parmi des sources distantes.
- **Calculer la taille d'un dossier dans le navigateur** : le navigateur ne voit pas le disque, et le faire au fil des tuiles multiplierait les parcours.
- **Historique illimité** : les images vivraient indéfiniment dans la base pour un choix qui reste visuel ; quatre versions suffisent à revenir en arrière.

## Conséquences

- Une migration ajoute `folder_logo_choices` et `folder_logo_versions` (00107). Les images sont dans la base, comme les avatars, et suivent la sauvegarde.
- Le contrat HTTP gagne l'import, la remise à l'automatique, l'historique, la restauration et les propriétés. La version active apparaît aussi dans l'explorateur (`logo_version`).
- Les propriétés d'un gros dossier coûtent un parcours du disque et une lecture par lots des clips ; elles restent bornées et explicites, comme l'analyse et l'indexation.
- Les colonnes Durée, Propriétaire et Ajouté le restent vides pour une vidéo jamais indexée : les propriétés expliquent cet état au lieu de le masquer.
