import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("une Personne liée à son compte voit ses clips et supprime son profil vocal", async ({
  page,
}) => {
  await prepareDiarizedClip(page);
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const voices = page.getByRole("region", { name: "Voix du clip" });
  const speakers = voices.getByRole("list", { name: "Locuteurs locaux" });
  const first = speakers.locator("xpath=./li").nth(0);
  await first.getByLabel("Nouvelle Personne").fill("Alice Synthétique");
  await first.getByRole("button", { name: "Valider", exact: true }).click();
  await expect(first.getByText("Validation enregistrée.")).toBeVisible();

  // Without a link the account has no voice card; an administrator links it.
  await expect(page.getByRole("region", { name: "Ma voix" })).toHaveCount(0);
  const accounts = page.locator("#person-accounts");
  await accounts.scrollIntoViewIfNeeded();
  await accounts
    .getByRole("button", { name: "Recharger les Personnes" })
    .click();
  await accounts
    .getByLabel("Personne à lier")
    .selectOption({ label: "Alice Synthétique" });
  await accounts.getByLabel("Nom du compte").fill("administrator");
  await accounts.getByRole("button", { name: "Lier au compte" }).click();
  await expect(accounts.getByText("est liée au compte")).toBeVisible();

  await page.reload();
  const mine = page.getByRole("region", { name: "Ma voix" });
  await expect(mine).toContainText("Alice Synthétique");
  await expect(mine).toContainText("1 échantillon(s) vocal(aux) validé(s)");
  await expect(mine.getByRole("list", { name: "Mes clips" })).toContainText(
    "synthetic-voices.mp4",
  );

  // The deletion asks for a confirmation, then erases the profile.
  await mine
    .getByRole("button", { name: "Supprimer mon profil vocal" })
    .click();
  await mine.getByRole("button", { name: "Garder mon profil" }).click();
  await expect(mine).toContainText("1 échantillon(s) vocal(aux) validé(s)");
  await mine
    .getByRole("button", { name: "Supprimer mon profil vocal" })
    .click();
  await mine.getByRole("button", { name: "Confirmer la suppression" }).click();
  await expect(mine).toContainText("Profil vocal supprimé");
  await expect(mine).toContainText("Snitch ne vous reconnaît plus");
  await expect(mine).toContainText("Aucun clip visible pour vous.");

  // The voice is unknown again and the Person can no longer receive validations.
  await page.getByText("Détails et lecture", { exact: true }).click();
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const voicesAfter = page.getByRole("region", { name: "Voix du clip" });
  const firstAfter = voicesAfter
    .getByRole("list", { name: "Locuteurs locaux" })
    .locator("xpath=./li")
    .nth(0);
  await expect(firstAfter).toContainText("Inconnu");
  await expect(firstAfter).not.toContainText("Validé par un humain");
});
