import { test, expect } from "@playwright/test";

test("un administrateur ajoute une source et voit ses clips validés", async ({
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
    page.getByRole("heading", { name: "Sources dossier" }),
  ).toBeVisible();
  await page.getByLabel("Nom de la source").fill("Captures de test");
  await page
    .getByLabel("Chemin du dossier autorisé")
    .fill(process.env.SNITCH_TEST_SOURCE_ROOT!);
  await page.getByRole("button", { name: "Ajouter le dossier" }).click();
  await expect(
    page.getByText("Captures de test", { exact: true }),
  ).toBeVisible();
  await page
    .getByText("Règle de piste audio pour cette source", { exact: true })
    .click();
  await page.getByLabel("Titre de piste exact").fill("Voix");
  await page.getByLabel("Langue exacte").fill("fra");
  await page.getByLabel("Disposition").fill("stereo");
  await page.getByLabel("Piste audio de repli (index global)").fill("2");
  await page.getByRole("button", { name: "Enregistrer la règle" }).click();
  await expect(page.getByRole("status")).toContainText(
    "Règle de piste enregistrée",
  );

  await expect
    .poll(async () => {
      await page
        .getByRole("button", { name: "Actualiser les sources" })
        .click();
      return await page.getByText(/browser-fixture\.mp4.*ready/).count();
    })
    .toBe(1);
  await expect(page.getByText(/4\.25 s · h264/)).toBeVisible();
  const sourceRuleSelection = await page.evaluate(async () => {
    const sources = await fetch("/api/admin/sources").then((response) =>
      response.json(),
    );
    const source = sources.items.find(
      (item: { name: string }) => item.name === "Captures de test",
    );
    const files = await fetch(`/api/admin/sources/${source.id}/files`).then(
      (response) => response.json(),
    );
    const ready = files.items.find(
      (file: { status: string; clip_id?: number }) =>
        file.status === "ready" && file.clip_id,
    );
    return fetch(`/api/clips/${ready.clip_id}`).then((response) =>
      response.json(),
    );
  });
  expect(sourceRuleSelection.audio_track_selection).toMatchObject({
    status: "selected",
    audio_stream_index: 2,
    selection_source: "source_rule",
    rule_version: 1,
  });
});
