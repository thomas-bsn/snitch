# 0044 — Changement du modèle d'empreinte : transition versionnée, aperçu et bascule atomique

Statut : proposé : à relire par l'utilisateur. Complète l'[ADR 0038 (modèles locaux pilotés par Snitch)](0038-modeles-locaux-pilotes-par-snitch.md), l'[ADR 0022](0022-modele-vocal-actif-global.md) et l'[ADR 0011](0011-locuteurs-personnes-et-attributions.md) ; respecte l'[ADR 0031](0031-voix-ignoree-random-et-refus-d-identification.md). Ticket #311 (parent #265, spec #303).

## Contexte

Une empreinte est un vecteur produit par un modèle donné. Deux vecteurs de modèles différents (ou de versions, ou de dimensions différentes) ne sont pas comparables. Les empreintes portent déjà `(model, model_version, dimensions)` et le rapprochement ne compare qu'à l'intérieur d'un groupe ; #310 a ajouté le marqueur `benchmark`, qui sort un groupe de l'identification. Mais rien n'organise le changement du modèle d'empreinte actif : #308 le refuse (`voice_replacement_kind_unsupported`) et aucune action n'écrit `voice_active_model`.

La « référence » d'une Personne n'est pas un vecteur stocké à part : c'est l'empreinte, dans le modèle actif, du locuteur local d'un échantillon validé (`voice_samples`). Changer le modèle actif sans recalcul rendrait toutes les références invisibles du rapprochement.

## Décision

1. **Versionnage.** Le groupe `(model, model_version, dimensions)` reste l'unité de comparaison. L'espace de production est le groupe des empreintes `benchmark = false`. Seul le modèle d'empreinte actif y écrit ; aucune action ne modifie `voice_active_model` en dehors de la bascule ci-dessous. Un résultat de production calculé par un autre modèle *du catalogue* que l'actif est refusé comme périmé (`stale_voice_result`), pour ne jamais ranger une empreinte dans le mauvais groupe ; un modèle hors catalogue (fournisseur synthétique) n'est pas concerné.
2. **Une transition explicite et unique.** Une table `voice_fingerprint_transitions` porte au plus une transition ouverte (source, cible, phase `preparing`, `recomputing`, `switched` ou `cancelled`, auteur, dates, nombre de références annoncé). Elle est lue à la demande avec sa progression ; l'état est dans la base, donc observable et reprenable.
3. **Aperçu d'impact avant confirmation.** `GET` d'aperçu : nombre de références à recalculer (locuteurs locaux d'échantillons validés ayant une empreinte exploitable dans l'espace actuel), clips concernés, Personnes concernées, autres empreintes à recalculer ensuite, Workers qui ont déjà le modèle cible prêt. Le démarrage exige l'écho du nombre de références (`confirm_references`) ; s'il a changé entre-temps, refus `voice_transition_preview_stale`. Choix : l'écho évite une confirmation sur un chiffre périmé.
4. **Recomputation dans l'espace en attente, via le modèle demandé de #310.** Les références sont recalculées par des jobs d'empreinte avec `requested_model_id = cible`, donc attribués seulement à un Worker qui a la cible `ready`, et rangés en `benchmark = true` : la production continue sur l'ancien modèle, sans dégradation, pendant toute la transition. Choix non trivial : pas de double écriture en production, pas de bascule au début.
5. **Que devient chaque chose.** Échantillons validés : lignes inchangées (ils désignent un locuteur, pas un modèle). Attributions validées (décision humaine) : inchangées. Attributions machine : réévaluées par le rapprochement après la bascule, avec le nouveau `match_model` ; sans empreinte cible elles sont retirées (jamais conservées sur un modèle périmé). Profil vocal : ses échantillons et attributions suivent ; sa suppression efface les empreintes de tous les groupes, y compris l'espace en attente et l'ancien. Une Personne dont le profil est supprimé n'est jamais recalculée.
6. **Migration essentielle = les références.** Quand chaque référence a son empreinte cible, la bascule se fait d'un bloc dans une transaction (verrou du modèle vocal et verrou du rapprochement) : `voice_active_model` passe à la cible, les empreintes cible passent en production, celles de l'ancien modèle passent hors production (`benchmark = true`, conservées). Choix non trivial : les autres empreintes (voix à rapprocher) ne bloquent pas la bascule ; elles sont recalculées ensuite par des jobs de production, et leurs attributions machine reviennent au fil des résultats. Alternative d'attendre 100 % rejetée (bibliothèque entière sur un NAS).
7. **Reprise après interruption.** L'avancement se déduit des données (empreintes cible présentes, jobs actifs), jamais d'un compteur en mémoire. Une fonction idempotente replanifie les références sans empreinte cible ni job actif, constate la fin et bascule. Elle est appelée au démarrage de la transition, à chaque résultat d'empreinte demandée, à chaque appel d'un Worker et à la lecture de l'état ; une reprise explicite (`resume`) relance aussi les jobs morts.
8. **Ancien modèle conservé.** Tant que la transition n'est pas `switched`, ni la source ni la cible ne peuvent être désinstallées ni désactivées (`voice_model_in_use`). L'installation de la cible à côté de la source réutilise le remplacement de #308, autorisé pour l'empreinte seulement quand une transition ouverte relie les deux modèles (sinon `voice_transition_required`). Le remplacement ne bascule pas seul ; après la bascule, son ordre `remove` suit les règles de #308 (différé tant qu'un job l'utilise). Choix non trivial : le retour arrière après bascule est une nouvelle transition inverse (les empreintes de l'ancien modèle sont conservées, donc peu à recalculer), pas une annulation.
9. **Jamais de comparaison silencieuse.** Le rapprochement, les groupes de voix et l'automatisation ne lisent que `benchmark = false` et par groupe `(modèle, version, dimensions)` ; la bascule change les deux d'un seul coup. Un test vérifie qu'après bascule aucune comparaison n'a lieu avec l'ancien groupe, et qu'avant bascule la cible n'a aucun effet sur l'identification.

## Alternatives écartées

- **Bascule immédiate puis recalcul** : références invisibles, identifications dégradées pendant des heures.
- **Convertir les vecteurs d'un espace à l'autre** : pas de projection fiable entre modèles.
- **Tout recalculer avant de basculer** : trop long ; les références suffisent à retrouver des rapprochements fiables.
- **Un nouveau marqueur de groupe à la place de `benchmark`** : le marqueur de #310 dit déjà « hors espace de production » ; le doubler créerait deux vérités.
- **Recalcul décidé par le Worker** : contraire au principe de l'ADR 0038 (Snitch décide).

## Conséquences

- Pendant la transition un Worker porte deux modèles d'empreinte (source en service, cible en préparation) : c'est la seule exception à « un modèle par type et par Worker », déjà admise pour les remplacements.
- Une grosse bibliothèque garde un moment des voix sans attribution machine après la bascule, jusqu'au rattrapage ; les décisions humaines ne bougent pas.
- Les empreintes de l'ancien modèle restent en base (hors production) tant qu'aucune purge explicite n'existe : à traiter dans un ticket à part.
- Le jeu de nombres de l'aperçu est calculé sur les données ; aucune lecture de média.
