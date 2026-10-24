# Snitch — Phase 3 : préparation audio, transcription et recherche textuelle

Issue GitHub : https://github.com/thomas-bsn/snitch/issues/19 (`ready-for-agent`).

## Problem Statement

La bibliothèque permet de retrouver et lire les clips, et la file peut distribuer des tâches à un worker. Les paroles ne sont pas encore extraites : il faut parcourir manuellement les vidéos pour retrouver une phrase, et un traitement par lots ne peut pas encore distinguer proprement la voix des autres pistes audio.

## Solution

Ajouter un pipeline de phase 3 qui prépare l’audio des clips lisibles, transcrit les paroles en français avec des timestamps au niveau des mots, et rend ces paroles consultables depuis Snitch. L’administrateur peut configurer le vocabulaire de jeu et les règles de sélection des pistes audio. Les tâches restent durables, versionnées et visibles dans la file existante. Les utilisateurs peuvent rechercher une expression, distinguer les correspondances exactes des approximations, puis ouvrir le clip au moment correspondant.

Le traitement s’exécute sur le worker via HTTPS. Les modèles et fournisseurs restent remplaçables ; la phase ne désigne pas de modèle final ni ne promet de qualité ou de débit avant le benchmark de phase 7. Les tests utilisent des médias synthétiques dédiés et un transcripteur déterministe. Aucun clip de la vraie bibliothèque n’est traité avant l’autorisation séparée du benchmark.

## User Stories

