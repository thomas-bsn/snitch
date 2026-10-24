import { expect, test } from "@playwright/test";

test("un administrateur suspend la file et règle sa concurrence globale", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.SNITCH_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();

  const controls = page.getByRole("region", { name: "Contrôle de la file" });

  await expect(
    controls.getByText("Acquisitions actives", { exact: true }),
  ).toBeVisible();
  await expect(controls.getByLabel("Parallélisme global")).toHaveValue("1");

  await controls.getByLabel("Parallélisme global").fill("3");
  await controls.getByLabel("Suspendre les nouvelles acquisitions").check();
  await controls
    .getByRole("button", { name: "Enregistrer les contrôles" })
    .click();
  await expect(page.getByRole("status")).toHaveText(
    "Contrôles de file enregistrés.",
  );
  await expect(
    controls.getByText("Acquisitions en pause", { exact: true }),
  ).toBeVisible();
  await expect(controls.getByLabel("Parallélisme global")).toHaveValue("3");

  await controls.getByLabel("Suspendre les nouvelles acquisitions").uncheck();
  await controls
    .getByRole("button", { name: "Enregistrer les contrôles" })
    .click();
  await expect(
    controls.getByText("Acquisitions actives", { exact: true }),
  ).toBeVisible();
});
