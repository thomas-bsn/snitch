import { test, expect } from "@playwright/test";

test("installer sans Discord, refuser un code invalide et retrouver sa session", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByLabel("Code d’installation").fill("incorrect-code");
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();
  await expect(page.getByRole("alert")).toContainText("code invalide / expiré");
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.RECEIPTS_TEST_CODE!);
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();
  await expect(page.locator(".sidebar-user strong")).toHaveText(
    "administrator",
  );
  await expect(
    page.getByText("Discord est facultatif.", { exact: false }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Sources dossier" }),
  ).toBeVisible();
  await expect(
    page.getByText("Worker : phase 2", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Embeddings : phase 4", { exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(page.locator(".sidebar-user strong")).toHaveText(
    "administrator",
  );
  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await expect(
    page.getByRole("heading", { name: "Retrouver votre espace" }),
  ).toBeVisible();
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("incorrect-password");
  await page.getByRole("button", { name: "Se connecter", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText(
    "Identifiants incorrects ou inscription en attente",
  );
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Se connecter", exact: true }).click();
  await expect(page.locator(".sidebar-user strong")).toHaveText(
    "administrator",
  );
  expect((await page.request.get("/api/auth/me")).status()).toBe(200);
});
