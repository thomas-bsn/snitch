# Prototype jetable : engines par Worker (#326, spec #348)

**Référence visuelle seulement** : ne jamais copier ce HTML tel quel dans `web/`, ne pas le fusionner dans `main`.

Question posée : comment montrer l'état de chaque version d'engine sur chaque Worker, et les actions possibles (Installer, Annuler, Réessayer, Remplacer, Mettre à jour, Revenir, Désinstaller), avec la raison exacte de chaque refus ?

Lancer : ouvrir `prototype.html` dans un navigateur (aucune dépendance, données synthétiques en mémoire). Changer de variante avec les flèches du bas ou `?variant=A|B|C`.

- **A** matrice versions × Workers.
- **B** Worker par Worker.
- **C** version par version, avec stepper et raisons de refus en clair.

Le simulateur en haut fait avancer les opérations, injecte des échecs et recharge trois scénarios (dérive entre Workers, tout vide, pannes).

Cas couverts : Worker hors ligne (ordre `requested` livré à la reconnexion), Worker sans GPU, VRAM insuffisante, version révoquée (installation et retour arrière refusés), désinstallation refusée si un modèle est lié, drainage si des jobs tournent, mise à jour avec bascule de la liaison seulement après inférence réussie, nettoyage des fichiers incomplets, remplacement depuis `Failed`.

## v2 : tenue à l'échelle (`prototype-echelle.html`)

Les variantes A à C de la v1 montrent toutes les cellules : inutilisables à 40 engines × 30 Workers. La v2 génère cette flotte (1 200 combinaisons) et teste trois façons de la rendre lisible, avec des actions en lot dont les refus sont regroupés par raison.

- **D** À traiter : seules les exceptions sont affichées, regroupées par cause (échecs, en cours, versions révoquées encore liées, ordres en attente).
- **E** Carte de la flotte : 1 case = 1 version sur 1 Worker, recherche, filtre de groupe, sélection de cellules, de lignes ou de groupes, panneau d'actions en lot.
- **F** Déploiement par version : une ligne par version avec barre de santé, déploiement par cible (tous, Worker pilote, groupe), seuls les Workers à problème sont listés.

## v3 : Worker d'abord (`prototype-worker.html`)

Choix retenu après la v1 et la v2 : la structure maître/détail de la variante C, avec les Workers en maître. On clique un Worker, on coche des engines dans son détail, on applique l'installation ou la mise à jour.

- Une ligne par famille d'engine (14) et non par version (40) ; l'action découle de l'état (absent : installer, ancien : mettre à jour, échec : réessayer).
- Le plan chiffre le disque cumulé et regroupe les refus par raison avant d'appliquer ; la portée peut être ce Worker, les Workers cochés ou tout le groupe.

## v4 : où l'intégrer dans Administration → Voix (`prototype-integration.html`)

L'en-tête, la bande de production, les onglets et la matrice imitent l'onglet Voix réel (mêmes jetons de couleur, clair et sombre). Trois emplacements possibles pour la gestion des engines, avec `?variant=A|B|C` :

- **A** sous-onglet « Engines » (maître/détail Worker d'abord).
- **B** bouton « Engines » par ligne de la matrice, qui ouvre un tiroir.
- **C** ligne de Worker dépliable dans la matrice (puces d'état et actions rapides).

Un essai en React avec faux backend a été abandonné (trop long pour la question posée) ; ses fichiers ne sont pas dans le dépôt.

Non couvert : droits (Super-administrateur et Administrateur), historique d'ordres, i18n.
