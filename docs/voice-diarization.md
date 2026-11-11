# Diarisation et empreintes vocales (phase 5, ticket 32)

Ce document décrit le socle vocal : Locuteurs locaux, empreintes et étapes de file. Les Personnes, validations et rapprochements viennent des tickets suivants de la phase 5. Il n'identifie encore personne : toute voix reste un Locuteur local non identifié.

## Étapes de file

Deux étapes versionnées de la file existante, planifiées uniquement par l'administrateur :

| Étape | Version d'entrée | Rôle |
| --- | --- | --- |
| `voice.diarization.<piste>` | `d5-a<piste>-p<version du pipeline vocal>` | Sépare les voix du clip en Locuteurs locaux et tours de parole. |
| `voice.fingerprint.<piste>` | `f5-a<piste>-p<version du pipeline vocal>-d<tâche de diarisation>` | Calcule une empreinte par Locuteur local et par tour, sur la Parole nette, avec ses mesures. |

- `POST /api/admin/clips/{id}/voice` planifie la diarisation d'un clip déjà transcrit (la piste par défaut est la piste sélectionnée). Il n'existe ni scan ni traitement nocturne ; la transcription et la vidéo ne sont jamais retraitées.
- L'étape d'empreintes est ajoutée automatiquement, dans la transaction du résultat de la diarisation, pour cette diarisation précise. Une nouvelle diarisation remplace l'ancienne et invalide l'étape d'empreintes qui en dépendait, sauf si un Locuteur local de la piste a été validé par un utilisateur : elle est alors refusée (voir [Personnes et validation](voice-persons.md)).
- Changer le pipeline vocal ne fait qu'ajouter une version : seules ces étapes sont rejouées, à la demande.

## Pipeline vocal

`GET/PUT /api/admin/voice-pipeline` (administrateur) gère des snapshots immuables : fournisseur (`pyannote`, `nemo` ou `fake` ; le fournisseur d'un modèle du catalogue est son moteur), modèle, version du modèle (la révision épinglée du catalogue), durée minimale d'une empreinte et qualité minimale. Le modèle de référence prévu est `pyannote/speaker-diarization-community-1`, qui demande un jeton et l'acceptation de ses conditions. La case « accès confirmé » n'existe plus : un modèle pyannote se choisit dans le catalogue (`GET /api/admin/voice-models`), chaque Worker le prépare (téléchargement, chargement, inférence synthétique) et ne prend de travaux vocaux que s'il détient le modèle actif prêt ; l'administrateur n'active un candidat que si au moins un Worker l'a prêt (`voice_model_not_ready` sinon), et l'activation crée une nouvelle version du pipeline (ADR 0022). Le jeton Hugging Face reste dans un fichier secret du Worker (`SNITCH_HF_TOKEN_FILE`), lu par le seul sous-processus de téléchargement ; son absence ou un accès refusé deviennent les causes `hf_token_missing` et `hf_access_denied`. Aucun téléchargement n'est jamais déclenché implicitement. Le catalogue contient aussi `nvidia/diar_streaming_sortformer_4spk-v2.1` (moteur `nemo`, 4 locuteurs au plus, pas de `num_speakers`, licence NVIDIA Open Model) : l'adaptateur NeMo lit l'archive `.nemo` hors ligne dans le dossier du modèle actif et rend les mêmes locuteurs et intervalles que pyannote ; les empreintes restent celles de WeSpeaker. Le worker fournit trois fournisseurs : `fake`, `pyannote` et `nemo`. Le fournisseur `pyannote` (pour le reste de ce paragraphe) (diarisation `pyannote/speaker-diarization-community-1` et empreintes WeSpeaker ResNet34, modèles lus depuis des chemins locaux, voir [worker vocal réel](real-voice-worker.md)). Tout autre fournisseur, ou un `pyannote` dont les modèles, les bibliothèques ou le GPU manquent, échoue avec le code expurgé `provider_unavailable`. Limite honnête : le code `pyannote` n'a pas été exécuté de bout en bout (ni GPU, ni modèle, ni jeton Hugging Face, ni média réel dans l'environnement de développement) ; seuls des pipelines simulés sont testés.

