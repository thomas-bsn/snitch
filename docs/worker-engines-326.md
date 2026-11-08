# #326 — bilan de la première tranche

Cette tranche ajoute le registre signé, le provisionnement isolé sur ordre et le branchement des adaptateurs d'empreintes. Elle ne valide pas l'ensemble de l'epic. Voir [ADR 0055](adr/0055-runtimes-vocaux-approuves.md) pour le protocole, la confiance et les limites opérationnelles.

| Défaut / risque | Gravité | Correction ou limite |
| --- | --- | --- |
| TitaNet/ECAPA refusés par la matrice et les contrats de jobs | Haute | Familles explicitement reconnues, enums générés et sélection du moteur d'empreintes indépendante de la diarisation. |
| Collision des dépendances ML dans le Python du Worker | Haute | Environnement isolé par manifeste, sous-processus d'inférence, installation hors index et sans résolution implicite. |
| Exécution de code arbitraire HF / installation non épinglée | Critique | Adaptateurs fixes, métadonnées strictes, configurations filtrées, chargement des poids sécurisé, roues signées et contrôlées. L'éditeur des roues reste une autorité de confiance. |
| Changement de version entre mise en file et exécution | Haute | Manifeste figé dans l'ordre et contrôle de son idempotence. |
| Échec de remplacement détruisant un runtime fonctionnel | Haute | Répertoires distincts, validation avant liaison et conservation des anciennes versions. |
| Modèle marqué Ready avec runtime absent lors d'une préparation | Haute | Refus explicite sans téléchargement pendant l'inférence ou une simple réconciliation. La synchronisation contrôle aussi la présence du runtime des modèles déjà déclarés Ready. |
| Mauvaise version Python dans le contrat initial | Haute | Python 3.12 aligné sur l'image Docker, tests de compatibilité. |
| Révocation sans effet sur les livraisons en attente | Haute | Annulation transactionnelle des ordres non acquittés ; drainage des ordres acquittés. |

## Exploitation

1. Configurer les clés publiques de l'éditeur dans `SNITCH_ENGINE_TRUST_KEYS` côté serveur. Garder les clés privées hors de Snitch.
2. Produire les manifestes avec des roues exactes, incluant toutes les dépendances et un budget disque couvrant téléchargement et décompression. Signer les octets JSON exacts avec Ed25519.
3. Dans Administration → Voix, importer l'enveloppe signée. La liste affiche version, contraintes et licence.
3 bis. Dans l'onglet Engines, choisir un Worker, « Ajouter un engine », cocher une version approuvée et confirmer le plan chiffré : l'ordre installe le runtime seul, sans poids. L'état de chaque Worker (demandé, téléchargement, installation, vérification, prêt, échec) et son historique y sont visibles.
4. Choisir le modèle, les Workers et la version du runtime dans l'installation existante. L'ordre est explicite ; aucun modèle n'est téléchargé par l'import du manifeste.
5. En cas d'échec, conserver le modèle actif précédent et consulter la cause de l'ordre. Réessayer via les actions existantes. Révoquer une release empêche les nouvelles installations ; cela ne désinstalle pas les environnements utilisés.

Aucune clé de confiance, release signée, réinitialisation de Worker ou suppression de modèle existant n'est introduite automatiquement.

## Validation et restant

Les tests hors réseau couvrent signatures, permissions, révocation, persistance des manifestes, versions concurrentes, intégrité, budgets, environnement sans secrets, absence de téléchargement implicite et compatibilité legacy. Validation locale : tests Go/DB ciblés avec détecteur de races, 352 tests Worker (21 ignorés), build TypeScript, lint et reproductibilité des fichiers générés réussis. La suite Go complète a été lancée puis interrompue après plus de dix minutes, pendant les calculs Argon2 du test existant `delete_clip` ; elle n'est pas déclarée verte. La CI doit confirmer la suite complète. Les E2E locaux sont volontairement omis à la demande de l'utilisateur et laissés à la CI GitHub.

Livré par #349 : ordre d'installation d'un engine indépendant des poids, inventaire par Worker (états, progression, causes, historique, générations) et onglet Administration → Voix → Engines. Restent nécessaires pour fermer #326 : annulation, remplacement, retour arrière et retrait des engines, dispatch tenant compte de leur inventaire, nettoyage avec drainage, mises à jour des runtimes de modèles Ready, releases publiées et vraies inférences TitaNet/ECAPA, tests Docker/WSL2/GPU et benchmark autorisé. Aucun scan de bibliothèque réelle n'a été effectué.
