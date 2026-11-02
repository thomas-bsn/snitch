import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import {
  completeQueuedTranscription,
  correctText,
  failQueuedTranscription,
  nameVoice,
  transcribeAndDiarize,
  uploadClip,
} from "./library-fixture";
import {
  ADMIN,
  MEMBER,
  createApprovedMember,
  installAdministrator,
  sessionFor,
  setAccountLanguage,
} from "./shell-fixture";

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

test("réanalyser un clip traité en confirmant les conséquences", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-reanalysis.mp4", "shared");
  const [first] = await transcribeAndDiarize(clipId);
  await correctText(clipId, 0, "Bien joué, Alice !", "synthetic");
  await nameVoice(first, "Alice");
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto(`/clips/${clipId}`);
  const analysis = page.getByRole("region", { name: "Clip analysis" });
  await analysis.getByRole("button", { name: "More analysis options" }).click();
  await page.getByRole("menuitem", { name: "Reanalyze…" }).click();
  const dialog = page.getByRole("dialog", { name: "Reanalyze this clip" });
  await expect(dialog).toContainText("no confirmation is needed");
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);

  // Choosing steps requires at least one of them.
  await dialog.getByLabel("Only the steps I choose").check();
  await expect(
    dialog.getByRole("button", { name: "Start reanalysis" }),
  ).toBeDisabled();
  await dialog.getByLabel("Voice detection").check();
  await expect(dialog).toContainText("validated voices");

  // The consequences are explained before anything starts; going back changes nothing.
  await dialog.getByRole("button", { name: "Start reanalysis" }).click();
  await expect(dialog).toContainText("will make human work obsolete");
  await expect(dialog).toContainText("1 validated voices");
  await expect(dialog).toContainText("Nothing is deleted");
  await dialog.getByRole("button", { name: "Back" }).click();
  await expect(
    analysis.getByRole("button", { name: "Cancel analysis" }),
  ).toHaveCount(0);

  // Everything again: corrections and validations are both listed; confirming starts the run.
  await dialog.getByLabel("Everything again").check();
  await dialog.getByRole("button", { name: "Start reanalysis" }).click();
  await expect(dialog).toContainText("1 corrected segment texts");
  await expect(dialog).toContainText("1 validated voices");
  await dialog.getByRole("button", { name: "Confirm and reanalyze" }).click();
  await expect(dialog).toHaveCount(0);
  await expect(analysis).toContainText("(run again)");
  await expect(
    analysis.getByRole("button", { name: "Cancel analysis" }),
  ).toBeVisible();
  // The current text stays visible until the new transcription succeeds.
  await expect(page.getByText("Bien joué, Alice !").first()).toBeVisible();
  await analysis.getByRole("button", { name: "Cancel analysis" }).click();
  await context.close();
});

test("l'administrateur lit la cause d'échec d'une étape, pas l'utilisateur ordinaire", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-failure.mp4", "shared");
  await createApprovedMember();
  const admin = await browser.newContext({
    storageState: await sessionFor(ADMIN),
    viewport: { width: 390, height: 800 },
  });
  const page = await admin.newPage();
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto(`/clips/${clipId}`);
  const analysis = page.getByRole("region", { name: "Clip analysis" });
  await analysis.getByRole("button", { name: "More analysis options" }).click();
  await page.getByRole("menuitem", { name: "Transcribe only" }).click();
  await analysis.getByLabel("Audio track").selectOption({ index: 1 });
  await analysis.getByRole("button", { name: "Confirm track" }).click();
  await failQueuedTranscription("invalid_input");

  const details = analysis.getByRole("group", { name: "Failure details" });
  await expect(details).toContainText("Transcription", { timeout: 15000 });
  await expect(details).toContainText(
    "The clip's media could not be processed.",
  );
  await expect(details).toContainText("Check the clip's media file.");
  await expect(details.locator("time")).toHaveAttribute("datetime", /^20/);
  await expect(details).not.toContainText("invalid_input");
  const retry = analysis.getByRole("button", { name: /Retry/ });
  expect((await retry.boundingBox())!.height).toBeGreaterThanOrEqual(44);
  expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);

  // The same details, in French, from the same persisted data.
  await setAccountLanguage(ADMIN, "fr");
  await page.reload();
  const detailsFr = page
    .getByRole("region", { name: "Analyse du clip" })
    .getByRole("group", { name: "Détails de l'échec" });
  await expect(detailsFr).toContainText(
    "Le média du clip n'a pas pu être traité.",
  );
  await expect(detailsFr).toContainText("Vérifier le fichier média du clip.");
  await setAccountLanguage(ADMIN, "en");

  // After a retry that fails again, the details describe the current attempt: the
  // failure time moves on instead of keeping the earlier attempt's.
  await page.reload();
  const firstFailure = await details.locator("time").getAttribute("datetime");
  await analysis.getByRole("button", { name: /Retry/ }).click();
  await expect(details).toHaveCount(0);
  await failQueuedTranscription("invalid_input");
  await expect(details.locator("time")).not.toHaveAttribute(
    "datetime",
    firstFailure!,
    { timeout: 15000 },
  );
  await expect(details).toContainText(
    "The clip's media could not be processed.",
  );

  // An ordinary user sees the state, and nothing technical.
  const member = await browser.newContext({
    storageState: await sessionFor(MEMBER),
  });
  const view = await member.newPage();
  await view.goto(`/clips/${clipId}`);
  const region = view.getByRole("region", { name: "Clip analysis" });
  await expect(region).toContainText("Failed");
  await expect(region.getByRole("group")).toHaveCount(0);
  await expect(region).not.toContainText("media could not be processed");
  await expect(region).not.toContainText("invalid_input");
});
