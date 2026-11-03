# 0035 — Analyse complète automatique à l'ajout

Statut : accepté. Modifie l'[ADR 0024](0024-analyse-de-clip-durable.md) (analyse lancée par un administrateur) pour les nouveaux ajouts.

## Contexte

À l'ajout d'une vidéo, seule la transcription est planifiée ; la détection des voix doit être lancée à la main. Un fichier téléversé vers un nouveau chemin ne déclenche même aucun scan.

## Décision

- Un réglage d'administration, **Analyse complète automatique**, actif par défaut : chaque nouvel ajout (vidéo prête dans la Racine vidéo, publication d'un téléversement, sélection depuis l'import) démarre une Analyse de clip de périmètre `full`. Désactivé, le comportement reste celui d'aujourd'hui.
- Si les voix ne peuvent pas être traitées (aucun Worker voix prêt, pas de jeton), la transcription est livrée et les étapes vocales attendent avec leur cause, puis reprennent seules dès qu'un Worker compatible est prêt. Elles n'échouent pas.
- Le réglage ne s'applique qu'aux nouveaux ajouts ; les clips déjà transcrits ne sont pas relancés en masse.

## Alternatives écartées

- **Désactivé par défaut** : contraire au comportement attendu par l'utilisateur.
- **Échec des étapes vocales sans Worker voix** : obligerait à tout relancer à la main.

## Conséquences

- La charge GPU augmente à chaque import ; la file de tâches reste le point de contrôle (annulation, priorités existantes).
