# Audit des Workers — 8 novembre 2026

Périmètre : résolution et chargement vocal, catalogue, ordres persistants, remplacement,
contrats Go/TypeScript/Python, interface d'administration, baux des jobs, téléchargement
Hugging Face, GPU et superviseur de mise à jour. Audit du code et tests sur fixtures ;
aucune intervention sur les Workers installés, leurs modèles ou la bibliothèque réelle.

## Bugs avérés et corrections

| Gravité | Défaut | Correction et preuve |
| --- | --- | --- |
| Haute | NeMo/pyannote contrôlaient encore les chemins legacy et exigeaient l'embedder pour une diarisation. Des modèles validés dans le catalogue restaient inutilisables. | Résolution dépôt + SHA depuis le catalogue, contrôles propres à chaque étape. Tests `test_voice_catalog_resolution.py`. |
| Haute | Le job d'empreintes ne recevait pas sa référence catalogue ; le chemin et la version provenaient de l'environnement legacy, même pour un modèle candidat. | L'entrée du job transmet le modèle demandé, sinon l'actif, indépendamment du moteur de diarisation. Test d'intégration `voice_requested_model_dispatch` vérifiant le contenu loué. |
| Haute | Un embedder déjà chargé pouvait rester en cache après changement de référence. | Cache invalidé au changement de chemin épinglé ; test de deux révisions successives. |
| Haute | La réception d'un ordre et la fin d'un autre réécrivaient `orders.json` sans verrou commun : perte possible d'identité ou de résultat après redémarrage. | Verrou de lecture/modification/persistance partagé, sans verrouiller le téléchargement. Test concurrent déterministe et rejeu après redémarrage. |
| Haute | Le thread de file pouvait terminer pendant qu'un nouvel ordre était ajouté et laisser ce dernier sans exécuteur. | Publication de la fin du thread sous le verrou de la file. |
| Haute | L'état Failed pouvait bloquer tout autre remplacement, ou suggérer la suppression du prédécesseur pour libérer du disque. | Retarget autorisé uniquement si le successeur a échoué en préparation ; prédécesseur conservé, raccourci destructif retiré. Retry, changement et désinstallation disponibles selon l'état ; annulation avant bascule. Tests API et navigateur. |
| Moyenne | Annuler un remplacement échoué effaçait son inventaire alors que des fichiers partiels pouvaient subsister. | Conserver l'inventaire Failed pour une désinstallation explicite ; annuler la relation de remplacement sans toucher au prédécesseur. |
| Moyenne | Une suppression legacy pouvait annoncer Removed sans retirer les poids réellement utilisés, ou après une erreur de permissions. | Propagation du type, résolution exacte du legacy et contrôle de l'existence après retrait ; échec explicite si les fichiers persistent. |
| Moyenne | Une installation déjà validée pouvait être refusée pour manque d'espace avant le contrôle Ready. | Réutilisation avant estimation disque ; test avec espace insuffisant et zéro téléchargement. |
| Moyenne | La déclaration des modèles pouvait choisir arbitrairement la première révision installée. | Déclarations fondées sur les références actives complètes ; pas de déclaration d'une révision épinglée absente. Test avec deux révisions. |
| Moyenne | Une même clé d'idempotence pouvait être réutilisée pour une autre action ; certaines validations acceptaient un saut de ligne final. | Identité enrichie action/type/moteur côté Worker, action vérifiée côté serveur, correspondances intégrales. Compatibilité des journaux existants conservée. |
| Haute | Après la première mise à jour Docker par digest, le calcul du dépôt depuis `current%:*` refusait les cibles suivantes. | Extraction dépôt/tag/digest et reconnaissance d'un digest déjà installé pour éviter les redémarrages répétés. Tests shell avec Docker simulé, sans toucher au vrai Worker. |

## Transitions et compatibilité

- Installation : ordre explicite → téléchargement → vérification synthétique → Ready.
  Un modèle déjà validé est réutilisé, sans téléchargement. Aucun accès réseau ajouté à l'inférence.
