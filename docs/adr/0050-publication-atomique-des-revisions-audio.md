# 0050 — Publication atomique des révisions audio

Statut : accepté pour le ticket #340 du parent #274. Complète l'[ADR 0047](0047-plans-audio-et-revisions-compatibles.md) et l'[ADR 0012](0012-corrections-de-segments.md).

## Contexte

Une révision candidate (#337) n'est pas une preuve de complétude. Les lecteurs (affichage, recherche exacte, plein texte, approximative, hybride, embeddings, exports, couverture des voix) lisaient chacun le dernier job de transcription réussi par leur propre requête.

## Décision

- Une table `audio_publications` (une ligne par clip) désigne la révision visible. Sans ligne, la projection historique `legacy:<job>` reste visible : les anciens clips et le comportement de production ne changent pas.
- Les vues `audio_visible_revisions`, `audio_visible_segments` et `audio_visible_transcripts` sont l'unique source des lecteurs. Un segment visible garde l'identité de sa source (job, indice de segment) : les corrections, les tâches d'embeddings et les index actif et candidat gardent leurs clés. Le rang visible est une présentation seulement ; aucun remappage de texte ou de temps n'est inventé.
- `audioplans.Store.PreviewPublication` calcule l'aperçu (révision attendue, corrections actives, validations de voix, pistes remplacées) et une garde, empreinte de l'historique des corrections et des attributions. L'aperçu ne dépend pas du mode du Run : changer de pistes en mode « seulement ce qui manque » exige la même confirmation.
- `audioplans.Store.PublishRevision(id, aperçu, confirmé)` s'exécute dans une transaction qui verrouille le clip, donc sérialisée avec les corrections. Elle exige : Run actif et Plan le plus récent, décisions toutes résolues, diagnostics terminaux valides, enfants de transcription terminés, chaque segment source représenté exactement une fois. Une révision vide n'est publiable que si chaque piste examinée est explicitement `no_speech`. Un enfant ne publie jamais seul.
- Si la garde a changé depuis l'aperçu (correction tardive), `ErrPublicationChanged` impose un nouvel aperçu ; la correction n'est pas écrasée. Sans confirmation alors que du travail humain est remplacé : `ErrConfirmationRequired`. Les erreurs ne modifient rien : l'ancienne révision et ses corrections restent visibles.
- À la bascule seulement, les corrections actives deviennent obsolètes (historique conservé) et les tâches d'embeddings des profils actif et candidat sont créées par `ON CONFLICT DO NOTHING`. Republier la même révision est idempotent.
- Les routes de correction verrouillent le clip, résolvent l'indice visible dans la révision visible et refusent `expected_revision_id` périmé (409 `transcription_changed`). Une révision de plan exige cette garde.
- Les voix et la couverture sont calculées par piste pour une révision de plan.

## Conséquences

Le déclenchement (#339), la sélection et son interface (#341) appellent ces méthodes ; l'activation d'Auto reste à #344. Un nouveau lecteur doit consommer les vues, jamais `jobs` directement.
