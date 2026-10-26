# Diarisation et empreintes vocales (phase 5, ticket 32)

Ce document décrit le socle vocal : Locuteurs locaux, empreintes et étapes de file. Les Personnes, validations et rapprochements viennent des tickets suivants de la phase 5. Il n'identifie encore personne : toute voix reste un Locuteur local non identifié.

## Étapes de file

Deux étapes versionnées de la file existante, planifiées uniquement par l'administrateur :

| Étape | Version d'entrée | Rôle |
| --- | --- | --- |
| `voice.diarization.<piste>` | `d5-a<piste>-p<version du pipeline vocal>` | Sépare les voix du clip en Locuteurs locaux et tours de parole. |
| `voice.fingerprint.<piste>` | `f5-a<piste>-p<version du pipeline vocal>-d<tâche de diarisation>` | Calcule une empreinte par Locuteur local et par tour. |

- `POST /api/admin/clips/{id}/voice` planifie la diarisation d'un clip déjà transcrit (la piste par défaut est la piste sélectionnée). Il n'existe ni scan ni traitement nocturne ; la transcription et la vidéo ne sont jamais retraitées.
- L'étape d'empreintes est ajoutée automatiquement, dans la transaction du résultat de la diarisation, pour cette diarisation précise. Une nouvelle diarisation remplace l'ancienne et invalide l'étape d'empreintes qui en dépendait, sauf si un Locuteur local de la piste a été validé par un utilisateur : elle est alors refusée (voir [Personnes et validation](voice-persons.md)).
- Changer le pipeline vocal ne fait qu'ajouter une version : seules ces étapes sont rejouées, à la demande.

## Pipeline vocal

`GET/PUT /api/admin/voice-pipeline` (administrateur) gère des snapshots immuables : fournisseur (`pyannote` ou `fake`), modèle, version du modèle, `access_confirmed`, durée minimale d'une empreinte et qualité minimale. Le modèle de référence prévu est `pyannote/speaker-diarization-community-1`, qui demande un jeton et l'acceptation de ses conditions. Tant que l'administrateur n'a pas confirmé l'accès (`access_confirmed`), aucune tâche n'est planifiée (`voice_model_access_unconfirmed`). Aucun téléchargement n'est jamais déclenché implicitement. Le worker fournit deux fournisseurs : `fake` et `pyannote` (diarisation `pyannote/speaker-diarization-community-1` et empreintes WeSpeaker ResNet34, modèles lus depuis des chemins locaux, voir [worker vocal réel](real-voice-worker.md)). Tout autre fournisseur, ou un `pyannote` dont les modèles, les bibliothèques ou le GPU manquent, échoue avec le code expurgé `provider_unavailable`. Limite honnête : le code `pyannote` n'a pas été exécuté de bout en bout (ni GPU, ni modèle, ni jeton Hugging Face, ni média réel dans l'environnement de développement) ; seuls des pipelines simulés sont testés.

Le fournisseur factice (`fake`) est déterministe et réservé aux tests : N locuteurs (1 à 8) alternent sur des tranches égales de la durée du clip, avec des empreintes déterministes de dimension configurable.

## Résultat et reprise

Le worker récupère sa configuration dans l'entrée de la tâche (`voice`) et ne parle qu'à l'API HTTPS avec son jeton limité. Le résultat est validé (modèle et version identiques au snapshot, étiquettes uniques, temps valides, dimensions cohérentes) puis écrit dans la **même transaction** que l'achèvement de la tâche : l'ancien résultat du clip est supprimé puis remplacé, sans doublon à la reprise. Un résultat d'empreintes ne laisse aucun vecteur dans `jobs.result` : la tâche ne garde qu'un résumé (modèle, dimensions, nombre d'empreintes) et un condensé SHA-256 du résultat reçu pour reconnaître une reprise identique (migration 00031, qui nettoie aussi les lignes existantes) ; les vecteurs ne vivent que dans `voice_fingerprints`. Un lease expiré ou une version d'entrée périmée est refusé (`lease_not_owned`), de même qu'un résultat d'empreintes calculé sur une diarisation remplacée (`stale_voice_result`). Une diarisation sans voix est un succès. Une voix inconnue ne bloque jamais l'achèvement.

## Données

- `local_speakers` et `speaker_turns` : voix détectées par clip et par piste ; supprimées avec le clip.
- `voice_fingerprints` : vecteur pgvector avec `model`, `model_version`, `dimensions`, durée, qualité et `usable`. Les empreintes trop courtes (`too_short`) ou de qualité insuffisante (`low_quality`) restent visibles mais ne servent ni de référence ni de base de propagation. Les vecteurs de modèles, versions ou dimensions différents ne doivent jamais être comparés.

## Consultation

`GET /api/clips/{id}/voice` (utilisateur connecté, clips visibles) retourne l'état des étapes (statut et dernier code d'erreur expurgé), les Locuteurs locaux et les métadonnées d'empreintes, jamais les vecteurs. `GET /api/clips/{id}` ajoute `local_speaker_id` et `local_speaker_label` à chaque segment de transcription : le Locuteur local dont les tours recouvrent le plus le segment. Un segment sans recouvrement n'a pas de Locuteur. La consultation et la recherche fonctionnent worker éteint.

Dans l'interface, le panneau « Voix » d'un clip affiche l'état des étapes, les Locuteurs locaux (non identifiés tant qu'une Personne ne leur est pas attribuée, voir [Personnes et validation](voice-persons.md)) et un bouton administrateur « Diariser les voix » ; la carte « Pipeline vocal » règle le pipeline. Les étapes `voice.*` apparaissent dans la file d'administration comme les autres.