1. En tant qu’administrateur, je veux planifier la préparation audio et la transcription d’un clip prêt, afin de lancer le traitement depuis la file durable existante.
2. En tant qu’administrateur, je veux planifier le traitement d’un ensemble de clips répondant aux filtres de la bibliothèque, afin d’amorcer un lot sans devoir créer chaque tâche séparément.
3. En tant qu’exploitant, je veux qu’un clip en attente, illisible ou sans original disponible ne soit pas transcrit comme s’il était prêt, afin d’éviter des résultats incomplets.
4. En tant qu’exploitant, je veux que les tâches de transcription n’aient besoin que d’un worker déclarant cette capacité, afin qu’un worker factice ou incompatible ne prenne pas le travail.
5. En tant qu’exploitant, je veux que les tâches restent en attente lorsque le worker adapté est absent, afin qu’elles puissent reprendre quand il revient.
6. En tant qu’exploitant, je veux que le pipeline reprenne après un arrêt ou l’expiration d’un lease, afin qu’un clip ne reste pas bloqué par un worker disparu.
7. En tant qu’administrateur, je veux voir l’étape, la version d’entrée, les tentatives, l’état et l’erreur expurgée pour chaque traitement audio, afin de comprendre ce qui doit être corrigé ou relancé.
8. En tant qu’administrateur, je veux relancer une transcription après avoir corrigé sa configuration, afin de remplacer son résultat sans dupliquer les segments.
9. En tant qu’exploitant, je veux qu’une modification de piste, de vocabulaire, de préparation audio ou de modèle invalide le résultat de transcription concerné, afin qu’un ancien résultat ne soit pas présenté comme actuel.
10. En tant qu’exploitant, je veux qu’une nouvelle version de transcription n’invalide pas les métadonnées du clip ni les tâches d’autres étapes, afin de limiter le retraitement.
11. En tant qu’utilisateur, je veux que Snitch inspecte les pistes audio disponibles d’un clip, afin de pouvoir distinguer la voix du jeu et les pistes de mixage.
12. En tant qu’administrateur, je veux consulter les numéros, titres, langues et dispositions de canaux des pistes détectées, afin de choisir la bonne piste sans deviner à partir du nom du fichier.
13. En tant qu’administrateur, je veux définir une règle de sélection des pistes par source et une règle de repli, afin d’appliquer une configuration cohérente à un lot.
14. En tant qu’administrateur, je veux pouvoir sélectionner une piste précise pour un clip ambigu, afin de corriger une exception sans modifier tous les clips.
15. En tant qu’exploitant, je veux que la sélection ne supprime jamais silencieusement une piste audio inconnue, afin de ne pas perdre des paroles présentes sur une autre piste.
16. En tant qu’exploitant, je veux qu’un clip avec plusieurs pistes et sans règle de sélection non ambiguë soit signalé comme nécessitant une sélection, afin qu’une piste de jeu ne soit pas prise à tort pour le microphone.
17. En tant qu’utilisateur, je veux que la préparation convertisse la piste retenue en audio normalisé à 16 kHz mono, afin que les fournisseurs de transcription reçoivent un format prévisible.
18. En tant qu’exploitant, je veux que la préparation conserve la relation entre les échantillons transformés et le temps du clip, afin que les timestamps restent alignés sur la vidéo originale.
19. En tant qu’exploitant, je veux que la détection et la suppression des silences n’altèrent pas les timestamps restitués, afin que les segments ouvrent au bon instant.
20. En tant qu’administrateur, je veux que les paramètres de détection d’activité vocale soient versionnés et configurables, afin d’adapter la préparation aux clips sans masquer les paroles.
21. En tant qu’utilisateur, je veux qu’un clip sans parole obtienne un état terminé sans segments, afin qu’il ne soit pas relancé indéfiniment.
22. En tant qu’administrateur, je veux qu’une piste absente, invalide ou impossible à décoder produise une erreur compréhensible sur ce clip, afin de diagnostiquer l’exception sans interrompre le lot.
23. En tant qu’utilisateur, je veux que les paroles soient transcrites en français, y compris le langage courant des conversations de jeu, afin que la recherche corresponde à mes clips.
24. En tant qu’utilisateur, je veux que la transcription conserve le texte et les timestamps de début et de fin pour chaque mot, afin de retrouver précisément le passage prononcé.
25. En tant qu’utilisateur, je veux que les mots soient également regroupés en segments lisibles avec ponctuation et contexte, afin de consulter une phrase sans perdre le repérage mot à mot.
26. En tant qu’utilisateur, je veux que le segment indique la piste audio qui l’a produit, afin de distinguer les résultats issus de pistes différentes.
27. En tant qu’exploitant, je veux conserver la version du fournisseur, du modèle et des paramètres ayant produit une transcription, afin de comparer et invalider les résultats de façon ciblée.
28. En tant qu’administrateur, je veux configurer un vocabulaire de jeu avec termes, variantes et graphies préférées, afin d’améliorer la reconnaissance des noms, cartes, personnages et expressions courantes.
29. En tant qu’administrateur, je veux activer ou retirer un terme du vocabulaire sans modifier les transcriptions historiques, afin de pouvoir contrôler une nouvelle version du pipeline.
30. En tant qu’exploitant, je veux que le vocabulaire actif soit inclus dans la version d’entrée du travail, afin qu’un changement de vocabulaire planifie uniquement les transcriptions concernées.
31. En tant qu’utilisateur, je veux rechercher un mot ou une expression exacte dans les paroles, afin de retrouver rapidement un clip dont je connais la formulation.
32. En tant qu’utilisateur, je veux que la recherche plein texte tolère les accents et la casse, afin que « réanimé » et « reanime » puissent mener au même passage.
33. En tant qu’utilisateur, je veux obtenir des résultats approximatifs pour une faute ou une variante proche, afin de retrouver une parole même si la transcription est imparfaite.
34. En tant qu’utilisateur, je veux distinguer clairement les correspondances exactes, plein texte et approximatives, afin de juger la fiabilité du résultat.
35. En tant qu’utilisateur, je veux voir un extrait des paroles autour du terme trouvé, afin de vérifier le contexte avant d’ouvrir le clip.
36. En tant qu’utilisateur, je veux que le terme recherché soit mis en évidence dans l’extrait, afin de repérer sa position rapidement.
37. En tant qu’utilisateur, je veux ouvrir un résultat au timestamp du segment, afin que la vidéo démarre près de la parole recherchée.
38. En tant qu’utilisateur, je veux naviguer d’un résultat au clip complet et revenir à la liste, afin de continuer ma recherche sans perdre son contexte.
39. En tant qu’utilisateur, je veux filtrer les résultats par date, source, visibilité et état du clip, afin de réduire une recherche à la partie pertinente de la bibliothèque.
40. En tant qu’utilisateur, je veux que la recherche respecte les permissions partagées et privées existantes, afin qu’une parole d’un clip privé ne soit jamais divulguée.
41. En tant qu’utilisateur, je veux que les compteurs de résultats correspondent exactement aux clips et segments retournés, afin de pouvoir faire confiance à la pagination.
42. En tant qu’utilisateur, je veux que les résultats répétitifs provenant de plusieurs mots d’une même phrase soient regroupés de manière lisible, afin de ne pas voir la même occurrence plusieurs fois sans raison.
43. En tant qu’administrateur, je veux consulter les paramètres du pipeline actif et les versions de résultats, afin de savoir quelle configuration a produit l’index courant.
44. En tant qu’exploitant, je veux que la recherche textuelle continue de fonctionner si le PC worker est éteint, afin que les transcriptions déjà enregistrées restent accessibles.
45. En tant qu’exploitant, je veux que le worker n’accède qu’aux entrées des clips et tâches qui lui sont loués, afin qu’il ne puisse pas parcourir la bibliothèque entière.
46. En tant que contributeur, je veux que les opérations de transcription et de recherche soient décrites dans OpenAPI et reflétées dans les clients générés, afin que le serveur, la SPA et le worker utilisent le même contrat.
47. En tant que propriétaire du projet, je veux utiliser uniquement des clips synthétiques et des fixtures dédiées avant la phase 7, afin qu’aucune vraie bibliothèque ne soit traitée prématurément.
48. En tant que propriétaire du projet, je veux qu’aucun traitement nocturne de la vraie bibliothèque ne soit déclenché automatiquement, afin de garder le démarrage du lot sous contrôle explicite.

