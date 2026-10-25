import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// The two synthetic voices of the fixture are close (cosine 0.96): matching proposes
// them as one group, which stays a proposal until a human names it.
test("un groupe de voix proches se nettoie puis se nomme comme une Personne", async ({
  page,
}) => {
  await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  await page.reload();
  const card = page.locator("#voice-groups");
  await card.scrollIntoViewIfNeeded();
  await expect(card).toContainText("Groupe proposé");
  await expect(card).toContainText("2 voix · 1 clip");
  await expect(
    card.getByLabel("Extrait de SPEAKER_00 (synthetic-voices.mp4)"),
  ).toBeVisible();

  // Excluding one voice leaves the other alone; counts come from the server.
  await card
    .getByRole("button", { name: "Exclure SPEAKER_01 du groupe" })
    .click();
  await expect(card.getByText("Voix exclue du groupe.")).toBeVisible();
  await expect(card.getByText("Aucun groupe de voix inconnues.")).toBeVisible();
  await card.getByLabel("Groupes proposés uniquement").uncheck();
  await expect(card).toContainText("Voix exclue");
  await expect(card).toContainText("Voix seule");
  await card.getByRole("button", { name: "Réintégrer SPEAKER_01" }).click();
  await expect(card.getByText("Voix réintégrée.")).toBeVisible();
  await expect(card).toContainText("Groupe proposé");
  await expect(card).toContainText("2 voix · 1 clip");

  // Naming validates every member as the Person.
  await card.getByLabel("Nom du groupe").fill("Groupe Synthétique");
  await card.getByRole("button", { name: "Nommer le groupe" }).click();
  await expect(card).toContainText(
    "Groupe nommé Groupe Synthétique : 2 voix validée(s)",
  );
  await expect(card.getByText("Aucun groupe de voix inconnues.")).toBeVisible();
});
