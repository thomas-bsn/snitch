# Modèle vocal actif global, préparé par les Workers

Statut : accepté.

## Contexte et décision

Le modèle de diarisation et le modèle d'empreinte étaient des chaînes saisies dans l'administration, sans lien avec ce que les Workers avaient réellement chargé. Une case « accès au modèle confirmé » tenait lieu de vérification.

Snitch choisit un **modèle vocal actif** par type (diarisation, empreinte), commun à tous les Workers. Il est choisi dans un **catalogue de modèles vocaux** figé dans le code : dépôt, révision épinglée, moteur et adapter connus. Aucun dépôt saisi librement n'est exécuté.

Un changement passe par un **modèle candidat**. Chaque Worker récupère la demande par le même mécanisme de tirage que la configuration de transcription (ADR 0020), télécharge le modèle, le charge et réalise une inférence de test sur des données synthétiques, puis signale son état : non installé, téléchargement, vérification, prêt ou échec avec une cause. Un modèle simplement téléchargé n'est pas prêt. L'administrateur ne peut activer un candidat que si au moins un Worker l'a prêt.

Un Worker qui n'a pas le modèle actif prêt ne prend plus de travaux vocaux. Les travaux en cours gardent la version avec laquelle ils ont commencé. Les nouveaux travaux attendent un Worker prêt, et l'interface le signale. Les clips déjà traités ne sont jamais rediarisés automatiquement.

Le jeton Hugging Face reste sur la machine du Worker, dans un fichier secret lu seulement par le processus de téléchargement. Il ne transite jamais par Snitch et n'est jamais présent dans le processus d'inférence.

L'ancien modèle actif reste installé pour permettre un retour arrière ; rien n'est supprimé automatiquement.

## Alternatives écartées

- **Un modèle actif par Worker** : un même clip donnerait des résultats différents selon le Worker qui l'a traité.
- **Un champ libre pour un dépôt Hugging Face** : on retombe sur une chaîne qui ne prouve rien, et un dépôt arbitraire n'est pas exécutable sans adapter.
- **Un jeton saisi dans l'interface puis relayé par l'API** : le secret passerait par le serveur, alors que l'installation garantit qu'il n'est jamais envoyé à Snitch.
- **Un bouton d'installation par Worker séparé de l'activation** : il crée des états « installé mais jamais utilisé » sans intérêt, puisque le modèle actif est global.
- **Des boutons « Tester » et « Installer » séparés** : ils créent un état « téléchargé mais non vérifié » que l'interface devrait expliquer.

## Conséquences

- L'attribution des travaux vocaux doit vérifier l'état réel du modèle actif sur le Worker, et plus seulement les étapes annoncées.
- Un nouveau moteur demande un adapter, et éventuellement une image avec ses dépendances. En revanche, ajouter un modèle d'un moteur déjà pris en charge se fait sans reconstruire l'image.
- Le catalogue initial couvre, pour la diarisation, pyannote community-1 et NVIDIA Sortformer streaming v2.1, et, pour l'empreinte, wespeaker-voxceleb-resnet34-LM. Le catalogue est décrit comme des données (moteur, dépôt, révision) et non codé dans chaque adapter : un ajout depuis l'administration reste possible plus tard sans refonte. L'image du Worker contient les deux moteurs, et le surcoût de taille est accepté : une image séparée imposerait de réinstaller le Worker pour changer de moteur. Sortformer ne distingue pas plus de 4 voix ; le catalogue l'affiche, et Snitch ne bascule jamais automatiquement vers un autre modèle selon le clip.
- La case « accès au modèle confirmé » disparaît. Un refus d'accès à un dépôt gated devient une cause d'échec signalée par le Worker.
- Changer de modèle d'empreinte impose de recalculer les empreintes et les références. Ce cycle fera l'objet d'une décision séparée. D'ici là, le modèle d'empreinte actif est seulement affiché.
