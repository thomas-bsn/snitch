# Revue finale #274

Périmètre : changements depuis `53583abcdc1ee80f9c9a014a7020f2bfc69480af`, intégration initiale `f0e9e23f`, puis corrections de cette reprise. Deux revues indépendantes ont examiné Standards et Spec sur cette intégration. La contre-vérification des corrections est réalisée par l'agent intégrateur ; elle n'est pas une seconde revue indépendante.

## Standards

Références : AGENTS.md, CONTRIBUTING.md, docs/development.md, consignes métier et tracker, GLOSSARY.md, roadmap et ADR concernés.

Deux défauts P2 corrigés :

- Un offset inconnu devenait zéro dans le contrat de diagnostic et la sélection. Sa nullabilité est maintenant conservée jusqu'au Worker ; le diagnostic reste ambigu (`timeline_unknown`). Les clients Go, TypeScript et Python sont régénérés.
- Les paires acoustiques explicites tentaient de construire un ensemble de dictionnaires Python. La normalisation déduplique désormais les identifiants avant de retrouver les événements ; une régression couvre ce chemin.

La migration 118 ne modifie plus la politique JIT de PostgreSQL : elle simplifie l'accès à la première source dans la vue de transcription. Le test de lecture sur 3 000 clips et 36 000 segments vérifie le résultat avec JIT activé. La migration appartient à cette intégration non publiée ; son remplacement ne constitue pas une procédure de mise à jour d'une installation qui aurait déjà appliqué l'ancienne version expérimentale.

## Spec

Références : parent #274, tickets #337 à #344, ADR 0024 et 0047 à 0054, guides des Plans et du Worker.

Six défauts P2 corrigés :

1. Les choix de portée, mode et étapes sont appliqués. Une réanalyse limitée réutilise seulement un Plan publié compatible ; sans transcription réutilisable, le vocal seul est refusé avec `transcription_required`.
2. Le vocal traite les pistes retenues ayant terminé leur reconnaissance, sans programmer les pistes exclues par le diagnostic.
3. La confirmation de remplacement conserve sa borne temporelle et refuse d'écraser un travail humain plus récent.
4. Une analyse complète reste active jusqu'aux étapes d'indexation et vocales ; attente de Worker, annulation et reprise restent visibles. Une relance forcée renouvelle diarisation et empreinte.
5. La fusion acoustique accepte l'union de preuves directes couvrant intégralement un segment, sans déduction transitive.
6. Les observations fiables peuvent être réutilisées entre Plans lorsque l'empreinte complète des entrées correspond ; les observations incertaines sont recalculées.

Les régressions couvrent notamment réanalyse limitée, remplacement concurrent, reprise vocale, échec terminal, relance forcée, cache et preuves fractionnées. Les validations et leurs limites figurent dans le [bilan](issue-274-report.md). Aucun défaut connu de ces deux revues n'est laissé ouvert ; cela ne constitue pas une mesure de qualité sur une bibliothèque réelle.

Bilan des constats initiaux : Standards, 2 défauts P2 ; Spec, 6 défauts P2. Tous corrigés et contre-vérifiés par l’intégrateur.
