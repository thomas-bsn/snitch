import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// The synthetic fingerprints of the fixture have a cosine similarity of 0.96: above the
// default automatic threshold (0.90), so the second voice is a suggestion until the
// administrator turns the automation on, and a suggestion again once it is off.
test("l'administrateur mesure, active puis désactive l'attribution automatique", async ({
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
  await first.getByLabel("Nouvelle Personne").fill("Alice Synthétique");
  await first.getByRole("button", { name: "Valider", exact: true }).click();
  await expect(first.getByText("Validation enregistrée.")).toBeVisible();

  const card = page.locator("#voice-automation");
  await card.scrollIntoViewIfNeeded();
  await expect(card).toContainText("État : désactivée");

  // Saving the criteria also reloads the figures computed by the server.
  await card.getByLabel("Confirmations minimales").fill("1");
  await card
    .getByLabel("Précision minimale pour proposer l’activation")
    .fill("0.5");
  await card
    .getByRole("button", { name: "Enregistrer les critères d’activation" })
    .click();
  await expect(card.getByText("Critères enregistrés.")).toBeVisible();

  // One reference and no test: nothing is measured, nothing is claimed.
  const figures = card.getByRole("list", { name: "Mesure de précision" });
  await expect(figures).toContainText("Précision mesurée : non mesurée");
  await expect(figures).toContainText("Confirmations : 0");
  await expect(card).toContainText("ne garantissent aucune qualité");
  const library = card.getByRole("list", { name: "Bibliothèque" });
  await expect(library).toContainText("Voix automatiques ou éligibles : 1");
  await expect(library).toContainText("Couverture automatique : 100 %");
  await expect(card.getByText("Snitch vous propose d’activer")).toHaveCount(0);
  await expect(
    card.getByRole("table", { name: "Précision par Personne" }),
  ).toContainText("Alice Synthétique");

  // Below the criteria the activation needs an explicit confirmation.
  const activate = card.getByRole("button", {
    name: "Activer l’attribution automatique",
  });
  await expect(activate).toBeDisabled();
  await card.getByLabel("Activer malgré des critères non atteints").check();
  await expect(activate).toBeEnabled();
  await activate.click();
  await expect(card.getByText("Attribution automatique activée")).toBeVisible();
  await expect(card).toContainText("État : activée");
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(second).toContainText("Attribution automatique");
  await expect(second).toContainText("Alice Synthétique");
  await expect(second).not.toContainText("Validé par un humain");
  await expect(
    card.getByRole("list", { name: "Historique de l’automatisation" }),
  ).toContainText("Activation par administrator (critères non atteints)");

  // Deactivation brings the automatic attribution back to a suggestion.
  await card
    .getByRole("button", { name: "Désactiver l’attribution automatique" })
    .click();
  await expect(card.getByText("redevenues des suggestions")).toBeVisible();
  await expect(card).toContainText("État : désactivée");
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(second).toContainText("Suggestion");
  await expect(second).not.toContainText("Attribution automatique");
});
