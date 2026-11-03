# Lot B — Chaîne d'analyse

Fait partie des évolutions des lots A à D (voir [ROADMAP](../../ROADMAP.md)). Données synthétiques uniquement.

## Problème

À l'ajout d'une vidéo, seule la transcription part ; la détection des voix se lance à la main. Un téléversement vers un nouveau chemin ne déclenche aucun scan. L'analyse d'un clip ne peut pas être supprimée sans supprimer le clip. Il n'existe aucune donnée pour dessiner une Forme d'onde.

## Solution

Analyse complète automatique ([ADR 0035](../adr/0035-analyse-complete-automatique.md)) ; Suppression d'analyse par étapes ([ADR 0030](../adr/0030-suppression-d-analyse-par-etapes.md)) ; pics de forme d'onde calculés par le Worker ([ADR 0034](../adr/0034-pics-de-forme-d-onde.md)).

## Comportements

- **Réglage** « Analyse complète automatique » dans l'administration, actif par défaut. Un ajout (vidéo prête dans la Racine vidéo, publication d'un téléversement, sélection depuis l'import) démarre une Analyse de clip `full`. Désactivé, on garde le comportement actuel.
- **Correction** : la publication d'un téléversement vers un nouveau chemin déclenche l'indexation, puis l'analyse selon le réglage.
- **Voix indisponibles** : la transcription est livrée ; les étapes vocales attendent avec leur cause (« aucun Worker voix prêt », « jeton absent ») et reprennent seules.
- **Suppression d'analyse** : `DELETE` d'étapes choisies (ou de toutes), réservé au Super-administrateur, refusé (409) pendant une analyse active, avec la cascade de l'ADR 0030. Un calcul préalable des pertes (segments, corrections, voix, voix validées, Personnes touchées) alimente la confirmation. Une trace d'audit est conservée au niveau du clip.
- **Pics** : le Worker envoie les pics par piste avec le résultat de la transcription ou de la diarisation ; le serveur les stocke et les sert avec la fiche. Un ancien Worker reste compatible.

## Critères d'acceptation

- Réglage actif : un ajout produit une analyse `full` ; réglage inactif : seulement la transcription (tests d'intégration PostgreSQL).
- Un téléversement publié vers un nouveau chemin crée le clip et lance l'analyse.
- Sans Worker voix, l'analyse montre la transcription faite et les étapes vocales en attente avec leur cause, puis se termine quand un Worker voix fictif arrive.
- Supprimer la diarisation d'un clip qui a une voix validée réussit, retire ses échantillons de la Référence et laisse la Personne ; une suppression hors de ce chemin reste refusée par le trigger.
- La cascade est respectée pour chaque étape (test par étape) ; 409 pendant une analyse active.
- Les pics sont stockés et renvoyés ; un résultat sans pics est accepté.

## Hors périmètre

Relance en masse des clips déjà transcrits ; spectrogramme.

## Tickets

1. **Analyse complète automatique et déclenchement à l'ajout** : réglage, déclencheurs, correction du téléversement, attente des étapes vocales. Aucun blocage.
2. **Suppression d'analyse par étapes** : API, calcul des pertes, cascade, levée contrôlée du verrou, audit. Aucun blocage.
3. **Pics de forme d'onde** : calcul côté Worker, contrat, stockage, lecture. Aucun blocage.
