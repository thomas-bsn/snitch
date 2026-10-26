import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("retranscrire un clip corrigé exige une confirmation et le lot l’exclut par défaut", async ({
  page,
}) => {
  await prepareDiarizedClip(page);
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const transcript = page.getByRole("region", {
    name: "Transcription horodatée",
  });
  await transcript
    .getByRole("button", { name: "Modifier le texte du segment 1" })
    .click();
  await transcript
    .getByRole("textbox", { name: "Texte du segment 1" })
    .fill("Bien joué, Alice !");
  await transcript
    .getByRole("button", { name: "Enregistrer le texte du segment 1" })
    .click();
  await expect(transcript.getByText("Correction enregistrée.")).toBeVisible();

  // Retranscribing warns about what would be replaced; cancelling changes nothing.
  await page.getByRole("button", { name: "Transcrire", exact: true }).click();
  const warning = page.getByRole("region", { name: "Corrections à remplacer" });
  await expect(warning).toContainText(
    "1 texte(s) corrigé(s) par administrator",
  );
  await expect(warning).toContainText("marquées obsolètes");
  await warning
    .getByRole("button", { name: "Annuler la retranscription" })
    .click();
  await expect(warning).toHaveCount(0);
  await expect(transcript).toContainText("Bien joué, Alice !");

  // The explicit confirmation schedules the retranscription; the visible text and its
  // correction stay until it succeeds.
  await page.getByRole("button", { name: "Transcrire", exact: true }).click();
  await warning
    .getByRole("button", {
      name: "Confirmer la retranscription et remplacer les corrections",
    })
    .click();
  await expect(page.getByRole("status")).toContainText(
    "Transcription planifiée",
  );
  await expect(warning).toHaveCount(0);
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(transcript).toContainText("Bien joué, Alice !");

  // A new vocabulary version makes the corrected clip eligible again: the batch
  // reports it and excludes it unless explicitly asked.
  await page.evaluate(async () => {
    const me = await fetch("/api/auth/me").then((response) => response.json());
    await fetch("/api/admin/vocabulary", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": me.csrf_token,
        Origin: location.origin,
      },
      body: JSON.stringify({
        term: "Terme synthétique",
        variants: [],
        preferred_spelling: "Terme synthétique",
        active: true,
      }),
    });
  });
  const batch = page.getByLabel("Planification par lot");
  await batch.getByRole("button", { name: "Prévisualiser le lot" }).click();
  await expect(batch).toContainText("0 éligible(s)");
  await expect(batch).toContainText(
    "1 clip(s) corrigé(s) (1 correction(s)) exclu(s) du lot",
  );
  await expect(
    batch.getByRole("button", { name: /Confirmer et planifier/ }),
  ).toBeDisabled();
  await batch
    .getByLabel(
      "Inclure les clips corrigés (leurs corrections seront remplacées)",
    )
    .check();
  await batch.getByRole("button", { name: "Prévisualiser le lot" }).click();
  await expect(batch).toContainText("1 éligible(s)");
  await expect(batch).toContainText("inclus à votre demande");
  page.once("dialog", (dialog) => dialog.accept());
  await batch.getByRole("button", { name: "Confirmer et planifier 1" }).click();
  await expect(
    page.getByText("1 tâche(s) ajoutée(s) à la file.", { exact: true }),
  ).toBeVisible();
});
