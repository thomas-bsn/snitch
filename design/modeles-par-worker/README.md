# Référence visuelle : modèles vocaux par Worker (#265, spec #303)

Direction A (tableau dense, tokens du dépôt) choisie parmi cinq. **Référence visuelle seulement** : ne jamais copier ce HTML tel quel dans `web/`, ne pas le fusionner comme code de production (même règle que le prototype de la phase 6 bis).

- `prototype-a.html` : prototype interactif jetable, données synthétiques.
- `goal-a-light.png`, `goal-a-dark.png` : images cibles pour `/frontend-audit` (écran principal).
- `goal-a-add.png` : formulaire d'ajout d'un dépôt Hugging Face (scénario « repo connu »).

Décisions : la séparation est repliée dans un bandeau tant que #268 n'est pas livré ; l'implémentation (lot 6) reprend les composants existants de `web/src/app/admin/voice/`.
