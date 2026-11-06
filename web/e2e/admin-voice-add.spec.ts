import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// Le serveur de test pointe l'inspection sur un faux Hugging Face de trois dépôts synthétiques.

test.beforeAll(async () => {
  await installAdministrator();
});

async function open(browser: import("@playwright/test").Browser) {
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  await page.getByRole("button", { name: "Add a model" }).click();
  return page;
}

async function inspect(page: Page, source: string) {
  const dialog = page.getByRole("dialog", {
    name: "Add a model to the catalogue",
  });
  await dialog.getByLabel("Hugging Face repository").fill(source);
  await dialog.getByRole("button", { name: "Inspect" }).click();
  return dialog;
}

test("un dépôt connu : type proposé, moteur déduit, ajout au catalogue", async ({
  browser,
}) => {
  const page = await open(browser);
  const dialog = await inspect(page, "nvidia/synthetic-sortformer");
  await expect(dialog).toContainText("Proposed type: Diarization");
  await expect(
    dialog.getByRole("button", { name: "Diarization" }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(dialog.getByText("NVIDIA NeMo")).toBeVisible();
  await expect(dialog).toContainText("Compatible engine");
  await expect(dialog).toContainText("cd03eee…");
  await expect(dialog).toContainText("apache-2.0");
  // Le moteur n'est jamais un choix : aucun sélecteur de moteur n'existe.
  await expect(dialog.getByRole("combobox")).toHaveCount(0);
  await expect(dialog.getByRole("radio")).toHaveCount(0);
  const { violations } = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(violations).toEqual([]);
  await dialog.getByRole("button", { name: "Add to the catalogue" }).click();
  await expect(
    page.getByText("nvidia/synthetic-sortformer was added to the catalogue."),
  ).toBeVisible();
});

test("un type incertain : la personne choisit parmi les trois types", async ({
  browser,
}) => {
  const page = await open(browser);
  const dialog = await inspect(page, "acme/unsure-pyannote");
  await expect(dialog).toContainText("does not state its type reliably");
  for (const kind of ["Diarization", "Fingerprint", "Separation"])
    await expect(
      dialog.getByRole("button", { name: kind, exact: true }),
    ).toHaveAttribute("aria-pressed", "false");
  const add = dialog.getByRole("button", { name: "Add to the catalogue" });
  await expect(add).toBeDisabled();
  await dialog.getByRole("button", { name: "Fingerprint" }).click();
  await expect(add).toBeEnabled();
  // Le moteur ne sait pas exécuter la séparation : refus même avec ce type choisi.
  await dialog.getByRole("button", { name: "Separation" }).click();
  await expect(dialog).toContainText("cannot run a Separation model");
  await expect(add).toBeDisabled();
});

test("un dépôt incompatible : raison affichée, ajout impossible", async ({
  browser,
}) => {
  const page = await open(browser);
  const dialog = await inspect(page, "acme/transformers-only");
  await expect(dialog).toContainText("Not supported");
  await expect(dialog).toContainText("It cannot be installed");
  await dialog.getByRole("button", { name: "Diarization" }).click();
  await expect(
    dialog.getByRole("button", { name: "Add to the catalogue" }),
  ).toBeDisabled();
});

test("un dépôt inconnu ou une adresse étrangère sont refusés avec un message clair", async ({
  browser,
}) => {
  const page = await open(browser);
  const dialog = await inspect(page, "acme/does-not-exist");
  await expect(dialog.getByRole("alert")).toContainText("does not know");
  await dialog
    .getByLabel("Hugging Face repository")
    .fill("https://evil.example/acme/x");
  await dialog.getByRole("button", { name: "Inspect" }).click();
  await expect(dialog.getByRole("alert")).toContainText(
    "Paste a Hugging Face repository",
  );
});

test("le changement d'empreinte montre l'impact réel puis attend un ordre de préparation explicite", async ({
  browser,
}) => {
  const page = await open(browser);
  const add = await inspect(page, "acme/unsure-pyannote");
  await add.getByRole("button", { name: "Fingerprint", exact: true }).click();
  await add
    .getByRole("button", { name: "Add to the catalogue", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.getByRole("tab", { name: /Catalogue/ }).click();
  const card = page.getByRole("article", {
    name: "acme/unsure-pyannote",
    exact: true,
  });
  await card.getByRole("button", { name: "Details", exact: true }).click();
  await page
    .getByRole("button", { name: "Change fingerprint model", exact: true })
    .click();
  const transition = page.getByRole("dialog", {
    name: "Change fingerprint model",
    exact: true,
  });
  await expect(transition).toContainText("References to recalculate");
  await expect(transition).toContainText(
    "0 Worker(s) already have the target ready",
  );
  const start = transition.getByRole("button", {
    name: "Start transition",
    exact: true,
  });
  await expect(start).toBeDisabled();
  await transition
    .getByLabel("I confirm recalculating the 0 displayed references.")
    .check();
  await start.click();
  await expect(transition).toContainText(
    "Preparing Workers. The old model remains active.",
  );
  await expect(
    transition.getByRole("button", { name: "Prepare on Workers", exact: true }),
  ).toBeVisible();
  await transition
    .getByRole("button", { name: "Cancel transition", exact: true })
    .click();
  await page
    .getByRole("alertdialog")
    .getByRole("button", { name: "Cancel transition", exact: true })
    .click();
  await expect(transition).toContainText("Transition cancelled.");
});
