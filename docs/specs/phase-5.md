# Snitch — Phase 5 : voix, personnes et attribution progressive

Issue GitHub : https://github.com/thomas-bsn/snitch/issues/31 (`ready-for-agent`).

## Problem Statement

Snitch retrouve ce qui a été dit, mais pas qui l'a dit. Les transcriptions ne distinguent pas les voix d'un clip, et rien ne relie la voix d'un clip à celle d'un autre. Pour savoir qui parle, il faut écouter chaque clip. La bibliothèque représente environ 1 To de clips courts : confirmer chaque occurrence de chaque personne dans chaque clip est impraticable.

## Solution

Distinguer les voix au sein de chaque clip, les relier à des Personnes et propager ces identités sur le reste de la bibliothèque, sans retraiter les vidéos.

Le worker produit, pour chaque clip, des Locuteurs locaux et des empreintes vocales. La piste micro dédiée est attribuée directement à son propriétaire, sans diarisation. Une voix inconnue ne bloque jamais le pipeline. Les utilisateurs connectés écoutent des extraits, créent des Personnes, valident des Échantillons vocaux validés et corrigent. Les validations humaines constituent les références de chaque Personne ; le rapprochement s'exécute ensuite en base, sans le PC et sans la vidéo.

Le lot initial sert à constituer et à évaluer les références, pas à tout confirmer. Snitch mesure la précision sur des clips distincts de ceux qui ont servi aux références. Lorsque les critères configurables sont atteints, il propose l'activation de l'attribution automatique ; seul l'administrateur l'active. Les utilisateurs vérifient alors surtout les cas inconnus ou ambigus. Les attributions automatiques restent distinctes des validations humaines et ne deviennent jamais d'elles-mêmes des échantillons validés (voir ADR 0008).

Une Personne ayant un compte voit les clips où elle est identifiée et peut supprimer son profil vocal ; ses attributions redeviennent inconnues. Snitch ne promet pas une identification parfaite.

Cette phase utilise uniquement des données synthétiques, des empreintes déterministes et un fournisseur de test. Aucun média réel, scan de la bibliothèque, téléchargement implicite de modèle ni benchmark n'est autorisé.

## User Stories

### Distinguer les voix d'un clip

1. En tant qu'utilisateur, je veux que les voix d'un clip soient séparées en Locuteurs locaux, afin de savoir combien de voix distinctes y figurent.
2. En tant qu'utilisateur, je veux que chaque segment transcrit indique son Locuteur local, afin de lire qui parle dans le passage.
3. En tant qu'utilisateur, je veux que la piste micro dédiée soit attribuée à son propriétaire sans diarisation, afin d'obtenir une identité fiable sans calcul incertain.
4. En tant qu'administrateur, je veux désigner la piste micro et son propriétaire par source à partir des pistes réellement détectées, afin de ne pas deviner à partir d'un nom de piste.
5. En tant qu'administrateur, je veux remplacer ce choix pour un clip particulier, afin de corriger une exception.
6. En tant qu'utilisateur, je veux qu'une voix inconnue soit conservée comme Locuteur local non identifié, afin que le clip reste exploitable.
7. En tant qu'exploitant, je veux qu'une voix inconnue ne bloque jamais l'achèvement d'un clip, afin que la file ne s'arrête pas sur une identité manquante.
8. En tant qu'exploitant, je veux que la diarisation et les empreintes soient des étapes versionnées de la file existante, afin qu'un changement de modèle n'invalide que cette étape.
9. En tant qu'exploitant, je veux que la reprise d'une étape vocale remplace ses résultats sans doublon, afin qu'un crash ou un retry ne multiplie pas les locuteurs.
10. En tant qu'exploitant, je veux que le clip reste utilisable et consultable lorsque le worker est éteint, afin que le NAS serve la recherche et la validation sans le PC.
11. En tant qu'exploitant, je veux que le modèle de diarisation soit configurable et que ses conditions d'accès soient vérifiées explicitement, afin qu'aucun téléchargement implicite n'ait lieu.
12. En tant qu'administrateur, je veux voir l'état de l'étape vocale d'un clip et ses erreurs expurgées, afin de diagnostiquer un échec.

### Empreintes et rapprochement