- Remplacement : ancien Ready → préparation du nouveau → bascule après validation → retrait
  ordonné par Snitch. Si la préparation échoue, l'ancien reste utilisable ; aucun retour
  arrière destructif n'est nécessaire puisqu'il n'a pas été retiré.
- Failed pendant la préparation : Retry, choix d'un autre modèle compatible, Cancel ou
  Uninstall du candidat. Une transition d'empreinte conserve ses protections métier.
- Après bascule, l'interface n'offre plus une annulation que le serveur refuserait.
- Les chemins legacy restent admis pour une référence identifiée par le manifeste,
  et pour les anciennes configurations sans référence SHA. Une référence catalogue
  manquante ne bascule jamais sur un modèle legacy sans rapport.
- Le contrat ajoute des champs facultatifs ; aucune migration SQL, suppression de volume,
  réinitialisation ou installation de modèle n'est nécessaire pour appliquer ce code.

## Vérifications transversales

Les contrôles existants restent en place : compatibilité type/moteur à la réception,
révisions SHA, jeton HF dans un fichier privé et réservé au sous-processus de téléchargement,
inférence hors ligne, erreurs structurées sans texte brut des bibliothèques, baux et
abandon des résultats après perte de bail, retries bornés avec temporisation et relance des
threads de claim, sélection explicite CPU/CUDA, validation du digest et rollback du superviseur.
Les tests de ces domaines font partie des suites existantes ; cela ne constitue pas une
certification de sécurité exhaustive ni un test du matériel de production.

## Validation

- Worker : 341 tests exécutés, aucun échec, 21 ignorés (dépendances/runtime optionnels).
- Intégration catalogue/ordres/remplacement/transitions d'empreinte : suite ciblée verte,
  PostgreSQL éphémère, détection des courses Go activée.
- Entrée d'empreintes : modèle actif et modèle demandé contrôlés via l'API d'un job loué.
- Mise à jour : tests avec commandes Docker simulées (digest courant et évolution depuis un digest).
- TypeScript et build Vite verts ; contrats régénérés et reproductibilité vérifiée.
- E2E : 9 scénarios verts d'administration vocale et remplacement sur Workers simulés,
  dont échec disque, maintien du prédécesseur, annulation, bascule/retrait après validation
  et remplacement d'un modèle échoué sans prédécesseur.
- Détecteur Impeccable : aucun résultat sur les deux composants modifiés.

Le résultat final de la suite Go complète est consigné dans la PR.

## Risques et limites restants

1. **Matériel/runtime (moyen)** : pas de GPU NVIDIA disponible dans cet environnement macOS.
   Le chargement réel de Sortformer/WeSpeaker, la VRAM et les performances CUDA doivent être
   confirmés sur le Worker cible après déploiement ; les fixtures ne prouvent pas cela.
2. **Intégrité du volume (moyen)** : Ready repose sur le marqueur de validation existant,
   pas sur une vérification cryptographique répétée de tous les poids. Une modification
   externe du volume après validation peut toujours provoquer un échec de chargement.
3. **Exclusivité (moyen)** : le verrou d'ordres protège un processus. Deux instances partageant
   simultanément le même volume et la même identité Worker restent hors contrat ; aucun
   verrou interprocessus ni migration d'identité introduit ici.
4. **Annulation et fichiers partiels (faible)** : Cancel interrompt la relation de remplacement
   échouée, sans effacer ses fichiers ; Uninstall déclenche le nettoyage explicite. Un
   téléchargement en cours n'est pas interrompu brutalement. L'inventaire conserve l'échec,
   mais la matrice affiche prioritairement le prédécesseur prêt après annulation.
5. **Legacy (faible)** : les anciens journaux d'ordres ne contiennent pas action/type/moteur.
   Leur identité historique reste acceptée pour préserver le rejeu sans réinstallation.
6. **Mise à jour (faible)** : une image précédemment lancée par digest ne fournit pas sa
   version lisible ; le rapport de rollback emploie la version tentée avec le digest restauré.
   Les pulls de métadonnées Docker peuvent se répéter ; ils ne téléchargent aucun modèle vocal.
7. Le build frontend conserve son avertissement préexistant de taille de bundle ; aucune
   refonte du découpage frontend n'est incluse dans cet audit.
