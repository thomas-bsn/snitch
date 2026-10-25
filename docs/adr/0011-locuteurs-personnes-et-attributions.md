# Locuteurs locaux, Personnes et attributions

Statut : accepté pour la phase 5. Précise l'[ADR 0008](0008-identification-progressive.md) sans la contredire.

## Contexte

Un clip contient des voix dont l'identité est inconnue. La même personne apparaît dans de nombreux clips, plusieurs utilisateurs valident en parallèle, et une identification automatique peut se tromper. Il faut distinguer ce qui est observé, ce qu'un humain a confirmé et ce que Snitch déduit, sans jamais promouvoir une déduction en preuve.

## Décision

**Modèle.** Quatre notions distinctes, alignées sur le glossaire :

- le Locuteur local est une voix distinguée dans un clip, créée par la diarisation ; il peut rester sans identité ;
- la Personne est une identité humaine commune à plusieurs clips ; elle n'est jamais un compte, un lien facultatif et unique pouvant la relier à un compte ;
- l'Échantillon vocal validé est un extrait confirmé par un utilisateur ; l'ensemble de ceux d'une Personne constitue ses seules références ;
- l'Attribution relie un Locuteur local à une Personne avec une confiance, un niveau de décision (humain, automatique, suggestion, ambigu, inconnu) et un état de validation.

Un Locuteur local porte au plus une attribution validée par un humain et au plus une attribution machine. L'attribution effective est la validée si elle existe, sinon la machine, sinon « inconnu ». Les attributions machine sont des données dérivées, recalculées en base à partir des empreintes déjà extraites ; elles ne créent jamais d'Échantillon vocal validé et n'enrichissent donc pas les références. Seules une validation ou une correction humaine le font. Tout changement (validation, correction, rapprochement, fusion, séparation, suppression de profil) est consigné dans un historique immuable sans donnée vocale.

**Première validation.** La première validation d'un Locuteur local fait foi. L'arbitrage des validations concurrentes est confié à PostgreSQL, par un index unique partiel sur les attributions validées d'un Locuteur local (le même mécanisme existe pour la ligne machine) : la validation perdante est refusée, annule ses effets et reçoit l'attribution existante avec son auteur. Une correction explicite reste possible ensuite et exige que l'utilisateur ait vu l'attribution courante ; elle retire l'ancien échantillon des références et en crée un nouveau.

**Évaluation sur clips distincts.** La précision de l'automatisation se mesure en rejouant le rapprochement de production sur les échantillons validés, avec une règle de partage : un échantillon sert de référence ou de test, jamais des deux, et aucun test ne partage un clip avec une référence (le candidat concerné est écarté et compté). Seules les attributions qui seraient automatiques comptent pour la précision. Précision minimale et nombre minimal de confirmations sont configurables ; Snitch propose l'activation lorsqu'ils sont atteints et seul l'administrateur active ou désactive. La désactivation ramène les attributions automatiques à des suggestions.

**Confidentialité.** Le profil vocal d'une Personne peut être supprimé par son compte lié ou par un administrateur. Un marqueur sans donnée vocale empêche le rapprochement de le reconstituer.

## Alternatives écartées

- Verrou applicatif ou lecture avant écriture pour départager les validations : fragile entre plusieurs processus et requêtes ; la contrainte en base est la seule garantie qui ne dépend pas du code appelant.
- Dernière validation gagnante : une validation peut en écraser silencieusement une autre.
- Évaluer sur les clips des références : mesure optimiste, car le rapprochement retrouverait ses propres exemples.
- Promouvoir les attributions automatiques fortes en références : amplification de ses propres erreurs, contraire à l'ADR 0008.
- Identifier une Personne par un compte : la plupart des voix n'ont pas de compte.
- Réentraîner un modèle à chaque validation : non nécessaire, les références suffisent.

## Conséquences

- La contrainte PostgreSQL est l'arbitre des validations concurrentes ; le client doit gérer le refus et afficher l'auteur du gagnant.
- Une piste dont un Locuteur local est validé ne peut plus être rediarisée, afin qu'une validation humaine ne soit jamais perdue en cascade ; une validation erronée se corrige ou se retire par la suppression de Personne.
- Les seuils, critères et valeurs initiales (par exemple 95 % et 50 confirmations) sont des valeurs de départ, non des garanties ; le benchmark de phase 7 les ajuste. La mesure d'une phase 5 sur échantillons synthétiques ne prouve aucune qualité réelle.
- Le rapprochement actuel est un scan exact exécuté de façon synchrone ; son passage à l'échelle de la bibliothèque est à mesurer en phase 7.
- Une suppression de profil vocal n'a pas de levée : la rétablir serait une décision ultérieure.