13. En tant qu'exploitant, je veux une empreinte vocale par Locuteur local, et par tour de parole assez long, avec sa durée et sa qualité, afin d'écarter les extraits trop courts ou trop bruités.
14. En tant qu'exploitant, je veux que les empreintes soient stockées avec leur modèle et leur version, afin qu'aucune comparaison ne mélange des empreintes incompatibles.
15. En tant qu'exploitant, je veux que le rapprochement vocal s'exécute en base, afin que la propagation n'exige ni PC ni retraitement vidéo.
16. En tant qu'exploitant, je veux qu'un changement de références ou de seuils réexamine les attributions à partir des empreintes déjà extraites, afin de ne jamais relire les vidéos.
17. En tant qu'utilisateur, je veux que chaque attribution porte un niveau de confiance et un niveau de décision, afin de distinguer automatique, suggestion, ambigu et inconnu.
18. En tant qu'administrateur, je veux régler les seuils entre ces niveaux, afin de les ajuster après mesure sans changer le code.
19. En tant qu'exploitant, je veux que deux voix proches ne soient jamais fusionnées sans validation humaine, afin d'éviter de confondre deux personnes.
20. En tant qu'utilisateur, je veux qu'un Locuteur local proche de plusieurs Personnes soit marqué ambigu avec ses candidats, afin de trancher moi-même.
21. En tant qu'exploitant, je veux que le rapprochement ignore les empreintes de qualité insuffisante, afin de ne pas propager une attribution douteuse.

### Personnes et échantillons

22. En tant qu'utilisateur connecté, je veux créer une Personne, afin de lui attribuer des voix.
23. En tant qu'utilisateur, je veux lier une Personne à un compte, afin qu'elle retrouve ses clips et gère son profil vocal.
24. En tant qu'utilisateur, je veux écouter un extrait représentatif d'un Locuteur local, afin de reconnaître la voix avant de valider.
25. En tant qu'utilisateur, je veux exclure un extrait inadapté d'un groupe, afin qu'il ne pollue pas les références.
26. En tant qu'utilisateur, je veux valider qu'un Locuteur local est une Personne donnée, afin de créer un Échantillon vocal validé.
27. En tant qu'utilisateur, je veux valider même lorsque la confiance automatique est haute, afin de constituer des références humaines.
28. En tant qu'utilisateur, je veux qu'une validation humaine enrichisse les références de la Personne, afin d'améliorer le rapprochement sans réentraînement.
29. En tant qu'utilisateur, je veux voir combien d'échantillons validés une Personne possède, afin de juger la solidité de ses références.
30. En tant qu'utilisateur, je veux fusionner deux Personnes créées par erreur, afin de rattacher leurs attributions à une seule identité.
31. En tant qu'utilisateur, je veux séparer ou supprimer une Personne créée par erreur, afin que ses attributions redeviennent inconnues.

### Groupes d'inconnues

32. En tant qu'utilisateur, je veux voir des groupes de voix inconnues proches, afin de les nommer en une seule opération.
33. En tant qu'utilisateur, je veux qu'une voix inconnue reste seule tant qu'aucun rapprochement ne la lie à un groupe, afin de ne pas former de faux groupes.
34. En tant qu'utilisateur, je veux écouter plusieurs extraits d'un groupe et en exclure certains, afin de nettoyer un groupe avant de le nommer.
35. En tant qu'utilisateur, je veux nommer un groupe entier comme une Personne, afin de valider plusieurs Locuteurs locaux d'un coup.
36. En tant qu'utilisateur, je veux que l'exclusion d'un membre ne modifie pas les autres, afin de corriger un seul extrait.
37. En tant qu'utilisateur, je veux que les nombres de membres et de clips d'un groupe proviennent de la base, afin de ne jamais voir un total inventé.
38. En tant qu'utilisateur, je veux trier les groupes par nombre de clips, afin de nommer d'abord les voix les plus fréquentes.

### Validation partagée et historique

