# Performance des cartes de la file de tâches

Mesure faite pour le ticket #181 (épopée #117) sur des clips et des tâches synthétiques insérés directement en base. Aucune donnée de la vraie bibliothèque. La mesure a été faite sur la branche d'intégration de #117, avant tout changement du contrat de cartes.

## Conditions

- Serveur et PostgreSQL 18 locaux (conteneur Colima), aucune latence réseau, pool de 4 connexions, portable Apple Silicon. Les durées sont donc plutôt un plancher.
- Page : Chromium sans interface ni GPU piloté par Playwright. La réponse enregistrée de la mesure serveur (3000 cartes actives) est rejouée par le navigateur ; les autres appels (contrôles, workers) sont réels.
- Chaque durée est la médiane de 5 appels de `GET /api/admin/task-cards` par un administrateur.

## Serveur : durée, requêtes SQL, taille

| Jeu de données | Clips avec tâches | Durée médiane | Requêtes SQL | Réponse |
|---|---|---|---|---|
| A. transcrits seulement (aucune carte active, aucun terminé) | 3000 | 3,3 s | 15 008 | 97 octets |
| B. 3000 cartes actives (1500 en file, 300 en cours, 600 en voix distinctes, 300 épuisées, 300 annulées) | 3000 | 3,5 s | 15 009 | 2,4 Mo (environ 810 octets par carte) |
| C. B plus 3000 clips transcrits seulement | 6000 | 6,9 s | 30 009 | 2,4 Mo |

Constats :

- Le coût est de 5 requêtes SQL par clip examiné (pistes audio du clip, surcharge de piste, règles de piste de la source, deux lectures de la désignation du micro), soit environ 1,1 ms par clip. Le nombre de requêtes croît linéairement avec le nombre de clips qui ont une tâche, pas avec le nombre de cartes renvoyées.
- Les clips transcrits mais jamais analysés (jeu A) coûtent autant que des cartes actives alors qu'ils ne produisent aucune carte : la colonne Terminé parcourt tous les clips « finis » jusqu'à en trouver trois réellement terminés, et construit une carte complète pour chacun.
- La page interroge toutes les 5 s. Avec 3,5 à 7 s de calcul serveur, la file occupe une connexion de base en continu.

## Page : 3000 cartes actives

| Mesure | Valeur |
|---|---|
| Nœuds DOM | 69 247 (environ 23 par carte) |
| Cartes montées | 3000 (aucune virtualisation, comme l'implémentation actuelle) |
| Premier affichage de la première carte (réponse locale) | 0,45 s |
| Mémoire JS | 58 Mo |
| Hauteur de la page | environ 255 000 px |
| Défilement de haut en bas (120 pas) | image médiane 38 ms, 95e centile 56 ms, pire image 170 ms |
| Tâches longues pendant le chargement et le défilement | 4, la plus longue 149 ms |
| Un cycle de polling (application de 3000 cartes) | 2 tâches longues, la plus longue 119 ms |

Les miniatures restent bornées : seules les cartes proches de la fenêtre montent une vidéo (parcours de la file, miniatures différées). Les 3000 observateurs d'intersection sont créés, mais aucune vidéo n'est montée hors fenêtre.

## Jugement

- Serveur : mauvais. 3 à 7 secondes et 15 000 à 30 000 requêtes SQL par rafraîchissement ne tiennent pas un polling de 5 s sur une grosse file, ni sur une grosse bibliothèque de clips transcrits.
- Taille de réponse : mauvaise pour un polling (2,4 Mo toutes les 5 s, soit près de 0,5 Mo/s), sans compression mesurée.
- Page : passable à 3000 cartes (0,45 s au premier affichage, environ 26 images par seconde au défilement, une tâche longue de 120 ms à chaque rafraîchissement), mais 69 000 nœuds et aucune virtualisation, contrairement à la spec de #117.

## Recommandations chiffrées (contrat inchangé)

1. Supprimer les 5 requêtes par clip : charger en une requête par table les pistes, surcharges, règles de source et désignations de micro des clips concernés (cinq requêtes au total, quel que soit le nombre de clips). Objectif : moins de 20 requêtes par appel et moins de 300 ms pour 3000 cartes.
2. Borner la colonne Terminé : sélectionner en SQL les trois clips les plus récemment terminés (ou les premiers candidats par date de dernière tâche) au lieu de construire une carte pour chaque clip fini. Le jeu A et la part « transcrits seulement » du jeu C tombent alors à un coût constant.
3. Réduire la réponse : omettre les champs vides et le détail des étapes quand il se déduit de la colonne, et activer la compression HTTP. Objectif : moins de 300 Ko non compressés pour 3000 cartes.
4. Éviter de tout renvoyer à chaque cycle : requête conditionnelle (ETag) ou, avec #131, événements ne portant que les cartes modifiées.
5. Page : fenêtrer le rendu (ne monter que les cartes proches de la fenêtre, ou au minimum `content-visibility: auto` sur les cartes) pour passer de 69 000 à quelques milliers de nœuds. Non mesuré ici.

La mesure se rejoue avec `SNITCH_PERF=1` (serveur, jeu de données synthétique, test `task cards performance` de `TestFoundation`) et, pour la page, `SNITCH_PERF_CARDS=<fichier>` avec `SNITCH_PERF_DUMP=<fichier>` (test « mesure » du parcours de la file). Les deux sont ignorés par la CI.
