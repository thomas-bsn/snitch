import { expect, test } from "@playwright/test";

test("un administrateur crée un worker puis révoque son jeton", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.RECEIPTS_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();

  await page.getByLabel("Nom du worker").fill("synthetic-worker");
  await page.getByLabel("Concurrence maximale").fill("2");
  await page.getByRole("button", { name: "Créer un worker" }).click();
  const token = page
    .locator('p[role="status"]')
    .filter({ hasText: "Jeton à copier maintenant" });
  await expect(token).toContainText("Jeton à copier maintenant: rw_");
  await expect(page.getByRole("list", { name: "Workers" })).toContainText(
    "synthetic-worker",
  );

  await page.getByRole("button", { name: "Révoquer le jeton" }).click();
  await expect(page.getByRole("list", { name: "Workers" })).toContainText(
    "Révoqué",
  );
  await expect(token).toHaveCount(0);
});
