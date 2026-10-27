import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
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
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
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
  const member = page.getByRole("row", { name: /member-user/ });
  await expect(member).toContainText("User");
  await member.getByRole("button", { name: "Change role" }).click();
  await expect(page.getByRole("alertdialog")).toContainText("Administrator");
  await page.getByRole("button", { name: "Confirm" }).click();
  await expect(member).toContainText("Administrator");
  await member.getByRole("button", { name: "Delete member" }).click();
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
