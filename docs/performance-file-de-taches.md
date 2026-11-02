# Performance de la file de tâches (#117)

Mesure de la Task Queue (`AnalysisBoard`, `GET /api/admin/analysis/queue` et son flux
SSE) sur 3000 exécutions d'analyse actives synthétiques, insérées directement en base
(jamais la vraie bibliothèque) : 6 étapes par exécution, des jobs réalistes (en attente,
en cours sur un Worker, en échec avec cause, en nouvel essai, voix à vérifier), plus
3 exécutions terminées et 2 tâches hors exécution.

Reproduire, depuis `app/` (le SPA doit être construit ; `SNITCH_PERF_PAGE=1` ouvre la
page dans Chromium via Playwright) :
`SNITCH_PERF=1 SNITCH_PERF_PAGE=1 go test -tags integration ./internal/httpapi -run 'TestFoundation/task_queue_performance' -count=1 -v`.
La mesure n'est jamais lancée sans `SNITCH_PERF=1`. Machine de mesure : portable de
développement, PostgreSQL 18 en conteneur ; les chiffres donnent un ordre de grandeur,
pas un seuil.

## Avant

| Mesure | Valeur |
| --- | --- |
| Durée de `GET /api/admin/analysis/queue` (médiane de 5) | 570 ms |
| Requêtes SQL par appel | 10 (constant, pas de N+1) |
| Temps SQL dominant | 470 ms sur la requête des tâches hors exécution (balayage de `clip_analysis_steps` pour chaque job) |
| Taille de la réponse brute | 5,8 Mo (1,9 Ko par exécution) |
| Taille compressée (gzip mesuré ; le serveur ne compresse pas lui-même) | 99 Ko |
| Premier snapshot du flux SSE | 5,8 Mo |
| Évènement SSE après le changement d'une étape | 5,8 Mo (snapshot complet), 650 ms de latence |
| Limite sur les exécutions actives | `LIMIT 100000` silencieux |
| Page : nœuds DOM | 150 581 (3003 cartes montées) |
| Page : mémoire JS après ramasse-miettes | 66,7 Mo |
| Page : tâches longues au chargement | 2 (470 ms au total, 287 ms au plus) |
| Page : tâche longue par mise à jour | 1 (142 ms) |

## Corrigé

- **Requête des tâches hors exécution** : l'exclusion « suivie par une exécution » était une
  sous-requête corrélée sans index ; elle est devenue une exclusion à ensemble calculé une
  fois. Même résultat, 470 ms de moins. Aucun index ni migration ajouté.
- **`LIMIT 100000` silencieux** : supprimé. Les exécutions actives ne sont jamais limitées
  côté serveur (le contrat disait déjà « sans pagination »). Les trois exécutions terminées
  gardent leur nombre fixe, qui est une règle métier.
- **Flux SSE** : sans paramètre, il envoie toujours l'état complet (contrat inchangé, pour
  tout autre consommateur). La page demande `?delta=1` : un premier snapshot complet, puis
  des évènements `delta` qui portent l'ordre des exécutions actives (3000 entiers) et le
  contenu des seules exécutions modifiées. Un évènement dont la base n'est pas l'état tenu
  par le client fait rouvrir le flux, donc repartir d'un snapshot complet. Seule la page de la
  file consomme ce sujet ; la page d'un clip et le centre de notifications ont leurs propres
  flux et n'ont pas changé.
- **Page** : chaque colonne dessine 50 cartes d'abord et affiche « N autres cartes de cette
  colonne ne sont pas encore affichées » avec un bouton « Afficher 100 de plus (M restantes) » ;
  le compteur de la colonne et le titre « Analyses en cours · N » restent exacts. Les cartes
  sont mémorisées : un delta ne redessine que celles qui ont changé. `content-visibility`
  seul ne suffisait pas (il ne réduit pas le nombre de nœuds, ni la mémoire).

## Après

| Mesure | Avant | Après |
| --- | --- | --- |
| Durée de `GET /api/admin/analysis/queue` (médiane de 5) | 570 ms | 92 ms (87 à 112 ms selon les passes) |
| Requêtes SQL par appel | 10 | 10 |
| Réponse brute | 5,8 Mo | 5,8 Mo (inchangée : l'état complet reste complet) |
| Réponse compressée (gzip mesuré) | 99 Ko | 99 Ko |
| Évènement SSE après le changement d'une étape (page) | 5,8 Mo, 650 ms | 18 Ko (delta), 120 ms |
| Évènement SSE sans `delta` (autres clients éventuels) | 5,8 Mo, 650 ms | 5,8 Mo, 131 ms |
| Limite sur les exécutions actives | `LIMIT 100000` silencieux | aucune |
| Page : nœuds DOM | 150 581 | 10 214 (203 cartes montées sur 3003) |
| Page : mémoire JS après ramasse-miettes | 66,7 Mo | 13,0 Mo |
| Page : tâches longues au chargement | 2 (470 ms) | 0 à 1 (au plus 51 ms) |
| Page : tâche longue par mise à jour | 142 ms (rechargement complet) | aucune (delta, une étape toutes les 500 ms) ; aucune non plus au rechargement complet |
| Page : premier affichage d'une carte | 1237 ms | 282 à 378 ms |

Objectifs indicatifs : moins de 300 ms (atteint), une vingtaine de requêtes au plus
(atteint, 10), moins de 300 Ko compressés (atteint, 99 Ko).

## Laissé volontairement

- **La réponse brute reste de 5,8 Mo** : l'application ne compresse pas ses réponses, c'est
  le rôle du reverse proxy. Si le déploiement n'en a pas, ajouter une compression gzip aux
  réponses JSON et au flux est le prochain levier (rapport 60 pour 1 mesuré).
- **Le flux ne filtre pas côté base** : à chaque changement le serveur relit la file entière
  (92 ms à 3000 exécutions) avant d'en tirer le delta ; au plus un évènement par seconde et
  par client. Un delta calculé en base demanderait un journal des changements par exécution.
- **Pas de virtualisation à défilement** : la fenêtre par colonne suffit aux mesures ; une
  colonne que l'administrateur déploie entièrement (3000 cartes) retrouve le coût d'avant.
- **Pas de limite de sécurité côté serveur** : à ce volume rien ne la justifie ; si elle
  devenait nécessaire, elle devrait ajouter au contrat un compteur « N autres » affiché.
