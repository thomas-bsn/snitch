import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { nameVoice, transcribeAndDiarize, uploadClip } from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// The instance is installed once for the file: every test shares it.
test.beforeAll(async () => {
  await installAdministrator();
});

async function prepare(page: Page, name: string, peaks: boolean) {
  const clipId = await uploadClip(name, "shared");
  await transcribeAndDiarize(clipId, { peaks });
  await page.context().addCookies((await sessionFor(ADMIN)).cookies);
  await page.goto(`/clips/${clipId}`);
  await expect(page.getByRole("tab", { name: "Voices" })).toBeVisible();
  return clipId;
}

async function speakerIds(page: Page, clipId: number) {
  const voice = await (
    await page.request.get(`/api/clips/${clipId}/voice`)
  ).json();
  return voice.speakers.map((speaker: { id: number }) => speaker.id);
}

/** The decision card of one voice, listed in the Voices tab. */
function card(page: Page, label: string) {
  return page
    .getByRole("list", { name: "Voices of the clip" })
    .getByRole("listitem")
    .filter({ hasText: label });
}

function firstSegment(page: Page) {
  return page
    .getByRole("list", { name: "Transcript segments" })
    .getByRole("listitem")
    .first();
}

test("une piste par voix, colorée, cliquable pour se placer dans l'audio", async ({
  page,
}) => {
  const clipId = await prepare(page, "fixture-voices-tracks.mp4", true);
  await page.getByRole("tab", { name: "Voices" }).click();

  const tracks = page.getByRole("button", { name: /^Track of SPEAKER_0/ });
  await expect(tracks).toHaveCount(2);
  await expect(tracks.first().locator("path")).toHaveAttribute("d", /M/);

  // Keyboard: Enter places the reader at the first turn of the voice (2.2 s).
  await tracks.nth(1).focus();
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(new RegExp(`/clips/${clipId}\\?t=2(&|$)`));

  // Pointer: a click anywhere on a track moves the reader there.
  await tracks.first().click();
  await expect(page).toHaveURL(/[?&]t=[1-9]/);
});

test("une piste sans pics est affichée sans tracé", async ({ page }) => {
  await prepare(page, "fixture-voices-no-peaks.mp4", false);
  await page.getByRole("tab", { name: "Voices" }).click();

  const tracks = page.getByRole("button", { name: /^Track of SPEAKER_0/ });
  await expect(tracks).toHaveCount(2);
  await expect(tracks.first().locator("path")).toHaveCount(0);
});

test("Random et Ignorer : libellé ou absence de locuteur dans la transcription, puis Rétablir", async ({
  page,
}) => {
  await prepare(page, "fixture-voices-random.mp4", false);
  await page.getByRole("tab", { name: "Voices" }).click();

  await card(page, "SPEAKER_00")
    .getByRole("button", { name: "Random", exact: true })
    .click();
  await expect(card(page, "SPEAKER_00")).toContainText("Random");

  await page.getByRole("tab", { name: "Transcript" }).click();
  await expect(firstSegment(page)).toContainText("Random");

  await page.getByRole("tab", { name: "Voices" }).click();
  await card(page, "SPEAKER_00")
    .getByRole("button", { name: "Restore", exact: true })
    .click();
  await page.getByRole("tab", { name: "Transcript" }).click();
  await expect(firstSegment(page)).toContainText("SPEAKER_00");

  await page.getByRole("tab", { name: "Voices" }).click();
  await card(page, "SPEAKER_00")
    .getByRole("button", { name: "Ignore", exact: true })
    .click();
  await page.getByRole("tab", { name: "Transcript" }).click();
  await expect(firstSegment(page)).toContainText("Bien joué");
  await expect(firstSegment(page)).not.toContainText("SPEAKER_00");
  await expect(firstSegment(page)).not.toContainText("Random");
});

test("Ce n'est pas X retire l'attribution de la voix, Annuler le refus la rend proposable", async ({
  page,
}) => {
  const clipId = await prepare(page, "fixture-voices-refusal.mp4", false);
  const [first] = await speakerIds(page, clipId);
  await nameVoice(first, "Camille");
  await page.reload();
  await page.getByRole("tab", { name: "Voices" }).click();

  await card(page, "SPEAKER_00")
    .getByRole("button", { name: "It is not Camille" })
    .click();
  await expect(
    card(page, "SPEAKER_00").getByRole("button", { name: "It is not Camille" }),
  ).toHaveCount(0);
  const refused = card(page, "SPEAKER_00").getByRole("list", {
    name: "People refused for SPEAKER_00",
  });
  await expect(refused).toContainText("Not Camille");

  await refused
    .getByRole("button", { name: "Cancel the refusal of Camille" })
    .click();
  await expect(refused).toHaveCount(0);
  const voice = await (
    await page.request.get(`/api/clips/${clipId}/voice`)
  ).json();
  expect(voice.speakers[0].refused_persons).toEqual([]);
});

test("la fiche des voix passe l'analyse d'accessibilité", async ({ page }) => {
  await prepare(page, "fixture-voices-axe.mp4", true);
  await page.getByRole("tab", { name: "Voices" }).click();
  await expect(card(page, "SPEAKER_00")).toBeVisible();
  const results = await new AxeBuilder({ page }).analyze();
  expect(results.violations).toEqual([]);
});

test("un segment à deux voix montre ses deux parts, et la correction annonce son effet", async ({
  page,
}) => {
  const clipId = await uploadClip("fixture-voices-parts.mp4", "shared");
  await transcribeAndDiarize(clipId, { peaks: true, sharedSegment: true });
  await page.context().addCookies((await sessionFor(ADMIN)).cookies);
  await page.goto(`/clips/${clipId}`);

  const segment = firstSegment(page);
  const parts = segment
    .getByRole("list", { name: "Parts of the segment" })
    .getByRole("listitem");
  await expect(parts).toHaveCount(2);
  await expect(parts.first()).toContainText("SPEAKER_00");
  await expect(parts.first()).toContainText("Bien joué !");
  await expect(parts.nth(1)).toContainText("SPEAKER_01");
  await expect(parts.nth(1)).toContainText("Salut");
  // A single-voice segment keeps its one chip and its text, without a list of parts.
  const single = page
    .getByRole("list", { name: "Transcript segments" })
    .getByRole("listitem")
    .filter({ hasText: "Merci" });
  await expect(single.getByRole("list")).toHaveCount(0);
  await expect(single).toContainText("SPEAKER_01");

  // The waveform follows the parts: the second voice speaks from 2.25 s.
  await page.getByRole("tab", { name: "Voices" }).click();
  await expect(
    page.getByRole("button", { name: "Track of SPEAKER_01" }).locator("path"),
  ).toHaveAttribute("d", /^M2\.25 0/);
  await page.getByRole("tab", { name: "Transcript" }).click();

  await segment.getByRole("button", { name: "Edit text" }).click();
  await expect(segment).toContainText(
    "This segment is read as 2 parts: a corrected text keeps a single part",
  );
  await segment.getByRole("button", { name: "Cancel" }).click();

  await segment.getByRole("button", { name: "Edit speaker" }).click();
  await expect(segment).toContainText(
    "This segment is read as 2 parts: the correction applies to the whole segment",
  );
  expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
  await segment.getByLabel("Speaker").selectOption({ label: "SPEAKER_01" });
  await segment.getByRole("button", { name: "Save speaker" }).click();
  await expect(segment).toContainText("Speaker corrected by administrator");
  await expect(segment.getByRole("list")).toHaveCount(0);
  await expect(segment).toContainText("Bien joué ! Salut");
});
