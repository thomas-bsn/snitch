# Identité et réconciliation des clips

Statut : accepté pour la phase 1.

## Contexte

Un contenu identique peut être découvert sous plusieurs chemins, sources ou uploads. Un nouveau chemin doit pouvoir être rapproché d'un clip existant, tandis qu'une source montée mais momentanément absente ne doit pas vider l'historique.

## Décision

Le SHA-256 du contenu stable est l'identité unique du clip ; chaque chemin de source garde un enregistrement de provenance vers ce clip. Une contrainte unique PostgreSQL protège les scans concurrents. Taille et date de modification permettent d'éviter un nouveau hash pour un chemin prêt et inchangé ; une nouvelle copie ou une modification détectée est sondée et hashée après stabilité.

Une disparition est appliquée uniquement à la fin d'une réconciliation complète réussie. Un échec de parcours, d'accès au fichier ou à la racine conserve les états précédents. Les chemins disparus restent reliés à leur clip et ne suppriment aucune donnée. La source affiche un avertissement lorsque les emplacements introuvables atteignent 20 % de l'historique connu, et conserve cet avertissement tant que la proportion reste élevée.

## Alternatives écartées

- Une entrée clip par chemin duplique le même média lors d'un renommage ou d'une deuxième copie.
- Une vérification complète du hash à chaque passage relit tout le volume, même si taille et date de modification n'ont pas bougé.
- Traiter une racine inaccessible comme un dossier vide produit de fausses disparitions massives.
- Supprimer les métadonnées ou l'enregistrement du clip à la disparition détruit la provenance et l'historique.

## Conséquences

- La contrainte d'unicité PostgreSQL est l'arbitre des dédoublements concurrents.
- La réconciliation complète peut relire les noms et métadonnées de fichiers, mais ne rehash pas les emplacements inchangés.
- Une modification conservant exactement taille et date de modification ne peut pas être détectée par cette réconciliation légère ; un nouveau chemin, ou une modification détectée par taille/date, est hashé.
- 20 % est un seuil opérationnel initial pour alerter, non une preuve qu'une source ou ses données sont perdues. Les fichiers sont conservés et aucune suppression automatique n'en découle.
