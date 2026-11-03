import { expect, test, type Page } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// Unknown voices: Random and Ignore for the whole group, from the naming panel (ticket 222).

let page: Page;

test.describe.serial("groupe de voix : disposition", () => {
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("ignorer tout le groupe le retire de la liste", async () => {
    await page.goto("/people/groups");
    await page.getByRole("button", { name: /^Listen and name/ }).click();
    await expect(
      page.getByRole("heading", { name: "Proposed grouping of 2 voices" }),
    ).toBeVisible();
    await expect(page.getByText("Heard in 1 video(s)")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Random for the whole group" }),
    ).toBeVisible();

    await page.getByRole("button", { name: "Ignore the whole group" }).click();
    await expect(page.getByRole("status").last()).toContainText(
      "Ignored: 2 voice(s) set aside.",
    );
    await expect(page.getByText("No group left to name.")).toBeVisible();
    const unknown = await page.evaluate(async () => {
      const v = await fetch("/api/voice-verification").then((r) => r.json());
      return v.counts.unknown;
    });
    expect(unknown).toBe(0);
  });
});