## Implementation Decisions

- Le monolithe Go reste propriétaire de PostgreSQL, des permissions, du contrat API, de la planification des tâches et de la recherche textuelle. Le worker Python réalise la préparation audio et appelle un fournisseur de transcription via l’interface de capacités de phase 2.
- Le worker communique uniquement avec Snitch via HTTPS sortant et les jetons worker existants. Il n’accède jamais directement à PostgreSQL et ne reçoit que les médias autorisés par un lease actif.
- Le pipeline comporte des étapes versionnées de sélection/préparation audio et de transcription. Les paramètres d’entrée incluent au minimum la piste retenue, la préparation, la langue, le fournisseur/modèle et la version du vocabulaire applicable.
- Les tâches sont planifiées seulement pour des clips prêts dont un original est lisible. Les clips indisponibles restent visibles dans la bibliothèque ; leur transcription n’est pas marquée réussie.
- L’inspection audio réutilise les métadonnées déjà extraites pour les clips et peut examiner la structure du média autorisé au worker. Aucun original n’est modifié, déplacé, renommé ou supprimé.
- L’administrateur peut sélectionner une piste par source à partir des métadonnées de pistes, et remplacer ce choix pour un clip particulier. Les règles peuvent utiliser les numéros de piste et les titres réellement présents ; aucun titre Medal n’est présumé dans cette spec.
- Si plusieurs pistes sont présentes et qu’aucune règle ne permet un choix non ambigu, le clip est signalé pour sélection explicite. Une piste inconnue n’est jamais abandonnée silencieusement. La sélection d’une piste de mixage complète reste possible si l’administrateur la choisit.
- La préparation produit un dérivé de travail WAV 16 kHz mono côté worker. Les dérivés temporaires ne deviennent pas des originaux de bibliothèque. La détection d’activité vocale peut éviter le calcul sur les silences mais doit conserver la correspondance avec la chronologie du média source.
- Le worker annonce les identifiants des étapes qu’il sait traiter. La capacité de transcription et son fournisseur sont distincts de `fixture.*`; un worker non compatible ne prend pas ces tâches.
- Les fournisseurs de transcription sont derrière une interface remplaçable et identifient précisément leur modèle/version. La spec n’impose pas le modèle final : les candidats et réglages restent configurables et leur choix définitif/qualité/débit sont tranchés au benchmark de phase 7.
- Aucun service externe n’est requis pour les tests. Les tests utilisent un fournisseur déterministe qui rend des segments synthétiques avec timestamps au niveau des mots.
- La persistance garde le texte, les temps de début/fin, la piste audio source, les segments lisibles, la provenance de modèle/configuration, les dates et l’état. Les contraintes et transactions rendent la répétition d’un résultat idempotente et empêchent qu’une ancienne version remplace la version courante.
- Une transcription valide sans parole est un résultat réussi vide. Les erreurs de décodage, de fournisseur ou de format restent visibles et suivent les retries bornés de la file existante.
- Le vocabulaire est administrable dans Snitch. Les termes actifs et leurs variantes font partie de la version de configuration utilisée pour chaque travail ; une modification ne déclenche pas d’invalidation de métadonnées ou d’autres étapes.
- La recherche utilise PostgreSQL : normalisation française des accents et de la casse, recherche plein texte, puis correspondance floue avec `pg_trgm`. Les embeddings et la recherche sémantique appartiennent à la phase 4.
- Les résultats indiquent leur type de correspondance et retournent des extraits, les mots/segments pertinents, leurs timestamps, et les clips autorisés. Les compteurs et la pagination sont calculés dans les requêtes SQL, pas estimés par la SPA.
- La lecture ouvre le clip au temps du segment à l’aide du lecteur vidéo existant et de l’accès média authentifié. Les permissions de bibliothèque privée sont appliquées à la liste de recherche, aux détails et au média.
- La SPA ajoute une recherche de paroles, les filtres de résultats, l’accès au passage et l’administration du vocabulaire, des règles de pistes et des paramètres de pipeline. La file existante expose les nouvelles étapes et leurs erreurs sans ajouter un orchestrateur séparé.
- Les migrations PostgreSQL sont versionnées et séparées du démarrage de l’API. OpenAPI demeure la source du contrat et les clients Go/TypeScript/Python sont générés et vérifiés.
- Les médias de test sont générés ou dédiés au dépôt. Aucun chemin de la bibliothèque réelle, scan NAS, benchmark représentatif, batch complet ou traitement nocturne réel n’est permis dans cette phase.
- README, documentation du worker et de l’exploitation, roadmap et bilan de phase sont mis à jour avant de demander la validation de fin de phase.

