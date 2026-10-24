import { test, expect } from "@playwright/test";

test("nouvelle installation et étapes reportées", async ({ page }) => {
  await page.goto("/installation");
  await expect(
    page.getByRole("heading", { name: "Installer votre espace" }),
  ).toBeVisible();
  await expect(
    page.getByText("Dossiers : phase 1", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Worker : phase 2", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Embeddings : phase 4", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Hugging Face : phase 5", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("La bibliothèque sera disponible à la phase 1.", {
      exact: true,
    }),
  ).toBeVisible();
});
