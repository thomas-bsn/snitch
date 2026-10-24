# Fournisseurs interchangeables et verrou de traitement

Statut : accepté dans le plan de phase 0.

## Contexte et décision

Les endpoints et modèles futurs sont configurables ; le cœur ne présume aucun fournisseur. Ollama démarre sans modèle en phase 0. Aucun accès à la vraie bibliothèque avant validation du benchmark de phase 7.

## Alternatives écartées

Choisir maintenant un modèle définitif figerait une décision sans mesures sur le matériel cible.

## Conséquences

Les phases suivantes utilisent exclusivement des fixtures dédiées. Le benchmark comparera qualité, latence, RAM et VRAM avant un batch réel.
