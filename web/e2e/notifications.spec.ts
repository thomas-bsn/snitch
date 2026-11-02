import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// The notification bell and the voices to verify: the synthetic clip leaves two unknown
// voices, announced by one notification that leads to the Verification list filtered on
// the clip. Reading does not resolve it; handling the voices does.

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];

let page: Page;
let clipId: number;

test.describe.serial("notifications : voix à vérifier", () => {
  test.beforeAll(async ({ browser }) => {
    page = await (await browser.newContext()).newPage();
    clipId = await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  });
  test.afterAll(async () => {
    await page.context().close();
  });

  test("la cloche compte les non-lues, la lecture ne résout pas l'action", async () => {
    await page.goto("/library");
    const bell = page.getByRole("button", { name: "Notifications, 1 unread" });
    await expect(bell).toBeVisible();
    await bell.click();
    const menu = page.getByRole("menu");
    await expect(menu).toContainText(
      "2 voices to verify in synthetic-voices.mp4",
    );
    await expect(menu).toContainText("1 to handle");
    const { violations } = await new AxeBuilder({ page })
      .withTags(TAGS)
      .analyze();
    expect(violations.map((v) => v.id)).toEqual([]);

    await menu.getByRole("menuitem", { name: "Mark all as read" }).click();
    await expect(
      page.getByRole("button", { name: "Notifications" }),
    ).toBeVisible();
    await expect(page.getByTestId("bell-count")).toHaveCount(0);
    // Read, but still an action to handle.
    await expect(menu).toContainText("1 to handle");
    await page.keyboard.press("Escape");
  });

  test("la notification mène à la vérification filtrée sur le clip, puis se résout", async () => {
    await page.goto("/library");
    await page.getByRole("button", { name: "Notifications" }).click();
    await page
      .getByRole("menuitem", { name: /2 voices to verify in synthetic-voices/ })
      .click();
    await expect(page).toHaveURL(
      new RegExp(`/people/verification\\?clip=${clipId}$`),
    );
    await expect(page.getByTestId("verification-clip-filter")).toContainText(
      "Voices of synthetic-voices.mp4",
    );
    const cases = page.getByRole("list", { name: "Voices to decide" });
    await expect(cases.locator("xpath=./li")).toHaveCount(2);

    // Clearing the filter keeps the same list (one clip only).
    await page.getByRole("button", { name: "Show all clips" }).click();
    await expect(page.getByTestId("verification-clip-filter")).toHaveCount(0);

    // Handling the voices resolves the shared action.
    await page
      .getByRole("button", { name: "Decide who SPEAKER_00 is" })
      .click();
    const panel = page.getByRole("dialog", { name: "Who is SPEAKER_00?" });
    await panel.getByRole("radio", { name: "Create a person" }).check();
    await panel.getByLabel("Name of the new person").fill("Dana Synthétique");
    await panel.getByRole("button", { name: "Confirm" }).click();
    await expect(page.getByText("Nothing left to decide here.")).toBeVisible();

    await page.goto("/library");
    await page.getByRole("button", { name: "Notifications" }).click();
    const menu = page.getByRole("menu");
    await expect(menu).toContainText("Voices verified in synthetic-voices.mp4");
    await expect(menu).toContainText("Resolved");
    await expect(menu).not.toContainText("to handle");
  });
});
