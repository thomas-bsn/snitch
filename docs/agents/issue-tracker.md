# Issue tracker

Issues et specs : https://github.com/thomas-bsn/snitch.
Utiliser gh avec --repo thomas-bsn/snitch explicitement, même si le remote origin est configuré, afin de désigner le dépôt sans ambiguïté.

Publier une spec signifie créer une issue GitHub.
Pour les textes multilignes, utiliser --body-file.

## Opérations

- Créer : gh issue create --repo thomas-bsn/snitch --title "..." --body-file <fichier> --label <label>.
- Lire : gh issue view <numéro> --repo thomas-bsn/snitch --json number,title,body,labels,comments.
- Lister : gh issue list --repo thomas-bsn/snitch --state open --json number,title,body,labels.
- Commenter : gh issue comment <numéro> --repo thomas-bsn/snitch --body-file <fichier>.
- Modifier les labels : gh issue edit <numéro> --repo thomas-bsn/snitch --add-label <label> ou --remove-label <label>.
- Fermer : gh issue close <numéro> --repo thomas-bsn/snitch.

Les PR externes ne sont pas une surface de triage.

La publication de la spec et le label ready-for-agent ne déclenchent pas la reprise du code : la phase 0 attend la validation de sa spec et le feu vert utilisateur.
