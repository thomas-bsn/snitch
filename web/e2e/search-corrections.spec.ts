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

  await page.goto("/search");
  const query = page.getByRole("searchbox", {
    name: "Search the transcripts",
  });

  // The replaced text is gone, even in exact search.
  await query.fill("joue");
  await query.press("Enter");
  await expect(
    page.getByRole("heading", { name: "Nothing found" }),
  ).toBeVisible();
  await expect(
    page.getByRole("list", { name: /Passages found in/ }),
  ).toHaveCount(0);

  // Exact search takes a phrase: its words follow each other, punctuation is ignored.
  await query.fill("tres bien");
  await query.press("Enter");
  const results = page.getByRole("list", {
    name: "Passages found in synthetic-voices.mp4",
  });
  await expect(results).toContainText("Très bien dit");
  await expect(results).toContainText("Corrected");
  // The segment start is unchanged: the result still opens at its timestamp.
  await expect(
    results.getByRole("link", {
      name: "Open synthetic-voices.mp4 at 0:01",
    }),
  ).toBeVisible();
});
