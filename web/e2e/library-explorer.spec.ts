import fs from "node:fs";
import path from "node:path";
import AxeBuilder from "@axe-core/playwright";
import { expect, request, test, type Page } from "@playwright/test";
import {
  ADMIN,
  MEMBER,
  createApprovedMember,
  installAdministrator,
  sessionFor,
} from "./shell-fixture";

// The explorer of the Bibliothèque (#195, ADR 0032) over a synthetic tree: an imported clip, unindexed
// videos, a subfolder and a private folder. Opening an unindexed video indexes it without analysis.
// The root is configured through the API; no real media is ever read.

const ROOT = process.env.SNITCH_TEST_EXPLORER_ROOT;

async function expectNoViolations(page: Page) {
  const { violations } = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(
    violations.map(
      (v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(" | ")}`,
    ),
  ).toEqual([]);
}

// The rows of the list view only: the tree of folders repeats the same names.
const row = (page: Page, name: string) =>
  page.getByRole("table").getByRole("row").filter({ hasText: name });

test.describe.serial("explorateur de la bibliothèque", () => {
  test.skip(!ROOT, "requires the explorer harness");

  test.beforeAll(async () => {
    await installAdministrator();
    const admin = await sessionFor(ADMIN);
    const api = await request.newContext({
      baseURL: process.env.SNITCH_TEST_URL!,
      extraHTTPHeaders: { Origin: process.env.SNITCH_TEST_URL! },
      storageState: admin,
    });
    const me = await (await api.get("/api/auth/me")).json();
    const headers = { "X-CSRF-Token": me.csrf_token };
    const configured = await api.put("/api/admin/video-root", {
      headers,
      data: { path: ROOT },
    });
    expect(configured.status()).toBe(200);
    const sources = await (await api.get("/api/admin/video-root")).json();
    const folders = await api.put("/api/admin/library/folders/privacy", {
      headers,
      data: { source_id: sources.root.id, folder: "secret", private: true },
    });
    expect(folders.status()).toBe(200);
    await api.dispose();
    await createApprovedMember();
  });

  test.beforeEach(async ({ context }) => {
    await context.addCookies((await sessionFor(ADMIN)).cookies);
  });

  test("le dossier privé garde son nom et son cadenas, les vidéos non indexées le disent", async ({
    page,
  }) => {
    await page.goto("/library");
    await expect(
      page.getByRole("heading", { level: 1, name: "Library" }),
    ).toBeVisible();
    await expect(row(page, "top.mp4")).toContainText("Not indexed");
    await expect(row(page, "league")).toBeVisible();
    await expect(row(page, "secret")).toContainText("(private folder)");
    await expect(page.getByText("plan.mp4")).toHaveCount(0);
    await expectNoViolations(page);
  });

  test("ouvrir une vidéo non indexée l'indexe sans l'analyser, puis ouvre sa fiche", async ({
    page,
  }) => {
    await page.goto("/library?folder=league%2F2026");
    await row(page, "other.mp4")
      .getByRole("button", { name: "other.mp4", exact: true })
      .click();
    await expect(page).toHaveURL(/\/clips\/\d+$/);
    await expect(
      page.getByRole("heading", { name: "other.mp4" }),
    ).toBeVisible();
    await page.goto("/library?folder=league%2F2026");
    await expect(row(page, "other.mp4")).toContainText("Not analysed");
  });

  test("une fois un seul clip importé, le dossier et ce clip restent accessibles et s'ouvrent", async ({
    page,
  }) => {
    await page.goto("/library?folder=league%2F2026");
    await expect(row(page, "sub")).toBeVisible();
    await expect(row(page, "match.mp4")).toContainText("Not indexed");
    await expect(row(page, "other.mp4")).toContainText("Not analysed");
    await row(page, "sub")
      .getByRole("button", { name: "sub", exact: true })
      .click();
    await expect(page).toHaveURL(/folder=league%2F2026%2Fsub/);
    await expect(row(page, "inner.mp4")).toContainText("Not indexed");
    await page.goto("/library?folder=league%2F2026");
    await row(page, "match.mp4")
      .getByRole("button", { name: "match.mp4", exact: true })
      .click();
    await expect(page).toHaveURL(/\/clips\/\d+$/);
    await expect(
      page.getByRole("heading", { name: "match.mp4" }),
    ).toBeVisible();
  });

  test("l'arbre liste tous les sous-dossiers, au-delà d'une page de 100 (#271)", async ({
    page,
  }) => {
    await page.goto("/library?folder=bulk");
    // The list shows 100 per page, so only the tree can show the 120th folder here.
    await expect(
      page.getByRole("button", { name: "dossier-120", exact: true }),
    ).toBeVisible();
  });

  test("un dossier créé sur le disque apparaît après Actualiser, sans rescan (#271)", async ({
    page,
  }) => {
    await page.goto("/library");
    const created = path.join(ROOT!, "cree-apres-ouverture");
    await expect(
      page.getByRole("button", { name: "cree-apres-ouverture" }),
    ).toHaveCount(0);
    fs.mkdirSync(created);
    try {
      await page.getByRole("button", { name: "Refresh", exact: true }).click();
      await expect(
        page.getByRole("button", { name: "cree-apres-ouverture" }).first(),
      ).toBeVisible();
      fs.rmdirSync(created);
      await page.getByRole("button", { name: "Refresh", exact: true }).click();
      await expect(
        page.getByRole("button", { name: "cree-apres-ouverture" }),
      ).toHaveCount(0);
    } finally {
      fs.rmSync(created, { recursive: true, force: true });
    }
  });

  test("un filtre donne des résultats à plat avec leur dossier", async ({
    page,
  }) => {
    await page.goto("/library?filter=inner");
    await expect(
      page.getByRole("columnheader", { name: "Folder" }),
    ).toBeVisible();
    await expect(row(page, "inner.mp4")).toContainText("league/2026/sub");
    await expect(row(page, "top.mp4")).toHaveCount(0);
  });

  test("le filtre survit au rafraîchissement, au retour/avant et se combine avec un autre filtre", async ({
    page,
  }) => {
    const box = page.getByRole("searchbox", { name: "Search in this folder" });
    await page.goto("/library?filter=inner&visibility=shared");
    await expect(box).toHaveValue("inner");
    await page.reload();
    await expect(box).toHaveValue("inner");
    await expect(page.getByLabel("Visibility", { exact: true })).toHaveValue(
      "shared",
    );
    await box.fill("top");
    await expect(page).toHaveURL(/filter=top/);
    await expect(page).toHaveURL(/visibility=shared/);
    await page.goBack();
    await expect(box).toHaveValue("inner");
    await page.goForward();
    await expect(box).toHaveValue("top");
  });

  test("un ancien lien ?q= n'applique aucun filtre à la bibliothèque", async ({
    page,
  }) => {
    await page.goto("/library?q=inner");
    await expect(
      page.getByRole("searchbox", { name: "Search in this folder" }),
    ).toHaveValue("");
    await expect(row(page, "top.mp4")).toBeVisible();
  });

  test("la vue icônes et la vue liste se basculent et restent accessibles", async ({
    page,
  }) => {
    await page.goto("/library?folder=league%2F2026");
    await page.getByRole("button", { name: "Icons" }).click();
    await expect(page.getByRole("button", { name: "Icons" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    await expect(page.getByRole("link", { name: /match\.mp4/ })).toBeVisible();
    await expectNoViolations(page);
    await page.getByRole("button", { name: "List" }).click();
    await expect(page.getByRole("table")).toBeVisible();
  });

  test("en vue icônes, une vidéo indexée affiche sa miniature, une non indexée garde la tuile en pointillés", async ({
    page,
  }) => {
    const api = await request.newContext({
      baseURL: process.env.SNITCH_TEST_URL!,
      extraHTTPHeaders: { Origin: process.env.SNITCH_TEST_URL! },
      storageState: await sessionFor(ADMIN),
    });
    const me = await (await api.get("/api/auth/me")).json();
    const indexed = await api.post("/api/library/explorer/index", {
      headers: { "X-CSRF-Token": me.csrf_token },
      data: { path: "media/clip.mp4" },
    });
    expect(indexed.ok()).toBe(true);
    await api.dispose();
    // The indexing is asynchronous: the clip exists once the file is stable.
    await expect(async () => {
      await page.goto("/library?folder=media&view=icons");
      await expect(page.getByRole("link", { name: /clip\.mp4/ })).toBeVisible({
        timeout: 1000,
      });
    }).toPass({ timeout: 15000 });
    const withClip = page.getByRole("link", { name: /clip\.mp4/ });
    await expect(withClip).toBeVisible();
    await expect(withClip.locator("[data-thumbnail]")).toHaveAttribute(
      "data-thumbnail",
      "ready",
    );
    const image = withClip.locator("img");
    await expect(image).toHaveCount(1);
    expect(
      await image.evaluate(
        (img: HTMLImageElement) => img.complete && img.naturalWidth > 0,
      ),
    ).toBe(true);
    // Computed styles: the picture fills the 80x56 tile like the dark tile it replaces.
    const box = await image.evaluate((img: HTMLImageElement) => {
      const tile = getComputedStyle(img.parentElement!);
      const style = getComputedStyle(img);
      return {
        tile: [tile.width, tile.height],
        fit: style.objectFit,
        radius: style.borderTopLeftRadius,
        natural: [img.naturalWidth, img.naturalHeight],
      };
    });
    expect(box.tile).toEqual(["80px", "56px"]);
    expect(box.fit).toBe("cover");
    expect(box.radius).not.toBe("0px");
    const plain = page.getByRole("button", { name: /plain\.mp4/ });
    await expect(plain).toBeVisible();
    await expect(plain.locator("img")).toHaveCount(0);
    await expect(plain.locator("[data-thumbnail]")).toHaveCount(0);
    await expect(plain.locator("span.border-dashed").first()).toBeVisible();
  });

  test("un membre ne voit ni le contenu ni le chemin d'un dossier privé", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(MEMBER),
    });
    const page = await context.newPage();
    await page.goto("/library");
    await expect(row(page, "secret")).toContainText("(private folder)");
    await expect(page.getByText("plan.mp4")).toHaveCount(0);
    await page.goto("/library?folder=secret");
    await expect(page.getByText("plan.mp4")).toHaveCount(0);
    await context.close();
  });

  test("l'analyse d'un dossier indexe d'abord ses vidéos non indexées", async ({
    page,
  }) => {
    await page.goto("/library?folder=league%2F2026");
    await expect(row(page, "sub")).toBeVisible();
    await row(page, "sub")
      .getByRole("button", { name: "Actions for folder sub" })
      .click();
    await page
      .getByRole("menuitem", { name: "Analyse the unanalysed videos" })
      .click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText(
      "Videos not indexed yet (1) are indexed first, then analysed.",
    );
    await dialog.getByRole("button", { name: "Analyse 1 videos" }).click();
    await expect(dialog).toBeHidden({ timeout: 30000 });
    // The video of the folder now has a clip: it is no longer unindexed.
    await page.goto("/library?folder=league%2F2026%2Fsub");
    await expect(row(page, "inner.mp4")).not.toContainText("Not indexed");
  });

  test("la pastille d'une vidéo en cours d'indexation s'anime puis se met à jour seule", async ({
    page,
  }) => {
    // Synthetic answers: the third reading of the folder finds the video indexed.
    let readings = 0;
    await page.route("**/api/library/explorer?*", async (route) => {
      const url = new URL(route.request().url());
      if (
        url.searchParams.get("recursive") === "1" ||
        url.searchParams.get("path") !== "league/2026"
      ) {
        return route.fallback();
      }
      readings += 1;
      const items = [
        {
          name: "live.mp4",
          path: "league/2026/live.mp4",
          folder: "league/2026",
          directory: false,
          size_bytes: 1,
          private: false,
          locked: false,
          state: readings >= 3 ? "analyzed" : "in_progress",
          clip_id: readings >= 3 ? 1 : null,
        },
      ];
      await route.fulfill({
        json: {
          source_id: 1,
          root_configured: true,
          items,
          total: items.length,
          states: [],
        },
      });
    });
    await page.goto("/library?folder=league%2F2026");
    const pill = row(page, "live.mp4").locator("[data-indexing]");
    await expect(pill).toBeVisible();
    await expect(row(page, "live.mp4")).toContainText("In progress");
    // No reload of the page, no click: the listing is read again and the pill turns.
    await expect(row(page, "live.mp4")).toContainText("Analysed", {
      timeout: 15000,
    });
    await expect(pill).toHaveCount(0);
  });

  test("une vidéo encore en cours d'indexation après le délai n'est pas un échec", async ({
    page,
  }) => {
    // Synthetic answers: the video is never indexed within the wait, as on a busy scan.
    const stuck = {
      name: "stuck.mp4",
      path: "stuck.mp4",
      folder: "",
      directory: false,
      size_bytes: 1,
      private: false,
      locked: false,
      state: "in_progress",
      clip_id: null,
    };
    let indexRequests = 0;
    await page.route("**/api/library/explorer/index", async (route) => {
      indexRequests += 1;
      await route.fulfill({
        status: 202,
        json: { path: "stuck.mp4", state: "in_progress" },
      });
    });
    await page.route("**/api/library/explorer?*", async (route) => {
      const url = new URL(route.request().url());
      if (
        url.searchParams.get("recursive") !== "1" ||
        !url.searchParams.get("path")
      ) {
        return route.fallback();
      }
      const state = url.searchParams.get("state");
      const items =
        (state === "in_progress" || !state) && indexRequests > 0
          ? [stuck]
          : state === "not_indexed" && indexRequests === 0
            ? [{ ...stuck, state: "not_indexed" }]
            : [];
      await route.fulfill({
        json: {
          source_id: 1,
          root_configured: true,
          items,
          total: items.length,
          states: [],
        },
      });
    });
    await page.clock.install();
    await page.goto("/library?folder=league%2F2026");
    await row(page, "sub")
      .getByRole("button", { name: "Actions for folder sub" })
      .click();
    await page
      .getByRole("menuitem", { name: "Analyse the unanalysed videos" })
      .click();
    const dialog = page.getByRole("alertdialog");
    await dialog.getByRole("button", { name: /^Analyse 1 videos/ }).click();
    // Each wait of the loop is a new timer: move the clock until the 60 polls have elapsed.
    for (let tick = 0; tick < 80; tick += 1) {
      await page.clock.fastForward(2000);
      await page.waitForTimeout(40);
    }
    await expect(
      page.getByText("1 videos are still being indexed: try again"),
    ).toBeVisible();
    await expect(page.getByText("refused")).toHaveCount(0);
    // The next click resumes the video instead of dropping it.
    await row(page, "sub")
      .getByRole("button", { name: "Actions for folder sub" })
      .click();
    await page
      .getByRole("menuitem", { name: "Analyse the unanalysed videos" })
      .click();
    await expect(page.getByRole("alertdialog")).toContainText(
      "Videos not indexed yet (1)",
    );
  });
});
