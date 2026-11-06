# Traitement vocal — interface Matrice des Workers

La variante B a été choisie par l’utilisateur parmi les huit propositions basées
sur `main` à `91bced1` (ticket #265, specs #303 et #312). Cette implémentation
reprend sa composition dans la véritable interface, avec les API existantes et
sans les commandes de laboratoire des prototypes.

## Parcours

- **Workers** : matrice avec sélection des machines, connexion, diarisation,
  empreinte et séparation. Les modèles prêts distinguent production et réserve.
  Révision, volume, progression, causes d’échec et retrait différé sont visibles.
  Installation, remplacement, nouvel essai, annulation et désinstallation restent
  des actions explicites ; rien ne se télécharge à l’ouverture de la page.
- **Catalogue** : cartes de références, recherche et filtre par type, détail et
  plan de déploiement. Le Super-administrateur peut ajouter une référence après
  inspection Hugging Face ou modifier nom, licence, taille, VRAM et disponibilité.
  Dépôt, révision, type et moteur restent issus de l’inspection. Les références
  incompatibles et désactivées ne peuvent pas être déployées.
- **Historique** : ordres de tous les Workers, ou d’une machine choisie, avec
  auteur, date, référence, révision et résultat. Les boutons de la page et de la
  matrice ouvrent aussi cet historique en panneau latéral.
- **Production** : un bandeau commun affiche les modèles actifs. Les réglages
  d’identification, les seuils, les mesures et l’accès HF s’ouvrent depuis deux
  cartes visibles sous le titre, avec l’état réel de l’attribution automatique. Les fonctions précédentes restent accessibles.

Les onglets ont une adresse partageable (`?view=catalogue`, `?view=history`) et
se parcourent avec les flèches, Début et Fin. La matrice défile horizontalement
sur téléphone sans élargir le document. Les libellés sont traduits en français
et en anglais ; les couleurs viennent des jetons Atelier clairs et sombres.

## Ordres et transition d’empreinte

Le plan reprend les Workers sélectionnés et, depuis une carte, le modèle choisi.
Il exclut les machines déjà prêtes pour cette cible et celles qui ont une
opération incompatible en cours. Un remplacement annonce l’ancien modèle qui
reste en service. Un aperçu affiche un ordre par Worker ; une confirmation
explicite précède l’envoi. Les résultats et les échecs sont affichés séparément,
avec des clés d’idempotence conservées lors d’une nouvelle tentative.

Le changement d’empreinte appelle les API de l’ADR 0044 : aperçu d’impact réel,
confirmation du nombre de références, préparation explicite des Workers,
progression, reprise et annulation avant bascule. Le serveur conserve la décision
de recalculer et de basculer ; aucun état de traitement n’est simulé par le front.
La séparation reste visible mais attend l’adaptateur de #268. Speaches reste
extérieur au catalogue des modèles locaux.

Une correction de lecture accompagne la refonte : le catalogue marque désormais
le modèle d’empreinte de `voice_active_model` comme actif. Auparavant seul le
modèle de diarisation recevait ce marqueur, ce qui présentait une empreinte de
production comme étant en réserve.

## Validation et limites

Les vérifications utilisent des fixtures synthétiques et une base PostgreSQL
isolée. Aucun média réel, aucun poids de modèle et aucun benchmark de phase 7
ne sont utilisés. Les tests couvrent les installations, les remplacements,
l’inspection HF simulée, l’édition persistante du catalogue, la sélection et la
prévisualisation des ordres, l’historique et la transition d’empreinte.

Le serveur de validation utilise le binaire de l’application et une base séparée,
avec quatre Workers synthétiques ; leurs pouls sont simulés, mais ils ne prennent
et n’exécutent aucun travail. Le reste de l’application conserve ses API réelles.
Le code est sur `feat/voice-processing-matrix` ; push et PR attendent la validation
visuelle de l’utilisateur. Aucun changement de contrat ou de migration n’est requis.

## Résultats du 6 novembre 2026

- `npm run lint` et `npm run build` : verts.
- `go vet ./internal/httpapi` et `go test ./internal/httpapi` : verts.
- 13 parcours Playwright contre Go/PostgreSQL : 7 pour l’administration vocale,
  5 pour l’inspection et la transition, 1 pour le remplacement. Le parcours
  mobile vérifie la matrice à 375 px avant d’ouvrir les réglages.
- 5 tests d’intégration de transition d’empreinte : verts, dont le marqueur actif
  du catalogue.
- Les trois vues à 1600 et 375 px, en clair et en sombre : 12 audits axe sans
  violation WCAG 2 A/AA, 2.1 AA ou 2.2 AA détectée et sans débordement du document.
  Contrôle complémentaire en paysage à 812 × 375 px, et des dialogues de plan
  et de confirmation sur téléphone.

Restant : validation visuelle de l’utilisateur avant push et ouverture de PR.
La validation avec de vrais modèles n’a pas été effectuée et ne fait pas partie
de cette refonte.

## Affinage visuel

Les accès Identification et Hugging Face sont placés avant les onglets. Les icônes
Lucide accompagnent les onglets, les types de modèles et les Workers. Le logo HF
est fourni localement, sans requête vers un tiers à l’ouverture de la page, depuis
les [ressources officielles Hugging Face](https://huggingface.co/brand).
Les retours au survol et à la sélection restent discrets ; une courte entrée anime
la matrice et le catalogue. `prefers-reduced-motion` désactive ces animations.
Les fenêtres et les confirmations gardent leurs parcours existants.

Vérification de cet affinage : compilation et formatage verts, 7 parcours
Playwright repassés, 12 audits clair/sombre et desktop/mobile sans violation,
paysage sans débordement. En mode mouvement réduit, animation et transition
calculées sont désactivées. Ouverture du panneau HF vérifiée sur le serveur local.
