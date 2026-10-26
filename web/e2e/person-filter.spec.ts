import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un utilisateur filtre la recherche et la bibliothèque par Personne et voit l’état inconnu", async ({
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

  // A Person without any voice exists too.
  const me = await (await page.request.get("/api/auth/me")).json();
  const created = await page.request.post("/api/persons", {
    data: { display_name: "Carol Synthétique" },
    headers: {
      "X-CSRF-Token": me.csrf_token,
      Origin: new URL(page.url()).origin,
    },
  });
  expect(created.status()).toBe(201);
  await page.reload();

  const search = page.getByRole("region", {
    name: "Recherche dans les paroles",
  });
  const results = search.locator("ol.search-results");
  const searchFor = async (
    text: string,
    person: string,
    mode = "Plein texte",
  ) => {
    await search.getByLabel("Rechercher dans les paroles").fill(text);
    await search.getByLabel("Type de recherche").selectOption({ label: mode });
    await search.getByLabel("Personne de la recherche").selectOption({
      label: person,
    });
    await Promise.all([
      page.waitForResponse((r) => r.url().includes("/api/search?")),
      search.getByRole("button", { name: "Rechercher" }).click(),
    ]);
  };

  // Each result names its Person, or the unknown state, and opens at its timestamp.
  await searchFor("joué", "Toutes");
  await expect(results).toContainText("Personne : Alice Synthétique");
  await expect(results).toContainText("Validé par un humain");
  await expect(
    results.getByRole("button", { name: "Ouvrir à 00:01" }),
  ).toBeVisible();
  await searchFor("merci", "Toutes");
  await expect(results).toContainText("Personne : inconnue");

  // The filter keeps only the chosen Person, in exact and full-text modes.
  await searchFor("merci", "Alice Synthétique");
  await expect(search.getByText("Aucun résultat")).toBeVisible();
  await searchFor("Bien", "Alice Synthétique", "Mot exact");
  await expect(results).toContainText("Personne : Alice Synthétique");
  await searchFor("Merci", "Alice Synthétique", "Mot exact");
  await expect(search.getByText("Aucun résultat")).toBeVisible();
  await searchFor("merci", "Voix inconnues");
  await expect(results).toContainText("Personne : inconnue");
  await searchFor("joué", "Voix inconnues");
  await expect(search.getByText("Aucun résultat")).toBeVisible();
  await searchFor("joué", "Carol Synthétique");
  await expect(search.getByText("Aucun résultat")).toBeVisible();

  // Library: the card lists the identified Persons and the list is filterable.
  const library = page.locator("#library");
  const cards = library.locator("ul.clip-grid");
  await expect(
    cards.getByRole("list", { name: "Personnes identifiées" }).first(),
  ).toContainText("Alice Synthétique");
  const applyLibraryFilters = () =>
    Promise.all([
      page.waitForResponse((r) => r.url().includes("/api/clips?")),
      // Focusing the form reveals the advanced filters and moves the button: submit by keyboard.
      library
        .getByRole("button", { name: "Appliquer les filtres" })
        .press("Enter"),
    ]);
  const filter = library.getByLabel("Personne de la bibliothèque");
  await filter.selectOption({ label: "Carol Synthétique" });
  await applyLibraryFilters();
  await expect(
    library.getByText("Aucun clip visible pour ces filtres."),
  ).toBeVisible();
  await filter.selectOption({ label: "Alice Synthétique" });
  await applyLibraryFilters();
  await expect(cards.getByText("synthetic-voices.mp4")).toBeVisible();
  await filter.selectOption({ label: "Voix inconnues" });
  await applyLibraryFilters();
  await expect(cards.getByText("synthetic-voices.mp4")).toBeVisible();

  // The clip page shows the identified Persons as well.
  await cards.getByText("Détails et lecture", { exact: true }).click();
  await expect(
    cards
      .locator("details")
      .getByRole("list", { name: "Personnes identifiées" }),
  ).toContainText("Alice Synthétique");
});
