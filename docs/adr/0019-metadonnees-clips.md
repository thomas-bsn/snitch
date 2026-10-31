# Métadonnées de capture des clips

Statut : accepté pour l’issue #94. Cette décision remplace le repli sur la date de modification décrit par la spec de phase 1 ; elle conserve les règles des ADR 0009 et 0010.

## Contexte

Le conteneur et certains noms de capture peuvent renseigner le jeu et la date de prise de vue. La date de modification peut correspondre à une copie et ne prouve pas quand la vidéo a été enregistrée. Un même clip peut avoir plusieurs provenances.

## Décision

Pour chaque champ, une correction manuelle prime sur les métadonnées du conteneur puis sur un nom DVR reconnu ; sinon le champ est inconnu. Le nom original reste conservé. L’heure sans fuseau est interprétée dans le fuseau configuré pour l’installation ; sans configuration, ou si l’heure locale est ambiguë ou inexistante, elle reste inconnue. Le wizard propose un pays et un fuseau modifiable ; l’administration permet de les changer.

La première découverte fixe les valeurs extraites, même si un contenu identique est découvert sous d’autres noms. Un Administrateur peut corriger ou effacer chaque champ. Un retraitement explicitement demandé sonde les originaux encore disponibles et conserve les corrections manuelles. Un changement de fuseau ne modifie pas les valeurs déjà enregistrées. Les dates techniques de découverte et de modification restent distinctes de la date de capture et servent à l’ingestion et au tri de secours.

La fiche présente les provenances connues, et la carte la première connue. La date de première observation des anciennes sources est approximée par leur dernière stabilisation, faute d’historique plus précis. Une identité supprimée est affichée comme importateur inconnu.

## Alternatives écartées

- Utiliser la date de modification comme date de capture : elle change lors d’une copie ou d’un import.
- Supposer UTC ou choisir silencieusement entre deux heures locales identiques : cela attribue un instant non prouvé.
- Retraiter automatiquement après un changement de configuration : cela modifierait les résultats sans action explicite et lirait les médias.

## Conséquences

La date de capture est nullable dans le contrat, et les filtres de dates excluent les dates inconnues. Le tri utilise la date de découverte en repli. Les installations existantes doivent configurer leur localisation, puis déclencher le retraitement si elles le souhaitent. Les médias inaccessibles ne sont pas modifiés ; aucun traitement de bibliothèque réelle n’est exécuté pendant le développement.
