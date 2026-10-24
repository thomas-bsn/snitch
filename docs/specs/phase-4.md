# Snitch — Phase 4 : recherche sémantique et index vectoriel

Issue GitHub : https://github.com/thomas-bsn/snitch/issues/26 (`ready-for-agent`).

## Problem Statement

La recherche actuelle retrouve des mots présents dans les transcriptions. Elle ne retrouve pas toujours un passage lorsque l’utilisateur décrit son sens avec d’autres mots. Les embeddings sont prévus dans l’installation, mais ils ne sont pas encore calculés ni utilisés pour la recherche.

## Solution

Ajouter la recherche sémantique des segments transcrits et la combiner avec la recherche textuelle existante. Snitch calcule et stocke les embeddings dans PostgreSQL/pgvector, puis fusionne le classement lexical et sémantique. L’utilisateur conserve les filtres, les extraits et l’ouverture du clip au bon timestamp.

Le même profil d’embeddings — fournisseur, modèle, dimensions et version — sert à indexer les segments et les requêtes. L’index actif reste consultable pendant qu’un nouvel index est construit ; Snitch ne bascule vers ce dernier que lorsqu’il est complet et cohérent. Si le service d’embeddings est indisponible, la recherche textuelle existante continue de fonctionner.

Les fournisseurs et modèles restent configurables. Aucun modèle final, qualité, débit ou coût ne sont promis avant le benchmark de phase 7. Cette phase utilise uniquement des transcriptions synthétiques et des fournisseurs de test. Aucun média réel, scan de la bibliothèque ou batch réel n’est autorisé.

## User Stories

1. En tant qu’utilisateur, je veux rechercher une idée avec des mots différents de ceux prononcés, afin de retrouver un passage par son sens.
2. En tant qu’utilisateur, je veux conserver la recherche exacte et plein texte, afin de retrouver une formulation connue précisément.
3. En tant qu’utilisateur, je veux utiliser une recherche hybride qui combine sens et correspondances textuelles, afin de profiter des deux types de correspondance.
4. En tant qu’utilisateur, je veux que les résultats les plus pertinents apparaissent en premier, afin de parcourir moins de clips.
5. En tant qu’utilisateur, je veux voir si un résultat est lexical, sémantique ou hybride, afin de comprendre pourquoi il apparaît.
6. En tant qu’utilisateur, je veux voir le texte du segment et le mot correspondant lorsqu’il y en a un, afin de vérifier le résultat avant d’ouvrir le clip.
7. En tant qu’utilisateur, je veux ouvrir un résultat au timestamp de son segment, afin de réécouter immédiatement le passage.
8. En tant qu’utilisateur, je veux paginer les résultats sans doublons et avec un total cohérent, afin de parcourir les occurrences trouvées.
9. En tant qu’utilisateur, je veux filtrer les résultats par date, source, visibilité et état du clip, afin de réduire la recherche à une partie de ma bibliothèque.
10. En tant qu’utilisateur, je veux que la recherche sémantique respecte les permissions des clips privés et partagés, afin qu’aucun extrait privé ne soit révélé.
11. En tant qu’utilisateur, je veux continuer à chercher par texte si le service d’embeddings est arrêté ou en erreur, afin que la recherche existante reste utile.
12. En tant qu’utilisateur, je veux être informé lorsque les résultats sémantiques sont temporairement indisponibles, afin de savoir que la liste repose alors sur la recherche textuelle.
13. En tant qu’administrateur, je veux configurer l’endpoint et le modèle d’embeddings compatibles avec l’API OpenAI, afin d’utiliser le service exécuté sur le NAS.
14. En tant qu’administrateur, je veux que Snitch vérifie la dimension des vecteurs retournés, afin de détecter une configuration incompatible avant de l’utiliser.
15. En tant qu’administrateur, je veux voir le profil d’embeddings actif, son état et la progression d’un index en préparation, afin de comprendre la disponibilité de la recherche sémantique.
16. En tant qu’administrateur, je veux lancer explicitement l’indexation des transcriptions déjà présentes, afin d’activer la recherche sémantique sur les contenus existants sans retraiter les vidéos.
17. En tant qu’exploitant, je veux que chaque segment transcrit soit indexé de façon idempotente, afin qu’une reprise après arrêt ne crée pas de doublons.
18. En tant qu’exploitant, je veux que les embeddings soient liés au texte, au clip, au segment et à la version de transcription qui les ont produits, afin qu’un changement de transcription n’expose pas un vecteur périmé.
19. En tant qu’exploitant, je veux qu’un changement de modèle ou de paramètres crée une nouvelle version de l’index, afin que les vecteurs de profils incompatibles ne soient jamais mélangés.
20. En tant qu’administrateur, je veux reconstruire l’index avec un autre profil pendant que l’ancien reste actif, afin que la recherche reste disponible pendant la préparation.
21. En tant qu’administrateur, je veux que Snitch bascule atomiquement vers le nouvel index seulement après sa construction réussie, afin qu’aucune requête ne voie un index partiel.
22. En tant qu’administrateur, je veux que l’échec de reconstruction conserve l’ancien index actif et indique l’erreur, afin qu’un problème de modèle ne rende pas la recherche indisponible.
23. En tant qu’exploitant, je veux que les embeddings de clips nouveaux ou dont la transcription change soient ajoutés ou remplacés de manière ciblée, afin d’éviter une reconstruction complète inutile.
24. En tant qu’exploitant, je veux que les requêtes utilisent exactement le profil correspondant à l’index actif, afin que les distances vectorielles aient un sens.
25. En tant qu’exploitant, je veux que les recherches hybrides fusionnent les classements lexical et sémantique avec RRF, afin de combiner leurs rangs sans comparer directement des scores de nature différente.
26. En tant qu’administrateur, je veux que la recherche textuelle reste disponible lorsqu’aucun profil d’embeddings n’est configuré, afin de pouvoir installer Snitch sans modèle chargé.
27. En tant qu’exploitant, je veux que l’arrêt du worker de transcription n’empêche pas la recherche dans les index déjà présents, afin que le NAS serve les recherches sans dépendre du PC gamer.
28. En tant que contributeur, je veux que le contrat HTTP et les clients générés décrivent le mode hybride, l’état dégradé et les métadonnées de classement, afin que la SPA et l’API restent synchronisées.
29. En tant que propriétaire du projet, je veux que les tests d’indexation et de recherche utilisent des vecteurs déterministes et des transcriptions synthétiques, afin de ne pas appeler un modèle externe ni utiliser de vidéos réelles.
30. En tant que propriétaire du projet, je veux que la phase n’exécute aucun batch réel ni scan nocturne, afin de réserver la validation de performance et le traitement de la bibliothèque à des étapes autorisées séparément.