## Testing Decisions

Un bon test vérifie des résultats observables aux interfaces publiques et la cohérence persistée, pas des appels internes, une classe ou une structure de tables particulière. La couture principale est le parcours réel **API Go + PostgreSQL/pgvector Testcontainers + worker Python HTTPS** avec clip synthétique multipiste et fournisseur déterministe. Playwright vérifie les actions et résultats visibles dans la SPA avec la vraie API.

1. Avec un clip vidéo synthétique multipiste, vérifier l’inventaire des pistes, la sélection par règle de source, le remplacement par clip et le traitement des choix ambigus sans perdre une piste silencieusement.
2. Vérifier que le worker ne reçoit que l’entrée autorisée pour son lease, prépare un audio synthétique en WAV 16 kHz mono, préserve les offsets temporels après détection des silences et ne modifie pas l’original.
3. Parcourir le pipeline par HTTPS : acquisition compatible, heartbeat pendant le traitement, résultat de transcription enregistré, tâche terminée, retry après erreur et reprise après perte/expiration de lease.
4. Vérifier les segments avec plusieurs mots : texte, ponctuation, piste audio, début/fin de chaque mot, regroupement en segment lisible, absence de décalage après une zone silencieuse et clip sans parole terminé avec zéro segment.
5. Vérifier l’idempotence d’un résultat répété, le refus d’un ancien lease ou d’une ancienne version d’entrée, et l’invalidation ciblée après changement de piste, modèle, paramètres ou vocabulaire.
6. Vérifier les erreurs d’audio absent, illisible, interrompu ou sans piste admissible : statut et erreur compréhensibles, retries bornés, et traitement des autres clips poursuivi.
7. Vérifier la personnalisation du vocabulaire sur les variantes et graphies françaises, l’activation/retrait d’un terme et la nouvelle version des tâches concernées.
8. Par l’API réelle avec PostgreSQL, vérifier recherche exacte, accents/casse, plein texte et approximation floue, étiquette du type de résultat, extraits, pagination et compteurs SQL exacts.
9. Vérifier les filtres de recherche par date, source, visibilité et état, le regroupement des correspondances d’une même occurrence et l’ouverture au timestamp correspondant.
10. Vérifier les permissions avec membre, propriétaire, autre membre, admin et session absente : ni résultats ni extraits d’un clip privé ne sont exposés par la recherche ou ses détails.
11. Parcours Playwright : administrateur configure une règle de piste et un terme de vocabulaire, planifie un traitement de clips synthétiques, suit la file, puis un membre recherche une expression et ouvre le clip à l’occurrence.
12. Vérifier que la recherche enregistrée fonctionne PC worker éteint et sans fournisseur de transcription disponible.
13. Vérifier OpenAPI, clients Go/TypeScript/Python générés, migrations vierges/rejouées, `go test -race`, `go vet`, build web/app/worker et `pip check`.
14. N’utiliser aucun fournisseur externe ni média réel ; ne lancer aucun batch réel, scan NAS, téléchargement de modèle définitif ou traitement nocturne. Le choix final de modèle et le benchmark représentatif restent distincts et soumis à la phase 7.

