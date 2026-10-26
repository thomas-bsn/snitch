# Snitch — Phase 6 : corrections des transcriptions et des locuteurs de segments

Issue GitHub : https://github.com/thomas-bsn/snitch/issues/48 (`ready-for-agent`).

## Problem Statement

Snitch retrouve ce qui a été dit, mais une transcription automatique se trompe : un nom propre mal orthographié, un terme de jeu déformé, une phrase manquée. Aujourd'hui rien ne permet de corriger ce texte. L'erreur reste dans la recherche, fausse les résultats et ne peut être réparée qu'en retranscrivant tout le clip, ce qui efface aussi ce qui était juste. De même, le locuteur affiché sur un segment est déduit automatiquement du recouvrement des voix et ne peut pas être redressé quand il est faux. Enfin, l'évaluation de la qualité en phase 7 a besoin de comparer la transcription automatique à un texte vérifié par un humain : sans corrections conservées à côté de l'original, cette mesure est impossible.

## Solution

Un utilisateur connecté corrige le texte d'un segment et le locuteur qui lui est attribué, depuis la page du clip. La transcription automatique d'origine n'est jamais modifiée : la correction la recouvre, et la recherche comme l'affichage utilisent le texte corrigé. Chaque correction conserve l'ancienne valeur, la nouvelle, l'auteur et la date dans un historique immuable consultable ; on peut revenir à l'original ou à une version antérieure, ce qui est lui-même une correction historisée. Seul le segment concerné est réindexé pour la recherche sémantique, sans retranscrire ni retraiter la vidéo.

Les corrections ne sont jamais perdues sans avertissement : retranscrire un clip qui en porte exige une confirmation explicite. Snitch ne promet pas que le texte soit exact : il rend ce qui a été vérifié par un humain visible et traçable.

Cette phase utilise uniquement des données synthétiques, un fournisseur d'embeddings de test et des fixtures dédiées. Aucun média réel, scan de la bibliothèque ou benchmark n'est autorisé.

## User Stories

### Corriger le texte d'un segment

1. En tant qu'utilisateur connecté, je veux modifier le texte d'un segment transcrit depuis la page du clip, afin de corriger une erreur de transcription.
2. En tant qu'utilisateur, je veux voir le texte d'origine et le texte corrigé d'un segment, afin de savoir ce que Snitch avait compris et ce que j'ai changé.
3. En tant qu'utilisateur, je veux que le texte corrigé remplace l'original dans l'affichage du clip, afin de lire directement la bonne version.
4. En tant qu'utilisateur, je veux qu'un segment corrigé soit signalé visuellement avec son auteur et sa date, afin de distinguer le texte vérifié du texte automatique.
5. En tant qu'utilisateur, je veux annuler ma saisie avant de l'enregistrer, afin de ne rien modifier par erreur.
6. En tant qu'utilisateur, je veux que le texte soit refusé s'il est vide ou démesuré, afin de ne pas supprimer un segment par inadvertance.
7. En tant qu'utilisateur, je veux que les débuts et fins de segment restent inchangés après une correction, afin que l'ouverture au timestamp du résultat reste fiable.
8. En tant qu'utilisateur, je veux pouvoir corriger plusieurs segments d'un même clip successivement, afin de relire un clip entier.
9. En tant qu'utilisateur, je veux que seuls les clips que je peux voir soient corrigeables, afin de respecter la visibilité.
10. En tant qu'utilisateur anonyme, je ne peux rien corriger, afin qu'aucune modification n'ait lieu sans compte.

### Historique et retour en arrière

