# Références visuelles de la refonte UI

Captures des prototypes de la branche `prototype/fiche-video` (référence visuelle uniquement,
jamais fusionnée). Nom des fichiers : `<écran>/<état>-<thème>-<largeur>.png`, thèmes `light` et
`dark`, largeurs 1440 et 390 (pleine page). Le prototype porte sa barre de simulation en bas
de page : elle fait partie des captures.

`states.json` liste chaque état, son URL de prototype et la route correspondante de l'application.
Chaque état existe en 4 fichiers (2 thèmes x 2 largeurs).

| Écran (dossier) | État | URL source (prototype) |
| --- | --- | --- |
| `fiche-video` | `echec` (étape d'empreintes en échec) | `/prototype/fiche-video?variant=C` |
| `fiche-video` | `voix` | `/prototype/fiche-video?variant=C&tab=voices` |
| `people` | `annuaire` | `/prototype/people?variant=D` |
| `people` | `a-traiter` (clic sur l'onglet « À traiter ») | `/prototype/people?variant=D` |
| `people` | `voix-inconnues` (clic sur l'onglet) | `/prototype/people?variant=D` |
| `traitement` | `traitement` | `/prototype/people?variant=D&traiter=40` |
| `bibliotheque` | `liste` | `/prototype/library?variant=D` |
| `bibliotheque` | `icones` (clic sur « Icônes ») | `/prototype/library?variant=D` |
| `import` | `import` | `/prototype/import?variant=A` |

## Regénérer les références

```
git worktree add ../snitch-proto-fiche prototype/fiche-video   # ou un dossier existant
cd ../snitch-proto-fiche/web && npm run dev -- --port 5199
# dans un autre terminal, depuis ce dépôt :
PROTO_URL=http://localhost:5199 node web/scripts/ui-capture-references.mjs [écran]
```

## Comparer l'application aux références

```
DOCKER_HOST=unix:///Users/$USER/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true \
  scripts/ui-compare.sh [écran]
```

Le script construit le web, lance l'application sur Postgres (testcontainers) via le harnais
d'intégration (`SNITCH_E2E_UI_COMPARE=1`, suite `web/e2e/ui-compare.spec.ts`), sème le jeu de
données synthétique (`web/e2e/ui-compare-fixture.ts`), capture les mêmes états en français puis
compose une image « référence | application ». Sortie dans `ui-compare-out/` (ignoré par git) :
`current/` (captures) et `diff/` (comparaisons). Ouvrir les images `diff/` avec le skill
`frontend-audit` ou `Read`.

Jeu de données : clip « Réunion de préparation — tournage S12 » (5 voix, 10 segments, étape
d'empreintes en échec), 6 personnes, 5 autres clips à traiter, arborescence `2026/...` du
dossier source. Limite : sans empreintes sur le clip principal, ses voix sont « inconnues »
(les états « suggestion » et « ambiguë » du prototype ne sont pas reproduits).
