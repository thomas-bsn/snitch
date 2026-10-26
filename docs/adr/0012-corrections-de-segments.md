# Corrections des segments transcrits

Statut : accepté pour la phase 6. S'appuie sur l'[ADR 0011](0011-locuteurs-personnes-et-attributions.md) sans le modifier.

## Contexte

Une transcription automatique se trompe, et le locuteur déduit d'un segment aussi. Corriger à la main ne doit ni détruire ce que le modèle avait produit (la phase 7 mesurera l'erreur par rapport à des textes vérifiés), ni laisser deux utilisateurs s'écraser en silence, ni imposer de retranscrire tout un clip pour réparer une phrase, ni perdre un travail humain lors d'une retranscription.

## Décision

**Couche de correction distincte.** La transcription d'origine du worker n'est jamais modifiée. Une correction est une couche séparée qui recouvre le texte (et le locuteur) d'un segment ; l'affichage, la recherche et la réindexation lisent le texte effectif, c'est-à-dire la correction si elle existe, sinon l'original. Seul le texte et le locuteur sont corrigeables : les débuts, les fins et le nombre de segments ne changent pas. Une correction de locuteur ne touche ni les tours de parole, ni les empreintes, ni les attributions, ni les échantillons vocaux validés.

**Historique immuable.** Chaque correction, restauration ou changement de locuteur ajoute une entrée (ancienne valeur, nouvelle valeur, auteur, date, motif facultatif) qui ne peut ni être modifiée ni être supprimée ; seule la suppression du clip l'efface. Revenir en arrière est une nouvelle entrée, jamais une réécriture du passé.

**Concurrence optimiste.** Toute écriture sur un segment porte la version que l'utilisateur a vue. Si la version courante a changé, l'écriture est refusée avec l'état courant et son auteur, et le brouillon reste à l'utilisateur, qui choisit de le reprendre par-dessus ou de l'abandonner. Ce choix diffère volontairement de « la première validation fait foi » des voix (ADR 0011) : valider une voix est un geste d'identification où la première affirmation vaut jusqu'à correction explicite, alors qu'un texte se réécrit légitimement et plusieurs fois ; on empêche seulement qu'une correction en remplace une autre sans que son auteur l'ait vue. La contrainte reste arbitrée par PostgreSQL (verrou de la ligne du segment, insertion unique pour un segment jamais corrigé), pas par le code appelant.

**Réindexation ciblée.** Une correction de texte ne recalcule que le vecteur de ce segment, pour l'index actif et pour l'index candidat d'une reconstruction en cours, sans retranscrire ni retraiter la vidéo. La correction est enregistrée même si le service d'embeddings est indisponible ; le segment reste alors en attente et est repris plus tard avec un nombre borné de tentatives, visibles de l'administrateur. La recherche lexicale voit la correction immédiatement. Une correction de locuteur ne réindexe rien.

**Politique de retranscription.** Retranscrire un clip qui porte des corrections actives exige une confirmation explicite de l'administrateur, sur tous les chemins qui planifient une retranscription. Les lots excluent par défaut les clips corrigés, qui n'y entrent que sur demande explicite, et un changement automatique de règle de piste les ignore. Quand la nouvelle transcription aboutit, les corrections de l'ancienne deviennent obsolètes : elles restent dans l'historique mais ne s'appliquent plus, et ne sont jamais réappliquées au nouveau texte, même identique. Tant que la nouvelle transcription n'a pas réussi, le texte actuel et ses corrections restent actifs.

## Alternatives écartées

- Modifier la transcription d'origine : perd la référence nécessaire à la mesure de qualité et rend toute retranscription destructrice.
- Dernière écriture gagnante : une correction en écraserait une autre sans qu'on le voie.
- Première correction gagnante, comme pour les voix : un texte erroné corrigé une fois ne pourrait plus être amélioré sans geste d'exception.
- Verrou applicatif ou lecture avant écriture : fragile entre plusieurs processus.
- Réappliquer les corrections au nouveau texte après une retranscription : les segments ne correspondent plus nécessairement, une correction serait posée sur un texte qu'elle n'a pas relu.
- Réindexer tout le clip ou attendre une reconstruction globale : coût sans rapport avec une phrase corrigée.
- Réaligner les mots d'un texte corrigé : fausse précision ; le texte corrigé fait foi sans horodatage propre par mot.

## Conséquences

- Les corrections sont rattachées à une transcription précise ; une retranscription réussie les rend obsolètes sans les supprimer. Supprimer un clip efface ses corrections et leur historique, jamais le fichier vidéo ; supprimer une Personne ou un profil vocal ne touche pas aux corrections de texte.
- Les mots d'un segment corrigé sont datés au début du segment ; l'ouverture au timestamp y renvoie.
- Tant que la réindexation d'un segment corrigé est en attente, la recherche sémantique ne le propose pas, pour ne pas ressortir l'ancien texte par son ancien vecteur ; la recherche lexicale le trouve tout de suite.
- Snitch ne calcule aucun taux d'erreur : il exporte les paires texte d'origine / texte corrigé pour la mesure de la phase 7.
- Aucune correction n'a été éprouvée sur de vraies transcriptions ; la qualité et l'usage réels restent à observer.