11. En tant qu'utilisateur, je veux consulter l'historique d'un segment, afin de comprendre comment son texte est devenu ce qu'il est.
12. En tant qu'utilisateur, je veux que chaque entrée de l'historique indique l'ancienne valeur, la nouvelle, l'auteur et la date, afin de tracer chaque changement.
13. En tant qu'utilisateur, je veux que l'historique soit immuable, afin qu'aucune modification passée ne puisse être réécrite ou effacée.
14. En tant qu'utilisateur, je veux restaurer le texte d'origine d'un segment, afin de défaire une correction erronée.
15. En tant qu'utilisateur, je veux restaurer une version antérieure précise, afin de reprendre un texte déjà corrigé.
16. En tant qu'utilisateur, je veux qu'une restauration soit enregistrée comme une nouvelle entrée d'historique, afin de ne jamais réécrire le passé.
17. En tant qu'utilisateur, je veux pouvoir indiquer un motif facultatif avec une correction, afin d'expliquer mon changement.
18. En tant qu'utilisateur, je veux que le texte d'origine reste toujours consultable, même après plusieurs corrections.

### Édition simultanée

19. En tant qu'utilisateur, je veux qu'une correction soit refusée si quelqu'un a modifié le segment depuis que je l'ai ouvert, afin de ne pas écraser son travail.
20. En tant qu'utilisateur, je veux voir alors la version courante et son auteur, afin de décider de reprendre ou d'abandonner ma modification.
21. En tant qu'utilisateur, je veux que deux corrections simultanées du même segment n'en laissent qu'une gagnante clairement identifiée, afin qu'aucune ne disparaisse sans trace.
22. En tant qu'utilisateur, je veux que corriger deux segments différents en même temps ne crée aucun conflit, afin de travailler à plusieurs sur un même clip.

### Corriger le locuteur d'un segment

23. En tant qu'utilisateur, je veux indiquer quel Locuteur local parle sur un segment, afin de corriger un locuteur mal déduit.
24. En tant qu'utilisateur, je veux ne pouvoir choisir que parmi les Locuteurs locaux du clip, afin de ne pas inventer une voix.
25. En tant qu'utilisateur, je veux revenir au locuteur déduit automatiquement, afin de défaire une correction.
26. En tant qu'utilisateur, je veux que la correction du locuteur conserve l'ancienne valeur, la nouvelle, l'auteur et la date, afin de tracer le changement.
27. En tant qu'utilisateur, je veux que la correction du locuteur d'un segment ne modifie ni les tours de parole, ni les empreintes, ni l'attribution du Locuteur local à une Personne, afin de ne corriger que ce segment.
28. En tant qu'utilisateur, je veux que le filtre par Personne et l'affichage des résultats utilisent le locuteur corrigé, afin d'avoir un comportement cohérent.
29. En tant qu'utilisateur, je veux que corriger le locuteur d'un segment ne crée pas d'Échantillon vocal validé, afin de ne pas modifier les références sans action explicite.
30. En tant qu'utilisateur, je veux que la correction du locuteur d'un segment s'applique sans retraiter la vidéo ni la diarisation, afin d'obtenir un résultat immédiat.

### Recherche et réindexation

31. En tant qu'utilisateur, je veux que la recherche exacte trouve le texte corrigé, afin de retrouver ce qui a été vérifié.
32. En tant qu'utilisateur, je veux que la recherche plein texte et la recherche approximative utilisent aussi le texte corrigé, afin d'avoir un comportement cohérent.
33. En tant qu'utilisateur, je veux que la recherche hybride et sémantique reflètent le texte corrigé, afin que le sens retrouvé corresponde à ce qui a été vérifié.
34. En tant qu'utilisateur, je veux que le texte d'origine erroné ne soit plus retrouvé par la recherche une fois corrigé, afin de ne pas voir de faux résultats.
35. En tant qu'utilisateur, je veux que les résultats indiquent qu'un segment est corrigé, afin de distinguer le texte vérifié.
36. En tant qu'exploitant, je veux que seul le segment corrigé soit réindexé pour la recherche sémantique, afin de ne pas recalculer les autres segments du clip.
37. En tant qu'exploitant, je veux que la correction n'exige ni retranscription, ni retraitement de la vidéo, ni le PC, afin que le NAS suffise.
38. En tant qu'exploitant, je veux que la réindexation concerne l'index actif et, pendant une reconstruction, l'index candidat, afin que la bascule ne ramène pas l'ancien texte.
39. En tant qu'utilisateur, je veux que la recherche lexicale reflète la correction immédiatement, même si le service d'embeddings est arrêté, afin de ne pas dépendre de lui.
40. En tant qu'exploitant, je veux qu'un segment dont la réindexation sémantique n'a pas pu être faite soit repris sans doublon, afin qu'une panne temporaire ne laisse pas un vecteur périmé.
41. En tant qu'administrateur, je veux voir le nombre de segments en attente de réindexation, afin de diagnostiquer un retard.
42. En tant qu'utilisateur, je veux ouvrir un résultat corrigé au timestamp du segment, afin de vérifier immédiatement.

