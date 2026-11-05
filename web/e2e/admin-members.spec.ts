import AxeBuilder from "@axe-core/playwright";
import { expect, request, test } from "@playwright/test";
import {
  ADMIN,
  MEMBER,
  createApprovedMember,
  installAdministrator,
  sessionFor,
} from "./shell-fixture";

test("un administrateur gère les membres, les inscriptions et les exports", async ({
  browser,
}) => {
  await installAdministrator();
  await createApprovedMember();
  const memberStorage = await sessionFor(MEMBER);
  const avatarClient = await request.newContext({
    baseURL: process.env.SNITCH_TEST_URL,
    storageState: memberStorage,
  });
  const me = await (await avatarClient.get("/api/auth/me")).json();
  const uploaded = await avatarClient.put("/api/account/avatar", {
    headers: {
      Origin: process.env.SNITCH_TEST_URL!,
      "X-CSRF-Token": me.csrf_token,
      "Content-Type": "image/png",
    },
    data: Buffer.from(
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
      "base64",
    ),
  });
  expect(uploaded.status()).toBe(204);
  await avatarClient.dispose();
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
    viewport: { width: 390, height: 780 },
  });
  const page = await context.newPage();
  await page.goto("/admin/users");
  expect(
    (
      await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze()
    ).violations,
  ).toEqual([]);
  // The list stays behind the open panel, which hides it from the accessibility tree: select by markup.
  const member = page.locator("tr", { hasText: "member-user" });
  await expect(member).toContainText("User");
  const photo = member.locator("img");
  await expect(photo).toBeVisible();
  await expect
    .poll(() =>
      photo.evaluate((image) =>
        image instanceof HTMLImageElement ? image.naturalWidth : 0,
      ),
    )
    .toBe(256);
  const owner = page.locator("tr", { hasText: "administrator" });
  await expect(owner.locator("img")).toHaveCount(0);
  await expect(owner.locator('[aria-hidden="true"]').first()).toHaveText("a");
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(
    390,
  );
  const members = page.getByRole("region", { name: "Members", exact: true });
  await members.evaluate((element) => {
    element.scrollLeft = element.scrollWidth;
  });
  expect(
    await members.evaluate((element) => element.scrollLeft),
  ).toBeGreaterThan(0);
  await member.getByRole("button", { name: /^Edit member-user/ }).click();
  const panel = page.getByRole("dialog");
  await expect(panel.getByLabel("Display name")).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(
    390,
  );
  expect(
    (
      await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze()
    ).violations,
  ).toEqual([]);
  await page.setViewportSize({ width: 1280, height: 800 });
  await panel.getByLabel("Display name").fill("Member Fixture");
  await panel.getByLabel("Language").selectOption("fr");
  await panel.getByLabel("Role", { exact: true }).selectOption("admin");
  await panel.getByRole("button", { name: "Save" }).click();
  await expect(panel.getByRole("status")).toContainText("Changes saved");
  await expect(panel.getByRole("button", { name: "Save" })).toBeDisabled();
  await expect(member).toContainText("Member Fixture");
  await expect(member).toContainText("Administrator");
  await panel.getByRole("button", { name: "Set a temporary password" }).click();
  await page
    .getByRole("alertdialog")
    .getByRole("button", { name: "Generate" })
    .click();
  await expect(panel.locator("output")).toHaveText(/^\S{20}$/);
  await panel.getByRole("button", { name: "Delete member" }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog.getByLabel("Hide their clips")).not.toBeChecked();
  await dialog.getByLabel("Hide their clips").check();
  await dialog.getByRole("button", { name: "Delete" }).click();
  await expect(member).toHaveCount(0);
  const memberContext = await browser.newContext({
    storageState: memberStorage,
  });
  const memberPage = await memberContext.newPage();
  await memberPage.goto("/admin/users");
  await expect(memberPage).toHaveURL(/\/login/);
  await memberContext.close();
  await page.getByRole("link", { name: "Sign-ups" }).click();
  await expect(page.getByLabel("Require approval for sign-ups")).toBeVisible();
  await expect(page.getByLabel("Discord application ID")).toBeVisible();
  await expect(page.getByLabel("Discord return address")).toHaveValue(
    `${process.env.SNITCH_TEST_URL}/api/auth/discord/callback`,
  );
  await page.getByLabel("Discord application ID").fill("fixture-app");
  await page
    .getByLabel("Discord application secret")
    .fill("fixture-discord-secret");
  await page.getByRole("button", { name: "Save settings" }).click();
  await expect(page.getByRole("status")).toContainText("Changes saved");
  await page.reload();
  await expect(page.getByLabel("Discord application ID")).toHaveValue(
    "fixture-app",
  );
  await expect(page.getByLabel("Discord application secret")).toHaveValue("");
  const guest = await browser.newContext();
  const response = await guest.request.post(
    `${process.env.SNITCH_TEST_URL}/api/auth/register`,
    {
      headers: { Origin: process.env.SNITCH_TEST_URL! },
      data: { username: "pending-fixture", password: "synthetic-password-123" },
    },
  );
  expect(response.status()).toBe(201);
  await guest.close();
  await page.reload();
  await page.getByRole("button", { name: "Approve pending-fixture" }).click();
  await expect(
    page.getByRole("button", { name: "Approve pending-fixture" }),
  ).toHaveCount(0);
  expect(
    (
      await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze()
    ).violations,
  ).toEqual([]);
  await page.goto("/admin/exports");
  await expect(
    page.getByRole("link", { name: "Download corrected segments" }),
  ).toHaveAttribute("href", /format=ndjson/);
  expect(
    (
      await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze()
    ).violations,
  ).toEqual([]);
  await context.close();
});
