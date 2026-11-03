# Fiche vidéo refondue

Fait partie des évolutions des lots A à D (voir [ROADMAP](../../ROADMAP.md)). Référence visuelle : maquette C de la branche jetable [`prototype/fiche-video`](https://github.com/thomas-bsn/snitch/tree/prototype/fiche-video) (`/prototype/fiche-video?variant=C`), à réécrire proprement et jamais à fusionner.

## Problème

La fiche empile le lecteur, les métadonnées, la bannière de progression, l'analyse, la retranscription et les essais ; l'état de l'analyse ne bouge qu'en rechargeant. On ne peut ni écarter une fausse voix, ni marquer une voix volontairement non identifiée, ni dire « ce n'est pas cette personne » sans que la suggestion revienne.

## Solution

Lecteur et onglets Transcript / Voices côte à côte ; métadonnées repliables ; état de l'analyse sous forme de pastille vivante qui ouvre un panneau latéral ; Forme d'onde dans Voices ; Voix ignorée, Voix Random et Refus d'identification ([ADR 0031](../adr/0031-voix-ignoree-random-et-refus-d-identification.md)).

## Comportements

- **En-tête** : titre, fichier, propriétaire, date, durée, visibilité ; « Modifier les métadonnées » ouvre un panneau ; toutes les métadonnées (dossier, taille, formats, pistes, champs personnalisés) se déplient sous le lecteur.
- **Pastille d'analyse** dans l'en-tête, mise à jour en direct (en cours avec l'étape, en attente avec la cause, échec avec l'étape, complète, jamais analysée). Elle ouvre un panneau latéral avec :
  - les étapes en direct (statut, Worker, durée ; aucun pourcentage inventé) ;
  - pour une étape en échec : la cause, l'heure et les 50 dernières lignes de log (Administrateurs) ;
  - « Lancer l'analyse… » : choix d'étapes, raccourcis « Analyse complète » et « Seulement ce qui manque ». Les prérequis absents et les étapes dépendantes s'ajoutent automatiquement et visiblement. S'appuie sur les modes de réanalyse existants ;
  - « Relancer » l'étape en échec ; « Annuler la suite » pendant une analyse ;
  - « Supprimer… » (Super-administrateur) : choix d'étapes, cascade affichée, liste des pertes ; désactivé pendant une analyse.
- **Clip jamais analysé** (dont une Vidéo non indexée qui vient d'être indexée) : même fiche, sans transcription ni voix, avec le lancement en évidence.
- **Transcript** : écouter depuis un segment, corriger le texte (corrections existantes) ; libellé « Random » pour une Voix Random ; aucun locuteur pour une Voix ignorée.
- **Voices** : Forme d'onde colorée par voix, avec une piste par voix, cliquable pour se placer ; une pastille de couleur par voix ; par voix : « C'est X », « Ce n'est pas X » (Refus d'identification), Identifier (candidats, autres Personnes, nouvelle Personne), Random, Ignorer, Rétablir, écouter un extrait ; mention du Groupe de voix inconnues quand la voix y appartient. Le composant de décision sur une voix est unique et réutilisé par l'écran de traitement du lot C.

## Critères d'acceptation

- La pastille et le panneau changent d'état sans rechargement pendant une analyse simulée, y compris pour une erreur et une attente (parcours navigateur).
- Une Voix ignorée disparaît de la vérification, des groupes et du rapprochement, et ses segments n'ont plus de locuteur ; Rétablir la remet (tests API et navigateur).
- Après « Ce n'est pas X », un nouveau rapprochement ne propose plus X pour cette voix.
- Sans pics, Voices affiche les pistes sans tracé.
- Accessibilité WCAG 2.2 AA ; thèmes clair et sombre ; mobile.

## Dépendances

Lot A (logs de job, direct), lot B (suppression d'analyse, pics).

## Tickets

1. **Structure de la fiche** : en-tête, lecteur et onglets côte à côte, métadonnées repliables, panneau de modification. Aucun blocage.
2. **Panneau Analyse en direct** : pastille, étapes SSE, erreurs et logs, lancer, relancer, annuler, supprimer. Bloqué par 1, lot A ticket 5, lot B ticket 2.
3. **Voix ignorée, Voix Random et Refus d'identification** : modèle, API, effets sur vérification, groupes, rapprochement, statistiques et transcription. Aucun blocage.
4. **Onglet Voices et Transcript** : Forme d'onde, composant de décision partagé, libellés Random et ignorée. Bloqué par 1, 3, lot B ticket 3.
