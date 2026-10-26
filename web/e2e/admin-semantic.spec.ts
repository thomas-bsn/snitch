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
  await page.getByRole("button", { name: "Test and save" }).click();
  await expect(page.getByRole("alert")).toContainText("could not be reached");
  await page
    .getByLabel("Embeddings service address")
    .fill(process.env.SNITCH_TEST_EMBEDDINGS_URL!);
  await page.getByRole("button", { name: "Test and save" }).click();
  await expect(page.getByRole("status")).toContainText("saved");
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
