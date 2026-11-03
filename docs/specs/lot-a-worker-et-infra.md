# Lot A — Worker et infrastructure

Fait partie des évolutions des lots A à D (voir [ROADMAP](../../ROADMAP.md)). Ce n'est pas la phase 7, qui garde son autorisation propre ; données synthétiques uniquement.

## Problème

Un Worker passe Offline pendant un job long, alors qu'il travaille. Une perte de lease passe inaperçue jusqu'à la fin du job, ce qui permet un double traitement. Le téléchargement des modèles vocaux échoue à l'installation (`hf_access_denied`, #203). Le jeton Hugging Face doit être redéposé sur chaque Worker recréé (#194). On ne sait pas si le serveur de transcription est joignable, ni pourquoi il ne l'est pas. Les pages Workers et l'état d'un job ne se mettent à jour qu'en rechargeant, et les logs du Worker restent invisibles.

## Solution

Présence indépendante et perte de lease traitée tout de suite ([ADR 0029](../adr/0029-presence-du-worker-et-perte-de-lease.md)) ; installation des modèles réparée ; Token Hugging Face central ([ADR 0027](../adr/0027-token-hugging-face-central.md)) ; test de joignabilité de la transcription par Worker ; lignes de log des jobs visibles ([ADR 0028](../adr/0028-logs-de-job-visibles-et-extrait-d-erreur.md)) ; pages Workers en direct.

## Comportements

- **Présence** : pulse toutes les 10 s depuis un thread dédié ; seuil Online inchangé (30 s) ; réessai avec délai croissant plafonné à 10 s ; reconnexion automatique après une coupure réseau.
- **Lease** : erreur réseau → réessai jusqu'à l'expiration ; refus du serveur → annulation immédiate du job sans résultat ; lease expiré à la reconnexion → résultat jeté.
- **#203** : avec un jeton valide et les dépôts gated acceptés, `y` puis `accept` installent et valident les modèles, et le Worker démarre « voix prête ». Les causes d'échec sont distinctes : jeton absent ou invalide, accès gated refusé, dépôt ou révision invalide, réseau, stockage. Un échec des modèles ne bloque jamais l'enregistrement du Worker ni la transcription. Le résultat est le même qu'avec un téléchargement lancé depuis Voice Processing.
- **#194** : saisie par le Super-administrateur dans Voice Processing et dans une étape facultative du wizard ; stockage chiffré dans `settings` ; jamais réaffiché. Distribution versionnée avec accusé de réception ; provenance retenue par le Worker (un jeton local n'est jamais remplacé en silence) ; état de synchronisation par Worker.
- **Joignabilité de la transcription** : chaque Worker teste `GET /v1/models` toutes les 60 s, de façon asynchrone par rapport au pulse, et transmet le dernier résultat dans son pulse. L'onglet Transcription affiche par Worker « joignable / injoignable » et le détail du dernier échec : méthode, URL sans identifiants, statut HTTP, latence, type d'erreur, extrait borné de la réponse, heure. Le résultat vaut « depuis ce Worker ».
- **Logs de job** : lignes envoyées par lots (2 s ou 50 lignes), nettoyées ; 50 dernières lignes conservées par job ; diffusées par SSE ; supprimées avec le job ; réservées aux Administrateurs.
- **Direct** : la liste des Workers et la fiche d'un Worker se mettent à jour par le flux d'événements existant, sans rechargement.

## Critères d'acceptation

- Un job simulé de 5 minutes laisse le Worker Online tout du long ; une coupure réseau simulée de 40 s le fait passer Offline, puis Online à la reconnexion, sans redémarrage.
- Un refus de renouvellement de lease annule le job en moins de 10 s et n'envoie aucun résultat.
- #203 : tests du lanceur et du provisioning avec fournisseurs simulés, plus une validation réelle faite par l'utilisateur (la PR reste en brouillon jusque-là).
- #194 : aucune lecture d'API ne renvoie le jeton ; un Worker révoqué n'obtient rien ; un jeton local n'est pas remplacé ; tests API, Worker et E2E pour les cas sans jeton, jeton invalide, Worker révoqué et panne réseau.
- Les lignes de log conservées ne contiennent ni jeton, ni Bearer, ni URL avec identifiants, ni chemin de clip (test).
- Les pages Workers changent d'état sans rechargement (parcours navigateur).

## Hors périmètre

Logs internes du serveur de transcription ; pourcentage de progression d'un job (aucun n'est mesuré) ; migration des pollings Voice, VoiceTrials et VideoStorage vers le SSE ; #152.

## Tickets

1. **Présence du Worker indépendante et perte de lease** : thread de pulse, réessais, annulation sur refus, tests. Aucun blocage.
2. **#203 — hf_access_denied à l'installation** (issue existante). Aucun blocage.
3. **#194 — Token Hugging Face central** (issue existante) : stockage, Voice Processing, étape du wizard, distribution et provenance. Bloqué par 2.
4. **Joignabilité du serveur de transcription par Worker** : test périodique, résultat dans le pulse, affichage. Bloqué par 1.
5. **Lignes de log des jobs** : envoi par lots, nettoyage, conservation, SSE. Aucun blocage.
6. **Pages Workers en direct**. Bloqué par 1.
