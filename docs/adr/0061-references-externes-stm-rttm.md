# 0061 — Références externes (STM, RTTM) pour lever le biais de production

Date : 2026-11-09. Statut : accepté pour #287. Complète l'[ADR 0057](0057-corpus-de-benchmark-et-verite-terrain.md).

## Contexte

La référence d'un clip certifié vient du pipeline de production corrigé par des humains : le modèle de production est avantagé en transcription, diarisation et bout en bout (ADR 0057). Il faut pouvoir mesurer contre une référence écrite à la main.

## Décision

- **À la création du corpus**, `reference_imports: [{clip_id, stm?, rttm?}]` fournit, pour un clip de test, un texte STM et/ou des tours RTTM. Ils **remplacent** la référence de production du clip, qui compte comme certifié (annotation humaine) sans cocher « relu en entier ». Un clip n'est importé qu'une fois.
- **Remplacement complet** : pour un clip importé, seul ce qui est importé est figé. Un STM seul donne la transcription (le locuteur de chaque segment est celui du fichier) ; un RTTM seul donne la diarisation. Rien n'est complété par la production, qui réintroduirait le biais.
- **Parseurs stricts** (`benchmark_reference_import.go`) : une ligne douteuse refuse le fichier entier. STM `<fichier> <canal> <locuteur> <début> <fin> [<étiquettes>] texte`, RTTM `SPEAKER <fichier> <canal> <début> <durée> <NA> <NA> <locuteur> <NA> <NA>` ; lignes `;;` et vides ignorées ; le fichier et le canal ne sont pas lus. Codes d'erreur : `benchmark_import_format`, `_times` (début négatif, fin non postérieure, durée nulle), `_speakers` (STM et RTTM d'un même clip sans locuteur commun), `_size` (2 Mo au total), `_clip` (clip non test ou importé deux fois), `_track`.
- **La piste** est celle de la transcription de production du clip (à défaut, de ses locuteurs diarisés) si elle est unique ; sinon refus. Les temps sont ceux du flux audio, sans décalage de plan : une référence importée ne retire pas l'étape bout en bout.
- **Les locuteurs importés sont des noms libres**, jamais des Personnes ; l'étape empreinte n'en dépend pas.
- **Source mémorisée par clip** : `benchmark_dataset_clips.reference_source` (`production` ou `import`, migration 00135). La liste des corpus et le détail d'un run exposent `reference_sources`. L'avertissement de biais (écran de création, résultats) ne s'affiche que si `production` y figure.
- Le corps de `POST /api/admin/benchmarks/datasets` passe de 16 Ko à environ 8 Mo ; seuls les textes importés (2 Mo) en justifient.

## Alternatives écartées

- Compléter un import partiel par la production : biais conservé sans que l'écran puisse le dire.
- Importer au moment du run ou après coup : la référence est figée à la création (ADR 0057).
- Rattacher les locuteurs importés à des Personnes : c'est le rôle de l'étape empreinte, hors sujet ici.

## Conséquences

- Un corpus peut mélanger des clips de production et importés : le biais est alors annoncé (il subsiste pour les premiers).
- Un clip sans transcription ni locuteur diarisé, ou multipiste, ne peut pas être importé.
