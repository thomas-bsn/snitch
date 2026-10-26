import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("un utilisateur écoute un extrait, crée une Personne, valide puis corrige un Locuteur local", async ({
  page,
  request,
}) => {
  await prepareDiarizedClip(page);
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const voices = page.getByRole("region", { name: "Voix du clip" });
  const speakers = voices.getByRole("list", { name: "Locuteurs locaux" });
  const first = speakers.locator("xpath=./li").nth(0);
  const second = speakers.locator("xpath=./li").nth(1);

  // Unknown voices are shown as such, with their listenable extract.
  await expect(first).toContainText("SPEAKER_00 · non identifié");
  await expect(first).toContainText("Inconnu");
  const extract = first.getByLabel("Extrait de SPEAKER_00");
  const source = await extract.getAttribute("src");
  expect(source).toMatch(/^\/api\/local-speakers\/\d+\/excerpt$/);
  const audio = await page.request.get(source!);
  expect(audio.ok()).toBe(true);
  expect(audio.headers()["content-type"]).toBe("audio/wav");
  // Extracts are never served without a session.
  expect((await request.get(source!)).status()).toBe(401);

  // First validation creates the Person and a validated sample.
  await first.getByLabel("Nouvelle Personne").fill("Alice Synthétique");
  await first.getByRole("button", { name: "Valider", exact: true }).click();
  await expect(first.getByText("Validation enregistrée.")).toBeVisible();
  await expect(first).toContainText("Alice Synthétique");
  await expect(first).toContainText("1 échantillon validé");
  await expect(first).toContainText("Validé par un humain");
  await expect(first).toContainText("Validé par administrator le");
  await expect(second).toContainText("Inconnu");
  await expect(second).not.toContainText("Validé par un humain");

  // The other voice is validated elsewhere; this stale screen sees who won.
  const secondId = (await second
    .getByLabel("Extrait de SPEAKER_01")
    .getAttribute("src"))!.match(/(\d+)\/excerpt/)![1];
  const people = await page.request.get("/api/persons");
  const alice = (await people.json()).items[0];
  const me = await (await page.request.get("/api/auth/me")).json();
  const raced = await page.request.post(
    `/api/local-speakers/${secondId}/validation`,
    {
      data: { person_id: alice.id },
      headers: {
        "X-CSRF-Token": me.csrf_token,
        Origin: new URL(page.url()).origin,
      },
    },
  );
  expect(raced.status()).toBe(201);
  await second.getByLabel("Nouvelle Personne").fill("Perdante");
  await second.getByRole("button", { name: "Valider", exact: true }).click();
  await expect(
    second.getByText(
      "Déjà validé par administrator : Alice Synthétique. La première validation fait foi.",
    ),
  ).toBeVisible();
  await expect(second).toContainText("Validé par un humain");

  // Explicit correction keeps the history.
  await first.getByRole("button", { name: "Corriger" }).click();
  await first.getByLabel("Nouvelle Personne").fill("Bob Synthétique");
  await first
    .getByRole("button", { name: "Enregistrer la correction" })
    .click();
  await expect(first.getByText("Correction enregistrée.")).toBeVisible();
  await expect(first).toContainText("Bob Synthétique");
  await expect(first).toContainText("révision 2");
  await first.getByRole("button", { name: "Historique" }).click();
  const history = first.getByRole("list", { name: "Historique SPEAKER_00" });
  await expect(history).toContainText(
    "Correction : Alice Synthétique → Bob Synthétique par administrator",
  );
  await expect(history).toContainText(
    "Validation : Inconnu → Alice Synthétique par administrator",
  );

  // Sample counts follow the correction: Alice keeps the other voice only.
  await expect(first).toContainText("Bob Synthétique (1 échantillon validé)");
  await expect(second).toContainText(
    "Alice Synthétique (1 échantillon validé)",
  );

  // The transcript names the Person of each segment.
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const transcript = page.getByRole("region", {
    name: "Transcription horodatée",
  });
  await expect(
    transcript.getByText("SPEAKER_00 · Bob Synthétique"),
  ).toBeVisible();
  await expect(
    transcript.getByText("SPEAKER_01 · Alice Synthétique"),
  ).toBeVisible();
});
