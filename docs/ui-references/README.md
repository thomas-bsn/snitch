# Références visuelles de la refonte UI

Captures des prototypes de la branche `prototype/fiche-video` (référence visuelle uniquement,
jamais fusionnée). Nom des fichiers : `<écran>/<état>-<thème>-<largeur>.png`, thèmes `light` et
`dark`, largeurs 1440 et 390 (pleine page). La barre noire de simulation du prototype est
masquée avant capture.

**Largeur 390 : références peu fiables.** La mise en page mobile du prototype est dégradée,
surtout `fiche-video` (titre cassé lettre à lettre, métadonnées étroites, bandeau d'actions
débordant) : ses captures `*-390` (`echec`, `voix`) ne sont pas une cible. Les autres états à
390 n'ont pas été relus en détail : à confirmer avant de s'y fier. Les références à 1440 sont
la cible.

`states.json` liste chaque état, son URL de prototype et la route correspondante de l'application.
Chaque état existe en 4 fichiers (2 thèmes x 2 largeurs).

| Écran (dossier) | État | URL source (prototype) |
| --- | --- | --- |
| `fiche-video` | `echec` (étape d'empreintes en échec) | `/prototype/fiche-video?variant=C` |
| `fiche-video` | `voix` | `/prototype/fiche-video?variant=C&tab=voices` |
| `people` | `annuaire` | `/prototype/people?variant=D` |
| `people` | `annuaire-personne` (clic sur la carte de Marc Lefèvre : fiche, Référence bonne) | `/prototype/people?variant=D` |
| `people` | `annuaire-personne-faible` (clic sur la carte de Thomas Roux : fiche, alerte Référence faible) | `/prototype/people?variant=D` |
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
compose une image « référence | application » et calcule un vrai diff de pixels (`pngjs`,
`web/scripts/ui-diff.mjs`). Sortie dans `ui-compare-out/` (hors suivi) : `current/` (captures),
`diff/` (côte à côte), `heat/` (pixels différents en rouge) et `report.tsv` (pourcentage de
pixels différents et écart de hauteur de page par état, du pire au meilleur ; seuil de couleur
24/255). La bannière « Finir la configuration » de l'app compte dans l'écart. Ouvrir les images
avec le skill `frontend-audit` ou `Read`.

Jeu de données : clip « Réunion de préparation — tournage S12 » (5 voix, 10 segments, étape
d'empreintes en échec), 6 personnes, 5 autres clips à traiter, arborescence `2026/...`
configurée comme racine vidéo. Les attributions des voix du clip principal sont écrites en base
par le harnais Go (`app/internal/httpapi/ui_compare_seed_test.go`, échange par fichier avec la
suite) : SPEAKER_00 validée « Marc Lefèvre », SPEAKER_01 suggestion « Julie Martin » (candidats
Julie Martin et Claire Dubois), SPEAKER_02 ambiguë (Thomas Roux, Hugo Bernard), SPEAKER_03 et 04
inconnues. Les autres clips n'ont que des voix inconnues.
