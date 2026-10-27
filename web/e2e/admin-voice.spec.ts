import { expect, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

test("un administrateur règle le rapprochement et l'automatisation vocale", async ({
  browser,
}) => {
  await installAdministrator();
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  await expect(
    page.getByRole("heading", { name: "Voice processing" }),
  ).toBeVisible();

  const matching = page
    .getByRole("heading", { name: "Voice matching" })
    .locator("..");
  await expect(matching).toContainText(
    "Only human-validated samples are references",
  );
  await matching.getByLabel("Suggestion threshold").fill("0.72");
  await matching.getByRole("button", { name: "Save thresholds" }).click();
  await expect(page.getByRole("status")).toContainText("Settings saved.");

  await matching.getByRole("button", { name: "Match voices again" }).click();
  await expect(matching.getByRole("status")).toContainText("voices evaluated:");
  await expect(page.getByText("Automatic attribution: disabled")).toBeVisible();

  const automation = page
    .getByRole("heading", { name: "Automatic voice attribution" })
    .locator("..");
  await automation.getByLabel("Minimum confirmations").fill("1");
  await automation.getByLabel("Minimum precision").fill("0.5");
  await automation
    .getByRole("button", { name: "Save activation criteria" })
    .click();
  await expect(
    page.getByText("Settings saved.", { exact: true }),
  ).toBeVisible();

  const enable = automation.getByRole("button", {
    name: "Enable automatic attribution",
  });
  await expect(enable).toBeDisabled();
  await automation.getByLabel("Enable even though criteria are unmet").check();
  await enable.click();
  await expect(page.getByText("Automatic attribution: enabled")).toBeVisible();
  await automation
    .getByRole("button", { name: "Disable automatic attribution" })
    .click();
  await expect(page.getByText("Automatic attribution: disabled")).toBeVisible();

  await context.close();
});
