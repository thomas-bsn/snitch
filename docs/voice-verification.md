# Vérification des voix inconnues et ambiguës (phase 5, ticket 37)

Un utilisateur parcourt une vue centrée sur les voix à trancher : les inconnues et les ambiguës (avec leurs Personnes candidates), les plus fréquentes d'abord. Les tests utilisent des empreintes synthétiques.

## Principes

- **Cas.** Un Locuteur local diarisé visible dont le niveau effectif est inconnu ou ambigu : ni validation humaine, ni attribution automatique, ni suggestion. Une suggestion n'est pas un cas (elle reste proposée à la confirmation sur le clip), la piste micro non plus.
- **Priorisation par fréquence.** Tri par nombre de clips distincts du groupe proposé de la voix (ticket 36), puis nombre de voix de ce groupe, puis temps de parole décroissant. Nommer une voix fréquente réduit le plus vite l'incertitude. Une voix sans lien est un groupe à elle seule (fréquence 1).
- **Trancher = validation humaine.** Aucune nouvelle écriture : un cas se tranche avec `POST /api/local-speakers/{id}/validation` (ticket 33), donc avec la même règle de première validation, le même Échantillon vocal validé, le même historique. Le rapprochement est relancé ensuite et les compteurs relus en base ; un cas tranché quitte la vue, et d'autres voix proches peuvent devenir des suggestions.
- **Cinq états distingués partout.** Validation humaine, attribution automatique, suggestion, ambigu, inconnu portent chacun un libellé et un glyphe (✔ ⚙ ≈ ⇄ ?) en plus de la couleur : en-tête de compteurs, chaque cas, lignes de locuteurs d'un clip.
- **Compteurs depuis la base.** Nombre de voix par niveau effectif sur les clips visibles de l'utilisateur ; ils ne sont jamais conservés côté client.
- **Aucune promesse de qualité.** L'interface affiche des volumes de voix par état, jamais un taux de fiabilité déduit du nombre d'extraits ou d'échantillons.
- **Visibilité.** Les voix de clips invisibles ne sont ni listées ni comptées, y compris dans le groupe d'une voix.

## API

`GET /api/voice-verification?level=all|unknown|ambiguous&page&page_size` (tout utilisateur connecté) : `{items, total, counts}`. Chaque cas : voix, clip, temps de parole, niveau (`unknown` ou `ambiguous`), `candidates` (ambigu), groupe (`group_id`, `group_voice_count`, `group_clip_count`). Extraits : `GET /api/local-speakers/{id}/excerpt`. Niveau ou pagination invalides : `400`.

Interface : carte « Vérification des voix » (`#voice-verification`, lien « Vérifier les voix »). Parcours Playwright : `web/e2e/voice-verification.spec.ts` (flag `SNITCH_E2E_VOICE_VERIFICATION=1`) ; l'ambigu et les candidates sont couverts par le test d'intégration Go (`voice_verification_test.go`).

## Limites

La fréquence d'une voix ambiguë est celle de son groupe proposé, pas du nombre de voix partageant les mêmes candidates. Le calcul des groupes est complet à chaque lecture (phase 7 si les volumes l'exigent).
