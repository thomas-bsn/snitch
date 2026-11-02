import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

test("la configuration sémantique signale un fournisseur indisponible", async ({
  browser,
}) => {
  await installAdministrator();
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/semantic");
  await page
    .getByLabel("Embeddings service address")
    .fill("http://127.0.0.1:1/v1/embeddings");
  await page.getByLabel("Embeddings model").fill("synthetic-model");
  const save = page.getByRole("button", { name: "Save", exact: true });
  const test = page.getByRole("button", { name: "Test", exact: true });
  // Nothing is saveable before a successful Test.
  await expect(save).toBeDisabled();
  await test.click();
  await expect(page.getByRole("alert")).toContainText("could not be reached");
  await expect(save).toBeDisabled();
  await page
    .getByLabel("Embeddings service address")
    .fill(process.env.SNITCH_TEST_EMBEDDINGS_URL!);
  await expect(save).toBeDisabled();
  await test.click();
  await expect(page.getByText("Service reachable")).toBeVisible();
  await expect(save).toBeEnabled();
  // Changing a single character voids the validation.
  await page.getByLabel("Embeddings model").fill("synthetic-model-x");
  await expect(save).toBeDisabled();
  await page.getByLabel("Embeddings model").fill("synthetic-model");
  await expect(save).toBeDisabled();
  await test.click();
  await expect(save).toBeEnabled();
  await save.click();
  await expect(page.getByText("settings saved")).toBeVisible();
  await expect(save).toBeDisabled();
  await page.reload();
  await expect(page.getByLabel("Embeddings model")).toHaveValue(
    "synthetic-model",
  );
  const result = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(result.violations).toEqual([]);
  await context.close();
});
