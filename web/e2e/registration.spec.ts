import { test, expect } from "@playwright/test";

test("installation, connexion, approbation et refus anonymes", async ({
  page,
  request,
}) => {
  const anon = await request.get("/api/admin/settings");
  expect(anon.status()).toBe(401);
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Installer votre espace" }),
  ).toBeVisible();
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.SNITCH_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();
  await expect(
    page.getByRole("heading", { name: "Accès et Discord" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await page
    .getByRole("button", { name: "Créer un compte", exact: true })
    .click();
  await page.getByLabel("Nom d’utilisateur").fill("player");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-456");
  await page.getByRole("button", { name: "Demander l’accès" }).click();
  await expect(page.getByRole("status")).toContainText("en attente");
  await page.getByRole("button", { name: "Se connecter", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("en attente");
  expect((await page.request.get("/api/auth/me")).status()).toBe(401);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Se connecter", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Inscriptions en attente (1)" }),
  ).toBeVisible();
  await page
    .getByLabel("Approbation administrateur requise pour les nouveaux comptes")
    .uncheck();
  await page.getByRole("button", { name: "Enregistrer les réglages" }).click();
  await expect(page.getByRole("status")).toHaveText("Réglages enregistrés.");
  await expect(
    page.getByRole("button", { name: "Approuver player" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await page
    .getByRole("button", { name: "Créer un compte", exact: true })
    .click();
  await page.getByLabel("Nom d’utilisateur").fill("automatic-player");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-789");
  await page.getByRole("button", { name: "Demander l’accès" }).click();
  await expect(page.getByRole("status")).toHaveText(
    "Compte créé. Vous pouvez vous connecter.",
  );
  await page.getByRole("button", { name: "Se connecter", exact: true }).click();
  await expect(page.locator(".sidebar-user strong")).toHaveText(
    "automatic-player",
  );
  await expect(
    page.getByRole("heading", { name: "Accès et Discord" }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Se connecter", exact: true }).click();
  await page.getByRole("button", { name: "Approuver player" }).click();
  await expect(page.getByText("Aucune demande en attente.")).toBeVisible();
  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await page.getByLabel("Nom d’utilisateur").fill("player");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-456");
  await page.getByRole("button", { name: "Se connecter", exact: true }).click();
  await expect(page.locator(".sidebar-user strong")).toHaveText("player");
  await expect(
    page.getByRole("heading", { name: "Accès et Discord" }),
  ).toHaveCount(0);
  expect((await page.request.get("/api/admin/settings")).status()).toBe(403);
  expect(
    (
      await page.request.post("/api/auth/logout", {
        headers: { Origin: process.env.SNITCH_TEST_URL! },
      })
    ).status(),
  ).toBe(403);
});