Le fournisseur factice (`fake`) est déterministe et réservé aux tests : N locuteurs (1 à 8) alternent sur des tranches égales de la durée du clip, avec des empreintes déterministes de dimension configurable.

## Modèles déclarés par le worker

Le worker déclare à l'application, à chaque pouls, les modèles vocaux qu'il utilise ; la page Workers les affiche aux administrateurs. Un rôle vocal est déclaré dès que le chemin local de son modèle est configuré (`SNITCH_VOICE_DIARIZATION_MODEL_PATH`, `SNITCH_VOICE_EMBEDDING_MODEL_PATH`), avec le fournisseur `pyannote`, le nom du modèle (par défaut `pyannote/speaker-diarization-community-1` et `pyannote/wespeaker-voxceleb-resnet34-LM`, remplaçable par `SNITCH_VOICE_DIARIZATION_MODEL_NAME` et `SNITCH_VOICE_EMBEDDING_MODEL_NAME`) et la version facultative (`SNITCH_VOICE_DIARIZATION_MODEL_VERSION`, `SNITCH_VOICE_EMBEDDING_MODEL_VERSION`). Jamais de secret. Sans chemin, le rôle est affiché « non déclaré », y compris avec le fournisseur factice. Cette déclaration est informative : le modèle réellement appliqué à une tâche reste celui du pipeline vocal, enregistré avec chaque résultat et chaque empreinte. Détails et règles de validation : [transcription-worker.md](transcription-worker.md#modèles-déclarés).

## Résultat et reprise

Le worker récupère sa configuration dans l'entrée de la tâche (`voice`) et ne parle qu'à l'API HTTPS avec son jeton limité. Le résultat est validé (modèle et version identiques au snapshot, étiquettes uniques, temps valides, dimensions cohérentes) puis écrit dans la **même transaction** que l'achèvement de la tâche : l'ancien résultat du clip est supprimé puis remplacé, sans doublon à la reprise. Un résultat d'empreintes ne laisse aucun vecteur dans `jobs.result` : la tâche ne garde qu'un résumé (modèle, dimensions, nombre d'empreintes) et un condensé SHA-256 du résultat reçu pour reconnaître une reprise identique (migration 00031, qui nettoie aussi les lignes existantes) ; les vecteurs ne vivent que dans `voice_fingerprints`. Un lease expiré ou une version d'entrée périmée est refusé (`lease_not_owned`), de même qu'un résultat d'empreintes calculé sur une diarisation remplacée (`stale_voice_result`). Une diarisation sans voix est un succès. Une voix inconnue ne bloque jamais l'achèvement.

## Mesures d'une empreinte

Depuis #389 (ADR 0064), le Worker calcule chaque empreinte sur la Parole nette du Locuteur local (voir [worker vocal réel](real-voice-worker.md#résultats-produits)) et joint à chaque élément de `fingerprints` un objet `measures` facultatif :

```json
"measures": {"version": 1, "net_speech_seconds": 2.1, "overlap_ratio": 0.25, "overlapped": false,
             "clipped_ratio": 0.0004, "mean_level_dbfs": -23.5, "coherence": 0.81}
```

| Champ | Requis | Valeurs | Sens |
| --- | --- | --- | --- |
| `version` | oui | entier ≥ 1 | version des définitions du Worker ; 1 : intervalles nets d'au moins 0,5 s |
| `net_speech_seconds` | oui | ≥ 0 | Parole nette entrée dans le vecteur ; 0 pour une empreinte chevauchée |
| `overlap_ratio` | oui | 0 à 1 | part de la parole du tour (ou du Locuteur local) pendant laquelle un autre Locuteur local de la piste parle |
| `overlapped` | oui | booléen | aucun intervalle net suffisant : vecteur calculé sur les tours entiers, jamais éligible comme référence |
| `clipped_ratio` | non (`null`) | 0 à 1 | part d'échantillons 16 bits à pleine échelle dans l'audio embarqué |
| `mean_level_dbfs` | non (`null`) | −120 à 0 | niveau RMS moyen de l'audio embarqué ; un silence numérique vaut −120 |
| `coherence` | non (`null`) | −1 à 1 | cosinus moyen entre les vecteurs des tours du Locuteur local ; absent avec un seul tour et pour une empreinte de tour |

