# Analyse de clip durable

Statut : accepté pour #115 (ticket #165). Complète l'[ADR 0012](0012-corrections-des-segments-transcrits.md) et l'[ADR 0022](0022-modele-vocal-actif-global.md) sans les modifier. Numéro 0024 : les numéros 0020 à 0022 étaient déjà pris sur la branche d'intégration.

## Contexte

Un administrateur lançait séparément la transcription et l'analyse vocale. #115 demande une analyse lancée d'un clic, qui enchaîne les étapes applicables, survit à la fermeture du navigateur et au redémarrage, et explique l'état de chaque étape. Les jobs existent déjà (leases, délais, limites de retry) et ne doivent pas être remplacés. Les tickets suivants ajouteront les branches vocales, l'indexation, l'annulation, le retry, la réanalyse, les événements temps réel et les notifications.

## Décision

**Une exécution et des étapes persistées.** Les tables `clip_analysis_runs` et `clip_analysis_steps` portent l'exécution (clip, périmètre, statut `active`, `completed`, `failed` ou `cancelled`, auteur, dates) et ses étapes (statut, code de raison, job suivi, tentative, dates). Le job reste l'unité de travail ; une étape ne fait que le suivre. Une étape d'un autre type s'ajoute par une ligne et une règle de dérivation, sans changer le schéma.

**Une exécution active par clip.** Un index unique partiel sur `clip_id` pour `status='active'` arbitre la concurrence dans PostgreSQL. Un démarrage concurrent ou répété renvoie l'exécution existante (200) ; un périmètre différent est refusé (409 `analysis_scope_conflict`, avec l'exécution active) jusqu'à sa fin. Le périmètre est ce que l'administrateur demande depuis le clip : `transcription` (transcrire seulement), `voice` (détection des voix : diarisation, empreintes, rapprochement) ou `full` (transcription, indexation et voix). Chaque exécution liste toutes les étapes (`preparation`, `transcription`, `indexing`, `diarization`, `fingerprints`, `matching`) ; celles hors périmètre sont `not_applicable` avec la raison `outside_scope`. Le périmètre `voice` s'appuie sur une transcription valide existante et ne transcrit jamais en douce : sans elle, le démarrage est refusé (409 `transcription_required`) et aucune exécution n'est créée.

**Étapes pas encore orchestrées.** Dans #165, seule la chaîne préparation puis transcription est câblée. L'indexation et les étapes vocales de `voice` et `full` prennent le statut `not_yet_supported` (raison `orchestration_pending`) une fois la transcription acquise, et l'exécution se termine `incomplete` : jamais un faux succès, et le clip est libéré. #166 (voix) et #168 (indexation) remplaceront ce statut par la dérivation depuis leurs jobs, sans changer le schéma ni le contrat.

**Statut dérivé, jamais saisi.** Statuts : `waiting`, `running`, `retry`, `blocked`, `failed`, `done`, `not_applicable`, `not_yet_supported`. Les raisons sont des codes stables traduits par l'interface (`awaiting_worker`, `audio_track_selection_required`, `retry_scheduled`, `reused_existing`, `kept_for_corrections`, `job_dead`, `transcription_required`…). Aucun pourcentage n'est inventé. La préparation vaut « média prêt et piste audio résolue » ; un choix de piste indispensable bloque cette seule étape, l'étape suivante attend.

**Réutilisation et protection des corrections.** L'étape de transcription réutilise un job `done` de la version d'entrée courante. Si seule une version plus ancienne existe et que des corrections humaines sont actives, la transcription visible est conservée (`kept_for_corrections`) : l'analyse ne remplace jamais un travail humain, conformément à l'ADR 0012 ; la retranscription confirmée relèvera de la réanalyse. Sinon un job est planifié par le chemin existant, idempotent par clip, étape et version.

**Avancement idempotent, sans verrou long.** `advance` recalcule les étapes depuis les faits (clip, piste, jobs) et ne met à jour que ce qui change ; plusieurs appelants convergent. Il est déclenché après un résultat ou un échec de worker, après un choix de piste, au démarrage et par un ticker de 5 secondes (`Serve`), ce qui assure la reprise après redémarrage ou perte d'un déclenchement. Un job mort ou annulé fait échouer l'étape puis l'exécution, ce qui libère le clip ; une nouvelle exécution ignore les jobs terminés en échec avant son début.

**API.** `POST /api/admin/clips/{id}/analysis` et `GET /api/admin/clips/{id}/analysis`, réservés aux administrateurs. Les commandes existantes (transcription manuelle, jobs, piste audio) restent. La page du clip remplace le bouton de transcription manuelle par un bouton à menu : « Tout analyser » (`full`), « Transcrire seulement » et « Détection des voix » et interroge l'état avec un sondage modéré ; les événements SSE viendront plus tard sur le même modèle.

## Alternatives écartées

- **Dériver l'état à la lecture seulement, sans table.** Pas d'historique, pas de périmètre ni d'auteur, et pas d'identité d'exécution pour l'idempotence ni pour les tickets suivants.
- **Verrouiller la ligne de l'exécution pendant l'avancement.** Les crochets appellent le service de jobs, qui verrouille le clip ; l'avancement idempotent évite l'inversion d'ordre de verrous.
- **Remplacer les jobs par des étapes.** Perdrait les leases, délais et limites de retry existants.

## Addendum : réanalyse (#168)

**Trois modes.** Le démarrage accepte un mode : `missing` (ne produire que ce qui manque, par défaut), `full` (tout refaire) et `steps` (refaire des étapes choisies). Les étapes forçables sont `transcription`, `indexing` et `diarization` ; les empreintes et le rapprochement suivent la diarisation. L'exécution persiste le mode et les étapes forcées, et un historique des exécutions est lisible par clip.

**Enjeux et confirmation.** Avant de refaire quoi que ce soit, l'API calcule les enjeux : corrections humaines actives et attributions de voix qui seraient touchées. Si des enjeux existent, la réanalyse exige une confirmation explicite de l'administrateur ; sans elle, le démarrage est refusé avec un conflit qui décrit ces enjeux, et aucune exécution n'est créée. L'interface les présente dans une boîte de confirmation.

**Travail humain protégé jusqu'au succès.** Les corrections et attributions humaines ne sont ni supprimées ni remplacées tant que la nouvelle production n'a pas réussi : en cas d'échec ou d'annulation, la transcription et les voix visibles précédentes restent intactes, et le clip est libéré. Le remplacement n'a lieu qu'à l'acquisition d'un résultat valide, conformément à l'ADR 0012.
