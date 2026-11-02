import { expect, test } from "@playwright/test";
import {
  correctSpeaker,
  nameVoice,
  transcribeAndDiarize,
  uploadClip,
} from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

test("relancer la diarisation d'un clip validé exige une confirmation", async ({
  page,
}) => {
  await installAdministrator();
  const clipId = await uploadClip("synthetic-voices.mp4", "shared");
  const [first] = await transcribeAndDiarize(clipId);
  await nameVoice(first, "Alice");
  await correctSpeaker(clipId, 1, first);
  await page.context().addCookies((await sessionFor(ADMIN)).cookies);
  await page.goto(`/clips/${clipId}`);
  await page.getByRole("tab", { name: "Voices" }).click();
  await expect(page.getByText("Alice").first()).toBeVisible();

  // The warning says what is at stake; cancelling schedules nothing.
  await page.getByRole("button", { name: "Relaunch diarization" }).click();
  const warning = page.getByRole("alertdialog");
  await expect(warning).toContainText("1 speaker correction(s)");
  await expect(warning).toContainText("1 validated voice(s)");
  await expect(warning).toContainText("1 validated voice sample(s)");
  await warning.getByRole("button", { name: "Cancel" }).click();
  await expect(warning).toHaveCount(0);
  await expect(page.getByText("Diarization relaunched.")).toHaveCount(0);

  // Explicit confirmation schedules the work; the current voices stay visible.
  await page.getByRole("button", { name: "Relaunch diarization" }).click();
  await warning.getByRole("button", { name: "Confirm relaunch" }).click();
  await expect(page.getByText("Diarization relaunched.")).toBeVisible();
  await expect(warning).toHaveCount(0);
  await expect(page.getByText("Alice").first()).toBeVisible();
});
