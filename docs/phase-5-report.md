# Bilan de phase 5 — voix, Personnes et attribution progressive

La phase 5 est implémentée sur `integration/phase-5-voice` (tickets [#32](https://github.com/thomas-bsn/snitch/issues/32) à [#42](https://github.com/thomas-bsn/snitch/issues/42), issue parente [#31](https://github.com/thomas-bsn/snitch/issues/31), [spec](specs/phase-5.md), [ADR 0011](adr/0011-locuteurs-personnes-et-attributions.md)). Aucun média réel, scan, batch ni fournisseur externe n'a été utilisé. La validation utilisateur est attendue avant la phase 6 ; les tickets restent ouverts jusque-là.

## Réalisé

- **Diarisation et empreintes** (#32) : étapes versionnées de la file, résultats enregistrés en une transaction, Locuteurs locaux par segment, fournisseur factice déterministe, aucun téléchargement implicite de modèle. Voir [diarisation](voice-diarization.md).
- **Personnes et validation** (#33) : création de Personnes, première validation arbitrée par index uniques partiels PostgreSQL, correction explicite, historique immuable, extraits écoutables par les seuls utilisateurs connectés autorisés. Voir [Personnes](voice-persons.md).
- **Piste micro** (#34) : désignation par source parmi les pistes détectées, remplacement par clip, attribution sans diarisation. Voir [piste micro](voice-microphone-track.md).
- **Rapprochement en base** (#35) : références = échantillons validés, niveaux automatique / suggestion / ambigu / inconnu, seuils configurables, aucune fusion sans validation. Voir [rapprochement](voice-matching.md).
- **Groupes de voix inconnues** (#36) et **vérification** (#37) : groupes proposés jamais confirmés, exclusion par voix, nommage d'un groupe, comptes issus de la base, vue des cas inconnus ou ambigus. Voir [groupes](voice-groups.md) et [vérification](voice-verification.md).
- **Précision et automatisation** (#38) : mesure sur clips distincts de ceux des références, critères configurables, proposition d'activation, activation et désactivation réservées à l'administrateur. Voir [automatisation](voice-automation.md).
- **Filtre par Personne** (#39) : effectif dans les quatre modes de recherche et dans la bibliothèque, permissions avant pagination et comptes. Voir [filtre](voice-person-filter.md).
- **Gestion des Personnes** (#40) : fusion, séparation, suppression, historique, recalcul. Voir [corrections](voice-person-corrections.md).
- **Clips d'une Personne et profil vocal** (#41) : lien Personne-compte, écran « Ma voix », suppression du profil avec marqueur sans donnée vocale et trace. Voir [profil vocal](voice-profile.md).
- **Clôture** (#42) : ADR 0011, glossaire, README, ROADMAP, spec et ce bilan.

## Vérifications

Environnement : Docker via Colima. Réussies :

- `go test -race -tags integration ./... -count=1` avec `SNITCH_E2E=1` (PostgreSQL/pgvector réel en Testcontainers, parcours Playwright contre la vraie API, migrations vierges puis rejouées ; httpapi environ 225 s) : toutes les suites passent, y compris celles de `person-filter.spec.ts`.
- `go vet ./...` avec et sans le tag `integration` ; `gofmt -l` sans sortie.
- Lint (prettier) et build web ; `check_generated.py` (contrat OpenAPI, clients Go/TypeScript/Python, sqlc reproductibles) ; `pip check` sans conflit ; 16 tests du worker (`unittest`).
- Aucun média réel : les fixtures sont des octets synthétiques, des empreintes déterministes et un faux `ffmpeg` ; aucune référence à une bibliothèque réelle ; aucune tâche de scan ni de batch n'est planifiée.

Non exécutés ici : builds d'images Docker app/worker et smoke Compose (CI distante non exécutée), `pip check` dans l'image worker (il a été exécuté sur l'environnement Python local).

## Restant

- Benchmark réel, seuils et critères définitifs (95 %, 50 confirmations, 0,90 / 0,75 / 0,05 sont des valeurs de départ), choix du modèle de diarisation et d'empreintes, accès à un vrai modèle : phase 7, avec autorisation séparée.
- Passage à l'échelle : le rapprochement, le calcul des groupes et le filtre par Personne sont exacts et synchrones ; à rendre incrémentaux ou asynchrones et à mesurer en phase 7 si les volumes l'exigent.
- Titres réels des pistes Medal à demander à l'utilisateur ; la règle de piste micro n'est pas appliquée à l'ingestion d'un clip arrivé après coup, hors replanification.
- Levée du retrait du profil vocal, changement de piste ou de modèle sur un clip validé, correction des transcriptions (phase 6).

## Écarts et choix à confirmer

- **Liaison compte-Personne réservée aux administrateurs** : la spec ne le précisait pas ; la suppression du profil est ouverte au compte lié et aux administrateurs.
- **Tous les utilisateurs connectés** peuvent créer, valider, corriger, fusionner, séparer et supprimer des Personnes (conforme à « tous les utilisateurs connectés valident », mais étendu aux opérations destructrices de #40) : à confirmer, ou à restreindre.
- **Une Personne liée à un compte n'est pas supprimable**, pour ne pas supprimer silencieusement l'identité d'un utilisateur.
- **Pas de levée du retrait du profil vocal** : une Personne ayant supprimé son profil ne peut plus recevoir de validation ni de rapprochement ; la rétablir demande une décision.
- **Rapprochement synchrone dans la requête de validation** : la requête attend le recalcul de la bibliothèque (un échec est journalisé sans défaire la validation) ; acceptable pour la phase 5, à revoir en phase 7.
- **Verrou de re-diarisation** : une piste dont un Locuteur local est validé ne peut plus être rediarisée, ce que la spec n'avait pas prévu.
- **Playwright** : `person-filter.spec.ts` a été signalé comme éventuellement instable par un agent ; il a passé dans la dernière exécution complète, sans flake observé. À surveiller.
- **Numérotation des migrations** : 00026 et 00028 ne sont pas utilisées (tickets n'ayant pas eu besoin de migration) ; les migrations passent vierges et rejouées.
- **Mentions « aucun code avant accord » obsolètes** : la note de la spec de phase 5 et la ligne d'état de la roadmap sont mises à jour. HANDOFF.md n'a pas été modifié : ses mentions d'état restent historiques (voir AGENTS.md).

## À valider

Valider la phase 5 (et les choix ci-dessus) avant de commencer la phase 6. Les tickets #31 à #42 restent ouverts jusqu'à cette validation.
