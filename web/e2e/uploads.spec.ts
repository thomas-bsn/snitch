import { test, expect } from "@playwright/test";

test("un membre envoie une vidéo synthétique", async ({ page }) => {
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
    page.getByRole("heading", { name: "Ajouter à la bibliothèque" }),
  ).toBeVisible();
  await page.getByLabel("Fichier vidéo").setInputFiles({
    name: "synthetic-browser-upload.mp4",
    mimeType: "video/mp4",
    buffer: Buffer.from("synthetic-browser-video-content"),
  });
  await page.getByRole("button", { name: "Envoyer la vidéo" }).click();
  await expect(page.getByRole("status")).toContainText(
    "a été vérifiée et ajoutée",
  );
});
