import { expect, test, type Page } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// The three sections of the People screen: Directory (default), To process and Unknown
// voices, and the addresses that must stay valid. Synthetic clip of the shared fixture.

let page: Page;

test.describe.serial("personnes : onglets et adresses", () => {
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("l'annuaire est l'onglet par défaut et les trois onglets portent leurs noms", async () => {
    await page.goto("/people");
    const nav = page.getByRole("navigation", { name: "People sections" });
    await expect(nav.getByRole("link")).toHaveText([
      "Directory",
      /^To process\s*\d*$/,
      "Unknown voices",
    ]);
    await expect(nav.getByRole("link", { name: "Directory" })).toHaveAttribute(
      "aria-current",
      "page",
    );
  });

  test("les adresses de traitement et de groupe ouvrent le bon onglet", async () => {
    await page.goto("/people/verification");
    await expect(
      page.getByRole("link", { name: /^To process/ }),
    ).toHaveAttribute("aria-current", "page");
    await page.goto("/people/groups");
    await expect(
      page.getByRole("link", { name: "Unknown voices" }),
    ).toHaveAttribute("aria-current", "page");
  });
});
