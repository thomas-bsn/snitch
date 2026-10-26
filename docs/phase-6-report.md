# Bilan de phase 6 — corrections des transcriptions et des locuteurs de segments

La phase 6 est implémentée sur `integration/phase-6-corrections` (tickets [#49](https://github.com/thomas-bsn/snitch/issues/49) à [#57](https://github.com/thomas-bsn/snitch/issues/57), issue parente [#48](https://github.com/thomas-bsn/snitch/issues/48), [spec](specs/phase-6.md), [ADR 0012](adr/0012-corrections-de-segments.md)). Aucun média réel, scan, batch ni fournisseur externe n'a été utilisé. La validation utilisateur est attendue avant la phase 7 ; les tickets restent ouverts jusque-là.

## Réalisé

- **Texte d'un segment** (#49) : couche de correction distincte de la transcription d'origine, jamais modifiée ; texte effectif, auteur, date, motif, version. Voir [corrections de segments](segment-corrections.md).
- **Historique et restauration** (#50) : historique immuable, restauration du texte d'origine ou d'une version antérieure comme nouvelle entrée.
- **Corrections simultanées** (#51) : concurrence optimiste arbitrée par PostgreSQL ; le perdant reçoit la version courante et son auteur, son brouillon est conservé.
- **Recherche du texte corrigé** (#52) : les quatre modes lisent le texte effectif ; résultats corrigés marqués.
- **Réindexation du segment** (#53) : seul le vecteur du segment est recalculé (index actif et candidat), reprise bornée si le fournisseur d'embeddings est indisponible, compteurs d'administration.
- **Locuteur d'un segment** (#54) : désignation d'un Locuteur local du même clip ou retour au locuteur déduit, sans toucher aux tours de parole, empreintes ni attributions.
- **Retranscription d'un clip corrigé** (#55) : confirmation explicite, lots excluant les corrigés, corrections obsolètes après succès, jamais réappliquées.
- **Export et suppression** (#56) : export administrateur (JSON paginé ou NDJSON) des paires texte d'origine / texte corrigé, sans taux d'erreur ; suppression en cascade avec le clip.
- **Clôture** (#57) : ADR 0012, glossaire, README, ROADMAP, spec, consignes des agents, nettoyage des mentions de numéros de ticket dans le code et la documentation de la phase, et ce bilan.

## Vérifications

Environnement : Docker via Colima, `.venv` partagé. Résultats :

- `go test -race -tags integration ./... -count=1` avec `SNITCH_E2E=1` (PostgreSQL/pgvector réel en Testcontainers, parcours Playwright contre la vraie API, migrations vierges puis rejouées ; httpapi environ 286 s) : tous les paquets passent.
- `go vet ./...` avec et sans le tag `integration` ; `gofmt -l` sans sortie.
- Lint (prettier) et build web (`tsc` et `vite`) ; `check_generated.py` (contrat OpenAPI, clients Go/TypeScript/Python, sqlc reproductibles) ; `check_compose.py` (distribution Forgeyard valide, 31 fixtures négatives rejetées) ; `pip check` sans conflit ; 64 tests du worker (`unittest`).
- Le nettoyage des commentaires n'a changé aucun contrat ni fichier généré.

Aucun média réel : les fixtures sont des octets synthétiques, des empreintes et des embeddings déterministes ; aucune tâche de scan ni de batch n'est planifiée ; aucune référence à la bibliothèque réelle. Les migrations passent vierges et rejouées (suite d'intégration).

Non exécutés ici : builds d'images Docker app/worker, smoke Compose et CI distante.

## Restant

- Mesure de la qualité de transcription (taux d'erreur de mots) sur des textes corrigés humainement, benchmark, seuils et modèles : phase 7, avec autorisation séparée. Snitch n'y calcule aucun taux d'erreur.
- Validation de bout en bout de la phase 5 avec de vrais modèles (pyannote, empreintes) : non faite.
- Usage réel des corrections à observer : les écarts ci-dessous sont à arbitrer à la lumière de l'usage.

## Écarts et choix à confirmer

- **Filtre de la bibliothèque par Personne** : il lit les voix identifiées du clip et ne suit pas le locuteur corrigé d'un segment ; seul le filtre de la recherche (niveau segment) le suit.
- **Voie sémantique et réindexation en attente** : un segment corrigé dont la réindexation est en attente est exclu de la voie sémantique de la recherche hybride (pour ne pas ressortir l'ancien texte par l'ancien vecteur) ; la voie lexicale le trouve tout de suite, et il redevient trouvable sémantiquement une fois réindexé.
- **Corrections de locuteur et retranscription** : elles sont comptées comme « corrigé » pour la confirmation de retranscription et les lots, mais ce chemin précis n'est pas couvert par un test.
- **Suppression d'une Personne** : testée en revue (voir « Corrections de la revue »).
- **Changement de piste** : il est refusé sur un clip corrigé même si la transcription n'est pas prête, tant que la confirmation explicite n'est pas donnée.
- **Aucun taux d'erreur** n'est calculé ; seul l'export est fourni.
- **Playwright** : une spec a été renommée pour éviter une collision de filtre entre fichiers de tests de navigateur.
- **Données synthétiques seulement** : aucune correction n'a été testée avec de vraies transcriptions.
- **Phase 5** : la validation de bout en bout avec de vrais modèles n'a pas eu lieu (voir la roadmap) ; les tests de la phase 6 n'en dépendent pas, mais l'usage réel oui.
- **Mots d'un segment corrigé** : datés au début du segment, sans réalignement ; l'ouverture au timestamp y renvoie.
- **HANDOFF.md** n'a pas été modifié : ses mentions d'état restent historiques (voir AGENTS.md). Les mentions de numéros de ticket des documents de la phase 5 n'ont pas été touchées.

## Corrections de la revue

Corrigé : (1) course locuteur/indexation, avec une `text_version` distincte (migration 00034) que seules les écritures de texte incrémentent ; (3) comparaison du texte d'origine nettoyé (`text_unchanged`) ; (4) corrections obsolètes consultables (`GET /api/clips/{id}/obsolete-corrections`, interface « Corrections obsolètes », test d'intégration, pas de spec Playwright : aucun worker simulé dans ces specs) ; (5) lecture de clip en un instantané `REPEATABLE READ` (pas de test dédié : une course de lecture n'est pas reproductible de façon déterministe) ; (6) une nouvelle diarisation ou une piste microphone qui supprime un locuteur désigné laisse une entrée d'historique, et la suppression d'une Personne est testée.
Conservé et documenté : (2) version unique texte/locuteur (conflit spurieux possible entre un éditeur de texte et un changement de locuteur) ; (7) correction ajoutée pendant une retranscription confirmée : obsolète à l'achèvement ; (8) segment exclu de la voie sémantique après 5 échecs, déjà visible en administration (`reindex_failed`).
Code : squelette `correctSegment` commun aux trois écritures, constantes de codes et de types d'entrée, `shared.ts` côté web.
Écart restant : un refus propre à une écriture (locuteur inconnu, entrée introuvable) est évalué après le contrôle de version, et non avant.

## À valider

Valider la phase 6 (et les écarts ci-dessus) avant de commencer la phase 7. Les tickets #48 à #57 restent ouverts jusqu'à cette validation.
