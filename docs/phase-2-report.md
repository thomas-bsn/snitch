# Bilan de phase 2

La phase 2 est implémentée sur `integration/phase-2`. La validation utilisateur de fin de phase est attendue avant la phase 3. Aucun modèle n’a été téléchargé ou chargé, aucune vraie bibliothèque n’a été traitée et aucun worker n’a été déployé sur le PC GPU ou Forgeyard.

## Réalisé

- File durable PostgreSQL avec planification idempotente par clip/étape/version, priorité, états et interface administrateur pour créer, filtrer et consulter les tâches.
- Worker Python autonome qui se connecte uniquement par HTTPS sortant. Les jetons dédiés sont montrés une seule fois, stockés hachés, révocables et limités aux routes worker.
- Les workers annoncent les étapes qu’ils savent exécuter ; le worker factice ne réclame que `fixture.*`, et les étapes futures restent en attente. L’administration distingue workers en ligne (heartbeat récent), absents et révoqués.
- Acquisition atomique, capacité limitée, leases renouvelés, jeton par lease, récupération après expiration et rejet des opérations d’un ancien lease même si le même worker reprend le job.
- Résultats transactionnels/idempotents, rejet des versions d’entrée obsolètes et invalidation ciblée d’étape.
- Retry exponentiel borné, état dead, historique d’exécution, relance administrateur, annulation, pause/reprise des acquisitions et parallélisme global borné par les capacités worker.
- Interface admin de file, worker et contrôles ; contrats OpenAPI et clients Go/TypeScript/Python générés ; migrations versionnées ; worker absent du Compose NAS.

Le fournisseur livré est factice : il vérifie le parcours worker et retourne des données synthétiques. Il ne prépare pas l’audio, ne transcrit pas les clips et ne réalise aucune analyse IA ; ces étapes appartiennent aux phases suivantes.

## Vérifications

- `npm --prefix web run build` exécuté avant les tests afin que l’API embarque la SPA courante.
- `RECEIPTS_E2E=1 bash scripts/test.sh` : Go avec race detector, PostgreSQL/pgvector via Testcontainers et onze suites Playwright réussis.
- `go vet ./...`, format Go, lint/build web, tests Python du worker, `scripts/check_generated.py` et `scripts/check_compose.py` réussis.
- Build smoke local de l’image worker et `pip check` réussis ; tests HTTPS utilisent un certificat de test approuvé et vérifient le rejet d’un certificat non approuvé.
- Revue finale standards/spec effectuée ; aucun écart documenté restant. Le fournisseur factice n’accède pas aux médias et les étapes non prises en charge ne sont pas louées.

## Restant et écarts

- Le worker n’a pas été testé sur une RTX 4070 réelle ; consommation VRAM, parallélisme soutenable, débit et durées restent inconnus.
- Le fournisseur réel, préparation audio, transcription et les étapes IA restent à faire en phase 3 et suivantes. La file ne déclenche aucun traitement nocturne réel.
- Aucun déploiement sur PC/NAS, aucune CI distante, publication d’image ni scan de la vraie bibliothèque n’a été effectué. Les montages de dossiers Forgeyard et sauvegardes des médias gardent les limites décrites au bilan de phase 1.
- Le batch réel et son benchmark représentatif restent soumis à l’autorisation de phase 7 ; tous les tests de phase 2 utilisent seulement fixtures et données synthétiques.

La prochaine étape est la validation utilisateur de fin de phase 2. Aucune phase 3 ne commence avant cette validation.
