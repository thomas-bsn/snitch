import { test } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import {
  ADMIN,
  installAdministrator,
  sessionFor,
  setAccountLanguage,
} from "./shell-fixture";
import { seedUiCompare } from "./ui-compare-fixture";

// Comparaison visuelle (ticket #251) : capture l'application réelle, avec le jeu de données
// synthétique, dans les mêmes états que docs/ui-references, puis compose une image
// « référence | application » par état. Lancé par scripts/ui-compare.sh uniquement.

test.skip(
  process.env.SNITCH_UI_COMPARE !== "1",
  "lancé par scripts/ui-compare.sh",
);

const repo = path.resolve(process.cwd(), "..");
const references = path.join(repo, "docs/ui-references");
const out = process.env.SNITCH_UI_OUT ?? path.join(repo, "ui-compare-out");
const manifest = JSON.parse(
  fs.readFileSync(path.join(references, "states.json"), "utf8"),
) as {
  widths: number[];
  themes: ("light" | "dark")[];
  states: {
    screen: string;
    state: string;
    app: string;
    prototypeClick?: string;
  }[];
};

test("capture et comparaison des états", async ({ browser }) => {
  test.setTimeout(10 * 60 * 1000);
  await installAdministrator();
  const { meetingId } = await seedUiCompare();
  await setAccountLanguage(ADMIN, "fr");
  const cookies = (await sessionFor(ADMIN)).cookies;
  const only = process.env.SNITCH_UI_ONLY;
  const files: string[] = [];

  for (const s of manifest.states) {
    if (only && s.screen !== only) continue;
    for (const theme of manifest.themes) {
      for (const width of manifest.widths) {
        const context = await browser.newContext({
          viewport: { width, height: 900 },
          colorScheme: theme,
          reducedMotion: "reduce",
        });
        await context.addCookies(cookies);
        const page = await context.newPage();
        await page.goto(s.app.replace("{clip}", String(meetingId)), {
          waitUntil: "networkidle",
        });
        if (s.prototypeClick === "Icônes") {
          await page
            .getByRole("button", { name: /Icônes|Icons/ })
            .first()
            .click({ timeout: 3000 })
            .catch(() => undefined);
        }
        await page.waitForTimeout(400);
        const name = `${s.state}-${theme}-${width}.png`;
        const dir = path.join(out, "current", s.screen);
        fs.mkdirSync(dir, { recursive: true });
        await page.screenshot({ path: path.join(dir, name), fullPage: true });
        files.push(path.join(s.screen, name));
        await context.close();
      }
    }
  }

  // Composites « référence | application » rendus par le navigateur lui-même.
  const composer = await browser.newPage({
    viewport: { width: 3000, height: 900 },
  });
  for (const file of files) {
    const reference = path.join(references, file);
    if (!fs.existsSync(reference)) continue;
    const data = (p: string) =>
      `data:image/png;base64,${fs.readFileSync(p).toString("base64")}`;
    await composer.setContent(
      `<body style="margin:0;background:#888;display:flex;gap:12px;align-items:flex-start">
        <figure style="margin:0"><figcaption style="font:14px sans-serif;color:#fff">Référence (prototype)</figcaption><img src="${data(reference)}"></figure>
        <figure style="margin:0"><figcaption style="font:14px sans-serif;color:#fff">Application</figcaption><img src="${data(path.join(out, "current", file))}"></figure>
      </body>`,
    );
    const dir = path.join(out, "diff", path.dirname(file));
    fs.mkdirSync(dir, { recursive: true });
    await composer.screenshot({
      path: path.join(out, "diff", file),
      fullPage: true,
    });
  }
  await composer.close();
});
