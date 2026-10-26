import { test, expect } from "@playwright/test";

// The administrator configures Discord in the previous interface (still at /legacy/); the
// visitor signs in through the new interface. The server sends the browser back to the
// root address, which belongs to the new interface: its notices and sessions are checked
// there.
test("Discord simulé : attente, approbation, reconnexion, liaison et conflit sans fusion", async ({
  page,
  browser,
}) => {
  await page.goto("/legacy/");
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
    visitor.getByText("You will be redirected to Discord", { exact: false }),
  ).toBeVisible();
  await visitor.getByRole("button", { name: "Continue with Discord" }).click();
  await expect(
    visitor.getByRole("heading", { name: "Discord simulé" }),
  ).toBeVisible();
  await visitor.getByRole("link", { name: "Refuser", exact: true }).click();
  await expect(visitor.getByRole("status")).toContainText(
    "Discord sign-in was cancelled",
  );
  await expect(
    visitor.getByRole("button", { name: "Sign in", exact: true }),
  ).toBeVisible();
  await visitor.getByRole("button", { name: "Continue with Discord" }).click();
  await visitor
    .getByRole("link", { name: "Autoriser 123456", exact: true })
    .click();
  await expect(visitor.getByRole("status")).toContainText(
    "waiting for an administrator",
  );
  expect((await visitor.request.get(origin + "/api/auth/me")).status()).toBe(
    401,
  );
  await page.reload();
  await page
    .getByRole("button", { name: "Approuver discord_123456", exact: true })
    .click();
  await visitor.getByRole("button", { name: "Continue with Discord" }).click();
  await visitor
    .getByRole("link", { name: "Autoriser 123456", exact: true })
    .click();
  await expect(
    visitor.getByRole("button", { name: "Account menu" }),
  ).toBeVisible();
  // A matching provider username never signs into the existing administrator.
  expect(
    await (await visitor.request.get(origin + "/api/auth/me")).json(),
  ).toMatchObject({ user: { role: "user", username: "discord_123456" } });
  // Linking an identity that belongs to another account keeps the existing link, and the
  // refusal is shown in the new interface where the server sends the browser back.
  await page.goto("/account");
  await page.getByRole("button", { name: "Link Discord", exact: true }).click();
  await page
    .getByRole("link", { name: "Autoriser 123456", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText("already linked");
  await expect(
    page.getByRole("button", { name: "Account menu" }),
  ).toBeVisible();
  await page.goto("/account");
  await page.getByRole("button", { name: "Link Discord", exact: true }).click();
  await page
    .getByRole("link", { name: "Autoriser 654321", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText(
    "Your Discord identity is linked",
  );
  await page.goto("/legacy/");
  await page
    .getByRole("button", { name: "Se déconnecter", exact: true })
    .click();
  await page.goto("/login");
  await page.getByRole("button", { name: "Continue with Discord" }).click();
  await page
    .getByRole("link", { name: "Autoriser 654321", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Account menu" }),
  ).toBeVisible();
  expect(
    await (await page.request.get(origin + "/api/auth/me")).json(),
  ).toMatchObject({ user: { role: "super_admin", username: "administrator" } });
  await visitor.goto(origin + "/?notice=unavailable");
  await expect(visitor.getByRole("status")).toContainText(
    "temporarily unavailable",
  );
  await visitorContext.close();
});