Le contrat est additif : un résultat sans `measures` (Worker plus ancien) est accepté et ses mesures restent inconnues (`NULL`) ; des mesures hors de ces bornes font refuser le résultat entier (`invalid_voice_result`). Les mesures ne portent aucun temps : le décalage de piste d'un Plan audio ne s'applique qu'aux bornes, une seule fois (ADR 0052). Elles ne décident encore rien : `usable` suit toujours la durée et la qualité minimales du pipeline.

## Données

- `local_speakers` et `speaker_turns` : voix détectées par clip et par piste ; supprimées avec le clip.
- `voice_fingerprints` : vecteur pgvector avec `model`, `model_version`, `dimensions`, durée, qualité, `usable` (comparable) et `contributive`, décidés par la [règle d'éligibilité](voice-matching.md#éligibilité-des-empreintes) (#390). Les empreintes rejetées (`too_short`, `overlap_only`, `clipped`, ou `low_quality` avant la règle) restent visibles mais ne sont pas rapprochées seules. Les vecteurs de modèles, versions ou dimensions différents ne doivent jamais être comparés. Les mesures sont stockées dans `measures_version`, `net_speech_seconds`, `overlap_ratio`, `overlapped`, `clipped_ratio`, `mean_level_dbfs` et `coherence` (migration 00142), toutes `NULL` pour une empreinte enregistrée sans mesures.

## Consultation

`GET /api/clips/{id}/voice` (utilisateur connecté, clips visibles) retourne l'état des étapes (statut et dernier code d'erreur expurgé), les Locuteurs locaux et les métadonnées d'empreintes, jamais les vecteurs. `GET /api/clips/{id}` ajoute `local_speaker_id` et `local_speaker_label` à chaque segment de transcription : le Locuteur local dont les tours recouvrent le plus le segment. Un segment sans recouvrement n'a pas de Locuteur. La consultation et la recherche fonctionnent worker éteint.

`GET /api/clips/{id}` ajoute `speaker_coverage` à chaque transcription de piste diarisée (#273) : pour chaque Locuteur local non ignoré, un statut, ses secondes de parole (tours qui se chevauchent comptés une fois), ses secondes sous la parole d'autrui (`overlapped_seconds`), ses secondes sous aucun segment (`uncovered_seconds`) et son nombre de segments attribués. Statuts : `words` (au moins un segment lui est attribué), `speech_without_words` (il parle sous des segments attribués à d'autres : un segment n'a qu'un Locuteur, celui qui parle le plus), `no_transcript` (aucun segment ne recouvre ses tours, dans un trou de la transcription) et `outside_transcript` (idem, avant ou après toute la transcription : transcription tronquée, décalage, mauvaise piste). Le calcul partage la règle d'attribution des segments (`dominantSpeaker`) et suit les corrections manuelles. Dans l'interface, chaque voix du panneau « Voix » porte cette distinction. `GET /api/admin/clips/{id}/voice-trace` (administrateur) donne la trace brute de la piste visible : modèles et versions de la transcription et de la diarisation, tours bruts, segments (bornes seulement) avec les voix qui les recouvrent avant alignement et la voix retenue après, couverture, et les empreintes de production de la piste avec leurs mesures (`fingerprints[].measures`, `null` pour une empreinte sans mesures). Elle ne contient aucun texte transcrit ni aucun vecteur. Méthode et scénarios de référence : [voice-coverage-benchmark.md](voice-coverage-benchmark.md).

Dans l'interface, le panneau « Voix » d'un clip affiche l'état des étapes, les Locuteurs locaux (non identifiés tant qu'une Personne ne leur est pas attribuée, voir [Personnes et validation](voice-persons.md)) et un bouton administrateur « Diariser les voix » ; la carte « Pipeline vocal » règle le pipeline. Les étapes `voice.*` apparaissent dans la file d'administration comme les autres.