39. En tant qu'utilisateur connecté, je veux pouvoir valider une attribution, afin que tous contribuent à constituer les références.
40. En tant qu'utilisateur, je veux que la première validation d'un Locuteur local fasse foi, afin d'éviter des validations contradictoires.
41. En tant qu'utilisateur, je veux que deux validations simultanées d'un même Locuteur local aboutissent à un seul gagnant clair, afin qu'aucune attribution ne soit écrasée par accident.
42. En tant qu'utilisateur, je veux voir qui a validé une attribution et quand, afin de juger sa fiabilité.
43. En tant qu'utilisateur, je veux corriger explicitement une attribution déjà validée, afin de réparer une erreur.
44. En tant qu'utilisateur, je veux que la correction conserve l'ancienne valeur, la nouvelle, l'auteur et la date, afin de tracer chaque changement.
45. En tant qu'utilisateur, je veux consulter l'historique d'une attribution, afin de comprendre comment elle est devenue ce qu'elle est.
46. En tant qu'utilisateur, je veux qu'une correction retire l'ancien échantillon des références et en crée un nouveau, afin qu'une erreur ne continue pas à propager de fausses attributions.
47. En tant qu'utilisateur, je veux que la correction d'une Personne mette à jour les clips qui en dépendaient par rapprochement, afin de ne pas laisser d'erreurs en aval.
48. En tant qu'utilisateur, je veux distinguer visuellement validation humaine, attribution automatique, suggestion, ambigu et inconnu, afin de savoir ce que je dois vérifier.

### Évaluation et automatisation

49. En tant qu'administrateur, je veux consulter la précision globale et par Personne mesurée sur des clips distincts de ceux des références, afin de juger la qualité réelle.
50. En tant qu'administrateur, je veux voir le nombre de confirmations, la couverture automatique et le volume de vérification restant, afin de décider de l'automatisation.
51. En tant qu'administrateur, je veux configurer le seuil de précision et le nombre minimal de confirmations, afin d'adapter l'exigence après benchmark.
52. En tant qu'administrateur, je veux que Snitch propose l'activation lorsque les critères sont atteints, afin de ne pas surveiller les chiffres moi-même.
53. En tant qu'administrateur, je veux être le seul à activer ou désactiver l'attribution automatique, afin de garder le contrôle de la propagation.
54. En tant qu'utilisateur non administrateur, je veux voir si l'automatisation est active sans pouvoir la modifier, afin de comprendre l'origine des attributions.
55. En tant qu'administrateur, je veux désactiver l'attribution automatique et revenir aux suggestions, afin de réagir à une dégradation.
56. En tant qu'utilisateur, je veux qu'une attribution automatique ne devienne pas un échantillon validé sans mon action, afin d'éviter l'amplification de ses propres erreurs.
57. En tant qu'utilisateur, je veux une vue de vérification centrée sur les cas inconnus ou ambigus, afin de passer mon temps là où mon avis compte.
58. En tant qu'utilisateur, je veux que cette vue priorise les cas les plus fréquents ou les plus utiles, afin de réduire vite l'incertitude.
59. En tant que propriétaire du projet, je veux qu'aucun nombre d'extraits ne soit présenté comme garantie de qualité, afin de ne pas promettre une identification parfaite.

### Recherche, filtres et consultation

60. En tant qu'utilisateur, je veux filtrer les résultats de recherche par Personne, afin de retrouver ce qu'une personne a dit.
61. En tant qu'utilisateur, je veux voir la Personne ou l'état inconnu dans chaque résultat, afin de lire qui a parlé sans ouvrir le clip.
62. En tant qu'utilisateur, je veux que le filtre par Personne s'applique aux modes exact, plein texte, approximatif et hybride, afin d'avoir un comportement cohérent.
63. En tant qu'utilisateur, je veux que le filtre applique les permissions de visibilité avant de paginer et de compter, afin de ne révéler aucun extrait privé.
64. En tant qu'utilisateur, je veux voir les Personnes identifiées d'un clip dans la bibliothèque et sur sa page, afin de reconnaître son contenu.
65. En tant qu'utilisateur, je veux filtrer la bibliothèque par Personne, afin de retrouver les clips où elle figure.
66. En tant qu'utilisateur, je veux ouvrir un résultat au timestamp du segment et voir son locuteur, afin de vérifier immédiatement.

### Confidentialité

67. En tant que Personne ayant un compte, je veux voir les clips où je suis identifiée, afin de savoir où ma voix apparaît.
68. En tant que Personne ayant un compte, je veux supprimer mon profil vocal, mes empreintes et mes échantillons, afin de retirer mes données biométriques.
69. En tant que Personne ayant supprimé son profil, je veux que mes attributions redeviennent inconnues, afin que le rapprochement ne me retrouve plus.
70. En tant que Personne ayant supprimé son profil, je veux que le rapprochement ne reconstitue pas mon profil à partir de données restantes, afin que la suppression soit effective.
71. En tant qu'administrateur, je veux que cette suppression reste tracée sans conserver de donnée vocale, afin d'auditer l'opération sans conserver la biométrie.
72. En tant qu'utilisateur, je veux que les extraits ne soient écoutables que par des utilisateurs connectés, afin qu'aucun accès anonyme n'existe.
73. En tant qu'utilisateur, je veux que les extraits d'un clip privé ne soient accessibles qu'à ceux autorisés à voir ce clip, afin de respecter la visibilité.
74. En tant qu'administrateur, je veux que la suppression d'un clip efface ses empreintes et ses attributions dérivées tout en conservant l'original, afin de respecter l'invariant de la bibliothèque.
75. En tant qu'administrateur, je veux que la suppression d'une Personne efface ses références sans supprimer les clips, afin de garder la bibliothèque intacte.

