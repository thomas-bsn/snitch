# Rapprochement vocal en base (phase 5, ticket 35)

Snitch rapproche les empreintes des Locuteurs locaux des références des Personnes et propose ou attribue une Personne avec un niveau de décision. Le calcul s'exécute entièrement dans PostgreSQL (pgvector) : ni worker, ni PC, ni vidéo. Les tests utilisent des empreintes synthétiques déterministes (vecteurs dans un plan : la similarité cosinus est le cosinus de l'angle).

## Principes

- **Références.** Les références d'une Personne sont uniquement ses Échantillons vocaux validés actifs (`voice_samples.retired_at IS NULL`). Le rapprochement n'écrit jamais dans `voice_samples` : une attribution automatique ne devient jamais référence (ADR 0008).
- **Comparabilité.** Seules les empreintes `usable` (assez longues et nettes, décidé à l'enregistrement) de portée `speaker` sont comparées, et uniquement à l'intérieur d'un même triplet `(model, model_version, dimensions)`. Un groupe sans aucune référence n'est pas évalué : ses attributions automatiques sont retirées plutôt que laissées périmées.
- **Score.** Pour chaque Personne, la similarité cosinus maximale entre l'empreinte du Locuteur local et les empreintes de ses références.
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
| Résultat d'empreintes d'un clip enregistré | Ce clip |
| Changement des seuils (`PUT /api/admin/voice-matching`) | Toute la bibliothèque |
| `POST /api/admin/voice-matching/rematch` (administrateur) | Toute la bibliothèque |

L'exécution est sérialisée (mutex de processus et verrou consultatif PostgreSQL) et lit l'état validé. Le déclencheur est déjà validé quand le rapprochement démarre : un échec est journalisé sans défaire la validation, et le rapprochement suivant (ou l'explicite) le répare. Seules les lignes dont la Personne, le niveau ou les candidates changent sont réécrites ; un changement de Personne ou de niveau ajoute un événement `match` à l'historique du locuteur (sans auteur, sans donnée vocale).

Limite assumée de cette phase : le calcul compare toutes les cibles à toutes les références (scan exact). Il convient aux volumes de la phase 5 ; un calcul incrémental et la mesure sur la bibliothèque réelle relèvent de la phase 7.

## Réglages (`voice_matching_settings`, une ligne)

| Champ | Défaut | Rôle |
| --- | --- | --- |
| `auto_enabled` | `false` | Interrupteur de l'automatisation. Tant qu'il est faux, aucune attribution n'est `automatic` : seulement des suggestions. Sa modification par l'administrateur est livrée avec le ticket d'activation ; l'API actuelle ne l'expose qu'en lecture. |
| `automatic_threshold` | 0,90 | Similarité minimale d'une attribution automatique. |
| `suggestion_threshold` | 0,75 | Similarité minimale d'une suggestion (au plus égale au seuil automatique). |
| `ambiguity_margin` | 0,05 | Écart sous lequel deux Personnes sont jugées aussi proches. |
| `max_candidates` | 5 | Nombre maximal de candidates conservées (2 à 20). |

Les valeurs initiales ne sont pas des choix définitifs ; elles attendent le benchmark de la phase 7.

## API

- `GET /api/voice-matching` : tout utilisateur connecté (il voit si l'automatisation est active).
- `PUT /api/admin/voice-matching` : administrateurs ; les quatre seuils sont obligatoires, `auto_enabled` est refusé. `400 invalid_voice_matching` pour des seuils incohérents. Les empreintes déjà stockées sont rapprochées de nouveau.
- `POST /api/admin/voice-matching/rematch` : administrateurs ; renvoie le résumé (voix examinées par niveau, attributions retirées).
- `Attribution.candidates` : Personnes entre lesquelles le rapprochement hésite (ambiguë : celles dans la marge ; suggestion ou automatique : les meilleures alternatives) avec rang et score ; vide pour une validation humaine. L'historique d'un locuteur peut contenir l'action `match`.

## Données

- `speaker_attribution_candidates(attribution_id, person_id, rank, score)` : candidates, supprimées avec l'attribution ou la Personne.
- `speaker_attributions.match_model` et `match_model_version` : modèle d'empreinte à l'origine d'une attribution automatique ; `origin = 'matching'`.
- Un index HNSW partiel par `(model, model_version, dimensions)` est créé à la première comparaison d'un groupe (`voice_fingerprints_hnsw_<hash>`). Les empreintes de modèles ou de dimensions différents ne partagent jamais un index. Le rapprochement actuel compare à l'ensemble réduit des références ; l'index sert les recherches de plus proches voisins sur toutes les empreintes (groupes de voix inconnues, ticket suivant).

## Écrans

La carte « Rapprochement vocal » de l'administration règle les seuils, affiche l'état de l'automatisation (lecture seule) et relance le rapprochement. Dans le panneau des voix d'un clip, une suggestion ou une voix ambiguë liste ses candidates avec leur similarité (« pas une garantie ») et un bouton « Valider comme … » : seul ce clic humain crée un échantillon validé.

## Contrat et code

`app/internal/httpapi/voice_matching.go` (`registerVoiceMatchingRoutes`, `rematchVoices`), migration `00024_voice_matching.sql`, tests `voice_matching_test.go` (suite « voice matching ») et `web/e2e/matching.spec.ts` (`SNITCH_E2E_MATCHING=1`). Le client web est dans `web/src/voice/VoiceMatchingCard.tsx` et `SpeakerRow.tsx`.
