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

  test("la page dédiée sépare non-lues et actions, filtre, et suit le direct", async () => {
    await page.goto("/library");
    await page.getByRole("button", { name: "Notifications" }).click();
    await page
      .getByRole("menuitem", { name: "View all notifications" })
      .click();
    await expect(page).toHaveURL(/\/notifications$/);
    await expect(
      page.getByRole("heading", { name: "Notifications", level: 1 }),
    ).toBeVisible();
    await expect(page.getByTestId("count-unread")).toHaveText("0 unread");
    await expect(page.getByTestId("count-actions")).toHaveText(
      "1 actions to handle",
    );
    const rows = page.getByTestId("notification-row");
    await expect(rows).toHaveCount(1);
    // An active action cannot be removed.
    await expect(
      page.getByRole("button", { name: "Remove from history" }),
    ).toHaveCount(0);

    // Unread is individual and does not touch the open action count.
    await rows.getByRole("button", { name: "Mark as unread" }).focus();
    await page.keyboard.press("Enter");
    await expect(page.getByTestId("count-unread")).toHaveText("1 unread");
    await expect(page.getByTestId("count-actions")).toHaveText(
      "1 actions to handle",
    );

    // Filters.
    const filters = page.getByRole("group", { name: "Filter notifications" });
    await filters.getByRole("button", { name: "Errors" }).click();
    await expect(
      page.getByText("No notifications match this filter."),
    ).toBeVisible();
    await filters.getByRole("button", { name: "Unread" }).click();
    await expect(rows).toHaveCount(1);
    await filters.getByRole("button", { name: "Actions required" }).click();
    await expect(rows).toHaveCount(1);
    await expect(
      rows.getByRole("link", { name: "Verify the voices" }),
    ).toHaveAttribute(
      "href",
      new RegExp(`/people/verification\\?clip=${clipId}$`),
    );

    // Live: a second tab sees the change without reloading.
    const other = await page.context().newPage();
    await other.goto("/notifications");
    await expect(other.getByTestId("count-unread")).toHaveText("1 unread");
    await page
      .getByTestId("notification-row")
      .getByRole("button", { name: "Mark as read" })
      .click();
    await expect(page.getByTestId("count-unread")).toHaveText("0 unread");
    await expect(other.getByTestId("count-unread")).toHaveText("0 unread");
    await other.close();

    const { violations } = await new AxeBuilder({ page })
      .withTags(TAGS)
      .analyze();
    expect(violations.map((v) => v.id)).toEqual([]);

    // Phone width: no horizontal scroll and still accessible.
    await page.setViewportSize({ width: 375, height: 700 });
    await expect(page.getByTestId("notification-row")).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    const phone = await new AxeBuilder({ page }).withTags(TAGS).analyze();
    expect(phone.violations.map((v) => v.id)).toEqual([]);
    await page.setViewportSize({ width: 1280, height: 800 });
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
    // The processing screen of that clip: the queue holds its two voices.
    const screen = page.getByRole("region", { name: "Clip processing" });
    await expect(screen).toContainText("synthetic-voices.mp4");
    const current = screen.getByRole("region", {
      name: "Who is speaking here?",
    });
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_00",
    );
    await expect(
      screen
        .getByRole("list", { name: "Voices to process" })
        .getByRole("listitem"),
    ).toHaveCount(2);

    // Handling the voices resolves the shared action: the second one is only a suggestion
    // once the first is named, so it no longer waits for a decision.
    await current.getByRole("button", { name: "Identify" }).click();
    await current.getByLabel("Search people").fill("Dana Synthétique");
    await current
      .getByRole("button", {
        name: "Create the profile “Dana Synthétique” and validate",
      })
      .click();
    await expect(current.getByRole("heading", { level: 2 })).toHaveText(
      "SPEAKER_01",
    );

    await page.goto("/library");
    await page.getByRole("button", { name: "Notifications" }).click();
    const menu = page.getByRole("menu");
    await expect(menu).toContainText("Voices verified in synthetic-voices.mp4");
    await expect(menu).toContainText("Resolved");
    await expect(menu).not.toContainText("to handle");
  });

  test("l'historique résolu se vide en une action", async () => {
    await page.goto("/notifications");
    await expect(page.getByTestId("count-actions")).toHaveText(
      "0 actions to handle",
    );
    const history = page.getByRole("list", { name: "Resolved notifications" });
    await expect(history.getByTestId("notification-row")).toHaveCount(1);
    await page
      .getByRole("button", { name: "Remove resolved notifications" })
      .click();
    await expect(page.getByText("No notifications.")).toBeVisible();
    await expect(page.getByTestId("notification-row")).toHaveCount(0);
  });
});