### Exploitation, contrat et sécurité

76. En tant qu'exploitant, je veux que le worker envoie diarisation et empreintes via l'API HTTPS avec son jeton limité, afin qu'il n'accède jamais à PostgreSQL.
77. En tant qu'exploitant, je veux que le résultat d'un clip soit enregistré en une seule transaction, afin qu'un crash ne laisse aucune voix partielle.
78. En tant qu'exploitant, je veux qu'un résultat périmé ou provenant d'un lease expiré soit refusé, afin qu'un ancien traitement n'écrase pas un plus récent.
79. En tant qu'exploitant, je veux que le traitement vocal ne relance jamais la transcription ni ne retraite la vidéo, afin de limiter les coûts.
80. En tant qu'administrateur, je veux planifier explicitement les clips à traiter vocalement, afin qu'aucun scan ni traitement nocturne ne démarre sans action.
81. En tant que contributeur, je veux que le contrat OpenAPI et les clients générés décrivent Locuteurs locaux, Personnes, attributions, historique et seuils, afin que SPA, API et worker restent synchronisés.
82. En tant que propriétaire du projet, je veux que les tests utilisent empreintes et audio synthétiques déterministes, afin de n'utiliser ni média réel ni fournisseur externe.
83. En tant que propriétaire du projet, je veux que la phase n'exécute aucun batch réel, afin de réserver le traitement de la bibliothèque à l'autorisation de phase 7.
84. En tant que propriétaire du projet, je veux que le filtrage par Personne ne reconnaisse pas de voix en dehors de ce qui a été validé ou rapproché avec un niveau tracé, afin de garder l'origine de chaque attribution explicable.

## Implementation Decisions

