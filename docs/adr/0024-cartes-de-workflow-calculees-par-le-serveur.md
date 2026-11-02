# 0024 — Cartes de workflow calculées par le serveur

## Contexte

La file de tâches calcule aujourd'hui qu'un clip est « terminé » quand tous ses derniers jobs existants sont faits, en SQL et une seconde fois dans le web. Un clip dont seule la transcription est planifiée passe donc pour terminé, un essai de diarisation en attente le garde actif, et la colonne Recherche n'a aucun job à montrer car l'indexation n'est pas une tâche de la file. L'orchestrateur et le flux temps réel de #115 (#130, #131) n'existent pas encore, et #117 demande un tableau par clip.

## Décision

- Le serveur renvoie des **cartes** déjà calculées (clip, colonne, état, étapes, date de fin) par un nouveau point d'accès réservé aux Administrateurs. Le web n'en dérive plus rien.
- **Traitement terminé** est dérivé des **étapes applicables** (pistes, indexation active, règles de l'ADR 0011), pas des jobs existants. Il n'est pas persisté. Completed est une vue calculée des trois clips les plus récemment terminés.
- Les essais de diarisation sont exclus des cartes et du calcul.
- La colonne Recherche lit l'état de l'indexation ; sans objet si elle est désactivée.
- « En attente de Worker » est dérivé à la lecture, sans nouvel état en base. Annulé et mort gardent le clip visible, jamais terminé.
- Aucune pagination des cartes actives, forme compacte ; la pagination ne porte que sur l'Historique du clip, par clip.
- Le rafraîchissement intermédiaire est un polling de 5 s suspendu quand l'onglet est caché ; le bouton Refresh ne disparaît qu'avec le flux temps réel de #131.
- L'ordonnanceur n'est pas modifié.

## Alternatives écartées

- Persister l'état « terminé » et sa date : il faudrait le tenir à jour à chaque changement et le migrer pour les clips existants.
- Garder la dérivation dans le web : elle reste dupliquée et diverge de la règle serveur.
- Attendre #130 et #131 : rien ne sortirait avant deux épopées non commencées.
- Une étape fictive « index » dans la file : elle ferait vivre deux sources d'état pour l'indexation.

## Conséquences

- #130 remplacera la dérivation par une entité d'exécution sans changer l'écran ni le contrat des cartes.
- Le calcul des étapes applicables doit rester aligné avec l'ADR 0011 ; un changement d'admissibilité touche la file.
- Le coût de la lecture croît avec le nombre de clips actifs ; la forme compacte le borne.
- L'ancienne liste de jobs et son agrégation restent pour les autres usages.
