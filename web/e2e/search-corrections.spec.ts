import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("la recherche trouve le texte corrigé d’un segment et plus l’ancien", async ({
  page,
}) => {
  await prepareDiarizedClip(page);

  // The segment "Bien joué !" is corrected through the API, like the reading page does.
  const old = await page.evaluate(async () => {
    const me = await fetch("/api/auth/me").then((response) => response.json());
    const clips = await fetch("/api/clips").then((response) => response.json());
    const clipID = clips.items[0].id;
    const corrected = await fetch(`/api/clips/${clipID}/segments/0/text`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": me.csrf_token,
        Origin: location.origin,
      },
      body: JSON.stringify({ text: "Très bien dit", expected_version: 0 }),
    });
    if (!corrected.ok) throw new Error(`correction ${corrected.status}`);
    return fetch("/api/search?q=joue&mode=exact").then((response) =>
      response.json(),
    );
  });
  expect(old).toMatchObject({ total: 0, items: [] });

  const search = page.getByRole("region", {
    name: "Recherche dans les paroles",
  });
  await page.getByLabel("Rechercher dans les paroles").fill("joue");
  await page.getByLabel("Type de recherche").selectOption("full_text");
  await page.getByRole("button", { name: "Rechercher", exact: true }).click();
  await expect(search.getByText("Très bien dit")).toHaveCount(0);
  await expect(search.getByText("Corrigé", { exact: true })).toHaveCount(0);

  await page.getByLabel("Rechercher dans les paroles").fill("tres bien");
  await page.getByRole("button", { name: "Rechercher", exact: true }).click();
  await expect(search.getByText("1 résultat · page 1")).toBeVisible();
  await expect(search.getByText("Très bien dit")).toBeVisible();
  await expect(search.getByText("Corrigé", { exact: true })).toBeVisible();
  // The segment start is unchanged: the result still opens at its timestamp.
  await expect(
    search.getByRole("button", { name: "Ouvrir à 00:01" }),
  ).toBeVisible();
});
