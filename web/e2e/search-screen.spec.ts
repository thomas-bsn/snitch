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
// the third voice is only a suggestion) and one private clip with no transcript. Search is
// always exact: a word or a phrase, ignoring case, accents and the punctuation between words.

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

/** Types in the header field, the only way into the search screen. */
async function search(page: Page, query: string) {
  const box = page.getByRole("searchbox", { name: "Search the transcripts" });
  await box.fill(query);
  await box.press("Enter");
}

test.describe.serial("recherche par citation", () => {
  let sharedId = 0;
  let privateId = 0;

  test.beforeAll(async () => {
    await installAdministrator();
    sharedId = await uploadClip("synthetic-shared.mp4", "shared");
    privateId = await uploadClip("synthetic-private.mp4", "private");
    const [first] = await transcribeAndDiarize(sharedId);
    await nameVoice(first, "Alice");
    await correctText(sharedId, 0, "Bien joué, Alice !", "nom");
    await correctSpeaker(sharedId, 1, first);
  });

  const passage = (clip_id: number, clip_name: string, timestamp: number) => ({
    clip_id,
    clip_name,
    visibility: "shared",
    recorded_at: new Date().toISOString(),
    status: "ready",
    sources: [],
    match_type: "exact",
    context: "Il lance une grenade.",
    matched_text: "grenade",
    timestamp,
    corrected: false,
    person: null,
    decision_level: "unknown",
  });

  /** Five passages found, four of them in the same clip. */
  async function stubPassages(page: Page) {
    await page.route("**/api/search?**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          items: [
            passage(sharedId, "synthetic-shared.mp4", 13),
            passage(sharedId, "synthetic-shared.mp4", 9),
            passage(privateId, "synthetic-private.mp4", 2),
            passage(sharedId, "synthetic-shared.mp4", 5),
            passage(sharedId, "synthetic-shared.mp4", 1),
          ],
          page: 1,
          page_size: 20,
          total: 5,
        }),
      });
    });
  }

  test("le champ de l'en-tête mène à la recherche et un résultat ouvre le clip au bon moment", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/library");
    await search(page, "JOUE");
    await expect(page).toHaveURL(/\/search\?q=JOUE$/);
    await expect(
      page.getByRole("heading", { level: 1, name: "Search" }),
    ).toBeVisible();
    await expect(
      page.getByRole("searchbox", { name: "Search the transcripts" }),
    ).toHaveValue("JOUE");
    await expect(
      page.getByText("Exact search · case- and accent-insensitive"),
    ).toBeVisible();
    await expect(page.getByText(/1 passage in 1 clip for/)).toBeVisible();

    // The passages are grouped by clip: the group names the clip, the voices and the count.
    const region = page.getByRole("region", { name: "synthetic-shared.mp4" });
    await expect(region).toBeVisible();
    await expect(
      region.getByRole("heading", { name: "synthetic-shared.mp4" }),
    ).toBeVisible();
    await expect(region.getByText(/1 passage · 1 voice/)).toBeVisible();

    // One passage: the quote with the hit highlighted, the speaker and their picture.
    const item = region
      .getByRole("list", { name: "Passages found in synthetic-shared.mp4" })
      .getByRole("listitem");
    await expect(item).toHaveCount(1);
    await expect(item).toContainText("Bien joué, Alice !");
    await expect(item.locator("mark")).toHaveText("joué");
    await expect(item).toContainText("Alice");
    await expect(item).toContainText("Validated by a human");
    await expect(item).toContainText("Corrected");
    // The clip identity lives in the group header, not repeated in every passage.
    await expect(item).not.toContainText("synthetic-shared.mp4");

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

  test("la bande des clips suit les clips, pas les passages trouvés", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await stubPassages(page);

    await page.goto("/search?q=grenade");
    const items = page
      .getByRole("list", { name: /Passages found in/ })
      .getByRole("listitem");
    await expect(items).toHaveCount(5);

    // Opening any passage opens its clip; the strip counts the clip once, at its first
    // passage, and says how many were found.
    await items
      .first()
      .getByRole("link", { name: "Open synthetic-shared.mp4 at 0:13" })
      .click();
    await expect(page).toHaveURL(new RegExp(`/clips/${sharedId}\\?t=13$`));
    const tiles = page.locator("[data-strip-clip]");
    await expect(tiles).toHaveCount(2);
    const shared = page.locator(`[data-strip-clip="${sharedId}"]`);
    await expect(shared).toContainText("×4");
    await expect(shared).toContainText("at 0:01");
    await expect(shared.getByText("4 passages found")).toBeAttached();

    // The arrow goes to the other clip, at its own first passage, not to the next
    // passage of the same one.
    await page
      .getByRole("button", { name: "Next clip: synthetic-private.mp4" })
      .click();
    await expect(page).toHaveURL(new RegExp(`/clips/${privateId}\\?t=2$`));
    await expect(
      page.getByRole("heading", { level: 1, name: "synthetic-private.mp4" }),
    ).toBeVisible();
    await expect(tiles).toHaveCount(2);
    await expect(shared).toContainText("×4");
    await context.close();
  });

  test("la bande passe d'un clip à l'autre, se replie et ramène à la recherche", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await stubPassages(page);
    await page.goto("/search?q=grenade");
    await page
      .getByRole("list", { name: /Passages found in/ })
      .getByRole("listitem")
      .first()
      .getByRole("link", { name: "Open synthetic-shared.mp4 at 0:13" })
      .click();
    await expect(page).toHaveURL(new RegExp(`/clips/${sharedId}\\?t=13$`));
    await expectNoViolations(page);

    // The strip folds, and the choice stays on the device.
    await page.getByRole("button", { name: "Hide clips" }).click();
    await expect(page.locator("#clip-strip")).toHaveCount(0);
    await page.reload();
    await expect(
      page.getByRole("button", { name: "Show clips" }),
    ).toBeVisible();
    await expect(page.locator("#clip-strip")).toHaveCount(0);
    await page.getByRole("button", { name: "Show clips" }).click();
    await expect(page.locator("#clip-strip")).toBeVisible();

    // The arrow goes to the other clip, at its own first passage.
    await page
      .getByRole("button", { name: "Next clip: synthetic-private.mp4" })
      .click();
    await expect(page).toHaveURL(new RegExp(`/clips/${privateId}\\?t=2$`));
    await expect(
      page.getByRole("heading", { level: 1, name: "synthetic-private.mp4" }),
    ).toBeVisible();

    // Maj+P comes back, still clip by clip.
    await page.keyboard.press("Shift+P");
    await expect(page).toHaveURL(new RegExp(`/clips/${sharedId}\\?t=1$`));
    await expect(
      page.getByRole("heading", { level: 1, name: "synthetic-shared.mp4" }),
    ).toBeVisible();

    // The arrows replaced the fiche's entry: one step returns to the search.
    await page.getByRole("link", { name: "Search results" }).click();
    await expect(page).toHaveURL(/\/search\?q=grenade$/);
    await expect(
      page.getByRole("list", { name: /Passages found in/ }),
    ).toHaveCount(2);
    await context.close();
  });

  test("Bibliothèque et Recherche partagent les filtres dans une URL partageable", async ({
    browser,
  }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/search?q=jou%C3%A9&visibility=shared&state=ready");
    await expect(
      page.getByRole("searchbox", { name: "Search the transcripts" }),
    ).toHaveValue("joué");
    await expect(
      page.getByText("Exact search · case- and accent-insensitive"),
    ).toBeVisible();
    await page
      .getByRole("navigation", { name: "Main navigation" })
      .getByRole("link", { name: "Library" })
      .click();
    await expect(page).toHaveURL(/\/library\?/);
    await expect(page.getByLabel("Visibility", { exact: true })).toHaveValue(
      "shared",
    );
    // The search state "ready" is not an explorer state: the library ignores it.
    await expect(page.getByLabel("State")).toHaveValue("");
    const sharedAddress = page.url();
    await page.reload();
    await expect(page).toHaveURL(sharedAddress);

    // Search is reached from the header, and there is no mode to choose.
    await search(page, "joué");
    await expect(
      page.getByRole("searchbox", { name: "Search the transcripts" }),
    ).toHaveValue("joué");
    await expect(
      page.getByText("Exact search · case- and accent-insensitive"),
    ).toBeVisible();
    await page
      .getByRole("navigation", { name: "Main navigation" })
      .getByRole("link", { name: "Library" })
      .click();
    // The words searched in the transcripts never become a Library filter,
    // yet stay in the address so coming back to Search finds them again.
    await expect(page).toHaveURL(/\/library\?.*q=jou/);
    await expect(page).not.toHaveURL(/filter=/);
    await expect(
      page.getByRole("searchbox", { name: "Search in this folder" }),
    ).toHaveValue("");
    await expect(page.getByLabel("Visibility", { exact: true })).toHaveValue(
      "shared",
    );
    // A Library filter typed on purpose is independent of the words and
    // survives a round trip through Search.
    await page
      .getByRole("searchbox", { name: "Search in this folder" })
      .fill("clip");
    await expect(page).toHaveURL(/filter=clip/);
    await search(page, "joué");
    await expect(
      page.getByRole("searchbox", { name: "Search the transcripts" }),
    ).toHaveValue("joué");
    await page
      .getByRole("navigation", { name: "Main navigation" })
      .getByRole("link", { name: "Library" })
      .click();
    await expect(
      page.getByRole("searchbox", { name: "Search in this folder" }),
    ).toHaveValue("clip");
    await context.close();
  });

  test("les filtres et le filtre par Personne", async ({ browser }) => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
    });
    const page = await context.newPage();
    await page.goto("/search");
    await expect(
      page.getByText("Type a word or a phrase in the search field at the top."),
    ).toBeVisible();
    await expect(
      page.getByText("Exact search · case- and accent-insensitive"),
    ).toBeVisible();

    const query = page.getByRole("searchbox", {
      name: "Search the transcripts",
    });
    const items = page
      .getByRole("list", { name: /Passages found in/ })
      .getByRole("listitem");

    await query.fill("merci");
    await query.press("Enter");
    await expect(items).toHaveCount(1);
    await expect(items.first()).toContainText("Merci");

    // The Person filter: "À bientôt" is spoken by a voice nobody identified, "Merci" by Alice.
    const bientot = items.filter({ hasText: "À bientôt" });
    await query.fill("bientôt");
    await query.press("Enter");
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
    await query.press("Enter");
    await expect(items).toHaveCount(1);
    await expect(items.first()).toContainText("Alice");
    await page.getByLabel("Person").selectOption({ label: "Unknown voices" });
    await expect(items.filter({ hasText: "Merci" })).toHaveCount(0);
    await page.getByRole("button", { name: "Clear the filters" }).click();
    await expect(items).toHaveCount(1);
    await expect(items.first()).toContainText("Merci");

    // The other filters narrow the results too.
    await page
      .getByLabel("Visibility", { exact: true })
      .selectOption("private");
    await expect(
      page.getByRole("heading", { name: "Nothing found" }),
    ).toBeVisible();
    await page.getByLabel("Visibility", { exact: true }).selectOption("shared");
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
      await expect(
        page.getByRole("heading", { name: "Search the transcripts" }),
      ).toBeVisible();
      await expectNoViolations(page);
      await page.goto("/search?q=joue");
      await expect(
        page.getByRole("list", { name: /Passages found in/ }),
      ).toBeVisible();
      await expect(
        page.getByText("Exact search · case- and accent-insensitive"),
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
      page.getByRole("list", { name: /Passages found in/ }),
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
