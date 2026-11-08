# 0049 — Orchestration durable d'une analyse multipiste

Statut : accepté pour le ticket #339 du parent #274. Complète l'[ADR 0047](0047-plans-audio-et-revisions-compatibles.md) et l'[ADR 0024](0024-analyse-de-clip-durable.md). Non activé en production : seuls des appels explicites (`analysis.NewAudio`) créent un Plan ; le démarrage historique ne change pas.

## Contexte

Un Plan audio immuable (ADR 0047) doit être exécuté : diagnostiquer chaque piste, ne transcrire que les pistes retenues, agréger les enfants et remettre le résultat complet à une interface de publication. Une panne ne doit recalculer que l'enfant touché, un redémarrage ne doit rien dupliquer, et l'annulation ne doit pas toucher les jobs d'un autre plan.

## Décision

- L'étape durable `transcription` du Run d'un Plan est pilotée par `AudioService.Reconcile`, sérialisé par le verrou de ligne du Run. Le ticker historique ignore tout Run possédant un Plan audio : il ne planifie jamais de transcription monopiste pour lui.
- Le détecteur et le publieur sont injectés (`AudioDiagnostics`, `AudioPublisher`). Sans détecteur, l'étape est bloquée (`diagnostics_unavailable`) ; sans publieur, elle est bloquée (`publication_unavailable`) et aucun enfant ne remplace la transcription visible.
- Auto attend le diagnostic de toutes les pistes de l'inventaire avant de créer le moindre job d'ASR. Sont retenues les pistes `speech`, plus celles dont une décision `include` est forcée. Une piste `ambiguous` ou `failed` bloque le Run (`diagnostic_ambiguous`, `diagnostic_failed`) ; un Plan étant immuable, la décision de l'opérateur crée un autre Plan (manuel, indices exacts, forçage enregistré) après annulation. Si aucune piste n'est retenue, le résultat vide est publié sans job d'ASR.
- Chaque piste retenue possède un seul job de transcription, lié au Plan dans la transaction qui le crée. Sa version d'entrée porte le préfixe `mt1:` ; seuls les Workers déclarant la capacité `audio.multitrack.v1` peuvent le prendre, et la tâche (`audio_plan` dans l'entrée du job) est relue depuis le Plan immuable, jamais depuis le pipeline ou le vocabulaire courants.
- Le Worker rend des temps sur la ligne de temps du clip (`time_base: "clip"`) : l'instant local de chaque segment et mot reçoit le début de l'intervalle découpé puis le décalage du flux, une seule fois. Le serveur refuse tout résultat d'un autre Plan, d'une autre piste ou en temps local avant publication.
- L'agrégation ne se termine pas au premier enfant. Quand tous les enfants attendus sont terminés, `PublishAudio` est appelé avec la transaction du verrou : la publication et la fin du Run sont atomiques. Une erreur laisse le Run reprenable.
- `Retry(plan, piste, étape)` remet en file le seul job mort concerné (même job, compteur de tentatives remis à zéro pour la transcription ; un diagnostic garde son compteur pour que chaque observation reste immuable). `Cancel` annule les enfants non terminés du Plan, y compris ceux sous bail, bloque les étapes et libère le clip ; l'annulation historique d'un Run appelle la même routine.

## Alternatives écartées

- Un job « parent » synthétique : doublonne l'état déjà porté par le Run et les liens enfants.
- Publier chaque enfant puis fusionner : montrerait un résultat partiel et rendrait des corrections obsolètes.
- Modifier la décision d'un Plan bloqué : contredit l'immutabilité de l'ADR 0047.

## Conséquences

Le raccordement du détecteur réel et des points d'entrée HTTP appartient au ticket d'intégration (#344), la publication de la révision à #340, la sélection et l'aperçu à #341.
