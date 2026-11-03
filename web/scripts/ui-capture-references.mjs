// Capture les images de référence des prototypes (fiche C, People D, traitement D,
// Bibliothèque D, Import A) vers docs/ui-references/<écran>/<état>-<thème>-<largeur>.png.
// Usage : PROTO_URL=http://localhost:5199 node web/scripts/ui-capture-references.mjs [écran]
// Le serveur du prototype se lance depuis la branche prototype/fiche-video :
//   cd web && npm run dev -- --port 5199
import { chromium } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../..",
);
const manifest = JSON.parse(
  fs.readFileSync(path.join(root, "docs/ui-references/states.json"), "utf8"),
);
const base = process.env.PROTO_URL ?? "http://localhost:5199";
const only = process.argv[2]; // filtre optionnel sur l'écran

const browser = await chromium.launch();
for (const s of manifest.states) {
  if (only && s.screen !== only) continue;
  for (const theme of manifest.themes) {
    for (const width of manifest.widths) {
      const context = await browser.newContext({
        viewport: { width, height: 900 },
        colorScheme: theme,
        reducedMotion: "reduce",
      });
      const page = await context.newPage();
      await page.goto(base + s.prototype, { waitUntil: "networkidle" });
      if (s.prototypeClick) {
        await page
          .getByRole("button", { name: new RegExp("^" + s.prototypeClick) })
          .first()
          .click();
      }
      await page.waitForTimeout(300);
      const dir = path.join(root, "docs/ui-references", s.screen);
      fs.mkdirSync(dir, { recursive: true });
      await page.screenshot({
        path: path.join(dir, `${s.state}-${theme}-${width}.png`),
        fullPage: true,
      });
      await context.close();
    }
  }
}
await browser.close();
