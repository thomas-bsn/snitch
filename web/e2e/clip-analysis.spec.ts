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
  await analysis.getByRole("button", { name: "Analyze", exact: true }).click();
  await expect(analysis).toContainText("Choose the audio track to transcribe.");
  await analysis.getByLabel("Audio track").selectOption({ index: 1 });
  await analysis.getByRole("button", { name: "Confirm track" }).click();
  await expect(analysis).toContainText("Waiting for a compatible Worker.");
  // No second run can start while this one is active.
  await expect(analysis.getByRole("button", { name: /Analyze/ })).toHaveCount(
    0,
  );

  // The state survives closing the page.
  await page.reload();
  await expect(
    page
      .getByRole("region", { name: "Clip analysis" })
      .getByText("Waiting for a compatible Worker."),
  ).toBeVisible();
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);

  await completeQueuedTranscription();
  await expect(page.getByText("Bonjour").first()).toBeVisible({
    timeout: 15000,
  });
  await expect(page.getByRole("region", { name: "Clip analysis" })).toHaveCount(
    0,
  );
  await context.close();
});
