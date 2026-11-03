import { expect, test, type Page } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// "To process": one row per clip with a voice to settle, its filters, and the counter on
// the tab and in the sidebar. Synthetic clip of the shared fixture (two unknown voices).

let page: Page;
let clipId: number;

test.describe.serial("personnes : à traiter par clip", () => {
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    clipId = await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("une ligne par clip avec ses nombres, et le compteur de l'onglet et de la barre latérale", async () => {
    await page.goto("/people/verification");
    const list = page.getByRole("region", { name: "Clips to process" });
    const row = list.getByRole("listitem");
    await expect(row).toHaveCount(1);
    await expect(row).toContainText("2 unknown");
    await expect(
      page.getByRole("link", { name: /^To process\s*2$/ }),
    ).toBeVisible();
    await expect(
      page
        .getByRole("navigation", { name: "Main navigation" })
        .first()
        .getByRole("link", { name: "People" }),
    ).toHaveAccessibleDescription("2 voice(s) to decide");
  });

  test("les filtres s'appliquent côté serveur", async () => {
    await page.goto("/people/verification");
    const list = page.getByRole("region", { name: "Clips to process" });
    const filters = list.getByRole("group", { name: "Filter the clips" });
    await filters.getByRole("button", { name: /^To confirm/ }).click();
    await expect(list.getByRole("listitem")).toHaveCount(0);
    await expect(
      list.getByText("No clip to process for this filter."),
    ).toBeVisible();
    await filters
      .getByRole("button", { name: /^Unknown \/ ambiguous/ })
      .click();
    await expect(list.getByRole("listitem")).toHaveCount(1);
  });

  test("Traiter ouvre l'écran de traitement de ce clip", async () => {
    await page.goto("/people/verification");
    await page.getByRole("link", { name: /^Process / }).click();
    await expect(page).toHaveURL(
      new RegExp(`/people/verification\\?clip=${clipId}$`),
    );
    await expect(
      page.getByRole("region", { name: "Clip processing" }),
    ).toBeVisible();
    await expect(
      page
        .getByRole("region", { name: "Who is speaking here?" })
        .getByRole("heading", { name: /SPEAKER_/ }),
    ).toBeVisible();
  });
});
