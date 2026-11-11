# Rapprochement vocal en base (phase 5, ticket 35)

Snitch rapproche les empreintes des Locuteurs locaux de la Référence de chaque Personne et propose ou attribue une Personne avec un niveau de décision. Le calcul part des vecteurs stockés dans PostgreSQL : ni worker, ni PC, ni vidéo. Les tests utilisent des empreintes synthétiques déterministes (vecteurs dans un plan : la similarité cosinus est le cosinus de l'angle).

## Principes

- **Références.** La Référence d'une Personne est construite uniquement à partir de ses Échantillons vocaux validés actifs (`voice_samples.retired_at IS NULL`), tant que son Profil vocal n'est pas supprimé. Le rapprochement n'écrit jamais dans `voice_samples` : une attribution automatique n'entre jamais dans une Référence (ADR 0008, ADR 0064).
- **Comparabilité.** Seules les empreintes comparables (`usable`, décidé par la [règle d'éligibilité](#éligibilité-des-empreintes)) de portée `speaker` sont comparées, et uniquement à l'intérieur d'un même triplet `(model, model_version, dimensions)`. Un groupe sans aucune référence n'est pas évalué : ses attributions automatiques sont retirées plutôt que laissées périmées.
- **Score** (ADR 0064, [ci-dessous](#score-et-référence-cumulative)). Par défaut, la similarité cosinus entre l'empreinte du Locuteur local et la Référence cumulative de la Personne ; en mode « meilleur échantillon », la meilleure similarité avec l'un de ses échantillons comparables.
- **Décision**, à partir des seuils du groupe d'empreintes ([seuils par groupe](#seuils-par-groupe-dempreintes)) :
  - aucune Personne au-dessus du seuil de suggestion : `unknown` ;
  - au moins deux Personnes à moins de la marge d'ambiguïté de la meilleure : `ambiguous`, sans Personne, avec ses candidates (jamais de fusion de deux voix proches sans validation humaine) ;
  - meilleure similarité au moins égale au seuil automatique : `automatic` si l'automatisation est activée **et** que le groupe a des seuils appliqués, sinon `suggestion` ;
  - sinon `suggestion`.
- **Humain d'abord.** Un Locuteur local validé n'est jamais rapproché (sa ligne machine éventuelle est retirée). Les lignes machine d'une autre origine que `matching` (par exemple la piste micro, ticket 34) ne sont pas écrasées.

## Quand le rapprochement s'exécute

| Déclencheur | Portée | Exécution |
| --- | --- | --- |
| Validation ou correction d'un Locuteur local (ajout, retrait ou changement d'une référence) | Toute la bibliothèque | Arrière-plan (demande enregistrée avec la décision) |
| Nommage d'un groupe de voix, disposition, refus ou annulation d'un refus | Toute la bibliothèque | Arrière-plan |
| Fusion, séparation ou suppression d'une Personne, suppression d'un Profil vocal | Toute la bibliothèque | Arrière-plan |
| Suppression d'un clip, ou de la diarisation ou des empreintes d'un clip, qui portait un échantillon validé actif | Toute la bibliothèque | Arrière-plan |
| Changement du mode de score (`PUT /api/admin/voice-score-mode`) | Toute la bibliothèque | Arrière-plan |
| Changement des seuils (`PUT /api/admin/voice-matching`) | Toute la bibliothèque | Arrière-plan |
| Changement de la règle d'éligibilité qui modifie l'éligibilité d'au moins une empreinte (`PUT /api/admin/voice-eligibility`) | Toute la bibliothèque | Arrière-plan |
| Bascule du modèle d'empreinte (ADR 0044) | Toute la bibliothèque | Arrière-plan (demande enregistrée avec la bascule) |
| Résultat d'empreintes d'un clip enregistré (requête du Worker) | Ce clip | Arrière-plan (demande ciblée) |
| Étape « Rapprochement » de l'analyse d'un clip | Ce clip | Immédiate, dans la boucle d'analyse (l'étape attend son succès) |
| `POST /api/admin/voice-matching/rematch` (administrateur), interrupteur d'automatisation | Toute la bibliothèque | Immédiate : la réponse donne les chiffres du passage |
| Fin d'un recalcul des empreintes sur la Parole nette qui a recalculé au moins une empreinte (terminé ou annulé, #395) | Toute la bibliothèque | Passage complet lancé en arrière-plan à la fin du recalcul ; `rematched_at` une fois terminé |

**Rapprochement en arrière-plan (#392).** Une décision répond dès qu'elle est enregistrée : elle n'attend plus le rapprochement. Le déclencheur enregistre une demande (`voice_matching_requests`) dans la transaction de la décision quand c'est possible (validation, correction, bascule de modèle), sinon juste après son commit, puis réveille une boucle d'arrière-plan du serveur. Un passage prend toutes les demandes présentes une fois le verrou du rapprochement obtenu et relit le dernier état validé : plusieurs décisions rapprochées, ou prises pendant un passage, font un seul passage suivant. Une demande pour toute la bibliothèque donne un passage complet ; sinon chaque clip demandé est rapproché seul. Une demande n'est supprimée que par le passage qui l'a lue, après son commit : un passage interrompu (arrêt du serveur, échec) laisse ses demandes, reprises au démarrage suivant ou au nouvel essai automatique (toutes les 30 s), sans action de l'utilisateur. `voice_matching_status` garde le passage en cours, la fin, la durée et le nombre de demandes du dernier passage réussi, et le dernier échec.

**Verrous.** Le rapprochement prend un mutex de processus et le verrou consultatif `snitch.voice_matching`. La fusion, la séparation et la suppression d'une Personne, la suppression d'un Profil vocal et la bascule du modèle d'empreinte prennent les mêmes verrous et attendent donc un passage en cours ; une validation ou une correction n'en prend aucun et ne l'attend jamais. Les scores sont calculés avant toute écriture : les lignes d'attribution qu'un passage verrouille ne le sont que pendant ses écritures. Deux validations simultanées n'écrivent que leurs propres lignes et chacune ajoute sa demande : le passage suivant voit les deux.

L'exécution lit l'état validé. Le déclencheur est déjà validé quand le rapprochement démarre : un échec est journalisé sans défaire la validation, et la demande reste jusqu'à un passage réussi. Seules les lignes dont la Personne, le niveau ou les candidates changent sont réécrites ; un changement de Personne ou de niveau ajoute un événement `match` à l'historique du locuteur (sans auteur, sans donnée vocale).

Le calcul compare chaque cible à chaque Référence (scan exact, sans index) : avec une Référence par Personne, une voix coûte de l'ordre de 66 µs face à 200 Personnes en 256 dimensions (`BenchmarkRankLibrary`, Apple M3 Pro), soit environ 2 s de calcul pour 30 000 voix, lecture des vecteurs en sus. Le mode « meilleur échantillon » compare à chaque échantillon : environ 1 ms par voix pour 4 000 échantillons. La durée d'un passage complet mesurée sur une bibliothèque synthétique de 30 000 Locuteurs locaux et 200 Personnes est consignée dans [performance-rapprochement.md](performance-rapprochement.md).

## Score et Référence cumulative

Une seule fonction de score (`app/internal/httpapi/voice_score.go`) sert le rapprochement, la Précision mesurée et le benchmark d'empreinte (#397, voir [Benchmarks IA](ai-benchmarks-287.md)) : un seuil lu dans l'un vaut pour les autres.

- **Référence cumulative** (mode `cumulative`, par défaut). Pour un groupe `(model, model_version, dimensions)`, la moyenne des vecteurs normalisés des échantillons validés actifs dont l'empreinte de Locuteur local est *contributive*, pondérée par leur Parole nette (`net_speech_seconds`, à défaut la durée de l'empreinte) plafonnée à 10 s par échantillon. Plusieurs extraits brefs reconnaissent ainsi une voix qu'aucun ne reconnaît seul. Le score est le cosinus avec cette Référence.
- **Meilleur échantillon** (mode `best_sample`, le score de la phase 5). Chaque échantillon dont l'empreinte est *comparable* est une référence à part ; le score d'une Personne est la meilleure similarité avec l'un d'eux. Les échantillons seulement contributifs n'y entrent pas.

La Référence n'est pas stockée : elle est dérivée des échantillons à chaque rapprochement. Tout ce qui change les échantillons (validation, correction, fusion, séparation, suppression d'un Profil vocal, d'un clip ou de ses empreintes, requalification) est donc pris en compte par le rapprochement suivant, que ces actions déclenchent. Deux groupes ne sont jamais mélangés ; après un Changement du modèle d'empreinte (ADR 0044), les Références du nouveau groupe sont celles de ses empreintes, et la transition recalcule toutes les empreintes contributives des échantillons validés (pas seulement les comparables).

**Niveau de Référence de l'annuaire** (#396, `voice_reference_level.go`, `voice_person_reference.go`). `GET /api/persons` et `GET /api/persons/{id}` renvoient `reference` : ce que la Référence cumulative du groupe d'empreinte actif utilise (échantillons contributifs, clips distincts, Parole nette cumulée après plafond), les échantillons validés écartés par raison (`too_short`, `overlap_only`, `clipped`, `low_quality`, ou `no_fingerprint` sans empreinte de production dans ce groupe) et un niveau calculé par le serveur :

| Niveau | Règle |
| --- | --- |
| `none` | aucun échantillon contributif |
| `weak` | échantillons contributifs sur un seul clip, quel que soit le classement |
| `good` / `weak` | à partir de 3 Personnes avec Référence dans le groupe : `good` à partir de la médiane de leur Parole nette, `weak` en dessous ; avec moins de 3 Références, `good` (au moins 2 clips) |

La médiane porte sur toutes les Personnes du groupe, quels que soient le lecteur, la recherche et la page : le niveau est le même pour tous. Les Attributions automatiques n'entrent dans aucune Référence, donc pas dans le niveau. La carte de l'annuaire, la fiche et son alerte de Référence faible affichent ce niveau ; `GET /api/persons/{id}/voices` donne pour chaque voix `reference_use` (contributive ou raison de l'écart). Le *groupe actif* (`activeFingerprintGroup`, `voice_active_group.go`) est celui du modèle d'empreinte actif, avec la version et les dimensions de ses dernières empreintes de production ; si ce modèle n'en a encore aucune, le groupe qui a le plus d'empreintes de production de Locuteur local en tient lieu (modèles hors catalogue, jeux synthétiques). Le niveau suit la Référence cumulative quel que soit le mode de score actif.

**Mode de score versionné** (`voice_score_modes`, historique en ajout seul, la version la plus haute est active). La version 1, posée par Snitch, est `cumulative`. `PUT /api/admin/voice-score-mode` enregistre une nouvelle version avec son auteur si le mode change (rien sinon) puis rapproche de nouveau toute la bibliothèque. L'écran **Paramètres avancés → Score d'une voix** le règle et affiche l'historique ; la réinitialisation revient au mode cumulatif.

## Réglages (`voice_matching_settings`, une ligne)

| Champ | Défaut | Rôle |
| --- | --- | --- |
| `auto_enabled` | `false` | Interrupteur de l'automatisation (voir [Précision mesurée](voice-automation.md)). Tant qu'il est faux, aucune attribution n'est `automatic` : seulement des suggestions. |
| `automatic_threshold` | 0,90 | Valeur provisoire, jamais utilisée pour décider : reprise par la migration pour le groupe actif. |
| `suggestion_threshold` | 0,75 | Valeur provisoire : seuil de suggestion d'un groupe sans seuils appliqués (au plus égal au seuil automatique). |
| `ambiguity_margin` | 0,05 | Valeur provisoire : marge d'un groupe sans seuils appliqués. |
| `max_candidates` | 5 | Nombre maximal de candidates conservées (2 à 20), pour tous les groupes. |

## Seuils par groupe d'empreintes

L'échelle d'un score cosinus dépend du modèle d'empreinte (ADR 0064, #393). Le seuil de suggestion, le seuil automatique et la marge d'ambiguïté appartiennent donc à un groupe `(model, model_version, dimensions)` : table `voice_group_thresholds` (migration `00148`), en ajout seul, la version la plus haute d'un groupe étant en vigueur, avec l'origine (`migration`, `administrator`, `proposal`), le mode de score actif au moment de l'application, l'auteur (instantané, `NULL` pour Snitch) et la date.

- **Groupe sans seuils appliqués** : il ne produit que des suggestions, avec le seuil de suggestion et la marge provisoires ; aucune Attribution automatique, même interrupteur activé. Après un Changement du modèle d'empreinte (ADR 0044), le nouveau groupe démarre ainsi : l'automatisation reste suspendue jusqu'à ce qu'un administrateur lui applique des seuils, de préférence après la [calibration](voice-automation.md#calibration-des-seuils). Un groupe n'hérite jamais des seuils d'un autre.
- **Appliquer** est une action explicite de l'administrateur (`PUT /api/admin/voice-matching`) ; rien n'est appliqué automatiquement, pas même la proposition de la mesure. Réappliquer les mêmes valeurs ne crée pas de version.
- **Migration** : les seuils de l'installation (`voice_matching_settings`) sont repris, version 1 d'origine `migration`, pour le seul groupe actif ; une installation sans empreinte n'en reçoit aucun.
- **Groupe actif** (`activeFingerprintGroup`, `voice_active_group.go`) : le groupe des dernières empreintes de production comparables du modèle d'empreinte actif ; si ce modèle n'en a aucune, le groupe ayant le plus d'empreintes `speaker` de production (modèles hors catalogue : installations anciennes, Workers synthétiques). Aucun groupe sans empreinte.
- Les liens entre voix inconnues (`refreshVoiceGroupLinks`) suivent le seuil de suggestion de leur groupe.

## Éligibilité des empreintes

Depuis #390 (ADR 0064), Snitch décide de l'éligibilité d'une empreinte à partir de ses mesures et d'une règle versionnée, et non plus une fois pour toutes à l'enregistrement. Deux usages :

- **comparable** (colonne `usable`, conservée pour que les index partiels `WHERE usable` suivent la règle) : assez de Parole nette pour être rapprochée seule d'une Référence. Le rapprochement, la Précision mesurée et les groupes de voix inconnues ne lisent que celles-ci ;
- **contributive** (colonne `contributive`) : assez de Parole nette pour entrer dans une Référence une fois validée par un humain (Référence cumulative, #391). Une empreinte comparable est toujours contributive.

Règle appliquée, dans cet ordre, à chaque empreinte de production (portées `speaker` et `turn`) :

| Condition | `unusable_reason` | Comparable | Contributive |
| --- | --- | --- | --- |
| `overlapped` : aucun intervalle net d'au moins 0,5 s | `overlap_only` | non | jamais |
| part d'échantillons écrêtés au-dessus du maximum | `clipped` | non | non |
| Parole nette sous la durée comparable | `too_short` | non | si Parole nette ≥ durée contributive |
| sinon | (vide) | oui | oui |

`low_quality` ne vient plus que d'une décision antérieure à la règle (empreintes existantes non requalifiées, benchmark). La cohérence est stockée mais n'est pas un critère.

**Parole nette retenue** (`net_speech_seconds`, origine dans `net_speech_source`) : `worker` = mesure du Worker (#389) ; `turns` = empreinte sans mesures, Parole nette calculée en base depuis les tours stockés (tours du Locuteur local, ou du tour, moins ceux des autres Locuteurs locaux de la même diarisation, intervalles d'au moins 0,5 s ; fonction SQL `voice_turns_net_speech`), jamais plus que la durée de l'empreinte, c'est-à-dire la parole réellement embarquée ; `estimated` = ni mesures ni tours, par exemple un Locuteur local validé puis archivé par une nouvelle diarisation : la durée de l'empreinte, qui majore la Parole nette (avant #389 elle valait la somme des tours entiers), sans Chevauchement connu. Une Référence validée n'est donc jamais rejetée par la seule absence de ses tours, et l'estimation reste visible. Avant #389 une empreinte `too_short` se jugeait sur cette durée de tours entiers ; elle se juge désormais sur la Parole nette, plus courte en cas de Chevauchement.

**Règle versionnée** (`voice_eligibility_rules`, historique en ajout seul, la version la plus haute est active) : `comparable_min_seconds`, `contributive_min_seconds` (au plus la première, 0,5 à 60 s), `max_clipped_ratio` (0 à 1), auteur et date. La version 1 est provisoire, posée par Snitch : 2 s comparable, 1 s contributive, 1 % d'écrêtage au plus. Ces valeurs attendent l'aperçu chiffré (voir plus bas), qui compare 1 s, 1,5 s et 2 s ; elles ne changent aucun seuil de score.

**Requalification.** `PUT /api/admin/voice-eligibility` (administrateurs) crée une nouvelle version si les valeurs changent (rien sinon), puis applique la règle active à toutes les empreintes de production en une opération en base, sans job Worker ni audio, et relance le rapprochement de toute la bibliothèque si une éligibilité a changé. La réponse donne le nombre d'empreintes requalifiées (`requalified`). Chaque requalification, même sous une règle inchangée, est tracée (`voice_eligibility_requalifications`, #394) : auteur, version de règle, date, empreintes changées et rapport avant/après (empreintes de Locuteur local de production par éligibilité et Personnes ayant une Référence), avec le nombre d'empreintes requalifiées sans source audio connue, à Parole nette estimée et gardant leur vecteur d'origine. La relancer sous la même règle ne réécrit aucune empreinte ; aucune Personne, validation ni Échantillon vocal n'est touché. Une nouvelle empreinte de production est décidée par la règle active dans la transaction qui l'enregistre ; un verrou consultatif (`snitch.voice_eligibility`, partagé à l'enregistrement, exclusif au changement de règle) garantit qu'aucune empreinte n'est décidée par une règle remplacée. `GET /api/admin/voice-eligibility` donne la règle active, les vingt dernières versions, les valeurs provisoires et le décompte des empreintes de Locuteur local de production par éligibilité, dont celles décidées par une autre règle (`stale`).

**Non affectées.** Les empreintes de benchmark et celles d'une transition de modèle en attente (`benchmark = true`, ADR 0044, ADR 0056) gardent en base la décision de leur pipeline épinglé (durée et qualité minimales) et ne sont jamais requalifiées (`eligibility_rule_version` nul). À la bascule d'une transition, les empreintes du nouveau modèle entrent en production et la règle active les décide. Le benchmark d'empreinte ne lit pas cette décision stockée : il décide ses empreintes, en lecture seule, par la même instruction SQL que la requalification (`eligibilityDecidedOverSQL`) sous la règle enregistrée avec son run (#397).

**Empreintes existantes.** La migration `00144` garde la décision de chaque empreinte déjà stockée (`eligibility_rule_version` nul, `contributive = usable`) : elles ne sont requalifiées que lorsque l'administrateur enregistre la règle (après l'aperçu chiffré). Seule exception, une empreinte chevauchée n'est jamais éligible. La migration calcule dès maintenant la Parole nette des empreintes sans mesures, tant que leurs tours existent.

**Aperçu chiffré** (#394, `GET /api/admin/voice-eligibility/preview`, administrateurs). Sans aucune écriture (transaction en lecture seule) ni lecture de média, il décide les empreintes de Locuteur local de production sous la règle examinée (valeurs passées en paramètre, à défaut la règle active) et sous plusieurs durées minimales (`durations`, par défaut 1, 1,5 et 2 s, chacune appliquée aux deux usages avec l'écrêtage examiné). Il emploie l'instruction même de la requalification (`eligibilityDecidedSQL`, qui calcule aussi à la volée la Parole nette que la requalification remplirait) : les chiffres annoncés pour une durée sont ceux qu'une requalification à cette durée écrit. Par groupe `(modèle, version, dimensions)` : état enregistré, effet de la règle examinée et de chaque durée (comparables, contributives, rejetées par raison, gagnées et perdues, Personnes ayant une Référence), empreintes sans source audio connue, à Parole nette estimée, à vecteur d'origine, part de la parole retirée par les Chevauchements ; puis le détail par Personne (ses échantillons validés actifs). Une Personne a une Référence quand au moins un de ses échantillons validés actifs a une empreinte contributive dans le groupe.

- **Source audio absente** : d'après la base seulement (aucun fichier `ready` dans une source activée, aucun média téléversé) ; l'empreinte est requalifiée comme les autres et comptée à part.
- **Vecteur d'origine** : une empreinte sans mesures du Worker (`measures_version` nul, `net_speech_source` `turns` ou `estimated`) garde le vecteur calculé avant la Parole nette ; la requalification ne change que son éligibilité. Seul le recalcul (ci-dessous) le remplace.
- **Parole nette estimée** : Locuteur local archivé sans tours, compté à part ; son Chevauchement est inconnu et n'entre pas dans la part retirée.

L'écran **Administration → Voix → Identification et seuils → Paramètres avancés → Empreintes vocales** règle les trois valeurs, affiche la version active (provisoire ou auteur et date), le décompte et l'historique. « Voir l'aperçu chiffré » affiche l'aperçu des valeurs saisies ; « Requalifier avec cette règle » les applique, puis l'écran montre le rapport de la dernière requalification (avant/après).

**Recalcul sur la Parole nette** (#395, spec voix brèves section 6 temps 3). Des tâches d'empreinte recalculent les vecteurs d'origine à partir des diarisations déjà faites ; rien n'est rediarisé et aucun Locuteur local, validation, Personne ni Échantillon vocal n'est modifié.

- **Périmètre** : empreintes de Locuteur local de production à vecteur d'origine (`measures_version` nul), hors empreintes d'un modèle du catalogue qui n'est pas le modèle d'empreinte actif ; le recalcul n'écrit donc que dans le groupe du modèle actif (un résultat d'un autre modèle est refusé, `stale_voice_result`). Une tâche par diarisation, de priorité basse (−10), sans modèle demandé (ce ne sont pas des empreintes de benchmark).
- **Non recalculables, gardées en service et comptées** : source audio absente (même critère que l'aperçu chiffré) ; Locuteur archivé ou dont la diarisation n'est plus la courante (pas de tours, la tâche serait refusée).
- **Ancienne empreinte en service jusqu'au succès** : le résultat remplace l'empreinte dans la transaction de sa complétion, voix par voix (une voix que le Worker n'a pas réembarquée garde l'ancienne) ; un échec, une annulation ou un résultat pour une diarisation remplacée entre-temps la laisse en place. La nouvelle empreinte est décidée par la règle d'éligibilité active.
- **Jobs** : version d'entrée `f5-a<piste>-net-p<pipeline>-d<diarisation>` (le marqueur précède le pipeline : `VoiceInputVersion` lit toujours pipeline et diarisation, et l'analyse du clip, qui cherche la version de production exacte, ne prend jamais une tâche de recalcul pour la sienne ; la vue des voix du clip non plus). Le recalcul crée lui-même ses tâches et ne rattache que celles qu'il a créées (`voice_recompute_jobs`) : jamais pour un clip dont une tâche d'empreinte de production est active (elle n'est ni remplacée ni absorbée ; le clip est planifié quand elle se termine). Une diarisation d'un plan audio garde son décalage de piste : la tâche hors plan prend celui de sa diarisation.
- **Déroulé** : `GET /api/admin/voice-fingerprint-recompute/preview` (ampleur, sans écriture) ; `POST /api/admin/voice-fingerprint-recompute` avec `confirm_fingerprints` égal au chiffre de l'aperçu (`409 voice_recompute_preview_stale` sinon), refusé pendant une transition de modèle ouverte (`voice_transition_in_progress`, ADR 0044) ou un autre recalcul (`voice_recompute_in_progress`) ; une transition est refusée pendant un recalcul. La progression et le rapport sont lus dans les tâches et les empreintes (`voice_recompute_runs`, `voice_recompute_items` : instantané de chaque empreinte au lancement) ; le recalcul avance à chaque résultat, à chaque lecture de l'état et chaque minute, se termine quand plus rien n'attend, fige alors son rapport et relance le rapprochement de toute la bibliothèque (`rematched_at`), qui ne convertit jamais une suggestion en validation. `DELETE` annule le recalcul et ses seules tâches ; `POST …/resume` reprend le dernier recalcul en replanifiant les empreintes dont la tâche a échoué ou été annulée (une tâche réussie n'est jamais recréée ; relancer un nouveau recalcul ne reprend que les empreintes encore d'origine).
- **Rapport** : recalculées, dont récupérées (rejetées avant, comparables après), devenues contributives, toujours rejetées, perdues (comparables avant, plus après) ; erreurs, annulées, sources absentes, voix archivées, remplacées hors du recalcul (`obsolete`) : toutes gardent leur empreinte d'origine.

Écran : même section **Empreintes vocales**, bloc « Recalculer les empreintes sur la Parole nette » (aperçu, confirmation du chiffre, progression, annulation, reprise, rapport final et date du rapprochement relancé).

## API

- `GET /api/voice-matching` : tout utilisateur connecté (il voit si l'automatisation est active). Les seuils renvoyés sont ceux du groupe actif (`group`), ou les provisoires ; `thresholds_applied` dit si le groupe a des seuils appliqués.
- `PUT /api/admin/voice-matching` : administrateurs ; les quatre valeurs sont obligatoires, `auto_enabled` est refusé. Les seuils vont au groupe `group` s'il est donné, sinon au groupe actif (nouvelle version tracée, `source` `administrator` par défaut ou `proposal`) ; sans aucun groupe, ils deviennent les valeurs provisoires. `400 invalid_voice_matching` pour des seuils incohérents, `409 voice_group_unknown` pour un groupe sans empreinte de production. Les empreintes déjà stockées sont rapprochées de nouveau, en arrière-plan.
- `POST /api/admin/voice-matching/rematch` : administrateurs ; exécute tout de suite un passage complet (qui prend aussi les demandes en attente) et renvoie le résumé (voix examinées par niveau, attributions retirées).
- `GET /api/voice-matching/status` : tout utilisateur connecté ; `pending` (demandes en attente), `running` et `running_since`, `last_finished_at`, `last_duration_ms` et `last_requests` du dernier passage réussi, `last_failed_at` (null quand le dernier passage a réussi). `GET /api/voice-matching/status/events` : le même état en flux `text/event-stream` (sujet `voice_matching` de l'outbox `events`, alimenté par déclencheurs), session revérifiée avant chaque snapshot.
- `GET` et `PUT /api/admin/voice-eligibility` : administrateurs ; règle d'éligibilité (voir plus haut), avec les dix dernières requalifications (`requalifications`). `GET /api/admin/voice-eligibility/preview` : aperçu chiffré. `/api/admin/voice-fingerprint-recompute` (`GET`, `POST`, `DELETE`, `/preview`, `/resume`) : recalcul sur la Parole nette (voir plus haut). `400 invalid_voice_eligibility` pour une règle ou une durée hors bornes ou une durée contributive supérieure à la durée comparable.
- `GET` et `PUT /api/admin/voice-score-mode` : administrateurs ; mode de score actif (`cumulative` ou `best_sample`), version, auteur, date et vingt dernières versions. `400 invalid_voice_score_mode` pour un autre mode.
- `VoiceFingerprint` (voix d'un clip) et `VoiceTraceFingerprint` (trace vocale) portent `usable` (comparable), `contributive`, `unusable_reason`, `eligibility_rule_version`, `net_speech_seconds` et `net_speech_source`.
- `Attribution.candidates` : Personnes entre lesquelles le rapprochement hésite (ambiguë : celles dans la marge ; suggestion ou automatique : les meilleures alternatives) avec rang et score ; vide pour une validation humaine. L'historique d'un locuteur peut contenir l'action `match`.

## Données

- `speaker_attribution_candidates(attribution_id, person_id, rank, score)` : candidates, supprimées avec l'attribution ou la Personne.
- `speaker_attributions.match_model` et `match_model_version` : modèle d'empreinte à l'origine d'une attribution automatique ; `origin = 'matching'`.
- Un index HNSW partiel par `(model, model_version, dimensions)` est créé à la première comparaison d'un groupe (`voice_fingerprints_hnsw_<hash>`). Les empreintes de modèles ou de dimensions différents ne partagent jamais un index. Le rapprochement compare à l'ensemble réduit des Références sans index ; l'index sert les recherches de plus proches voisins sur toutes les empreintes (groupes de voix inconnues).

## Écrans

La carte « Rapprochement vocal » de l'administration affiche le modèle d'empreinte actif, ses seuils appliqués (version, origine, auteur, date) ou leur absence, l'historique, applique de nouveaux seuils à ce modèle et relance le rapprochement. L'en-tête signale une attribution automatique suspendue faute de seuils. Elle indique, comme la liste « À traiter » et l'écran de traitement d'un clip, si un rapprochement est en cours ou l'heure du dernier (« Suggestions à jour · dernier rapprochement à 14:32 »), et un échec repris automatiquement. La liste et l'écran de traitement relisent leurs voix à la fin de chaque passage : la suggestion qui suit une validation y apparaît d'elle-même. Dans le panneau des voix d'un clip, une suggestion ou une voix ambiguë liste ses candidates avec leur similarité (« pas une garantie ») et un bouton « Valider comme … » : seul ce clic humain crée un échantillon validé.

## Contrat et code

`app/internal/httpapi/voice_matching.go` (`registerVoiceMatchingRoutes`, `rematchVoices`, `loadReferenceSamples`), `voice_score.go` (fonction de score), `voice_matching_queue.go` (demandes, boucle d'arrière-plan, `matchingChanged`, `rematchNow`, état), `voice_score_mode.go`, `voice_fingerprint_recompute.go` (recalcul sur la Parole nette), migrations `00024_voice_matching.sql`, `00146_voice_score_modes.sql` `00147_voice_recompute.sql` et `00149_voice_matching_requests.sql`, tests `voice_matching_test.go` (suite « voice matching »), `voice_reference_test.go` (suite « voice reference »), `voice_matching_background_test.go` (suite « voice matching in the background » ; les autres suites rapprochent dans la requête, `matchingInline`), `voice_matching_perf_test.go` (mesure, `SNITCH_PERF=1`), `voice_score_test.go` (unitaires) et `web/e2e/matching.spec.ts` (`SNITCH_E2E_MATCHING=1`). Le client web est dans `web/src/voice/VoiceMatchingCard.tsx` et `SpeakerRow.tsx`.
