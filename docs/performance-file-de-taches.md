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
