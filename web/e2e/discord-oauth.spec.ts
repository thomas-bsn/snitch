import { test, expect } from "@playwright/test";

test("Discord simulé : attente, approbation, reconnexion, liaison et conflit sans fusion", async ({
  page,
  browser,
}) => {
  await page.goto("/");
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.SNITCH_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();
  await page
    .getByLabel("Identifiant de l’application Discord")
    .fill("fixture-app");
  await page
    .getByLabel("Secret Discord", { exact: true })
    .fill("fixture-secret");
  await page.getByRole("button", { name: "Enregistrer les réglages" }).click();
  await expect(page.getByRole("status")).toContainText("Réglages enregistrés");
  const origin = new URL(page.url()).origin;
  const visitorContext = await browser.newContext();
  const visitor = await visitorContext.newPage();
  await visitor.goto(origin);
  await expect(
    visitor.getByText("Vous serez redirigé vers Discord", { exact: false }),
  ).toBeVisible();
  await visitor.getByRole("button", { name: "Continuer avec Discord" }).click();
  await expect(
    visitor.getByRole("heading", { name: "Discord simulé" }),
  ).toBeVisible();
  await visitor.getByRole("link", { name: "Refuser", exact: true }).click();
  await expect(visitor.getByRole("status")).toContainText(
    "Connexion Discord annulée",
  );
  await expect(
    visitor.getByRole("button", { name: "Se connecter", exact: true }),
  ).toBeVisible();
  await visitor.getByRole("button", { name: "Continuer avec Discord" }).click();
  await visitor
    .getByRole("link", { name: "Autoriser 123456", exact: true })
    .click();
  await expect(visitor.getByRole("status")).toContainText(
    "en attente d’approbation",
  );
  expect((await visitor.request.get(origin + "/api/auth/me")).status()).toBe(
    401,
  );
  await page.reload();
  await page
    .getByRole("button", { name: "Approuver discord_123456", exact: true })
    .click();
  await visitor.getByRole("button", { name: "Continuer avec Discord" }).click();
  await visitor
    .getByRole("link", { name: "Autoriser 123456", exact: true })
    .click();
  await expect(visitor.locator(".sidebar-user strong")).toHaveText(
    "discord_123456",
  );
  // A matching provider username never signs into the existing administrator.
  expect(
    await (await visitor.request.get(origin + "/api/auth/me")).json(),
  ).toMatchObject({ user: { role: "user", username: "discord_123456" } });
  await page.getByRole("button", { name: "Lier Discord", exact: true }).click();
  await page
    .getByRole("link", { name: "Autoriser 123456", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText("déjà liée");
  await expect(page.locator(".sidebar-user strong")).toHaveText(
    "administrator",
  );
  await page.getByRole("button", { name: "Lier Discord", exact: true }).click();
  await page
    .getByRole("link", { name: "Autoriser 654321", exact: true })
    .click();
  await expect(page.locator(".sidebar-user strong")).toHaveText(
    "administrator",
  );
  await page
    .getByRole("button", { name: "Se déconnecter", exact: true })
    .click();
  await page.getByRole("button", { name: "Continuer avec Discord" }).click();
  await page
    .getByRole("link", { name: "Autoriser 654321", exact: true })
    .click();
  await expect(page.locator(".sidebar-user strong")).toHaveText(
    "administrator",
  );
  await visitor.goto(origin + "/?notice=unavailable");
  await expect(visitor.getByRole("status")).toContainText(
    "Le service est temporairement indisponible",
  );
  await visitorContext.close();
});
