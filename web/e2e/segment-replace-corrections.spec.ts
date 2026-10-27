import { expect, test } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

test("retranscrire un clip corrigé exige une confirmation", async ({
  page,
}) => {
  await prepareDiarizedClip(page);
  const transcript = page.getByRole("list", { name: "Transcript segments" });
  const firstSegment = transcript.getByRole("listitem").first();
  await firstSegment.getByRole("button", { name: "Edit text" }).click();
  await firstSegment.getByLabel("Segment text").fill("Bien joué, Alice !");
  await firstSegment.getByRole("button", { name: "Save text" }).click();
  await expect(firstSegment.getByText("Bien joué, Alice !")).toBeVisible();

  // Retranscribing warns about what would be replaced; cancelling changes nothing.
  await page.getByRole("button", { name: "Transcribe again" }).click();
  const warning = page.getByRole("alertdialog");
  await expect(warning).toContainText("1 text and 0 speaker correction(s)");
  await expect(warning).toContainText("administrator");
  await expect(warning).toContainText("obsolete");
  await warning.getByRole("button", { name: "Cancel" }).click();
  await expect(warning).toHaveCount(0);
  await expect(firstSegment).toContainText("Bien joué, Alice !");

  // Explicit confirmation schedules the work; current text remains until it succeeds.
  await page.getByRole("button", { name: "Transcribe again" }).click();
  await warning
    .getByRole("button", { name: "Confirm retranscription" })
    .click();
  await expect(page.getByText("Transcription scheduled.")).toBeVisible();
  await expect(warning).toHaveCount(0);
  await expect(firstSegment).toContainText("Bien joué, Alice !");
});
