# Bilan #274 — analyse multipiste

L'intégration réunit #337 à #344 : inventaire, diagnostic explicite et hors ligne, Plans immuables, sélection des pistes, reconnaissance par piste, fusion acoustique prudente, publication atomique, vocal multipiste et parcours de réanalyse. La PR #345 est destinée à la revue utilisateur ; sa fusion reste une décision séparée.

## Réalisé

Les diagnostics fiables sont réutilisables entre Plans compatibles. Un offset inconnu reste inconnu. Les exclusions ne déclenchent pas de traitement vocal. La publication conserve une provenance par source et la fusion ne supprime que les doublons entièrement couverts par des preuves acoustiques directes.

La réanalyse respecte portée, mode et étapes forcées. Le vocal seul exige une transcription réutilisable. L'analyse complète attend réellement indexation et voix, avec reprise et annulation ; les échecs terminaux ne provoquent pas de boucle de relance automatique. Une diarisation forcée renouvelle aussi l'empreinte. Le remplacement protège les validations et corrections humaines postérieures à la confirmation.

La migration 118 optimise la vue de transcription en lisant directement la première source du manifeste canonique. Aucun réglage JIT global n'est introduit. Le test de performance utilise 3 000 clips et 36 000 segments synthétiques, avec JIT activé.

La [revue Standards et Spec](issue-274-review.md) documente les huit défauts identifiés et leurs corrections.

## Validation

- Worker Python : 332 tests, dont 9 sauts liés aux dépendances ou modèles vocaux optionnels. Silero et la déduplication acoustique s'exécutent réellement sur des fixtures dédiées.
- Deux smokes Go / Worker Python : claim HTTP, téléchargement, ffmpeg, diagnostic Silero, complétion, publication et recherche. Le premier oppose parole et tonalité synthétique (3 jobs) ; le second déduplique deux copies de la même fixture vocale par preuve PCM réelle (5 jobs), en conservant les deux sources dans le manifeste. Le reconnaisseur ASR est une fixture déterministe injectée ; ce smoke ne mesure pas la précision d'un modèle de transcription.
- Régressions d'intégration : diagnostics réutilisés, réanalyse limitée, garde concurrente, reprise après échec, analyse complète et relance forcée des voix.
- CI Worker : le job installe explicitement `ffmpeg`, nécessaire aux quatre tests de décodage des médias synthétiques ; le job relancé passe (332 tests, 21 sauts dans cet environnement léger).
- Suivi après publication : le moteur réutilise la clé traduite `audio_plan_published` ; le parcours navigateur attend uniquement les pistes vocales retenues.
- Interface : build, lint et génération reproductible du contrat ; sept tests `admin-voice` passent avec le catalogue de modèles créé par les autres parcours.

La suite complète locale Go/PostgreSQL avec détection de courses passe pour tous les tests hors navigateur. La première exécution navigateur via Docker compte 40 suites réussies sur 45 : trois limites du relais local (Discord, contrôle des embeddings, répertoire temporaire non monté), une course de synchronisation du test du panneau d’analyse et un scénario qui devait choisir explicitement le mode de réanalyse complète. Ces deux scénarios sont corrigés et revérifiés. Les cinq shards E2E de la CI utilisent Chromium natif ; leurs résultats définitifs sont accessibles dans les contrôles de la PR #345.

Chromium natif est interdit par le sandbox macOS de cette session. Les E2E utilisent l'image Playwright Docker et un relais localhost temporaire hors dépôt. Le diagnostic du test `admin-voice` a montré une attente incorrecte d'un catalogue vide après les autres tests : l'attente utilise désormais le nombre de modèles effectivement disponibles.

## Restant et limites

Aucun média de bibliothèque réelle n'a été parcouru. Les tests utilisent des données synthétiques ou des fixtures dédiées. La qualité ASR, les performances sur bibliothèque réelle et les combinaisons de modèles vocaux optionnels ne sont pas établies par ces vérifications. Le benchmark de phase 7 exige toujours son autorisation propre.

La migration 118 remplace une version expérimentale non publiée de cette branche. Une base de développement ayant déjà appliqué cette ancienne migration doit être recréée ou réconciliée explicitement ; les tests utilisent des bases éphémères neuves.
