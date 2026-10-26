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

  // Without a link the account has no voice profile; an administrator links it.
  await page.goto("/account/voice");
  await expect(page.getByRole("main")).toContainText(
    "No person is linked to your account",
  );
  await page.goto("/legacy/");
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

  // The voice card is now in the account settings of the new interface.
  await page.goto("/account/voice");
  const mine = page.getByRole("main");
  await expect(mine).toContainText("Alice Synthétique");
  await expect(mine).toContainText("1 validated voice sample(s)");
  await expect(mine.getByRole("list", { name: "My clips" })).toContainText(
    "synthetic-voices.mp4",
  );

  // The deletion explains its effects, asks for a confirmation, then erases the profile.
  await expect(mine).toContainText("erases your validated voice samples");
  await mine.getByRole("button", { name: "Delete my voice profile" }).click();
  const dialog = page.getByRole("alertdialog", {
    name: "Delete your voice profile?",
  });
  await dialog.getByRole("button", { name: "Keep my profile" }).click();
  await expect(dialog).toHaveCount(0);
  await expect(mine).toContainText("1 validated voice sample(s)");
  await mine.getByRole("button", { name: "Delete my voice profile" }).click();
  await page
    .getByRole("alertdialog")
    .getByRole("button", { name: "Delete my voice profile" })
    .click();
  await expect(mine.getByRole("status")).toContainText(
    "1 sample(s), 1 fingerprint(s) and 1 attribution(s) erased",
  );
  await expect(mine).toContainText("Snitch no longer recognizes you");
  await expect(mine).toContainText("No clip visible to you.");
  await expect(
    mine.getByRole("button", { name: "Delete my voice profile" }),
  ).toHaveCount(0);

  // The voice is unknown again and the Person can no longer receive validations.
  await page.goto("/legacy/");
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
