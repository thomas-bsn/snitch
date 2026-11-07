# 0051 — Sélection des pistes audio et aperçu signé des pertes

Statut : accepté pour le ticket #341 du parent #274. Complète l'[ADR 0047](0047-plans-audio-et-revisions-compatibles.md) ; l'orchestration durable et la publication restent à #339 et #340.

## Contexte

Le choix d'une piste unique (`PUT /audio-track`) ne permet ni Auto, ni plusieurs pistes, ni diagnostic, ni forçage. Le choix doit être validé, comparé à l'inventaire réel et à l'état du travail humain, puis persisté dans un Plan audio immuable, sans qu'une confirmation donnée pour un aperçu puisse couvrir un autre état.

## Décision

- Trois routes administrateur : `GET /api/admin/clips/{id}/audio-selection` (inventaire réel lu dans `clips.tracks`, diagnostics et fraîcheur fournis par le moteur), `POST …/preview` (aperçu sans effet), `POST …/audio-selection` (démarrage). Sans moteur injecté, la lecture répond `enabled=false` et l'aperçu et le démarrage répondent 503 : le comportement de production antérieur est inchangé jusqu'à l'activation (#344).
- `version` de contexte : empreinte de l'inventaire, des diagnostics, de la version d'état du moteur et du dernier plan. Un choix fondé sur une autre version est refusé en 409 `audio_selection_stale` avant tout travail.
- Validation : sélection manuelle non vide, indices connus (les indices vidéo ou absents sont refusés), jamais de rôle déduit d'une position. Une piste `ambiguous` à diagnostic courant exige une décision (inclure ou écarter) ; une piste `no_speech` à diagnostic courant ne peut être choisie qu'avec `forced=true`, l'intention (action, forçage, motif, auteur, date) étant conservée dans le Plan. Un diagnostic périmé est montré, il ne contraint pas.
- Le moteur (`AudioSelectionEngine`, #339/#344) calcule les effets (étapes ajoutées ou invalidées, pistes, pertes, empreinte du travail humain `stake_version`). L'API ajoute l'horodatage d'aperçu et signe (HMAC de la clé maître, domaine `audio-selection-v1`) un reçu lié au clip, à l'administrateur, au choix et aux effets, valable 15 minutes.
- Le démarrage relit tout sous verrou de la ligne du clip, recalcule les effets et exige qu'ils soient identiques à ceux du reçu (409 `audio_preview_changed` sinon, même si les compteurs sont égaux : le calcul porte sur l'empreinte du travail), puis la confirmation explicite (409 `audio_confirmation_required`) quand des corrections ou validations seraient remplacées. Le run, le Plan, les jobs enfants liés par le moteur et le reçu sont validés dans une seule transaction ; une erreur du moteur annule tout. La confirmation ne vaut que pour ce reçu : toute modification du choix, tout changement d'inventaire ou de travail humain exige un nouvel aperçu.
- Le contrat étend les schémas de #337 sans les modifier (`AudioSelection*`, `AudioAnalysisChoice`).
- Interface : un champ « Pistes audio » unique (Auto ou choix multiple, métadonnées, diagnostic, fraîcheur, cause, décision, forçage) sert à la préparation (étape en attente de piste) et à la fenêtre Analyser / Relancer. Il ne s'affiche que si le serveur l'active ; modifier le choix revient au formulaire et oublie l'aperçu et sa confirmation.

## Alternatives écartées

- Réutiliser l'API de piste unique : ne porte ni décisions ni aperçu.
- Faire confiance au client pour l'inventaire : la validation relit les pistes réelles.
- Confirmation générale « j'accepte les pertes » : une modification ultérieure du travail humain serait couverte à tort.
- Stocker les aperçus en base : le reçu signé et la comparaison des effets recalculés suffisent et n'ajoutent pas d'état à nettoyer.

## Conséquences

L'activation (#344) injecte un moteur réel dans `Server.AudioSelection`. Le moteur doit lier ses jobs dans la transaction fournie, ne pas valider lui-même et faire rechecker à la publication (#340) `Effects.Losses` et `StakeVersion`.
