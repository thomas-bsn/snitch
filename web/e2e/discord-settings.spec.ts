import { test, expect } from "@playwright/test";

test("Discord facultatif : configurer, conserver le secret et revenir au local", async ({
  page,
}) => {
  await page.goto("/legacy/");
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.SNITCH_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();
  await expect(
    page.getByText(
      "Discord désactivé : les comptes locaux restent disponibles.",
    ),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Discord Developer Portal" }),
  ).toHaveAttribute("href", "https://discord.com/developers/applications");
  await expect(
    page.getByText("URL de retour à renseigner dans Discord", { exact: false }),
  ).toContainText(`${new URL(page.url()).origin}/api/auth/discord/callback`);
  await page.getByRole("button", { name: "Continuer sans Discord" }).click();
  await expect(page.getByRole("status")).toContainText("Discord désactivé");
  await page
    .getByLabel("Identifiant de l’application Discord")
    .fill("synthetic-app");
  await page.getByRole("button", { name: "Enregistrer les réglages" }).click();
  await expect(page.getByRole("alert")).toContainText("Saisissez le secret");
  await page
    .getByLabel("Secret Discord", { exact: true })
    .fill("fixture-discord-secret");
  await page.getByRole("button", { name: "Enregistrer les réglages" }).click();
  await expect(page.getByRole("status")).toContainText("Réglages enregistrés");
  await expect(page.getByLabel("Secret Discord", { exact: true })).toHaveValue(
    "",
  );
  await expect(
    page.getByText(
      "Secret configuré. Laissez le champ vide pour le conserver.",
      { exact: true },
    ),
  ).toBeVisible();
  const settings = await (await page.request.get("/api/admin/settings")).text();
  expect(settings).not.toContain("fixture-discord-secret");
  await page.reload();
  await expect(page.getByLabel("Secret Discord", { exact: true })).toHaveValue(
    "",
  );
  await page
    .getByLabel("Identifiant de l’application Discord")
    .fill("updated-app");
  await page.getByRole("button", { name: "Enregistrer les réglages" }).click();
  await expect(page.getByRole("status")).toContainText("Réglages enregistrés");
  expect(
    await (await page.request.get("/api/admin/settings")).json(),
  ).toMatchObject({ client_id: "updated-app", secret_configured: true });
  await page.getByRole("button", { name: "Désactiver Discord" }).click();
  await expect(page.getByRole("status")).toContainText("Discord désactivé");
  await expect(
    page.getByLabel("Identifiant de l’application Discord"),
  ).toHaveValue("");
  expect(
    await (await page.request.get("/api/admin/settings")).json(),
  ).toMatchObject({ client_id: "", secret_configured: false });
  await expect(
    page.getByRole("button", { name: "Lier Discord", exact: true }),
  ).toHaveCount(0);
});
