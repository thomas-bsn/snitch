import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import {
  correctSpeaker,
  correctText,
  nameVoice,
  transcribeAndDiarize,
  uploadClip,
} from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Journeys of the search screen of the new interface, on a synthetic library built through
// the API: one shared clip transcribed and diarized (Alice speaks the first two segments,
// the third voice is only a suggestion) and one private clip with no transcript.

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];

async function expectNoViolations(page: Page) {
  const { violations } = await new AxeBuilder({ page })
    .withTags(TAGS)
    .analyze();
  expect(
    violations.map(
      (v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(" | ")}`,
    ),
  ).toEqual([]);
}

test.describe.serial("recherche par citation", () => {
  let sharedId = 0;

  test.beforeAll(async () => {
    await installAdministrator();
    sharedId = await uploadClip("synthetic-shared.mp4", "shared");
    await uploadClip("synthetic-private.mp4", "private");
    const [first] = await transcribeAndDiarize(sharedId);
    await nameVoice(first, "Alice");
    await correctText(sharedId, 0, "Bien joué, Alice !", "nom");
    await correctSpeaker(sharedId, 1, first);
  });

  test("le champ de l'en-tête mène à la recherche et un résultat ouvre le clip au bon moment", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/library");
    await page.getByLabel("Search the transcripts").first().fill("JOUE");
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/search\?q=JOUE$/);
    await expect(
      page.getByRole("heading", { level: 1, name: "Search" }),
    ).toBeVisible();
    await expect(
      page.getByRole("searchbox", { name: "What was said" }),
    ).toHaveValue("JOUE");

    // The quote, with the passage found highlighted, its timecode, clip and speaker.
    const results = page.getByRole("list", { name: "Passages found" });
    const item = results.getByRole("listitem");
    await expect(item).toHaveCount(1);
    await expect(item).toContainText("Bien joué, Alice !");
    await expect(item.locator("mark")).toHaveText("joué");
    await expect(item).toContainText("synthetic-shared.mp4");
    await expect(item).toContainText("Alice");
    await expect(item).toContainText("Validated by a human");
    await expect(item).toContainText("Exact match");
    await expect(item).toContainText("Corrected");
    // The count and the mode in use are visible.
    await expect(page.getByText("1 result · Mode: Full text")).toBeVisible();

    // The timecode opens the clip at that moment.
    await item
      .getByRole("link", { name: "Open synthetic-shared.mp4 at 0:01" })
      .click();
    await expect(page).toHaveURL(new RegExp(`/clips/${sharedId}\\?t=1$`));
    await expect(
      page.getByRole("heading", { level: 1, name: "synthetic-shared.mp4" }),
    ).toBeVisible();

    // And leads back to the same search.
    await page.getByRole("link", { name: "Search results" }).click();
    await expect(page).toHaveURL(/\/search\?q=JOUE$/);
    await expect(item).toHaveCount(1);
    await context.close();
  });

  test("les modes, les filtres et le filtre par Personne", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/search");
    await expect(page.getByText("Type a word or a phrase")).toBeVisible();

    const query = page.getByRole("searchbox", { name: "What was said" });
    const items = page
      .getByRole("list", { name: "Passages found" })
      .getByRole("listitem");

    await query.fill("merci");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await expect(items).toHaveCount(1);
    await page
      .locator("label")
      .filter({ hasText: /^Exact$/ })
      .click();
    await expect(page).toHaveURL(/mode=exact/);
    await expect(items).toHaveCount(1);
    await expect(items.first()).toContainText("Exact match");
    await expect(page.getByText("Mode: Exact")).toBeVisible();
    await page
      .locator("label")
      .filter({ hasText: /^Full text$/ })
      .click();
    await expect(page).not.toHaveURL(/mode=/);

    // The Person filter: "À bientôt" is spoken by a voice nobody identified, "Merci" by Alice.
    const bientot = items.filter({ hasText: "À bientôt" });
    await query.fill("bientôt");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await expect(bientot).toHaveCount(1);
    await expect(bientot).toContainText("Unknown speaker");
    await page.getByLabel("Person").selectOption({ label: "Alice" });
    await expect(page).toHaveURL(/person=\d+/);
    await expect(bientot).toHaveCount(0);
    await page.getByLabel("Person").selectOption({ label: "Unknown voices" });
    await expect(page).toHaveURL(/person=unknown/);
    await expect(bientot).toHaveCount(1);
    await page.getByLabel("Person").selectOption({ label: "Alice" });
    await query.fill("merci");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await expect(items).toHaveCount(1);
    await expect(items.first()).toContainText("Alice");
    await page.getByLabel("Person").selectOption({ label: "Unknown voices" });
    await expect(items.filter({ hasText: "Merci" })).toHaveCount(0);
    await page.getByRole("button", { name: "Clear the filters" }).click();
    await expect(items).toHaveCount(1);
    await expect(items.first()).toContainText("Merci");

    // The other filters narrow the results too.
    await page.getByLabel("Visibility").selectOption("private");
    await expect(
      page.getByRole("heading", { name: "Nothing found" }),
    ).toBeVisible();
    await page.getByLabel("Visibility").selectOption("shared");
    await expect(items).toHaveCount(1);
    await page.getByLabel("State").selectOption({ label: "Missing" });
    await expect(
      page.getByRole("heading", { name: "Nothing found" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Clear the filters" }).click();
    await page.getByLabel("From").fill("2099-01-01");
    await expect(
      page.getByRole("heading", { name: "Nothing found" }),
    ).toBeVisible();
    await context.close();
  });

  test("l'indisponibilité des embeddings est dite et le type de correspondance affiché", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    let calls = 0;
    await page.route("**/api/search?**", async (route) => {
      calls += 1;
      const result = (match_type: string, timestamp: number) => ({
        clip_id: sharedId,
        clip_name: "synthetic-shared.mp4",
        visibility: "shared",
        recorded_at: new Date().toISOString(),
        status: "ready",
        sources: [],
        match_type,
        context: "Bien joué !",
        matched_text: "Bien joué",
        timestamp,
        corrected: false,
        person: null,
        decision_level: "unknown",
      });
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          calls === 1
            ? {
                items: [result("semantic", 1.5), result("approximate", 2.5)],
                page: 1,
                page_size: 20,
                total: 2,
              }
            : {
                items: [result("full_text", 1.5)],
                page: 1,
                page_size: 20,
                total: 1,
                degraded: true,
              },
        ),
      });
    });

    await page.goto("/search?q=intention&mode=hybrid");
    await expect(page.getByText("Semantic match")).toBeVisible();
    await expect(page.getByText("Approximate match")).toBeVisible();
    await expect(page.getByText("Mode: Hybrid")).toBeVisible();
    await expect(page.getByText("Semantic search is unavailable")).toHaveCount(
      0,
    );

    await page.getByRole("searchbox", { name: "What was said" }).fill("autre");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await expect(
      page.getByText(
        "Semantic search is unavailable: the results come from full-text search.",
      ),
    ).toBeVisible();
    await expect(page.getByText("Mode: Full text")).toBeVisible();
    await context.close();
  });

  for (const theme of ["light", "dark"] as const) {
    test(`accessibilité de la recherche, thème ${theme}`, async ({
      browser,
    }) => {
      const context = await browser.newContext({
        storageState: await sessionFor(ADMIN),
      });
      await context.addInitScript((value) => {
        localStorage.setItem("snitch-ui-theme", value);
      }, theme);
      const page = await context.newPage();
      await page.goto("/search");
      await expect(page.getByText("Type a word or a phrase")).toBeVisible();
      await expectNoViolations(page);
      await page.goto("/search?q=joue");
      await expect(
        page.getByRole("list", { name: "Passages found" }),
      ).toBeVisible();
      await expectNoViolations(page);
      await page.goto("/search?q=zzzzzz");
      await expect(
        page.getByRole("heading", { name: "Nothing found" }),
      ).toBeVisible();
      await expectNoViolations(page);
      await context.close();
    });
  }

  test("téléphone : pas de défilement horizontal", async ({ browser }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
      viewport: { width: 390, height: 780 },
    });
    const page = await context.newPage();
    await page.goto("/search?q=joue");
    await expect(
      page.getByRole("list", { name: "Passages found" }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await expectNoViolations(page);
    await context.close();
  });
});