## Implementation Decisions

- Le monolithe Go conserve la propriété de PostgreSQL, des permissions, de l’API, de l’indexation, de la recherche et de la bascule des versions. La recherche sémantique fonctionne sur le NAS et ne dépend pas de la présence du worker PC.
- Les embeddings sont demandés à un endpoint configurable compatible avec le contrat embeddings OpenAI. Le nom du modèle et ses dimensions font partie du profil versionné. Aucun modèle n’est choisi définitivement dans cette phase.
- L’index cible les segments de transcription pour conserver le contexte textuel et l’ouverture au timestamp. Chaque vecteur est associé à un clip, un segment, une empreinte du texte source, une version de transcription et un profil d’embeddings.
- Les vecteurs sont stockés dans PostgreSQL/pgvector avec une représentation halfvec et un index HNSW adapté au profil actif. La recherche vectorielle applique une distance cosinus.
- La construction de vecteurs est reprenable et idempotente. Elle ne relit ni ne transcode l’original vidéo. Une transcription déjà terminée peut être indexée sans réexécuter le modèle de transcription.
- L’index courant et le profil candidat sont distincts. Les nouveaux vecteurs sont préparés sous un profil isolé ; une bascule atomique change le profil actif après vérification de la complétude. Les requêtes utilisent le modèle du profil actif ; en cas d’échec de requête d’embedding, elles retombent sur le classement lexical.
- La recherche hybride fusionne les rangs lexical et sémantique par Reciprocal Rank Fusion (RRF). Les modes exact et plein texte existants restent disponibles. L’API indique le mode réellement servi et si une réponse est dégradée.
- Les filtres déjà disponibles — date, source, visibilité et état du clip — s’appliquent avant de paginer et de compter les résultats. Les permissions de visibilité sont appliquées dans les deux parcours de recherche. Le filtre par locuteur attend les données de voix de phase 5.
- La pagination, les totaux et le regroupement se font au niveau des résultats de segments visibles par l’utilisateur. Chaque résultat conserve ses informations de source et son timestamp.
- L’administration peut consulter la configuration active, lancer une reconstruction explicite, suivre sa progression et voir une erreur expurgée. Aucun changement de configuration ne lance implicitement un scan de bibliothèque ou un traitement nocturne.
- Le contrat OpenAPI reste la source des interfaces HTTP ; les clients Go, TypeScript et Python sont régénérés et vérifiés en CI.
- Les migrations sont exécutées par la commande de migration existante. L’application reste démarrable sans endpoint d’embeddings et la recherche lexicale reste utilisable.
- Aucun modèle, endpoint public, média réel, benchmark représentatif, traitement massif de clips ou publication d’image n’est déclenché par cette phase.