La phase 2 fournit les coutures à prolonger : serveur HTTP Go et PostgreSQL/pgvector Testcontainers, acquisition et résultat de job, client Python HTTPS avec certificat de test, fournisseur factice, contrats OpenAPI générés et parcours Playwright avec la vraie API.

## Out of Scope

- Embeddings, recherche vectorielle/hybride, recherche sémantique et indexation par vecteurs : phase 4.
- Diarisation, séparation et regroupement des Locuteurs locaux, identité de Personne, empreintes vocales, Échantillons vocaux validés et Attribution : phase 5.
- Correction manuelle de mots, timestamps ou Attributions et historique d’édition : phase 6.
- Résumés, chatbot, keyframes, analyse vidéo, événements de jeu ou recherche par image.
- Séparation de sources audio ou isolation automatique de la voix à partir d’un mix ; la phase sélectionne des pistes existantes et prépare l’audio.
- Choix final de modèle, promesse de qualité/débit, benchmark soutenu PC/NAS, échantillon représentatif de 200–500 clips, lot réel de 1 To et go/no-go : phase 7 et validations distinctes.
- Scan ou traitement de la vraie bibliothèque, transcriptions automatiques nocturnes, déploiement worker sur PC/NAS et publication d’images.

## Further Notes

- Les titres réels des pistes Medal ne sont pas connus dans les décisions actuelles. L’interface doit permettre de les inspecter sur un clip d’exemple et de configurer les règles sans codage en dur ; demander les titres disponibles à l’utilisateur au moment de leur intégration réelle si nécessaire.
- La préparation annoncée en phase 3 est un format de calcul pour le worker, pas un transcodage ou remplacement des vidéos originales.
- La présence de tâches en file n’implique pas que le PC worker est connecté, que le modèle définitif est choisi ou que la bibliothèque entière peut être traitée.
- Cette spec décrit et autorise la préparation de phase 3 après validation de la phase 2. Le traitement de la vraie bibliothèque et le batch complet restent interdits avant le benchmark de phase 7 et son autorisation distincte.
