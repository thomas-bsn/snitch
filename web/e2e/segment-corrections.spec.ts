import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un utilisateur corrige le texte d’un segment sans toucher à l’original", async ({
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
  await expect(transcript).not.toContainText("Corrigé");

  // Cancelling leaves the segment untouched.
  await transcript
    .getByRole("button", { name: "Modifier le texte du segment 1" })
    .click();
  await transcript
    .getByRole("textbox", { name: "Texte du segment 1" })
    .fill("Texte abandonné");
  await transcript
    .getByRole("button", { name: "Abandonner la modification du segment 1" })
    .click();
  await expect(transcript).toContainText("Bien joué");
  await expect(transcript).not.toContainText("Texte abandonné");

  // An empty text cannot be saved.
  await transcript
    .getByRole("button", { name: "Modifier le texte du segment 1" })
    .click();
  await transcript
    .getByRole("textbox", { name: "Texte du segment 1" })
    .fill("   ");
  await expect(
    transcript.getByRole("button", {
      name: "Enregistrer le texte du segment 1",
    }),
  ).toBeDisabled();

  // A correction shows the corrected text, its author and date, and the original.
  await transcript
    .getByRole("textbox", { name: "Texte du segment 1" })
    .fill("Bien joué, Alice !");
  await transcript
    .getByRole("textbox", { name: "Motif du segment 1 (facultatif)" })
    .fill("nom");
  await transcript
    .getByRole("button", { name: "Enregistrer le texte du segment 1" })
    .click();
  await expect(transcript.getByText("Correction enregistrée.")).toBeVisible();
  await expect(transcript).toContainText("Bien joué, Alice !");
  await expect(transcript).toContainText("Corrigé par administrator le");
  await expect(transcript.getByText("Texte d’origine :")).toHaveCount(0);
  await transcript
    .getByRole("button", { name: "Texte d’origine du segment 1" })
    .click();
  await expect(transcript).toContainText("Texte d’origine : Bien joué !");

  // The correction survives a reload; the other segment stays automatic and is
  // corrected independently. The segment start (timecode button) is unchanged.
  await page.reload();
  await page.getByText("Détails et lecture", { exact: true }).click();
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(transcript).toContainText("Bien joué, Alice !");
  await expect(transcript.getByText("Corrigé", { exact: true })).toHaveCount(1);
  await transcript
    .getByRole("button", { name: "Modifier le texte du segment 2" })
    .click();
  await transcript
    .getByRole("textbox", { name: "Texte du segment 2" })
    .fill("Merci beaucoup");
  await transcript
    .getByRole("button", { name: "Enregistrer le texte du segment 2" })
    .click();
  await expect(transcript).toContainText("Merci beaucoup");
  await expect(transcript.getByText("Corrigé", { exact: true })).toHaveCount(2);
  await expect(transcript.getByRole("button", { name: "00:01" })).toBeVisible();
});