## Testing Decisions

Un bon test observe le comportement aux interfaces publiques et la cohérence des résultats persistés ; il ne vérifie pas les appels privés ni la forme interne des tables. La couture principale est un parcours API Go avec PostgreSQL/pgvector réel en Testcontainers et un serveur HTTP d’embeddings déterministe. Un parcours navigateur vérifie les contrôles visibles dans la SPA avec la vraie API.

1. Vérifier l’indexation d’une transcription synthétique, le rattachement du vecteur au bon clip/segment/timestamp et l’absence de doublons après reprise.
2. Vérifier l’idempotence lorsque la même transcription est indexée plusieurs fois et le remplacement ciblé après modification de son texte ou de sa version.
3. Vérifier qu’une réponse d’embeddings de mauvaise dimension ou mal formée ne devient pas un vecteur actif et produit un état/une erreur compréhensible.
4. Vérifier qu’une recherche sémantique envoie le texte de requête au profil actif et retrouve un segment pertinent dont les mots diffèrent de la requête.
5. Vérifier que les modes exact et plein texte gardent leur comportement existant et que le mode hybride combine leurs classements avec le classement sémantique.
6. Vérifier les filtres date, source, visibilité et état, les comptes, la pagination et les timestamps sur les résultats combinés.
7. Vérifier l’isolation des clips privés pour chaque mode, y compris les extraits et compteurs.
8. Arrêter le serveur d’embeddings et vérifier que la recherche textuelle réussit avec un indicateur d’état dégradé ; vérifier également le cas où aucun endpoint n’est configuré.
9. Construire un profil candidat distinct, confirmer que l’ancien continue de servir les recherches, puis vérifier la bascule atomique après construction complète.
10. Faire échouer une reconstruction ou une bascule et vérifier que l’index antérieur reste actif et que le candidat incomplet n’apparaît pas dans les résultats.
11. Vérifier qu’une requête utilise le profil du vecteur actif et qu’aucun mélange de profils ou de dimensions ne produit de résultat.
12. Vérifier par Playwright la recherche hybride, l’indication de mode dégradé, les filtres et l’ouverture du résultat au timestamp.
13. Vérifier le contrat OpenAPI, les clients générés, les migrations vierges/rejouées, les tests Go avec race detector, go vet, build web/app/worker et pip check.
14. Utiliser seulement des embeddings factices reproductibles et des données synthétiques ; aucun appel à un fournisseur externe, média réel, scan du NAS ou batch réel.

Les tests prolongent les coutures existantes : intégration API Go + PostgreSQL/pgvector Testcontainers, contrat OpenAPI, fixtures d’endpoint HTTP local, et Playwright avec la vraie API. La recherche textuelle actuelle et sa suite de navigateur sont le comportement de référence à préserver.

## Out of Scope

- Diarisation, Locuteurs locaux, identité de Personne, empreintes vocales, échantillons vocaux validés et attribution automatique : phase 5.
- Filtrage effectif par locuteur avant la création de ces données en phase 5.
- Correction des transcriptions et historique des modifications : phase 6.
- Sélection définitive du modèle, mesure de qualité, latence ou empreinte NAS, benchmark de 200–500 clips et décision go/no-go : phase 7.
- Scan ou traitement de la bibliothèque réelle, indexation d’un lot de 1 To, traitement nocturne ou téléchargement implicite de modèles.
- Recherche dans l’image ou la vidéo, résumés, chatbot, génération ou réentraînement de modèles.
- Remplacement de PostgreSQL/pgvector par une base vectorielle séparée.

## Further Notes

- La phase 0 a préparé PostgreSQL/pgvector et une dépendance embeddings facultative. La phase 3 fournit les segments textuels et timestamps qui servent de contenu indexable.
- Le service d’embeddings tourne côté NAS ; le PC worker peut être éteint après la transcription. La qualité réelle et le coût de reconstruction seront évalués pendant le benchmark de phase 7.
- La spec suppose que la recherche hybride est le parcours principal lorsque le profil actif est disponible ; le lexical reste un mode explicite et un repli fiable.
- La phase 4 n’autorise pas à traiter le vrai lot de vidéos. La validation du benchmark réel et du batch complet reste distincte.
