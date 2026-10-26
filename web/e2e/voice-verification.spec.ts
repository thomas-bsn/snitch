import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// Two close synthetic voices stay unknown until a human settles one of them: the other
// then only becomes a suggestion, so it leaves the review without any human validation.
test("la vérification ne montre que les inconnus, se tranche et met à jour les compteurs", async ({
  page,
}) => {
  await prepareDiarizedClip(page, { secondFingerprintEnd: 4.2 });
  await page.reload();
  const card = page.locator("#voice-verification");
  await card.scrollIntoViewIfNeeded();

  // The five states are always told apart by label and glyph, with counts from the server.
  const counts = card.getByRole("list", { name: "Voix par état" });
  for (const label of [
    "Validé par un humain",
    "Attribution automatique",
    "Suggestion",
    "Ambigu",
    "Inconnu",
  ]) {
    await expect(counts.locator("li").filter({ hasText: label })).toHaveCount(
      1,
    );
  }
  await expect(
    counts.locator("li").filter({ hasText: "Inconnu" }),
  ).toContainText("2");
  await expect(card).toContainText("2 cas à vérifier");
  const cases = card.getByRole("list", { name: "Cas à vérifier" });
  await expect(cases.locator("xpath=./li")).toHaveCount(2);
  await expect(cases).toContainText("Groupe proposé : 2 voix dans 1 clip.");
  await expect(cases).not.toContainText("Validé par un humain");
  await expect(
    card.getByLabel("Extrait à vérifier de SPEAKER_00 (synthetic-voices.mp4)"),
  ).toBeVisible();

  // No case is ambiguous: the filter leaves nothing to review.
  await card.getByLabel("Cas affichés").selectOption("ambiguous");
  await expect(card).toContainText("Aucune voix à vérifier.");
  await card.getByLabel("Cas affichés").selectOption("all");
  await expect(card).toContainText("2 cas à vérifier");

  // Settling one case is a human validation; the counters follow.
  await card
    .getByLabel("Nouvelle Personne pour SPEAKER_00")
    .fill("Claire Synthétique");
  await card
    .getByRole("button", { name: "Trancher SPEAKER_00", exact: true })
    .click();
  await expect(card).toContainText(
    "cas tranché, validation humaine enregistrée.",
  );
  await expect(card).toContainText("Aucune voix à vérifier.");
  await expect(
    counts.locator("li").filter({ hasText: "Validé par un humain" }),
  ).toContainText("1");
  await expect(
    counts.locator("li").filter({ hasText: "Suggestion" }),
  ).toContainText("1");
  await expect(
    counts.locator("li").filter({ hasText: "Inconnu" }),
  ).toContainText("0");
});