### Retranscrire un clip corrigé

43. En tant qu'administrateur, je veux être averti qu'une retranscription remplacerait des corrections humaines, afin de ne pas les perdre par inadvertance.
44. En tant qu'administrateur, je veux confirmer explicitement avant de retranscrire un clip qui porte des corrections, afin de choisir en connaissance de cause.
45. En tant qu'administrateur, je veux que la planification par lot exclue par défaut les clips corrigés et les signale, afin qu'un lot n'efface pas le travail humain.
46. En tant qu'administrateur, je veux pouvoir inclure explicitement ces clips dans un lot, afin de les retraiter volontairement.
47. En tant qu'utilisateur, je veux que les corrections remplacées par une retranscription restent dans l'historique, marquées comme obsolètes, afin de ne jamais perdre une trace de ce qui a été vérifié.
48. En tant qu'utilisateur, je veux que la transcription visible ne change qu'après le succès de la retranscription, afin de ne pas perdre le texte actuel en cas d'échec.

### Exploitation, mesure et confidentialité

49. En tant qu'administrateur, je veux exporter les segments corrigés avec leur texte d'origine et leur texte corrigé, afin de préparer la mesure de qualité de la phase 7.
50. En tant que propriétaire du projet, je veux que Snitch ne calcule aucun taux d'erreur dans cette phase, afin de réserver la mesure au benchmark.
51. En tant qu'administrateur, je veux que la suppression d'un clip efface ses corrections et leur historique tout en conservant l'original vidéo, afin de respecter l'invariant de la bibliothèque.
52. En tant qu'utilisateur, je veux que l'historique d'un segment ne soit visible que par ceux qui peuvent voir le clip, afin de respecter la visibilité.
53. En tant qu'exploitant, je veux que la correction soit enregistrée en une seule transaction avec son historique, afin qu'un crash ne laisse ni texte sans trace ni trace sans texte.
54. En tant qu'exploitant, je veux que le worker n'intervienne jamais dans une correction, afin qu'il n'accède pas à ces données et reste éteignable.
55. En tant que propriétaire du projet, je veux qu'aucun média réel, scan ou batch ne soit utilisé, afin de réserver la bibliothèque à l'autorisation de la phase 7.

### Contrat et interface

56. En tant que contributeur, je veux que le contrat OpenAPI et les clients générés décrivent les corrections, leur historique et les conflits, afin que SPA et API restent synchronisées.
57. En tant qu'utilisateur, je veux modifier un segment sans quitter la lecture du clip, afin de vérifier à l'oreille en corrigeant.
58. En tant qu'utilisateur, je veux voir en un coup d'œil les segments corrigés d'un clip, afin de savoir ce qui a déjà été relu.
59. En tant qu'utilisateur, je veux des messages clairs en cas de conflit, de texte refusé ou de clip non visible, afin de savoir comment poursuivre.

## Implementation Decisions

