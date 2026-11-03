## Problem Statement

Les lots A à D et la fiche vidéo ont été livrés avec leurs comportements, mais **l'interface réelle ne ressemble pas aux prototypes validés** par l'utilisateur. Les tickets citaient la branche `prototype/fiche-video` comme référence visuelle, mais rien ne permettait de la comparer : les agents ont testé des fonctions (Go, Playwright), la revue de code ne contrôlait que les standards et la spec (des comportements, pas une mise en page). Un écran peut passer tous les tests et ne pas ressembler à la maquette. Exemple constaté : le bouton « Tous les clips » est resté de l'ancienne Bibliothèque alors que la maquette retenue n'en a pas.

## Solution

Aligner chaque écran sur son prototype, avec une **boucle de vérification visuelle obligatoire** : pour chaque écran, les états du prototype sont capturés comme références, l'écran réel est lancé en local avec les mêmes données de test, rendu et capturé, puis comparé à la référence avec `/frontend-audit` (couleurs, formes, espacements, styles calculés) et piloté avec `/agent-browser` (états, clics). On corrige jusqu'à un écart nul ou expliqué. L'utilisateur voit la comparaison côte à côte et valide **écran par écran** avant tout commit sur la PR.

Écrans et variantes retenues (branche `prototype/fiche-video`, serveur local `cd web && npm run dev -- --port 5199`) :

| Écran | Prototype | Variante |
|---|---|---|
| Fiche vidéo | `/prototype/fiche-video?variant=C` | C |
| People (annuaire, À traiter, Voix inconnues) | `/prototype/people?variant=D` | D |
| Écran de traitement d'un clip | `/prototype/people?variant=D&traiter=40` | D |
| Bibliothèque (explorateur) | `/prototype/library?variant=D` | D |
| Import | `/prototype/import?variant=A` | A |

## Règles de fidélité

- **Ce qui doit être identique** : structure et disposition, hiérarchie, typographie, couleurs (jetons du thème), espacements, formes, libellés, présence et absence d'éléments (ex. aucun bouton « Tous les clips »).
- **Ce qui est comparé dans chaque état** : thème clair et thème sombre ; largeurs 1440 px et 390 px.
- **Ce qui ne se compare pas pixel par pixel** : le lecteur vidéo (factice dans le prototype), les animations, les données du direct simulé. On compare le conteneur, pas le contenu animé.
- **Tolérance** : tout écart visible est soit corrigé, soit listé et accepté par l'utilisateur dans le compte rendu de l'écran.
- Le code du prototype n'est **jamais copié ni fusionné** : l'écran réel est écrit avec les composants de production.

## Données de test communes

Sans données identiques, la comparaison ne vaut rien. Un jeu de données **synthétique** reproduit celui du prototype : le clip « Réunion de préparation — tournage S12 », ses 5 voix, ses 10 segments, ses étapes d'analyse (dont un échec), les personnes de l'annuaire, la liste « À traiter », l'arborescence de l'explorateur. Aucune donnée réelle, aucun scan de la vraie bibliothèque.

## Tickets

1. **Infra de comparaison** : captures de référence des prototypes (états, thèmes, largeurs) rangées dans `docs/ui-references/`, jeu de données de test, script qui lance l'app avec ces données, capture et compare. Bloque tous les autres.
2. **Fiche vidéo** : aligner sur la variante C. Bloqué par 1.
3. **People et écran de traitement** : aligner sur la variante D. Bloqué par 1.
4. **Bibliothèque (explorateur)** : aligner sur la variante D, retirer « Tous les clips » et ce qui reste de l'ancienne page. Bloqué par 1.
5. **Import** : aligner sur la variante A. Bloqué par 1.
6. **File de tâches dans la barre latérale** : la « Task queue » quitte l'administration et devient une entrée permanente de la barre latérale gauche, visible seulement des administrateurs et du super-administrateur (les membres ne la voient pas et ne peuvent pas y accéder). Pas de prototype : critères de comportement, pas de comparaison pixel. L'ancienne adresse d'administration redirige vers la nouvelle. Sur téléphone, la barre d'onglets du bas passe à plus d'entrées : vérifier qu'elle reste lisible. Indépendant de 1.

## Critères d'acceptation (par écran)

- Une comparaison côte à côte (prototype / app réelle) pour chaque état et thème, jointe à la PR.
- Aucun écart non expliqué.
- Les tests existants restent verts ; la CI est verte.
- L'utilisateur a validé l'écran avant le commit.

## Hors périmètre

Les pages sans prototype : pages Workers, joignabilité de la transcription, étape Hugging Face de l'assistant (la file de tâches dans la barre latérale est traitée à part, ticket 6). Aucun changement de comportement métier.
