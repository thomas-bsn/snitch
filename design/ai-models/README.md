# Référence visuelle : Admin → AI Models (#287), v2

Prototype jetable, données synthétiques. **Référence visuelle seulement** : ne jamais copier ce HTML tel quel dans `web/`, ne pas le fusionner comme code de production (même règle que `design/modeles-par-worker/`).

- `prototype.html` : s'ouvre dans un navigateur. Contrôles en haut : thème, état (normal, chargement, vide, erreur). Jouable : créer un corpus, choisir l'étape et les modèles ou chaînes, lancer (progression simulée, annulation confirmée), lire un résultat.
- `goal-*.png` : images cibles pour `/frontend-audit`.

## Décisions (v2)

- **Un corpus de clips annotés** (texte corrigé et voix validées par des humains) sert à toutes les étapes. Il est figé à la création et réutilise les corrections et validations existantes.
- **Results s'ouvre sur « Meilleur modèle par étape »** (dernier benchmark terminé de chaque étape, écart avec le deuxième, différence avec la production) ; avertissement si les corpus diffèrent.
- **Étapes, par importance :** Transcription (WER/CER), Diarisation (DER), Empreinte vocale (EER, précision, rappel), puis **Bout en bout** (chaînes complètes, « qui a dit quoi » correct). Chaque étape affiche ses propres métriques ; une étape non mesurable avec le corpus est désactivée avec la raison.
- **Empreinte :** les entrées amont sont la vérité terrain par défaut (modèle isolé), avec l'option de la diarisation de production.
- **Bout en bout :** 2 à 4 chaînes (un modèle par étape) ; la chaîne A reprend la production ; c'est une vérification, car le meilleur modèle de chaque étape n'est pas forcément la meilleure chaîne.
- Pas d'onglet Models : il dupliquait Voice sans rien permettre de faire. L'installation et l'activation restent dans Voice (un seul catalogue, #265) ; « prêt sur X Workers » s'affiche dans le choix des modèles. Aucun bouton « Activer en production ».
- Aucun gagnant global : seuils propres à chaque modèle, avertissement sur les petits échantillons, couverture et résultats partiels visibles, « temps écoulé » file d'attente comprise, RAM/VRAM masquées.
- Jetons `--series-1` à `--series-4` proposés pour les graphiques (clair et sombre), à passer au test de contraste.

## Écart avec le backend livré (PR #365)

Le backend actuel ne connaît que le jeu de « voix validées » et l'étape empreinte. Pour suivre ce design : généraliser le jeu de données en corpus de clips annotés, ajouter les étapes transcription, diarisation et bout en bout, une route listant les clips annotés candidats, et l'état « prêt sur au moins un Worker » par modèle.
