import { expect, test } from "@playwright/test";

test("un administrateur planifie et filtre une tâche persistante", async ({
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

  await page.getByLabel("Fichier vidéo").setInputFiles({
    name: "synthetic-job-clip.mp4",
    mimeType: "video/mp4",
    buffer: Buffer.from("synthetic-job-video-content"),
  });
  await page.getByRole("button", { name: "Envoyer la vidéo" }).click();
  await expect(page.getByRole("status")).toContainText(
    "a été vérifiée et ajoutée",
  );
  const clipID = await page.evaluate(async () => {
    const response = await fetch("/api/clips?page=1&page_size=10");
    const page = await response.json();
    return page.items[0].id as number;
  });

  await page.getByLabel("ID du clip", { exact: true }).fill(String(clipID));
  await page.getByLabel("Étape", { exact: true }).fill("fixture.inspect");
  await page.getByLabel("Version d’entrée").fill("fixture-v1");
  await page.getByLabel("Priorité").fill("5");
  await page.getByRole("button", { name: "Planifier l’étape" }).click();
  await expect(page.getByRole("status")).toHaveText("Tâche planifiée.");
  await expect(
    page.getByRole("heading", { name: "File de tâches (1)" }),
  ).toBeVisible();
  await expect(
    page
      .getByRole("list", { name: "Tâches" })
      .getByText(new RegExp(`fixture\\.inspect.*clip ${clipID} · pending`)),
  ).toBeVisible();

  await page.getByLabel("ID du clip", { exact: true }).fill(String(clipID));
  await page.getByLabel("Étape", { exact: true }).fill("fixture.inspect");
  await page.getByLabel("Version d’entrée").fill("fixture-v1");
  await page.getByRole("button", { name: "Planifier l’étape" }).click();
  await expect(page.getByRole("status")).toHaveText(
    "Cette étape et cette version sont déjà dans la file.",
  );
  await expect(
    page.getByRole("heading", { name: "File de tâches (1)" }),
  ).toBeVisible();

  await page.getByLabel("Étape à filtrer").fill("fixture.inspect");
  await page.getByRole("button", { name: "Filtrer les tâches" }).click();
  await expect(
    page.getByRole("list", { name: "Tâches" }).getByText("fixture.inspect"),
  ).toBeVisible();
});