- Le monolithe Go reste propriétaire de PostgreSQL, des permissions, de l'API, du rapprochement vocal, de la validation et de l'historique. Le rapprochement et la propagation s'exécutent sur le NAS, sans worker ni vidéo. Le worker ne fait que calculer diarisation et empreintes.
- Le modèle conceptuel suit le glossaire sans collision : Locuteur local (voix d'un clip), Personne (identité), Échantillon vocal validé (extrait confirmé par un utilisateur), Attribution (lien voix–Personne avec niveau de confiance, niveau de décision et état de validation). Une Personne n'est jamais un compte ; un lien optionnel relie une Personne à un compte.
- La diarisation et l'extraction d'empreintes sont de nouvelles étapes de la file existante, avec version modèle/configuration par étape. Elles s'exécutent sur l'audio préparé en phase 3. Le changement d'un modèle invalide uniquement ces étapes ; aucune retranscription ni retraitement vidéo.
- La piste micro dédiée est désignée par source via les métadonnées de pistes déjà inventoriées en phase 3, avec remplacement par clip. Elle reçoit directement l'Attribution de son propriétaire, sans diarisation. Aucun titre de piste Medal n'est supposé ; les titres réels restent à demander à l'utilisateur au moment de leur intégration.
- Le fournisseur de diarisation est remplaçable. Le modèle de référence prévu est pyannote speaker-diarization-community-1, soumis à un jeton et à des conditions d'accès vérifiés explicitement par l'administrateur. Aucun téléchargement n'est déclenché implicitement ; les tests utilisent un fournisseur factice.
- Les empreintes sont calculées par Locuteur local, et par tour de parole suffisamment long, et enregistrent durée, qualité, modèle et version. Elles sont stockées dans PostgreSQL avec pgvector. Des empreintes de modèles ou versions différents ne sont jamais comparées entre elles. Les extraits trop courts ou de qualité insuffisante ne servent ni de référence ni de base de propagation.
- Les résultats vocaux d'un clip sont enregistrés en une transaction, remplacent l'ancien résultat sans doublon et sont rejetés pour un lease expiré ou une version périmée, conformément aux garanties de la phase 2.
- Les références d'une Personne sont l'ensemble de ses Échantillons vocaux validés. Le rapprochement compare les empreintes des Locuteurs locaux aux références et produit un niveau de décision : automatique, suggestion, ambigu ou inconnu. Les seuils de séparation sont configurables ; leurs valeurs initiales ne sont pas des choix définitifs et dépendent du benchmark. Une voix proche de plusieurs Personnes est ambiguë avec la liste de ses candidates ; deux voix proches ne sont jamais fusionnées sans validation humaine.
- Les attributions automatiques sont stockées comme telles et ne deviennent jamais des échantillons validés sans action humaine ; elles ne contribuent donc pas aux références. Seules les validations et corrections humaines les enrichissent (ADR 0008). Aucun réentraînement du modèle n'est nécessaire ; l'ajout, le retrait ou la correction d'un échantillon déclenche un nouveau rapprochement des empreintes existantes.
- Les voix inconnues forment d'abord des groupes singleton. Le rapprochement entre inconnues propose des groupes sans jamais les confirmer. Les extraits sont écoutables, un extrait peut être exclu d'un groupe, et un groupe peut être nommé comme une Personne. Les comptes de membres et de clips d'un groupe sont toujours calculés en base.
- La première validation d'un Locuteur local fait foi, arbitrée par une contrainte PostgreSQL ; une validation concurrente perd et voit l'attribution existante avec son auteur. Une correction explicite d'un utilisateur connecté est possible ensuite. Toute validation ou correction enregistre ancienne valeur, nouvelle valeur, auteur et date dans un historique immuable.
- Tous les utilisateurs connectés peuvent valider et corriger. Les extraits ne sont servis qu'aux utilisateurs connectés et uniquement pour des clips que leur visibilité permet de voir.
- L'évaluation de précision utilise des clips distincts de ceux qui ont servi aux références : un échantillon validé sert soit de référence, soit de test, pas des deux pour la même mesure. Snitch expose la précision globale et par Personne, le nombre de confirmations, la couverture automatique et le volume de vérification restant. Les critères d'activation (seuil de précision, nombre minimal de confirmations) sont configurables ; les exemples de 95 % et 50 confirmations ne sont pas des valeurs garanties. Snitch propose l'activation, seul l'administrateur l'active ou la désactive.
- L'interface de vérification cible les cas inconnus et ambigus, priorisés par fréquence. Elle distingue visuellement validation humaine, attribution automatique, suggestion, ambigu et inconnu. Aucune promesse de qualité n'est affichée à partir du nombre d'extraits.
- Le filtre par Personne devient effectif dans les recherches exacte, plein texte, approximative et hybride, ainsi que dans la bibliothèque. Les permissions et les autres filtres s'appliquent avant la pagination et les comptes. Chaque résultat indique la Personne ou l'état inconnu. Cette phase lève la réserve laissée en phase 4.
- Une Personne ayant un compte voit les clips où elle est identifiée et peut supprimer son profil vocal, ses empreintes et ses échantillons. Les attributions correspondantes redeviennent inconnues et le rapprochement ne peut pas reconstituer ce profil. La suppression est tracée sans conserver de donnée vocale. La suppression d'un clip efface ses empreintes et attributions dérivées sans toucher à l'original.
- Aucune ingestion, aucun scan ou traitement nocturne n'est déclenché automatiquement. L'administrateur planifie explicitement les clips à traiter, comme en phase 3.
- Le contrat OpenAPI reste la source des interfaces ; les clients Go, TypeScript et Python sont régénérés et vérifiés en CI. Les migrations passent par la commande existante. Le worker utilise exclusivement l'API HTTPS avec son jeton limité.
- Une décision technique non triviale de cette phase (modèle conceptuel locuteur/personne/attribution, arbitrage de la première validation, règle d'évaluation sur clips distincts) est consignée dans un ADR. Le glossaire peut recevoir les termes métier résolus (par exemple niveau de décision, groupe de voix inconnues) sans détails d'implémentation.

## Testing Decisions

Un bon test observe le comportement aux interfaces publiques et la cohérence des résultats persistés ; il ne vérifie ni les appels privés ni la forme interne des tables. Deux coutures existantes sont utilisées :

1. Un parcours API Go avec PostgreSQL/pgvector réel en Testcontainers, un worker simulé qui parle l'API HTTPS du worker et des empreintes vocales synthétiques déterministes.
2. Un parcours Playwright avec la vraie API pour les écrans de validation, l'écoute d'extraits et la vue de vérification.

Le worker Python reste couvert par ses tests unitaires de contrat de résultat existants.

Comportements à vérifier :

- Une piste micro dédiée est attribuée à son propriétaire sans diarisation ; remplacement par clip.
- Un clip avec voix inconnues atteint l'état terminé sans bloquer la file.
- La reprise d'une étape vocale et le rejet d'un résultat périmé ne créent aucun doublon ni voix partielle.
- Validation concurrente d'un même Locuteur local : un seul gagnant, l'autre voit l'attribution et l'auteur.
- Correction explicite : historique ancienne/nouvelle valeur, auteur, date ; retrait de l'ancien échantillon des références.
- Propagation par rapprochement en base sans worker démarré et sans média ; niveaux automatique, suggestion, ambigu, inconnu selon les seuils.
- Deux voix proches ne sont pas fusionnées ; une voix proche de plusieurs Personnes est ambiguë avec ses candidates.
- Une attribution automatique ne devient pas un échantillon validé et n'altère pas les références.
- Groupes d'inconnues : singleton par défaut, exclusion d'un extrait, nommage d'un groupe, comptages issus de la base.
- Évaluation sur clips distincts, précision globale et par Personne, proposition d'activation, activation et désactivation réservées à l'administrateur.
- Filtre par Personne dans les quatre modes de recherche et dans la bibliothèque, avec permissions, comptes et pagination cohérents.
- Suppression du profil vocal : empreintes et échantillons effacés, attributions redevenues inconnues, aucune reconstitution, trace conservée sans donnée vocale ; suppression d'un clip.
- Extraits : refus anonyme, refus pour un clip privé non autorisé.
- Contrat OpenAPI, clients générés, migrations vierges/rejouées, tests Go avec race detector, go vet, build web/app/worker, pip check.

Prior art : les tests d'intégration de la phase 4 (indexation, reconstruction, fixtures déterministes), les tests de lease et de résultats de la phase 2, les tests de pistes et de recherche de la phase 3, et les suites Playwright existantes.

## Out of Scope

- Correction des transcriptions et de leur historique : phase 6. Seule l'attribution du locuteur est corrigeable ici.
- Benchmark de 200–500 clips, mesure réelle de précision, choix définitif de modèle et de seuils, go/no-go : phase 7.
- Traitement ou scan de la vraie bibliothèque, batch de 1 To, traitement nocturne, benchmark sur échantillon réel.
- Séparation vocale ou isolation de la voix à partir d'un mix ; Demucs.
- Réentraînement ou fine-tuning de modèles.
- Identification par le visage, analyse visuelle ou vidéo.
- Garantie de qualité d'identification, quel que soit le nombre d'extraits.
- Titres réels des pistes Medal : à demander à l'utilisateur au moment utile.

## Further Notes

- Phase 5 implémentée (voir le [bilan](../phase-5-report.md)). Les validations de la phase 4 ont été exécutées avant cette phase.
- Les seuils, le nombre de confirmations et les critères d'activation sont des paramètres configurables dont les valeurs viennent du benchmark. ADR 0008 reste la référence sur la relation entre validations humaines, références et attributions automatiques.
- Tickets d'implémentation : [#32](https://github.com/thomas-bsn/snitch/issues/32) diarisation et Locuteurs locaux, [#33](https://github.com/thomas-bsn/snitch/issues/33) Personnes et validation, [#34](https://github.com/thomas-bsn/snitch/issues/34) piste micro, [#35](https://github.com/thomas-bsn/snitch/issues/35) rapprochement en base, [#36](https://github.com/thomas-bsn/snitch/issues/36) groupes de voix inconnues, [#37](https://github.com/thomas-bsn/snitch/issues/37) vérification, [#38](https://github.com/thomas-bsn/snitch/issues/38) précision et automatisation, [#39](https://github.com/thomas-bsn/snitch/issues/39) filtre par Personne, [#40](https://github.com/thomas-bsn/snitch/issues/40) gestion des Personnes, [#41](https://github.com/thomas-bsn/snitch/issues/41) profil vocal, [#42](https://github.com/thomas-bsn/snitch/issues/42) clôture. Décision d'architecture : [ADR 0011](../adr/0011-locuteurs-personnes-et-attributions.md).
