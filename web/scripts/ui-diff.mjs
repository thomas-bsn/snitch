// Diff de pixels entre docs/ui-references et les captures de l'application.
// Usage : node web/scripts/ui-diff.mjs <dossier-de-sortie>
// Écrit <sortie>/report.tsv (écart par état, trié du pire au meilleur) et, pour chaque état,
// <sortie>/heat/<écran>/<fichier>.png : la carte des pixels différents (rouge) sur la capture.
// L'écart est le pourcentage de pixels différents sur la zone commune (seuil de couleur 24/255)
// ; la différence de hauteur de page est donnée à part, car une page plus longue ou plus courte
// est déjà un écart de mise en page.
import { PNG } from "pngjs";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const references = path.join(root, "docs/ui-references");
const out = path.resolve(process.argv[2] ?? path.join(root, "ui-compare-out"));
const THRESHOLD = 24;

const read = (p) => PNG.sync.read(fs.readFileSync(p));
const rows = [];
for (const screen of fs.readdirSync(path.join(out, "current"))) {
  for (const file of fs.readdirSync(path.join(out, "current", screen))) {
    const ref = path.join(references, screen, file);
    if (!fs.existsSync(ref)) continue;
    const a = read(ref);
    const b = read(path.join(out, "current", screen, file));
    const w = Math.min(a.width, b.width);
    const h = Math.min(a.height, b.height);
    const heat = new PNG({ width: w, height: h });
    let diff = 0;
    for (let y = 0; y < h; y++) {
      for (let x = 0; x < w; x++) {
        const i = (y * a.width + x) * 4;
        const j = (y * b.width + x) * 4;
        const k = (y * w + x) * 4;
        const d = Math.max(
          Math.abs(a.data[i] - b.data[j]),
          Math.abs(a.data[i + 1] - b.data[j + 1]),
          Math.abs(a.data[i + 2] - b.data[j + 2]),
        );
        const bad = d > THRESHOLD;
        if (bad) diff++;
        heat.data[k] = bad ? 255 : b.data[j] >> 1;
        heat.data[k + 1] = bad ? 0 : b.data[j + 1] >> 1;
        heat.data[k + 2] = bad ? 0 : b.data[j + 2] >> 1;
        heat.data[k + 3] = 255;
      }
    }
    const dir = path.join(out, "heat", screen);
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(path.join(dir, file), PNG.sync.write(heat));
    rows.push({
      file: `${screen}/${file}`,
      pct: (100 * diff) / (w * h),
      dh: b.height - a.height,
    });
  }
}
rows.sort((x, y) => y.pct - x.pct);
const report = ["fichier\técart_pixels_%\tdelta_hauteur_px"]
  .concat(rows.map((r) => `${r.file}\t${r.pct.toFixed(1)}\t${r.dh}`))
  .join("\n");
fs.writeFileSync(path.join(out, "report.tsv"), report + "\n");
console.log(report);
