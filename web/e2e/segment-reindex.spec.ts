import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un administrateur voit les segments corrigés en attente de réindexation", async ({
  page,
}) => {
  await prepareDiarizedClip(page);
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const index = page.getByRole("region", { name: "Index sémantique" });
  await expect(index).toContainText(
    "Segments corrigés en attente de réindexation : 0",
  );

  // The synthetic stack has no embeddings service: the correction is recorded and
  // the segment waits for reindexing.
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
  await expect(transcript).toContainText("Bien joué, Alice !");
  await expect(index).toContainText(
    "Segments corrigés en attente de réindexation : 1",
    { timeout: 15000 },
  );
});
