import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un utilisateur fusionne, sépare et supprime des Personnes créées par erreur", async ({
  page,
}) => {
  await prepareDiarizedClip(page);
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const voices = page.getByRole("region", { name: "Voix du clip" });
  const speakers = voices.getByRole("list", { name: "Locuteurs locaux" });
  const first = speakers.locator("xpath=./li").nth(0);
  const second = speakers.locator("xpath=./li").nth(1);

  // The same identity was created twice, by mistake.
  await first.getByLabel("Nouvelle Personne").fill("Alice Synthétique");
  await first.getByRole("button", { name: "Valider", exact: true }).click();
  await expect(first.getByText("Validation enregistrée.")).toBeVisible();
  await second.getByLabel("Nouvelle Personne").fill("Alicia Synthétique");
  await second.getByRole("button", { name: "Valider", exact: true }).click();
  await expect(second.getByText("Validation enregistrée.")).toBeVisible();

  const card = page.locator("#persons");
  await card.scrollIntoViewIfNeeded();
  await card.getByRole("button", { name: "Actualiser les Personnes" }).click();
  const list = card.getByRole("list", { name: "Liste des Personnes" });
  await expect(list).toContainText("Alicia Synthétique");

  // Merge: nothing happens before the confirmation.
  await card
    .getByRole("button", { name: "Fusionner Alicia Synthétique" })
    .click();
  await card
    .getByLabel("Fusionner Alicia Synthétique dans")
    .selectOption({ label: "Alice Synthétique" });
  await card.getByRole("button", { name: "Confirmer la fusion" }).click();
  await expect(card).toContainText("Fusion de Alicia Synthétique faite");
  await expect(list).not.toContainText("Alicia Synthétique");
  await expect(list).toContainText("2 échantillons validés");
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(second).toContainText("Alice Synthétique");
  await second.getByRole("button", { name: "Historique" }).click();
  await expect(
    second.getByRole("list", { name: "Historique SPEAKER_01" }),
  ).toContainText(
    "Fusion de Personnes : Alicia Synthétique → Alice Synthétique",
  );
  await card.getByRole("button", { name: "Journal des corrections" }).click();
  await expect(
    card.getByRole("list", {
      name: "Historique des corrections de Personnes",
    }),
  ).toContainText("Fusion : Alicia Synthétique → Alice Synthétique");

  // Separation of one voice: asked, then confirmed.
  await second
    .getByRole("button", { name: "Séparer de cette Personne" })
    .click();
  await second.getByRole("button", { name: "Annuler la séparation" }).click();
  await expect(second).toContainText("Validé par un humain");
  await second
    .getByRole("button", { name: "Séparer de cette Personne" })
    .click();
  await second.getByRole("button", { name: "Confirmer la séparation" }).click();
  await expect(
    second.getByText(
      "Voix séparée de la Personne : elle est de nouveau inconnue.",
    ),
  ).toBeVisible();
  await expect(second).toContainText("Inconnu");
  await expect(second).not.toContainText("Validé par un humain");
  await expect(
    second.getByRole("list", { name: "Historique SPEAKER_01" }),
  ).toContainText("Séparation de la Personne : Alice Synthétique → Inconnu");

  // Deletion: the first voice is unknown again, the clip stays.
  await card.getByRole("button", { name: "Actualiser les Personnes" }).click();
  await card
    .getByRole("button", { name: "Supprimer Alice Synthétique" })
    .click();
  await card.getByRole("button", { name: "Annuler la correction" }).click();
  await expect(list).toContainText("Alice Synthétique");
  await card
    .getByRole("button", { name: "Supprimer Alice Synthétique" })
    .click();
  await card.getByRole("button", { name: "Confirmer la suppression" }).click();
  await expect(card).toContainText("Suppression de Alice Synthétique faite");
  await expect(list.locator("li")).toHaveCount(0);
  await expect(card).toContainText("Aucune Personne.");
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  await expect(first).toContainText("Inconnu");
  await expect(first).not.toContainText("Validé par un humain");
  await expect(voices).toBeVisible();
});
