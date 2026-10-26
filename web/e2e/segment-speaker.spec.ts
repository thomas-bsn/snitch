import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un utilisateur corrige le locuteur d’un segment puis revient au locuteur déduit", async ({
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
  const first = transcript.locator("p").first();
  await expect(first.locator(".speaker-badge")).toContainText("SPEAKER_00");

  // Nothing to save until another speaker is chosen.
  const save = transcript.getByRole("button", {
    name: "Enregistrer le locuteur du segment 1",
  });
  await expect(save).toBeDisabled();
  await transcript
    .getByLabel("Locuteur du segment 1", { exact: true })
    .selectOption({ label: "SPEAKER_01" });
  await save.click();
  await expect(transcript.getByText("Locuteur enregistré.")).toBeVisible();
  await expect(first.locator(".speaker-badge")).toContainText("SPEAKER_01");
  await expect(first).toContainText("Locuteur corrigé");

  // The history of the segment lists the speaker change.
  await transcript
    .getByRole("button", { name: "Historique du segment 1" })
    .click();
  const entries = transcript.locator(".segment-history-list li");
  await expect(entries).toHaveCount(1);
  await expect(entries.nth(0)).toContainText(
    "Locuteur : SPEAKER_00 → SPEAKER_01",
  );
  await expect(entries.nth(0)).toContainText("administrator");

  // Returning to the deduced speaker is a new history entry.
  await transcript
    .getByLabel("Locuteur du segment 1", { exact: true })
    .selectOption({ label: "Locuteur déduit des tours de parole" });
  await save.click();
  await expect(first.locator(".speaker-badge")).toContainText("SPEAKER_00");
  await expect(first).not.toContainText("Locuteur corrigé");
  await expect(entries).toHaveCount(2);
});