- Le monolithe Go reste propriétaire de PostgreSQL, des permissions, de l'API, de l'historique et de la réindexation. Le worker n'intervient pas : aucune tâche worker nouvelle, aucun accès du worker à ces données.
- La transcription produite par le worker n'est jamais modifiée. Une correction est une couche distincte qui recouvre le texte d'un segment ; le texte affiché et recherché est le texte corrigé s'il existe, sinon le texte d'origine. Cela conserve l'original (invariant de la bibliothèque, mesure de la phase 7) et évite de réécrire un résultat que la file remplace après chaque retranscription.
- L'unité de correction est le segment, identifié par sa transcription et son rang. Seuls le texte et le locuteur d'un segment sont corrigeables. Les débuts et fins de segment, leur nombre et leur découpage ne changent pas. Les horodatages par mot d'origine sont conservés pour la navigation ; le texte corrigé fait foi pour l'affichage et la recherche, sans prétendre réaligner les mots.
- L'historique est immuable : chaque correction enregistre ancienne valeur, nouvelle valeur, auteur, date, un motif facultatif et son type (correction de texte, correction de locuteur, restauration). Un déclencheur ou une contrainte empêche toute modification ou suppression directe d'une entrée. Une restauration, vers l'original ou vers une version antérieure, est une nouvelle entrée, jamais une suppression.
- Le contrôle de concurrence est optimiste : une demande de correction indique la version qu'elle croit remplacer. Si le segment a changé entre-temps, elle est refusée avec la version courante et son auteur, sans écrasement silencieux. Ce choix diffère de la validation vocale de la phase 5, où la première validation fait foi : un texte se corrige plusieurs fois à la suite, alors qu'un Locuteur local n'est validé qu'une première fois. Deux corrections de segments différents sont indépendantes.
- Tous les utilisateurs connectés qui peuvent voir un clip peuvent en corriger les segments, comme pour les validations vocales de la phase 5. Un utilisateur anonyme ne peut rien corriger. L'historique d'un segment suit la visibilité du clip.
- La correction du locuteur d'un segment est distincte de l'attribution d'un Locuteur local à une Personne (phase 5). Aujourd'hui le locuteur d'un segment est celui dont les tours de parole le recouvrent le plus ; une correction permet de désigner explicitement un autre Locuteur local du même clip, ou de revenir à la valeur déduite. Elle ne modifie ni les tours de parole, ni les empreintes, ni les attributions, ni les échantillons validés, et ne relance aucune diarisation. L'affichage et le filtre par Personne utilisent le locuteur corrigé.
- La recherche exacte, plein texte, approximative et hybride appliquent le texte corrigé. Les permissions et les autres filtres s'appliquent avant la pagination et les comptes, comme en phase 4. Chaque résultat indique si son segment est corrigé et s'ouvre au timestamp du segment.
- La réindexation sémantique est ciblée sur le segment corrigé : le vecteur de ce seul segment est recalculé et remplace l'ancien, pour l'index actif et, pendant une reconstruction, pour l'index candidat afin que la bascule ne ramène pas l'ancien texte. Elle ne relance ni la transcription ni les autres segments. Si le service d'embeddings est indisponible, la correction est enregistrée et la recherche lexicale la reflète aussitôt ; le segment reste marqué à réindexer et est repris sans doublon. Le nombre de segments en attente est visible des administrateurs. La correction d'un locuteur n'exige aucune réindexation sémantique, puisque l'embedding dépend du texte.
- Retranscrire un clip qui porte des corrections exige une confirmation explicite de l'administrateur. La planification par lot de la phase 3 exclut par défaut les clips corrigés, les signale et ne les inclut que sur demande explicite. Comme en phase 3, la transcription visible ne change qu'après la réussite de la tâche. Les corrections remplacées restent dans l'historique, marquées obsolètes, et ne sont jamais réappliquées silencieusement à un nouveau texte.
- Un export réservé aux administrateurs fournit les segments corrigés avec leur texte d'origine et leur texte corrigé, pour préparer la mesure de qualité du benchmark. Aucun taux d'erreur n'est calculé dans cette phase.
- La suppression d'un clip efface ses corrections et leur historique tout en conservant l'original, conformément à l'invariant de la bibliothèque. La suppression du profil vocal d'une Personne ou d'une Personne elle-même n'affecte pas les corrections de texte.
- Une correction est enregistrée en une seule transaction avec son entrée d'historique. La taille d'un texte corrigé est bornée ; un texte vide est refusé.
- Le contrat OpenAPI reste la source des interfaces ; les clients Go, TypeScript et Python sont régénérés et vérifiés en CI. Les migrations passent par la commande existante.
- Une décision technique non triviale de cette phase (couche de correction distincte de l'original, concurrence optimiste, réindexation ciblée, politique de retranscription) est consignée dans un ADR. Le glossaire peut recevoir les termes métier résolus (par exemple correction, transcription d'origine) sans détails d'implémentation.

