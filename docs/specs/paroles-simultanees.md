# Attribution des paroles au mot et séparation mesurée

Seconde des deux specs issues de [#268](https://github.com/thomas-bsn/snitch/issues/268) (Pipeline V2). La première est [Empreintes et reconnaissance sur voix brèves](voix-breves.md) ; cette spec en dépend pour sa partie séparation. Issue : [#388](https://github.com/thomas-bsn/snitch/issues/388).

## Problème

Sur des clips de discussion vocale, les échanges sont rapides et se recouvrent. Deux défauts distincts en résultent.

- **Un segment n'a qu'un locuteur.** La transcription fournit déjà des mots horodatés, mais chaque segment reçoit en entier le Locuteur local qui y parle le plus (`voice.go`, `voice_coverage.go`). Quand deux personnes se répondent dans le même segment, les mots de la seconde sont donnés à la première. C'est un défaut de code.
- **Des paroles en Chevauchement ne sont pas transcrites.** La diarisation détecte trois Locuteurs locaux, la transcription n'en restitue que deux : le troisième parle sous les autres et reste sans paroles. Aucun réglage du code ne retrouve des mots que le modèle de transcription n'a pas produits. C'est un défaut de modèle, que la séparation des voix peut réduire. Rien ne prouve encore qu'elle le fait sur ces clips.

## Objectif

Attribuer chaque mot au bon Locuteur local, et savoir par la mesure si la séparation retrouve des paroles perdues en Chevauchement, avant de la brancher sur la bibliothèque.

## Solution

### 1. Attribution au mot

- Chaque mot reçoit le Locuteur local dont les tours le recouvrent le plus, sur la piste du mot. À égalité, il garde le locuteur du mot précédent. Un mot qu'aucun tour ne recouvre prend celui de son voisin dans le segment ; un segment sans aucun tour reste sans locuteur, comme aujourd'hui.
- Une voix qui ne tiendrait qu'un mot isolé entre deux mots d'une même autre voix est rattachée à celle-ci : les horodatages de mots sont approximatifs et ne justifient pas une telle bascule.
- Le segment n'est pas redécoupé. Il est rendu comme une suite de **parts**, chacune avec son locuteur. Le nombre de segments, leurs bornes et leur identité ne changent pas, donc les Corrections et leur historique restent valides (ADR 0012).
- Une Correction de locuteur continue de s'appliquer au segment entier. Un segment dont le texte est corrigé n'a plus de mots horodatés fiables : il garde un seul locuteur, celui de la règle actuelle.
- La Couverture d'un Locuteur local est calculée sur les mots : une voix à qui des mots reviennent n'est plus comptée comme parlant sous les paroles des autres.
- Le filtre de recherche par Personne retient un segment dès qu'une de ses parts lui est attribuée.
- La règle de fusion est versionnée et s'applique à la lecture : aucune donnée n'est réécrite, et revenir à la règle par segment est un réglage.

Le benchmark de bout en bout (ADR 0060) mesure le cpWER avec la règle par segment et avec la règle au mot, par la même fonction que la production.

### 2. Séparation des voix : adaptateur

- Un adaptateur Worker de séparation est ajouté pour la famille d'engine qui la porte déjà (SpeechBrain). Le Catalogue de modèles vocaux peut alors déclarer un modèle de séparation supporté ; aujourd'hui une contrainte de la base et l'adaptateur le refusent.
- Les règles des autres modèles s'appliquent sans exception : ordre explicite de l'administrateur, aucun téléchargement implicite, inférence de test obligatoire avant l'état prêt, contrôle des ressources avant installation.
- Le choix des modèles candidats se fait au ticket, d'après leur documentation d'origine : fréquence d'échantillonnage, nombre de voix séparées, données d'entraînement. Les modèles publics sont surtout entraînés sur de la lecture propre mélangée artificiellement ; un clip de jeu compressé avec du son d'ambiance en est loin. Un résultat médiocre est une issue attendue, pas un échec de la spec.

### 3. Séparation des voix : mesure

La séparation n'existe d'abord que comme étape de benchmark. Rien de ce qu'elle produit n'atteint la bibliothèque.

- **Zones ciblées.** Seuls les Chevauchements d'une piste où un Locuteur local reste sans paroles sont traités, avec une marge de contexte. Une piste sans Chevauchement n'appelle jamais le modèle.
- **Transcription des sources.** Chaque Source séparée est transcrite par le service de transcription externe. Les horodatages sont ramenés au repère du clip.
- **Attribution d'une source.** Une Source séparée est rapprochée des Locuteurs locaux du même clip par ressemblance avec leur empreinte de Parole nette. Sans écart net entre les candidats, la source reste sans locuteur. Cette comparaison ne sort pas du clip et n'est pas conservée.
- **Fusion prudente.** Les mots de la transcription d'origine sont gardés. Seuls s'ajoutent les mots d'une Source séparée attribuée à un Locuteur local qui n'en avait aucun dans la zone.
- **Mesures**, sur les clips certifiés du corpus qui contiennent un Chevauchement, avec et sans séparation :
  - cpWER et WER ;
  - Locuteurs locaux sans paroles avant et après ;
  - mots ajoutés absents de la référence, et mots dupliqués entre deux sources ;
  - sources restées sans locuteur ;
  - durée d'inférence et pic de mémoire GPU, que le Worker ne remonte pas aujourd'hui.
- Le rapport du run dit si la séparation retrouve des paroles, à quel prix en erreurs ajoutées et en ressources.

Une Source séparée ne sert jamais à calculer une empreinte stockée, ni à enrichir une Référence.

## Hors périmètre

- **Le branchement de la séparation dans l'Analyse de clip.** Il fera l'objet d'une spec propre si la mesure montre un gain. Écrire ce branchement avant la mesure serait du code pour un modèle non prouvé.
- Le choix du modèle de transcription. Le benchmark de transcription existant (ADR 0058) sert à cela ; cette spec ne le change pas.
- Un modèle d'alignement de mots dédié, un détecteur de Chevauchement dédié, l'extraction d'une voix cible : aucun besoin démontré.
- Le redécoupage des segments et la reprise des Corrections sur un nouveau texte.

## Données

Développement et tests sur données synthétiques. Les runs de benchmark sont lancés par l'administrateur sur son instance ; aucun agent ne les lance sur la vraie bibliothèque.

## Critères d'acceptation

- Un segment où deux Locuteurs locaux se répondent est rendu en deux parts, chacune avec ses mots ; le nombre et les bornes des segments sont inchangés (test API).
- Un mot isolé ne crée pas de part à lui seul entre deux mots d'une même autre voix.
- Une Correction de locuteur s'applique au segment entier ; un segment au texte corrigé n'a qu'un locuteur ; aucune Correction existante n'est perdue ni déplacée.
- La Couverture d'un Locuteur local et le filtre par Personne suivent les parts.
- Le benchmark de bout en bout rapporte le cpWER des deux règles ; production et benchmark donnent la même attribution pour les mêmes entrées (test de non-dérive).
- Un modèle de séparation ne s'installe que sur ordre, passe une inférence de test, et son absence donne un état explicite sans téléchargement.
- Une piste sans Chevauchement ne déclenche aucune séparation (test Worker).
- Un run avec séparation n'écrit ni dans les transcriptions, ni dans les empreintes, ni dans les Références de la bibliothèque.
- Une Source séparée sans locuteur net n'ajoute aucun mot.
- Le rapport d'un run affiche les mesures listées, durée et pic de mémoire GPU compris ; un manque de mémoire GPU fait échouer la tâche avec une cause stable, sans nouvelle tentative en boucle.

## Tickets

Une seule PR, dans cet ordre.

1. **Attribution des paroles au mot et parts d'un segment** (#398). Aucun blocage.
2. **Parts d'un segment dans la fiche d'un clip** (#399). Bloqué par #398.
3. **cpWER par segment et au mot dans le benchmark de bout en bout** (#400). Bloqué par #398.
4. **Adaptateur de séparation des voix et modèle supporté au catalogue** (#401). Aucun blocage.
5. **Étape de benchmark avec séparation des Chevauchements** (#402). Bloqué par #401 et par #389 (Parole nette, première spec).
6. **Mesures et rapport de la séparation, avec durée et mémoire GPU** (#403). Bloqué par #402.
