# 0056 — Benchmarks IA isolés : jeux de données, runs et résultats

Date : 2026-11-09. Statut : accepté pour la première tranche de #287 (étape empreinte vocale). S'appuie sur l'[ADR 0038](0038-modeles-locaux-pilotes-par-snitch.md) (modèle demandé, groupe d'empreintes de benchmark) et l'[ADR 0044](0044-changement-du-modele-d-empreinte.md) (changer le modèle actif reste une transition distincte).

## Contexte

L'administrateur doit comparer plusieurs modèles d'une même étape sur les mêmes clips avant de choisir celui de production. Les empreintes d'un modèle demandé sont déjà isolées de la production. Il manquait de quoi décrire une vérité terrain, orchestrer un essai, le mesurer et conserver le résultat.

## Décision

- **Le jeu de données est un instantané.** Il est construit depuis des voix validées par un humain, jamais depuis une attribution automatique. La Personne de chaque voix est copiée à la création : corriger une Personne ensuite ne change pas un jeu existant, donc un essai reste reproductible. Chaque voix est soit *référence* (enrôlement), soit *test* ; une voix ne peut pas être les deux, et tout test a une référence de sa Personne. Il faut au moins deux Personnes, sans quoi aucun faux positif n'est mesurable. Une voix de test ne peut pas provenir du clip d'une référence de la même Personne : le score mesurerait l'enregistrement et non le modèle. Les voix de la piste microphone sont exclues : elles sont désignées, pas reconnues.
- **Un run n'écrit que dans ses propres tables.** Ses empreintes partagent le drapeau `benchmark` avec la zone de transition de l'ADR 0044 mais n'en font pas partie : les requêtes de la transition ignorent les empreintes issues des tâches d'un run (`benchmark_run_jobs`), qui ne sont donc ni comptées comme recalculées ni promues au basculement. Un run ne démarre pas pendant une transition ouverte. Il programme des tâches à modèle demandé (une par clip, piste et diarisation, par modèle). Ni l'identification, ni l'attribution, ni le modèle actif, ni les validations ne sont lus ou modifiés. Un seul run est actif à la fois : deux runs sur le même clip et le même modèle partageraient une tâche.
- **Les résultats sont calculés à la lecture**, quand toutes les tâches du run ont fini, dans une transaction qui verrouille le run : idempotent, repris après redémarrage, sans compteur à tenir. Démarrer un run règle d'abord celui qui est resté « en cours » sans que personne le lise, pour qu'il ne bloque pas le suivant. Seules les empreintes produites par les tâches du run comptent, jamais celles d'un essai précédent.
- **Métriques par modèle, jamais fusionnées.** Centroïde par Personne (moyenne des références normalisées), score cosinus du test contre chaque centroïde. *Remplacé par #397 (ADR 0064) : la Référence et le score sont ceux de la production, sous le mode et la règle d'éligibilité enregistrés avec le run ; les runs antérieurs s'affichent « centroïde (ancien) ».* EER et son seuil, exactitude top-1, puis décision ouverte au seuil EER propre au modèle : vrais positifs, faux positifs (mauvaise Personne acceptée), faux négatifs (rejeté ou mal nommé), précision, rappel. Les scores cosinus de deux modèles ne sont pas sur la même échelle : aucune comparaison de score brut, chaque modèle porte son seuil.
- **Un résultat dit sa couverture.** `complete`, `partial` (certaines voix sans empreinte) ou `failed` ; le modèle, sa version et ses dimensions sont recopiés à la fin du run.
- **Annulation** : le run passe à `cancelled` et ses seules tâches non terminées sont annulées. Aucune tâche de production n'est touchée. Une tâche qui finit entre-temps écrit des empreintes de benchmark que plus aucun run ne lit : sans effet sur la production.
- **Décision ouverte** : précision, rappel, faux positifs et faux négatifs se lisent au seuil EER du modèle sur la Personne la mieux notée de chaque test ; une mauvaise Personne acceptée compte à la fois comme faux positif et comme faux négatif. Les effectifs sont petits : ces nombres décrivent un jeu précis, sans garantie.
- **Accès administrateur uniquement**, y compris la lecture.

## Alternatives écartées

- **Vérité terrain lue en direct dans les validations** : un essai ne serait plus reproductible après une correction.
- **Comparer les scores cosinus entre modèles** : échelles différentes, comparaison trompeuse (demandé explicitement par #287).
- **Un moteur de run concurrent** : plusieurs essais simultanés se disputeraient des tâches partagées ; un seul actif suffit pour un MVP.
- **Résultats en mémoire ou compteur de progression** : perdus au redémarrage.

## Conséquences

- Un jeu perd la voix d'un clip supprimé ou re-diarisé (cascade) ; les résultats déjà calculés gardent leurs effectifs. Un jeu utilisé par un run ne peut pas être supprimé.
- Le temps écoulé d'un modèle va de la création de ses tâches à la fin de la dernière, file d'attente comprise : ce n'est pas une latence d'inférence. RAM et VRAM ne sont pas remontées par le Worker : `resources` vaut `null`.
- Hors de cette tranche : diarisation (DER), transcription via le service externe (WER/CER), séparation, recherche sémantique, l'écran Admin → AI Models (bloqué par #281) et l'activation du modèle de production avec plan de réanalyse (la transition de l'ADR 0044 en est la base).
