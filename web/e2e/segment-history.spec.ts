import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un utilisateur consulte l’historique d’un segment et le restaure", async ({
  page,
}) => {
  await prepareDiarizedClip(page);
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const transcript = page.getByRole("region", {
    name: "Transcription horodatée",
  });
  await expect(transcript).toContainText("Bien joué");

  // A never corrected segment has an empty history.
  await transcript
    .getByRole("button", { name: "Historique du segment 1" })
    .click();
  await expect(
    transcript.getByText("Aucune correction pour ce segment."),
  ).toBeVisible();
  await transcript
    .getByRole("button", { name: "Historique du segment 1" })
    .click();

  // Two successive corrections.
  for (const [text, reason] of [
    ["Bien joué, Alice !", "nom"],
    ["Bien joué, Alicia !", "orthographe"],
  ]) {
    await transcript
      .getByRole("button", { name: "Modifier le texte du segment 1" })
      .click();
    await transcript
      .getByRole("textbox", { name: "Texte du segment 1" })
      .fill(text);
    await transcript
      .getByRole("textbox", { name: "Motif du segment 1 (facultatif)" })
      .fill(reason);
    await transcript
      .getByRole("button", { name: "Enregistrer le texte du segment 1" })
      .click();
    await expect(transcript).toContainText(text);
  }

  // The history shows old and new values, author and reason, newest first.
  await transcript
    .getByRole("button", { name: "Historique du segment 1" })
    .click();
  const entries = transcript.locator(".segment-history-list li");
  await expect(entries).toHaveCount(2);
  await expect(entries.nth(0)).toContainText("version 2 par administrator");
  await expect(entries.nth(0)).toContainText("motif : orthographe");
  await expect(entries.nth(0)).toContainText("Avant : Bien joué, Alice !");
  await expect(entries.nth(0)).toContainText("Après : Bien joué, Alicia !");
  await expect(entries.nth(1)).toContainText("Avant : Bien joué !");
  await expect(
    transcript.getByRole("button", {
      name: "Restaurer la version 2 du segment 1",
    }),
  ).toHaveCount(0);

  // Restoring a past version is a new history entry.
  await transcript
    .getByRole("textbox", {
      name: "Motif de la restauration du segment 1 (facultatif)",
    })
    .fill("retour");
  await transcript
    .getByRole("button", { name: "Restaurer la version 1 du segment 1" })
    .click();
  await expect(transcript.getByText("Texte restauré.")).toBeVisible();
  await expect(entries).toHaveCount(3);
  await expect(entries.nth(0)).toContainText("Restauration version 3");
  await expect(entries.nth(0)).toContainText("motif : retour");
  await expect(entries.nth(2)).toContainText("Avant : Bien joué !");
  await expect(transcript.locator(".segment-body").first()).toHaveText(
    "Bien joué, Alice !",
  );

  // Restoring the original removes the correction label and keeps the trail.
  await transcript
    .getByRole("button", { name: "Restaurer le texte d’origine du segment 1" })
    .click();
  await expect(entries).toHaveCount(4);
  await expect(transcript.locator(".segment-body").first()).toContainText(
    "Bien joué",
  );
  await expect(transcript.locator(".segment-body").first()).not.toContainText(
    "Alice",
  );
  await expect(transcript.getByText("Corrigé", { exact: true })).toHaveCount(0);
  await expect(
    transcript.getByRole("button", {
      name: "Restaurer le texte d’origine du segment 1",
    }),
  ).toHaveCount(0);
  await expect(entries.nth(0)).toContainText("Restauration version 4");

  // The history survives a reload.
  await page.reload();
  await page.getByText("Détails et lecture", { exact: true }).click();
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await transcript
    .getByRole("button", { name: "Historique du segment 1" })
    .click();
  await expect(entries).toHaveCount(4);
});
