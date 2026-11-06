# Suppression définitive des clips

Statut : accepté à la demande explicite de l'utilisateur. Cette décision modifie l'invariant de conservation des originaux des ADR 0010, 0012 et 0017 pour une suppression explicitement confirmée par le Super-administrateur.

## Décision

Le Super-administrateur peut demander la suppression définitive d'un clip et de toutes ses copies indexées. Les autres rôles ne disposent pas de cette action. Le dialogue annonce la suppression des fichiers et des données associées. Chaque copie est vérifiée par son empreinte avant suppression ; un fichier différent ou une localisation absente bloque l'opération. Les sources montées en lecture seule ne sont pas modifiées si le système refuse l'écriture.

Les suppressions sur disque et en base ne forment pas une transaction unique. Un journal persistant suit chaque copie et permet de reprendre l'opération après une erreur ou une interruption. Les fichiers sont brièvement renommés dans leur répertoire pour que la copie vérifiée soit celle supprimée. Si l'emplacement est inaccessible ou a changé, l'interface signale l'échec et l'exploitant doit vérifier les fichiers avant de réessayer.

## Alternatives et conséquences

Conserver systématiquement les originaux ne satisfait pas la demande. Une suppression silencieuse sans vérification pourrait effacer un fichier remplacé depuis l'indexation. L'opération nécessite les droits d'écriture sur chaque dossier concerné et ne peut garantir l'effacement de copies que Snitch ne connaît pas.
