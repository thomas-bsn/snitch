import { expect, test } from "@playwright/test";

test("un administrateur désigne la piste micro d'une source puis la remplace pour un clip", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByLabel("Code d’installation")
    .fill(process.env.SNITCH_TEST_CODE!);
  await page.getByLabel("Nom d’utilisateur").fill("administrator");
  await page
    .getByLabel("Mot de passe", { exact: true })
    .fill("synthetic-password-123");
  await page.getByRole("button", { name: "Créer l’administrateur" }).click();

  await page.getByLabel("Nom de la source").fill("Captures micro");
  await page
    .getByLabel("Chemin du dossier autorisé")
    .fill(process.env.SNITCH_TEST_SOURCE_ROOT!);
  await page.getByRole("button", { name: "Ajouter le dossier" }).click();
  await expect
    .poll(async () => {
      await page
        .getByRole("button", { name: "Actualiser les sources" })
        .click();
      return await page.getByText(/browser-fixture\.mp4.*ready/).count();
    })
    .toBe(1);

  // The card lists the tracks really detected, never a built-in title.
  await page.reload();
  await page
    .getByText("Piste micro pour cette source", { exact: true })
    .click();
  const sourceCard = page.getByRole("region", {
    name: "Piste micro de la source",
  });
  await expect(sourceCard).toContainText("Aucune piste micro désignée.");
  const detected = sourceCard.getByRole("list", { name: "Pistes détectées" });
  await expect(detected).toContainText("Piste 1 · Micro · fra · 1 clip");
  await expect(detected).toContainText("Piste 2 · Voix · fra · 1 clip");

  await sourceCard.getByLabel("Piste micro détectée").selectOption("1");
  await sourceCard.getByLabel("son titre exact").check();
  await sourceCard
    .getByLabel("Nouvelle Personne propriétaire")
    .fill("Alice Synthétique");
  await sourceCard
    .getByRole("button", { name: "Désigner la piste micro" })
    .click();
  await expect(sourceCard).toContainText(
    "Piste micro enregistrée : 1 clip désigné.",
  );
  await expect(sourceCard).toContainText(
    "Désignée : titre « Micro » → Alice Synthétique",
  );

  // The clip shows how each track is treated and carries the owner without diarization.
  await page.getByText("Détails et lecture", { exact: true }).click();
  const clipCard = page.getByRole("region", { name: "Piste micro du clip" });
  await expect(clipCard).toContainText("Piste micro désignée");
  await expect(clipCard).toContainText("Propriétaire : Alice Synthétique");
  const tracks = clipCard.getByRole("list", { name: "Pistes audio du clip" });
  await expect(tracks).toContainText("Piste 1 · Micro · piste micro");
  await expect(tracks).toContainText("Piste 2 · Voix · traitée normalement");
  const speakers = page
    .getByRole("region", { name: "Voix du clip" })
    .getByRole("list", { name: "Locuteurs locaux" });
  await expect(speakers).toContainText("Alice Synthétique");
  await expect(speakers).toContainText("Attribution automatique");

  // Per-clip replacement is recorded in the history.
  await clipCard
    .getByRole("button", { name: "Pas de piste micro pour ce clip" })
    .click();
  await expect(clipCard).toContainText("Ce clip n’a pas de piste micro.");
  await expect(
    clipCard.getByRole("list", { name: "Historique de la piste micro" }),
  ).toContainText("pas de piste micro");
  await expect(speakers).toHaveCount(0);
  await clipCard
    .getByRole("button", { name: "Revenir à la règle de la source" })
    .click();
  await expect(clipCard).toContainText("Règle de la source rétablie.");
  await expect(clipCard).toContainText("Piste micro désignée");
  await expect(
    clipCard.getByRole("list", { name: "Historique de la piste micro" }),
  ).toContainText("par administrator");

  // Changing the rule moves the designation; removing it withdraws the attribution.
  await sourceCard.getByLabel("son numéro", { exact: true }).check();
  await sourceCard.getByLabel("Piste micro détectée").selectOption("2");
  await sourceCard.getByLabel("son numéro et son titre").check();
  await sourceCard.getByLabel("Propriétaire du micro").selectOption({
    label: "Alice Synthétique",
  });
  await sourceCard
    .getByRole("button", { name: "Désigner la piste micro" })
    .click();
  await expect(sourceCard).toContainText(
    "Piste micro enregistrée : 1 clip désigné.",
  );
  await sourceCard
    .getByRole("button", { name: "Retirer la désignation" })
    .click();
  await expect(sourceCard).toContainText("Désignation retirée");
  await expect(sourceCard).toContainText("Aucune piste micro désignée.");
});
