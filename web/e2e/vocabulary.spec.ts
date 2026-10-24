import { expect, test } from "@playwright/test";

test("admin gère les termes, variantes et graphies préférées", async ({
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

  const vocabulary = page.locator("#vocabulary");
  await vocabulary.scrollIntoViewIfNeeded();
  await expect(
    vocabulary.getByRole("heading", { name: "Vocabulaire de transcription" }),
  ).toBeVisible();
  await vocabulary.getByLabel("Terme").fill("Final Fantasy XIV");
  await vocabulary
    .getByLabel("Variantes, séparées par des virgules")
    .fill("FFXIV, Final Fantasy 14");
  await vocabulary.getByLabel("Graphie préférée").fill("FFXIV");
  await vocabulary.getByRole("button", { name: "Ajouter le terme" }).click();
  await expect(vocabulary.getByText("FFXIV", { exact: true })).toBeVisible();
  await expect(
    vocabulary.getByText("Final Fantasy XIV · FFXIV · Final Fantasy 14"),
  ).toBeVisible();

  await vocabulary.getByRole("button", { name: "Modifier" }).click();
  await vocabulary.getByLabel("Graphie préférée").fill("Final Fantasy XIV");
  await vocabulary
    .getByLabel("Actif pour les prochaines transcriptions")
    .uncheck();
  await vocabulary
    .getByRole("button", { name: "Enregistrer le terme" })
    .click();
  await expect(vocabulary.getByText("Inactif", { exact: true })).toBeVisible();
  await expect(
    vocabulary.getByText("Final Fantasy XIV", { exact: true }),
  ).toBeVisible();

  await vocabulary.getByRole("button", { name: "Supprimer" }).click();
  await expect(vocabulary.getByText("Aucun terme enregistré.")).toBeVisible();
});
