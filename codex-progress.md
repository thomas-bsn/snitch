# Reprise #274 — PR #345

Branche : integration/issue-274-multitrack ; HEAD initial f0e9e23f.

Autorisé : corrections, tests synthétiques, revue depuis 53583abc, intégration main nécessaire, push et PR ready. Aucun merge PR, scan réel ou benchmark phase 7.

En cours : validation complète finale et contre-revue des corrections, puis commit/push et passage de PR345 à ready.

Résultats antérieurs : /tmp/snitch-274/344-result.md. Ils ne remplacent pas les validations finales de cette reprise.

## Reprise finale

- Revue indépendante depuis 53583abc : 2 défauts fonctionnels Standards et 6 Spec identifiés ; corrections appliquées, contre-vérification en cours.
- Migration 118 remplacée par une optimisation de vue ; aucun réglage JIT global. Test sur 3 000 clips / 36 000 segments passant avec JIT activé.
- Corrections : offset inconnu conservateur, paires acoustiques explicites, couverture acoustique union des preuves directes, diagnostics réutilisables entre Plans, réanalyse limitée, pistes vocales réellement retenues, garde des validations concurrentes, cycle complet jusqu'aux voix.
- Génération reproductible validée. Worker final : 332 tests, 9 sauts de dépendances vocales optionnelles ; Silero et déduplication exécutés sur fixtures dédiées. Smoke serveur Go / Worker Python réel passant (ASR déterministe injecté).
- Chromium natif bloqué par le sandbox macOS. Contournement de test hors dépôt : Playwright Docker + relais localhost. Premier relais non sécurisé rejeté car crypto.randomUUID indisponible ; relais localhost Docker validé : les sept tests admin-voice passent.
- PR #345 reste draft jusqu'à validation complète, push et bilan. Aucun média réel traité.

- Dernier contrôle : la relance forcée de diarisation renouvelle aussi le même enfant empreinte ; régression ajoutée. Suite Go/E2E Docker complète en cours.

## Publication préparée

- Corrections committées : `d0eb2502` sur `integration/issue-274-multitrack`.
- Métadonnées temporaires : `/tmp/snitch-274/publish.git`, worktree source inchangé. Le sandbox interdit l'écriture dans le gitdir partagé situé sous `/Users/thomasboisson/dev/snitch/.git/worktrees/` ; les commits et le push utilisent donc ce gitdir temporaire. Le HEAD du gitdir partagé reste historique.
- Fetch GitHub effectué avec succès depuis ce gitdir. `main` = `53583abc` est entièrement intégré ; aucune fusion supplémentaire requise. La branche distante est ancêtre de l'intégration (push fast-forward prévu).
- Régression de relance forcée passant ; `go vet ./...` passant après la dernière correction.
- Suite complète Go/race/PostgreSQL + E2E Docker toujours en cours ; pas encore de passage ready.

- Deux smokes Python réels passent : parole/tonalité (3 jobs) et doublon vocal (5 jobs, diagnostic Silero + déduplication PCM réelle + provenance double). Journal : `/tmp/snitch-274/python-both-smokes.log`. ASR injecté déterministe.

- Push fast-forward effectué jusqu’à `f0d94458` sur la branche existante ; la CI GitHub a démarré (Go, cinq shards E2E, Worker, lint et images). PR #345 toujours draft pendant la validation locale.

- CI : premier run Worker rouge car ffmpeg absent du runner (4 tests de décodage). Correctif workflow `26b9cdb4` poussé ; Worker et lint verts sur le run `42434934502`. Autres jobs en cours.

- CI E2E a révélé un défaut de restitution après publication : `audio_published` n'avait pas de traduction. Le moteur réutilise maintenant `audio_plan_published`. L'assertion vocale du parcours d'activation attend uniquement la piste retenue, sans piste `no_speech`. Correctif `32bffa97` poussé ; tests du moteur d'analyse verts. Run courant : `42435321432`.

- Suite locale complète terminée : tous les sous-tests Go/PostgreSQL passent ; 40/45 suites navigateur passent. Cinq suites ont échoué : trois limites du relais Docker (Discord, contrôle embeddings, accès au répertoire temporaire), scénarios clip-analysis intermittents et sélection de réanalyse non forcée. Relance ciblée : recherche 8/8, panneau analyse 4/4, activation multipiste 3/3 passent. Le scénario destructif choisit explicitement le mode full (`fe35f2ff`, poussé). Les deux suites intermittentes sont répétées trois fois pour contrôle. CI relancée sur ce commit.

- Répétition locale : course reproduite dans clip-analysis, claim avant achèvement du POST de lancement ; synchronisation du test sur fermeture du dialogue et réponse de retry, sans temporisation arbitraire. Recherche CI intermittente observée séparément ; répétitions locales en cours.
