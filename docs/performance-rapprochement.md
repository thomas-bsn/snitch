# Performance du rapprochement en arrière-plan (#392)

Mesure d'un passage complet du rapprochement vocal ([rapprochement en arrière-plan](voice-matching.md#quand-le-rapprochement-sexécute))
sur une bibliothèque synthétique de l'ordre de la bibliothèque visée par la spec [voix brèves](specs/voix-breves.md),
insérée directement en base (jamais la vraie bibliothèque).

Reproduire, depuis `app/` (sans détecteur de concurrence, qui multiplie les temps) :
`SNITCH_PERF=1 go test -tags integration ./internal/httpapi -run 'TestFoundation/voice_matching_performance$' -count=1 -v -timeout 40m`
(avec `DOCKER_HOST` et `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE` comme dans `scripts/test.sh`). La mesure n'est jamais
lancée sans `SNITCH_PERF=1`. Code : `app/internal/httpapi/voice_matching_perf_test.go`.

## Configuration de la mesure

| | |
| --- | --- |
| Date | 2026-11-11 |
| Machine | Apple M3 Pro, 12 cœurs, 18 Go, macOS ; Go 1.27.1 darwin/arm64 (serveur Snitch dans le processus de test) |
| Base | PostgreSQL 18.6 + pgvector 0.8.7 en conteneur Testcontainers, dans une VM Colima de 2 CPU et 2 Go |
| Bibliothèque | 3 000 clips × 10 Locuteurs locaux = **30 000 Locuteurs locaux**, une empreinte de Locuteur local comparable et contributive chacun, 256 dimensions (modèle synthétique), Parole nette 5 s |
| Personnes | **200**, 10 échantillons validés chacune (2 000 échantillons) |
| Répartition | 4 voix sur 5 appartiennent à une Personne (son centre + bruit uniforme), 1 sur 5 est une voix inconnue |
| Réglages | seuils par défaut (suggestion 0,75, automatique 0,90, marge 0,05), automatisation désactivée |
| Mode de score | `cumulative` (défaut), puis `best_sample` |

Les chiffres donnent un ordre de grandeur sur un poste de développement, pas un seuil ; la base dispose de peu de
mémoire (2 Go pour toute la VM).

## Résultats

« Fin de » donne le temps écoulé depuis le début du passage à la fin de chaque phase : calcul des scores (lecture des
vecteurs et fonction de score en Go), écriture des attributions machine, commit (après le décompte par clip),
index HNSW, proximité des voix inconnues (`refreshVoiceGroupLinks`), annonces des voix à traiter
(`syncAllSpeakerReviews`). Le reste (index HNSW déjà présent, rafraîchissement des statistiques au premier passage)
prend au plus 2,4 s.

| Passage | Durée totale | Scores | Écritures | Commit | Proximité des inconnues | Annonces | Résultat |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Cumulatif, premier passage (22 000 suggestions écrites) | **72 s** | 1,8 s | 37,2 s | 3,6 s | 17,0 s | 10,3 s | 28 000 évaluées, 22 000 suggestions, 6 000 inconnues |
| Cumulatif, passage sans changement | **31 s** | 1,9 s | 0,1 s | 3,6 s | 17,0 s | 8,5 s | identique |
| Cumulatif, passage demandé par une validation (arrière-plan) | **32 s** | 1,8 s | 0,2 s | 3,4 s | 16,8 s | 9,5 s | 1 demande |
| Meilleur échantillon, premier passage (scores réécrits) | **62 s** | 13,9 s | 16,1 s | 4,5 s | 17,2 s | 10,2 s | 17 656 suggestions, 10 343 inconnues |
| Meilleur échantillon, passage sans changement | **44 s** | 13,9 s | 0,1 s | 3,4 s | 17,4 s | 8,9 s | identique |

- **La validation ne dépend plus du passage** : une validation envoyée pendant un passage complet a répondu en 12 ms.
  Elle s'est ajoutée au passage suivant, qui a pris en compte une seule demande.
- Le calcul des scores lui-même est court (1,8 s en cumulatif, 14 s en meilleur échantillon, qui compare chaque voix
  aux 2 000 échantillons au lieu de 200 Références).
- Le premier passage est dominé par l'écriture ligne par ligne de 22 000 attributions, de leurs candidates et de
  leurs événements ; ensuite, seules les lignes qui changent sont réécrites.
- Les deux traitements qui suivent le commit (proximité des voix inconnues, annonces) coûtent environ 26 s à chaque
  passage complet, même sans changement.

## Corrigé pendant la mesure

La première mesure a dépassé la limite de 5 minutes d'un rapprochement : écritures et commit en 42 s, puis la
proximité des voix inconnues a duré plus de quatre minutes et a été interrompue par la limite, et les annonces des voix
à traiter n'ont pas été faites. Deux causes, deux corrections :

- **Statistiques périmées** : 22 000 attributions venaient d'être insérées, l'autovacuum n'était pas encore passé
  et le planificateur choisissait un mauvais plan pour la requête de proximité. Un passage qui insère ou supprime au
  moins 1 000 attributions machine rafraîchit maintenant les statistiques (`ANALYZE speaker_attributions,
  speaker_attribution_candidates`, `analyzeChurn`). Proximité : plus de 255 s → 17 s.
- **Budget partagé** : les traitements après le commit héritaient de l'échéance de la transaction de rapprochement.
  Chacun a maintenant son propre budget de 5 minutes : une proximité lente ne prive plus les voix à traiter de leur
  annonce.

Premier passage complet : plus de 5 minutes, interrompu → **72 s**, complet.

## Laissé volontairement

- **Écriture ligne par ligne** des attributions (37 s pour 22 000 au premier passage) : un premier passage sur toute
  une bibliothèque est rare. Ensuite, seules les décisions qui changent sont écrites. Écrire par lots est le
  prochain levier si un premier passage gêne.
- **Proximité des voix inconnues et annonces recalculées en entier** à chaque passage complet (environ 26 s) : à
  rendre incrémentales si les passages deviennent fréquents. Le verrou du rapprochement reste tenu pendant ce temps :
  une fusion, une séparation ou une suppression de Personne attend la fin du passage en cours. Une validation
  n'attend jamais.
- **Décompte par clip avant le commit** (`clipMatchSummaries`, une requête par clip, environ 3,5 s pour 3 000 clips) :
  il ne sert qu'au journal.
