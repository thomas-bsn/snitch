import { test, expect } from "@playwright/test";

test("un administrateur ajoute une source et voit ses clips validés", async ({
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

  await expect(
    page.getByRole("heading", { name: "Sources dossier" }),
  ).toBeVisible();
  await page.getByLabel("Nom de la source").fill("Captures de test");
  await page
    .getByLabel("Chemin du dossier autorisé")
    .fill(process.env.RECEIPTS_TEST_SOURCE_ROOT!);
  await page.getByRole("button", { name: "Ajouter le dossier" }).click();
  await expect(
    page.getByText("Captures de test", { exact: true }),
  ).toBeVisible();

  await expect
    .poll(async () => {
      await page
        .getByRole("button", { name: "Actualiser les sources" })
        .click();
      return await page.getByText(/browser-fixture\.mp4.*ready/).count();
    })
    .toBe(1);
  await expect(page.getByText(/4\.25 s · h264/)).toBeVisible();
});
