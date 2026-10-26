import { readFile } from "node:fs/promises";
import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un administrateur télécharge l’export des segments corrigés", async ({
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

  const card = page.getByRole("region", { name: "Export des corrections" });
  await expect(card).toContainText("Snitch ne calcule aucun taux d’erreur");
  const downloading = page.waitForEvent("download");
  await card
    .getByRole("link", { name: "Télécharger l’export des segments corrigés" })
    .click();
  const download = await downloading;
  expect(download.suggestedFilename()).toBe("segment-corrections.ndjson");
  const lines = (await readFile((await download.path())!, "utf8"))
    .trim()
    .split("\n")
    .map((line) => JSON.parse(line));
  expect(lines).toHaveLength(1);
  expect(lines[0]).toMatchObject({
    index: 0,
    corrected_text: "Bien joué, Alice !",
    author: { username: "administrator" },
    version: 1,
  });
  expect(lines[0].original_text).not.toBe("Bien joué, Alice !");
});
