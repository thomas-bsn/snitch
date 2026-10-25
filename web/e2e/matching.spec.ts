import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// The synthetic fingerprints of the fixture are [0.6, 0.8] and [0.8, 0.6]: their
// cosine similarity is 0.96: above the default suggestion and automatic thresholds
// (0.75 and 0.90), yet only a suggestion because the automation is off by default.
test("une validation propose la voix proche, l'administrateur règle les seuils", async ({
  page,
}) => {
  await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const voices = page.getByRole("region", { name: "Voix du clip" });
  const speakers = voices.getByRole("list", { name: "Locuteurs locaux" });
  const first = speakers.locator("xpath=./li").nth(0);
  const second = speakers.locator("xpath=./li").nth(1);

  // The automation is off by default: matching can only suggest.
  const card = page.locator("#voice-matching");
  await card.scrollIntoViewIfNeeded();
  await expect(card).toContainText("Attribution automatique : désactivée");

  // Without any reference, voices stay unknown.
  await expect(second).toContainText("Inconnu");

  // One human validation creates the reference; the other voice is only suggested.
  await first.getByLabel("Nouvelle Personne").fill("Alice Synthétique");
  await first.getByRole("button", { name: "Valider", exact: true }).click();
  await expect(first.getByText("Validation enregistrée.")).toBeVisible();
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(second).toContainText("Suggestion");
  await expect(second).toContainText("Alice Synthétique");
  await expect(second).not.toContainText("Validé par un humain");
  const candidates = second.getByRole("list", {
    name: "Personnes proches de SPEAKER_01",
  });
  await expect(candidates).toContainText("Alice Synthétique · similarité 96 %");
  await second.getByRole("button", { name: "Historique" }).click();
  await expect(
    second.getByRole("list", { name: "Historique SPEAKER_01" }),
  ).toContainText("Rapprochement : Inconnu → Alice Synthétique");

  // Stricter thresholds re-examine the stored fingerprints: the voice is unknown again.
  await card.getByLabel("Seuil d’attribution automatique").fill("0.99");
  await card.getByLabel("Seuil de suggestion").fill("0.98");
  await card.getByRole("button", { name: "Enregistrer les seuils" }).click();
  await expect(card.getByText("Seuils enregistrés")).toBeVisible();
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(second).toContainText("Inconnu");
  await expect(second).not.toContainText("Suggestion");

  // Invalid thresholds are refused with an explicit message.
  await card.getByLabel("Seuil de suggestion").fill("0.995");
  await card.getByRole("button", { name: "Enregistrer les seuils" }).click();
  await expect(card.getByText("Seuils invalides")).toBeVisible();

  // Back to defaults, then confirm the suggestion: only the human click creates a sample.
  await card.getByLabel("Seuil d’attribution automatique").fill("0.9");
  await card.getByLabel("Seuil de suggestion").fill("0.75");
  await card.getByRole("button", { name: "Enregistrer les seuils" }).click();
  await expect(card.getByText("Seuils enregistrés")).toBeVisible();
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(second).toContainText("Suggestion");
  await second
    .getByRole("button", { name: "Valider comme Alice Synthétique" })
    .click();
  await expect(second.getByText("Validation enregistrée.")).toBeVisible();
  await expect(second).toContainText("Validé par un humain");
  await expect(second).toContainText("Alice Synthétique (2 échantillons");
});
