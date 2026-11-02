import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { completeQueuedTranscription, uploadClip } from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

test("analyser un clip jusqu'à sa transcription, durablement", async ({
  browser,
}) => {
  await installAdministrator();
  const clipId = await uploadClip("fixture-analysis.mp4", "shared");
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto(`/clips/${clipId}`);

  const analysis = page.getByRole("region", { name: "Clip analysis" });
  // Voice detection never transcribes behind the scenes: it explains what is missing.
  await analysis.getByRole("button", { name: "More analysis options" }).click();
  await page.getByRole("menuitem", { name: "Detect voices" }).click();
  await expect(page.getByRole("alert")).toContainText("transcribed first");
  await analysis.getByRole("button", { name: "More analysis options" }).click();
  await page.getByRole("menuitem", { name: "Transcribe only" }).click();
  await expect(analysis).toContainText("Choose the audio track to transcribe.");
  await analysis.getByLabel("Audio track").selectOption({ index: 1 });
  await analysis.getByRole("button", { name: "Confirm track" }).click();
  await expect(analysis).toContainText(
    "Waiting for a compatible Worker: none is connected.",
  );
  // No second run can start while this one is active.
  await expect(
    analysis.getByRole("button", { name: /Analyze all/ }),
  ).toHaveCount(0);

  // Cancelling frees the clip; a new analysis is a new run that waits again.
  await analysis.getByRole("button", { name: "Cancel analysis" }).click();
  await expect(analysis).toContainText("Analysis cancelled.");
  await analysis.getByRole("button", { name: "More analysis options" }).click();
  await page.getByRole("menuitem", { name: "Transcribe only" }).click();
  await expect(analysis).toContainText("Waiting for a compatible Worker");
  await expect(
    analysis.getByRole("link", { name: /View task #/ }),
  ).toBeVisible();

  // The state survives closing the page.
  await page.reload();
  await expect(
    page
      .getByRole("region", { name: "Clip analysis" })
      .getByText("Waiting for a compatible Worker: none is connected."),
  ).toBeVisible();
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);

  // A second tab follows the same analysis, live.
  const other = await context.newPage();
  await other.goto(`/clips/${clipId}`);
  await expect(
    other.getByRole("region", { name: "Clip analysis" }).getByRole("status"),
  ).toHaveText("Live updates on");

  await completeQueuedTranscription();
  await expect(
    other.getByRole("region", { name: "Clip analysis" }),
  ).toContainText("Done", { timeout: 15000 });
  await expect(page.getByText("Bonjour").first()).toBeVisible({
    timeout: 15000,
  });
  await expect(
    page.getByRole("region", { name: "Clip analysis" }),
  ).toContainText("Done");
  // With a transcription in place, the voice detection starts and waits for its Worker.
  const again = page.getByRole("region", { name: "Clip analysis" });
  await again.getByRole("button", { name: "More analysis options" }).click();
  await page.getByRole("menuitem", { name: "Detect voices" }).click();
  await expect(again).toContainText("Waiting for a compatible Worker");
  await again.getByRole("button", { name: "Cancel analysis" }).click();
  await expect(again).toContainText("Analysis cancelled.");
  await context.close();
});
