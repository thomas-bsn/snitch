# 0028 — Lignes de log d'un job visibles, extrait d'erreur nettoyé

Statut : accepté. Amende l'[ADR 0023](0023-diagnostic-par-causes-d-echec-persistees.md) sur deux points : il écartait la lecture des logs depuis l'interface et interdisait de journaliser le texte d'une erreur de fournisseur.

## Contexte

Un échec récurrent de `voice.fingerprint` (clip #40) restait inexplicable : `fingerprint()` masquait l'exception, et la cause persistée (`inference_failed`) ne disait ni l'étape ni la raison. L'utilisateur veut aussi voir, depuis la fiche d'un clip, les dernières lignes du Worker et ce que le Worker a reçu du serveur de transcription.

## Décision

- **Extrait d'erreur** : en cas d'échec, le Worker journalise la classe de l'exception, l'étape et un extrait du message nettoyé (jetons `hf_`, en-têtes d'autorisation, URL, chemins et chaînes opaques retirés, 240 caractères au plus). Livré par #204.
- **Lignes du job** : le Worker envoie les lignes de log d'un job par lots (toutes les 2 s ou 50 lignes), après le filtre central des clés sensibles et le même nettoyage. Le serveur conserve les 50 dernières lignes par job, les diffuse par le flux d'événements existant et les supprime avec le job. Seuls les Administrateurs les voient.
- **Transcription** : pour Whisper, on n'affiche que ce que le Worker observe de ses appels (méthode, URL sans identifiants, statut HTTP, latence, extrait borné du corps d'erreur), jamais les logs internes du serveur de transcription.
- Les causes d'échec restent le vocabulaire fermé de l'ADR 0023 ; les lignes de log ne les remplacent pas.

## Alternatives écartées

- **Flux stdout complet** : volumineux, et risque de fuite de secrets ou de contenu.
- **Événements structurés seuls** : ils ne montrent pas l'enchaînement qui mène à l'échec.

## Conséquences

- Le filtrage devient une exigence de sécurité testée : aucun jeton, Bearer, URL avec identifiants ni chemin de clip dans les lignes conservées.
- Une table bornée par job s'ajoute ; sa taille est plafonnée par le nombre de lignes.
