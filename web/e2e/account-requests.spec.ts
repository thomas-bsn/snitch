import AxeBuilder from "@axe-core/playwright";
import { expect, request, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// #283: a sign-up waiting for approval reaches the administrators' bell live and leads to the request, which
// they refuse from there; the Super-administrator then configures the Discord webhook (Discord is simulated by
// the test server: every call answers 204).

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];
const WEBHOOK =
  "https://discord.com/api/webhooks/111122223333444455/SyntheticWebhookToken_0123456789abcdefghijklmnop";

test("une demande d'accès arrive dans la cloche, se refuse, et le webhook Discord se configure", async ({
  browser,
}) => {
  await installAdministrator();
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/library");
  await expect(
    page.getByRole("button", { name: "Notifications" }),
  ).toBeVisible();

  // The visitor signs up while the page is open: the bell follows live.
  const guest = await request.newContext({
    baseURL: process.env.SNITCH_TEST_URL,
    extraHTTPHeaders: { Origin: process.env.SNITCH_TEST_URL! },
  });
  const registered = await guest.post("/api/auth/register", {
    data: { username: "visitor", password: "synthetic-password-789" },
  });
  expect(registered.status()).toBe(201);
  await guest.dispose();

  const bell = page.getByRole("button", { name: "Notifications, 1 unread" });
  await expect(bell).toBeVisible();
  await bell.click();
  const menu = page.getByRole("menu");
  await expect(menu).toContainText("New access request from visitor");
  await menu
    .getByRole("menuitem", { name: /New access request from visitor/ })
    .first()
    .click();
  await expect(page).toHaveURL(/\/admin\/users\/signups$/);

  await page.getByRole("button", { name: "Refuse visitor" }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog).toContainText("Refuse the access request from visitor?");
  await dialog.getByRole("button", { name: "Refuse visitor" }).click();
  await expect(page.getByText("No sign-ups awaiting approval.")).toBeVisible();
  await page.getByRole("button", { name: /^Notifications/ }).click();
  await expect(page.getByRole("menu")).toContainText("Access request refused");
  await page.keyboard.press("Escape");

  // The webhook: refused address, then a Discord one, enabled, tested.
  await page.goto("/admin/notifications");
  await expect(page.getByTestId("webhook-state")).toHaveText("Not configured");
  const address = page.getByLabel("Webhook address");
  await address.fill("https://example.org/api/webhooks/1/x");
  await page.getByLabel("Send notifications to Discord").check();
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "Paste the address of a Discord webhook",
  );
  await address.fill(WEBHOOK);
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByTestId("webhook-state")).toHaveText(
    "Enabled (webhook …4455)",
  );
  await expect(page.getByLabel("Access requests")).toBeChecked();
  await expect(page.locator("body")).not.toContainText("SyntheticWebhookToken");
  await page.getByRole("button", { name: "Send a test message" }).click();
  await expect(page.getByText("The test message was sent.")).toBeVisible();
  await expect(page.getByTestId("webhook-last-status")).toContainText("Sent");
  expect(
    (await new AxeBuilder({ page }).withTags(TAGS).analyze()).violations.map(
      (v) => v.id,
    ),
  ).toEqual([]);
  await context.close();
});
