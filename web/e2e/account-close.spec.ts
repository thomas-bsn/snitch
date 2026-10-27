import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import {
  ADMIN,
  MEMBER,
  createApprovedMember,
  installAdministrator,
  sessionFor,
} from "./shell-fixture";

test("fermeture de compte avec confirmation et protection du Super-administrateur", async ({
  browser,
}) => {
  await installAdministrator();
  await createApprovedMember();
  const owner = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const ownerPage = await owner.newPage();
  await ownerPage.goto("/account");
  await expect(
    ownerPage.getByText("The Super-administrator cannot close this account."),
  ).toBeVisible();
  await expect(
    ownerPage.getByRole("button", { name: "Delete my account" }),
  ).toHaveCount(0);
  const storage = await sessionFor(MEMBER);
  const first = await browser.newContext({ storageState: storage });
  const second = await browser.newContext({ storageState: storage });
  const page = await first.newPage();
  const other = await second.newPage();
  await page.goto("/account");
  await other.goto("/library");
  await page.getByRole("button", { name: "Delete my account" }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(
    dialog.getByRole("button", { name: "Delete account" }),
  ).toBeDisabled();
  await dialog.getByLabel("Type delete or supprimer").fill("other");
  await expect(
    dialog.getByRole("button", { name: "Delete account" }),
  ).toBeDisabled();
  await dialog.getByLabel("Type delete or supprimer").fill("SUPPRIMER");
  await expect(
    dialog.getByRole("button", { name: "Delete account" }),
  ).toBeEnabled();
  expect(
    (
      await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze()
    ).violations,
  ).toEqual([]);
  await dialog.getByRole("button", { name: "Delete account" }).click();
  await expect(page).toHaveURL(/\/login/);
  await other.reload();
  await expect(other).toHaveURL(/\/login/);
  await first.close();
  await second.close();
  await owner.close();
});
