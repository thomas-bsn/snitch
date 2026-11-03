import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
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

/** The header pill of the analysis, whatever its state and the language. */
// Hidden from the accessibility tree while the panel is open, hence includeHidden.
const pill = (page: Page) =>
  page.getByRole("button", {
    name: /^(Clip analysis|Analyse du clip)/,
    includeHidden: true,
  });

/** Opens the analysis panel from its pill and returns the panel's region. */
async function openPanel(page: Page) {
  await pill(page).click();
  return page.getByRole("region", {
    name: /^(Clip analysis|Analyse du clip)$/,
  });
}

async function closePanel(page: Page) {
  await page
    .getByRole("dialog", { name: /^(Clip analysis|Analyse du clip)$/ })
    .getByRole("button", { name: /^(Close|Fermer)$/ })
    .click();
}

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
  await expect(pill(page)).toContainText("Never analysed");

  let analysis = await openPanel(page);
  await analysis.getByRole("button", { name: "Launch analysis…" }).click();
  let dialog = page.getByRole("dialog", { name: "Launch analysis" });
  // Voice detection never transcribes behind the scenes: it explains what is missing.
  await dialog.getByLabel("Voices only").check();
  await dialog.getByRole("button", { name: "Start reanalysis" }).click();
  await expect(dialog.getByRole("alert")).toContainText("transcribed first");
  await dialog.getByLabel("Transcription only").check();
  await dialog.getByRole("button", { name: "Start reanalysis" }).click();
  await expect(analysis).toContainText("Choose the audio track to transcribe.");
  await analysis.getByLabel("Audio track").selectOption({ index: 1 });
  await analysis.getByRole("button", { name: "Confirm track" }).click();
  await expect(analysis).toContainText(
    "Waiting for a compatible Worker: none is connected.",
  );
  // The pill follows the run without a reload; no second run can start meanwhile.
  await expect(pill(page)).toContainText("Waiting: Waiting for a compatible");
  await expect(
    analysis.getByRole("button", { name: "Launch analysis…" }),
  ).toHaveCount(0);
  await expect(
    analysis.getByRole("button", { name: "Delete…" }),
  ).toBeDisabled();

  // Cancelling frees the clip; a new analysis is a new run that waits again.
  await analysis.getByRole("button", { name: "Cancel the rest" }).click();
  await expect(analysis).toContainText("Analysis cancelled.");
  await expect(pill(page)).toContainText("Analysis cancelled");
  await analysis.getByRole("button", { name: "Launch analysis…" }).click();
  dialog = page.getByRole("dialog", { name: "Launch analysis" });
  await dialog.getByLabel("Transcription only").check();
  await dialog.getByRole("button", { name: "Start reanalysis" }).click();
  await expect(analysis).toContainText("Waiting for a compatible Worker");
  await expect(
    analysis.getByRole("link", { name: /View task #/ }),
  ).toBeVisible();

  // The state survives closing the page.
  await page.reload();
  await expect(pill(page)).toContainText("Waiting: Waiting for a compatible");
  analysis = await openPanel(page);
  await expect(
    analysis.getByText("Waiting for a compatible Worker: none is connected."),
  ).toBeVisible();
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);
  await closePanel(page);

  // A second tab follows the same analysis, live.
  const other = await context.newPage();
  await other.goto(`/clips/${clipId}`);
  const followed = await openPanel(other);
  await expect(followed.getByRole("status")).toHaveText("Live updates on");

  await completeQueuedTranscription();
  await expect(followed).toContainText("Done", { timeout: 15000 });
  await expect(pill(other)).toContainText("Analysis complete", {
    timeout: 15000,
  });
  await expect(page.getByText("Bonjour").first()).toBeVisible({
    timeout: 15000,
  });
  await expect(pill(page)).toContainText("Analysis complete");
  // With a transcription in place, the voice detection starts and waits for its Worker.
  const again = await openPanel(page);
  await again.getByRole("button", { name: "Launch analysis…" }).click();
  dialog = page.getByRole("dialog", { name: "Launch analysis" });
  await dialog.getByLabel("Voices only").check();
  await dialog.getByLabel("Only the missing steps").check();
  await dialog.getByRole("button", { name: "Start reanalysis" }).click();
  await expect(again).toContainText("Waiting for a compatible Worker");
  await again.getByRole("button", { name: "Cancel the rest" }).click();
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
  const analysis = await openPanel(page);
  await analysis.getByRole("button", { name: "Launch analysis…" }).click();
  const dialog = page.getByRole("dialog", { name: "Launch analysis" });
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
    analysis.getByRole("button", { name: "Cancel the rest" }),
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
    analysis.getByRole("button", { name: "Cancel the rest" }),
  ).toBeVisible();
  // The current text stays visible until the new transcription succeeds.
  await expect(page.getByText("Bien joué, Alice !").first()).toBeVisible();
  await analysis.getByRole("button", { name: "Cancel the rest" }).click();
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
  let analysis = await openPanel(page);
  await analysis.getByRole("button", { name: "Launch analysis…" }).click();
  const dialog = page.getByRole("dialog", { name: "Launch analysis" });
  await dialog.getByLabel("Transcription only").check();
  await dialog.getByRole("button", { name: "Start reanalysis" }).click();
  await analysis.getByLabel("Audio track").selectOption({ index: 1 });
  await analysis.getByRole("button", { name: "Confirm track" }).click();
  await failQueuedTranscription("invalid_input");
  // The pill and the panel change without a reload.
  await expect(pill(page)).toContainText("Failed: Transcription", {
    timeout: 15000,
  });

  const details = analysis.getByRole("group", { name: "Failure details" });
  await expect(details).toContainText("Transcription", { timeout: 15000 });
  await expect(details).toContainText(
    "The clip's media could not be processed.",
  );
  await expect(details).toContainText("Check the clip's media file.");
  await expect(details.locator("time")).toHaveAttribute("datetime", /^20/);
  await expect(details).not.toContainText("invalid_input");
  // The last log lines of the failed task are there for an Administrator (none sent here).
  await expect(
    analysis.getByRole("region", { name: "Last 50 log lines" }),
  ).toContainText("No log line for this task yet.");
  const retry = analysis.getByRole("button", { name: /Retry/ });
  expect((await retry.boundingBox())!.height).toBeGreaterThanOrEqual(44);
  expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);

  // The same details, in French, from the same persisted data.
  await setAccountLanguage(ADMIN, "fr");
  await page.reload();
  analysis = await openPanel(page);
  const detailsFr = analysis.getByRole("group", {
    name: "Détails de l'échec",
  });
  await expect(detailsFr).toContainText(
    "Le média du clip n'a pas pu être traité.",
  );
  await expect(detailsFr).toContainText("Vérifier le fichier média du clip.");
  await setAccountLanguage(ADMIN, "en");

  // After a retry that fails again, the details describe the current attempt: the
  // failure time moves on instead of keeping the earlier attempt's.
  await page.reload();
  analysis = await openPanel(page);
  const firstFailure = await analysis
    .getByRole("group", { name: "Failure details" })
    .locator("time")
    .getAttribute("datetime");
  await analysis.getByRole("button", { name: /Retry/ }).click();
  await expect(
    analysis.getByRole("group", { name: "Failure details" }),
  ).toHaveCount(0);
  await failQueuedTranscription("invalid_input");
  await expect(
    analysis.getByRole("group", { name: "Failure details" }).locator("time"),
  ).not.toHaveAttribute("datetime", firstFailure!, { timeout: 15000 });
  await expect(
    analysis.getByRole("group", { name: "Failure details" }),
  ).toContainText("The clip's media could not be processed.");

  // An ordinary user sees the state, and nothing technical.
  const member = await browser.newContext({
    storageState: await sessionFor(MEMBER),
  });
  const view = await member.newPage();
  await view.goto(`/clips/${clipId}`);
  await expect(pill(view)).toContainText("Failed");
  const region = await openPanel(view);
  await expect(region).toContainText("Failed");
  await expect(region.getByRole("group")).toHaveCount(0);
  await expect(
    region.getByRole("region", { name: "Last 50 log lines" }),
  ).toHaveCount(0);
  await expect(region).not.toContainText("media could not be processed");
  await expect(region).not.toContainText("invalid_input");
});

test("le Super-administrateur voit les pertes avant de supprimer une étape", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-deletion.mp4", "shared");
  await transcribeAndDiarize(clipId);
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto(`/clips/${clipId}`);
  const analysis = await openPanel(page);
  await analysis.getByRole("button", { name: "Delete…" }).click();
  const dialog = page.getByRole("dialog", {
    name: "Delete part of the analysis",
  });
  await expect(
    dialog.getByRole("button", { name: "Delete the selection" }),
  ).toBeDisabled();
  await dialog.getByLabel("Transcription", { exact: true }).check();
  // The losses come from the server before anything is deleted.
  await expect(dialog).toContainText("transcript segments");
  await expect(dialog).toContainText("Nothing is deleted");
  await dialog.getByRole("button", { name: "Delete the selection" }).click();
  await expect(dialog).toHaveCount(0);
  await page.reload();
  await expect(page.getByText("Bonjour")).toHaveCount(0);
  await context.close();
});
