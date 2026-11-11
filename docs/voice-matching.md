# Rapprochement vocal en base (phase 5, ticket 35)

Snitch rapproche les empreintes des Locuteurs locaux de la Référence de chaque Personne et propose ou attribue une Personne avec un niveau de décision. Le calcul part des vecteurs stockés dans PostgreSQL : ni worker, ni PC, ni vidéo. Les tests utilisent des empreintes synthétiques déterministes (vecteurs dans un plan : la similarité cosinus est le cosinus de l'angle).

## Principes

- **Références.** La Référence d'une Personne est construite uniquement à partir de ses Échantillons vocaux validés actifs (`voice_samples.retired_at IS NULL`), tant que son Profil vocal n'est pas supprimé. Le rapprochement n'écrit jamais dans `voice_samples` : une attribution automatique n'entre jamais dans une Référence (ADR 0008, ADR 0064).
- **Comparabilité.** Seules les empreintes comparables (`usable`, décidé par la [règle d'éligibilité](#éligibilité-des-empreintes)) de portée `speaker` sont comparées, et uniquement à l'intérieur d'un même triplet `(model, model_version, dimensions)`. Un groupe sans aucune référence n'est pas évalué : ses attributions automatiques sont retirées plutôt que laissées périmées.
- **Score** (ADR 0064, [ci-dessous](#score-et-référence-cumulative)). Par défaut, la similarité cosinus entre l'empreinte du Locuteur local et la Référence cumulative de la Personne ; en mode « meilleur échantillon », la meilleure similarité avec l'un de ses échantillons comparables.
- **Décision**, à partir des seuils (voir plus bas) :
  - aucune Personne au-dessus du seuil de suggestion : `unknown` ;
  - au moins deux Personnes à moins de la marge d'ambiguïté de la meilleure : `ambiguous`, sans Personne, avec ses candidates (jamais de fusion de deux voix proches sans validation humaine) ;
  - meilleure similarité au moins égale au seuil automatique : `automatic` si l'automatisation est activée, sinon `suggestion` ;
  - sinon `suggestion`.
- **Humain d'abord.** Un Locuteur local validé n'est jamais rapproché (sa ligne machine éventuelle est retirée). Les lignes machine d'une autre origine que `matching` (par exemple la piste micro, ticket 34) ne sont pas écrasées.

## Quand le rapprochement s'exécute

| Déclencheur | Portée |
| --- | --- |
| Validation ou correction d'un Locuteur local (ajout, retrait ou changement d'une référence) | Toute la bibliothèque |
| Fusion, séparation ou suppression d'une Personne, suppression d'un Profil vocal | Toute la bibliothèque |
| Suppression d'un clip, ou de la diarisation ou des empreintes d'un clip, qui portait un échantillon validé actif | Toute la bibliothèque |
| Changement du mode de score (`PUT /api/admin/voice-score-mode`) | Toute la bibliothèque |
| Résultat d'empreintes d'un clip enregistré | Ce clip |
| Changement des seuils (`PUT /api/admin/voice-matching`) | Toute la bibliothèque |
| Changement de la règle d'éligibilité qui modifie l'éligibilité d'au moins une empreinte (`PUT /api/admin/voice-eligibility`) | Toute la bibliothèque |
| `POST /api/admin/voice-matching/rematch` (administrateur) | Toute la bibliothèque |

L'exécution est sérialisée (mutex de processus et verrou consultatif PostgreSQL) et lit l'état validé. Le déclencheur est déjà validé quand le rapprochement démarre : un échec est journalisé sans défaire la validation, et le rapprochement suivant (ou l'explicite) le répare. Seules les lignes dont la Personne, le niveau ou les candidates changent sont réécrites ; un changement de Personne ou de niveau ajoute un événement `match` à l'historique du locuteur (sans auteur, sans donnée vocale).

Le calcul compare chaque cible à chaque Référence (scan exact, sans index) : avec une Référence par Personne, une voix coûte de l'ordre de 66 µs face à 200 Personnes en 256 dimensions (`BenchmarkRankLibrary`, Apple M3 Pro), soit environ 2 s de calcul pour 30 000 voix, lecture des vecteurs en sus. Le mode « meilleur échantillon » compare à chaque échantillon : environ 1 ms par voix pour 4 000 échantillons.

## Score et Référence cumulative

Une seule fonction de score (`app/internal/httpapi/voice_score.go`) sert le rapprochement, la Précision mesurée et, à terme, le benchmark d'empreinte : un seuil lu dans l'un vaut pour les autres.

- **Référence cumulative** (mode `cumulative`, par défaut). Pour un groupe `(model, model_version, dimensions)`, la moyenne des vecteurs normalisés des échantillons validés actifs dont l'empreinte de Locuteur local est *contributive*, pondérée par leur Parole nette (`net_speech_seconds`, à défaut la durée de l'empreinte) plafonnée à 10 s par échantillon. Plusieurs extraits brefs reconnaissent ainsi une voix qu'aucun ne reconnaît seul. Le score est le cosinus avec cette Référence.
- **Meilleur échantillon** (mode `best_sample`, le score de la phase 5). Chaque échantillon dont l'empreinte est *comparable* est une référence à part ; le score d'une Personne est la meilleure similarité avec l'un d'eux. Les échantillons seulement contributifs n'y entrent pas.

La Référence n'est pas stockée : elle est dérivée des échantillons à chaque rapprochement. Tout ce qui change les échantillons (validation, correction, fusion, séparation, suppression d'un Profil vocal, d'un clip ou de ses empreintes, requalification) est donc pris en compte par le rapprochement suivant, que ces actions déclenchent. Deux groupes ne sont jamais mélangés ; après un Changement du modèle d'empreinte (ADR 0044), les Références du nouveau groupe sont celles de ses empreintes, et la transition recalcule toutes les empreintes contributives des échantillons validés (pas seulement les comparables).

**Mode de score versionné** (`voice_score_modes`, historique en ajout seul, la version la plus haute est active). La version 1, posée par Snitch, est `cumulative`. `PUT /api/admin/voice-score-mode` enregistre une nouvelle version avec son auteur si le mode change (rien sinon) puis rapproche de nouveau toute la bibliothèque. L'écran **Paramètres avancés → Score d'une voix** le règle et affiche l'historique ; la réinitialisation revient au mode cumulatif.

## Réglages (`voice_matching_settings`, une ligne)

| Champ | Défaut | Rôle |
| --- | --- | --- |
| `auto_enabled` | `false` | Interrupteur de l'automatisation. Tant qu'il est faux, aucune attribution n'est `automatic` : seulement des suggestions. Sa modification par l'administrateur est livrée avec le ticket d'activation ; l'API actuelle ne l'expose qu'en lecture. |
| `automatic_threshold` | 0,90 | Similarité minimale d'une attribution automatique. |
| `suggestion_threshold` | 0,75 | Similarité minimale d'une suggestion (au plus égale au seuil automatique). |
| `ambiguity_margin` | 0,05 | Écart sous lequel deux Personnes sont jugées aussi proches. |
| `max_candidates` | 5 | Nombre maximal de candidates conservées (2 à 20). |

Les valeurs initiales ne sont pas des choix définitifs ; elles attendent le benchmark de la phase 7.

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

**Règle versionnée** (`voice_eligibility_rules`, historique en ajout seul, la version la plus haute est active) : `comparable_min_seconds`, `contributive_min_seconds` (au plus la première, 0,5 à 60 s), `max_clipped_ratio` (0 à 1), auteur et date. La version 1 est provisoire, posée par Snitch : 2 s comparable, 1 s contributive, 1 % d'écrêtage au plus. Ces valeurs attendent l'aperçu chiffré (#394), qui comparera 1 s, 1,5 s et 2 s ; elles ne changent aucun seuil de score.

**Requalification.** `PUT /api/admin/voice-eligibility` (administrateurs) crée une nouvelle version si les valeurs changent (rien sinon), puis applique la règle active à toutes les empreintes de production en une opération en base, sans job Worker ni audio, et relance le rapprochement de toute la bibliothèque si une éligibilité a changé. La réponse donne le nombre d'empreintes requalifiées (`requalified`). Une nouvelle empreinte de production est décidée par la règle active dans la transaction qui l'enregistre ; un verrou consultatif (`snitch.voice_eligibility`, partagé à l'enregistrement, exclusif au changement de règle) garantit qu'aucune empreinte n'est décidée par une règle remplacée. `GET /api/admin/voice-eligibility` donne la règle active, les vingt dernières versions, les valeurs provisoires et le décompte des empreintes de Locuteur local de production par éligibilité, dont celles décidées par une autre règle (`stale`).

**Non affectées.** Les empreintes de benchmark et celles d'une transition de modèle en attente (`benchmark = true`, ADR 0044, ADR 0056) gardent la décision de leur pipeline épinglé (durée et qualité minimales) et ne sont jamais requalifiées (`eligibility_rule_version` nul). À la bascule d'une transition, les empreintes du nouveau modèle entrent en production et la règle active les décide.

**Empreintes existantes.** La migration `00144` garde la décision de chaque empreinte déjà stockée (`eligibility_rule_version` nul, `contributive = usable`) : elles ne sont requalifiées que lorsque l'administrateur enregistre la règle (après l'aperçu de #394). Seule exception, une empreinte chevauchée n'est jamais éligible. La migration calcule dès maintenant la Parole nette des empreintes sans mesures, tant que leurs tours existent.

L'écran **Administration → Voix → Identification et seuils → Paramètres avancés → Empreintes vocales** règle les trois valeurs, affiche la version active (provisoire ou auteur et date), le décompte et l'historique.

## API

- `GET /api/voice-matching` : tout utilisateur connecté (il voit si l'automatisation est active).
- `PUT /api/admin/voice-matching` : administrateurs ; les quatre seuils sont obligatoires, `auto_enabled` est refusé. `400 invalid_voice_matching` pour des seuils incohérents. Les empreintes déjà stockées sont rapprochées de nouveau.
- `POST /api/admin/voice-matching/rematch` : administrateurs ; renvoie le résumé (voix examinées par niveau, attributions retirées).
- `GET` et `PUT /api/admin/voice-eligibility` : administrateurs ; règle d'éligibilité (voir plus haut). `400 invalid_voice_eligibility` pour une règle hors bornes ou une durée contributive supérieure à la durée comparable.
- `GET` et `PUT /api/admin/voice-score-mode` : administrateurs ; mode de score actif (`cumulative` ou `best_sample`), version, auteur, date et vingt dernières versions. `400 invalid_voice_score_mode` pour un autre mode.
- `VoiceFingerprint` (voix d'un clip) et `VoiceTraceFingerprint` (trace vocale) portent `usable` (comparable), `contributive`, `unusable_reason`, `eligibility_rule_version`, `net_speech_seconds` et `net_speech_source`.
- `Attribution.candidates` : Personnes entre lesquelles le rapprochement hésite (ambiguë : celles dans la marge ; suggestion ou automatique : les meilleures alternatives) avec rang et score ; vide pour une validation humaine. L'historique d'un locuteur peut contenir l'action `match`.

## Données

- `speaker_attribution_candidates(attribution_id, person_id, rank, score)` : candidates, supprimées avec l'attribution ou la Personne.
- `speaker_attributions.match_model` et `match_model_version` : modèle d'empreinte à l'origine d'une attribution automatique ; `origin = 'matching'`.
- Un index HNSW partiel par `(model, model_version, dimensions)` est créé à la première comparaison d'un groupe (`voice_fingerprints_hnsw_<hash>`). Les empreintes de modèles ou de dimensions différents ne partagent jamais un index. Le rapprochement compare à l'ensemble réduit des Références sans index ; l'index sert les recherches de plus proches voisins sur toutes les empreintes (groupes de voix inconnues).

## Écrans

La carte « Rapprochement vocal » de l'administration règle les seuils, affiche l'état de l'automatisation (lecture seule) et relance le rapprochement. Dans le panneau des voix d'un clip, une suggestion ou une voix ambiguë liste ses candidates avec leur similarité (« pas une garantie ») et un bouton « Valider comme … » : seul ce clic humain crée un échantillon validé.

## Contrat et code

`app/internal/httpapi/voice_matching.go` (`registerVoiceMatchingRoutes`, `rematchVoices`, `loadReferenceSamples`), `voice_score.go` (fonction de score), `voice_score_mode.go`, migrations `00024_voice_matching.sql` et `00146_voice_score_modes.sql`, tests `voice_matching_test.go` (suite « voice matching »), `voice_reference_test.go` (suite « voice reference »), `voice_score_test.go` (unitaires) et `web/e2e/matching.spec.ts` (`SNITCH_E2E_MATCHING=1`). Le client web est dans `web/src/voice/VoiceMatchingCard.tsx` et `SpeakerRow.tsx`.