## Testing Decisions

Un bon test observe le comportement aux interfaces publiques et la cohérence des résultats persistés ; il ne vérifie ni les appels privés ni la forme interne des tables. Deux coutures existantes sont utilisées, aucune nouvelle :

1. Un parcours API Go avec PostgreSQL/pgvector réel en Testcontainers, des transcriptions synthétiques produites par un worker simulé qui parle l'API du worker, et un serveur d'embeddings déterministe.
2. Un parcours Playwright avec la vraie API pour l'édition d'un segment, l'historique et la restauration.

Comportements à vérifier :

- Une correction de texte remplace le texte affiché sans modifier la transcription d'origine, qui reste consultable.
- L'historique enregistre ancienne valeur, nouvelle valeur, auteur et date ; il est immuable ; une restauration crée une nouvelle entrée.
- Deux corrections simultanées du même segment : une seule aboutit, l'autre est refusée avec la version courante et son auteur ; deux segments différents ne se bloquent pas.
- Le texte vide, démesuré, un clip non visible et un utilisateur anonyme sont refusés.
- Les quatre modes de recherche trouvent le texte corrigé et plus l'ancien ; les résultats indiquent le segment corrigé ; permissions, comptes et pagination restent cohérents.
- La correction réindexe uniquement le segment concerné, pour l'index actif et l'index candidat pendant une reconstruction ; un service d'embeddings arrêté n'empêche pas la correction, la recherche lexicale la reflète, et le segment est repris ensuite sans doublon.
- La correction du locuteur d'un segment se limite à ce segment : tours, empreintes, attributions et échantillons restent inchangés ; l'affichage et le filtre par Personne l'appliquent ; le retour à la valeur déduite fonctionne.
- Retranscrire un clip corrigé exige une confirmation ; un lot exclut par défaut les clips corrigés ; les corrections remplacées restent dans l'historique, marquées obsolètes ; un échec de retranscription conserve le texte actuel.
- L'export administrateur donne texte d'origine et texte corrigé et est refusé aux non-administrateurs.
- La suppression d'un clip efface corrections et historique sans toucher à l'original.
- Contrat OpenAPI, clients générés, migrations vierges et rejouées, tests Go avec race detector, go vet, build web, `pip check`.

Prior art : les tests d'historique, de concurrence et de permissions de la phase 5 (validation, correction et historique des attributions), les tests de reconstruction et d'indexation de la phase 4, les tests de transcription et de recherche de la phase 3, et les suites Playwright existantes.

## Out of Scope

- Redécoupage, fusion, ajout ou suppression de segments, et correction des mots un par un ou de leurs horodatages.
- Remplacement en masse ou global d'un terme sur plusieurs clips.
- Correction automatique de la transcription, par modèle de langage ou autre.
- Circuit de relecture ou d'approbation (assignation, validation par un second utilisateur) et commentaires sur les segments.
- Alimentation automatique du vocabulaire de transcription à partir des corrections.
- Calcul du taux d'erreur, benchmark de 200 à 500 clips, mesure de qualité et go/no-go : phase 7.
- Traitement ou scan de la vraie bibliothèque, batch de 1 To, traitement nocturne.
- Nouvelle diarisation ou modification des empreintes et des échantillons vocaux validés.

## Further Notes

- La phase 5 doit avoir été validée de bout en bout avant d'aborder le code de la phase 6 (AGENTS.md) ; le code vocal réel n'a pas encore tourné avec de vrais modèles. Cette spec ne démarre aucun code.
- La phase 7 mesure le taux d'erreur de mots sur un sous-échantillon corrigé humainement : c'est la raison de conserver le texte d'origine à côté du texte corrigé et d'offrir un export.
- Les tickets d'implémentation sont à dériver de cette spec au moment de la phase, comme pour les phases précédentes.
- Pièges connus des phases précédentes à respecter dans les tests de navigateur : pas de `role="status"` (utiliser `aria-live`), noms de boutons uniques, aucun style qui déplace un contrôle au survol.
