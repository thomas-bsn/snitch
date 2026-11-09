import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import {
  adminApi,
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
} from "./shell-fixture";

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];

async function expectNoViolations(page: Page) {
  const { violations } = await new AxeBuilder({ page })
    .withTags(TAGS)
    .analyze();
  expect(violations).toEqual([]);
}

// The Go suite seeds a corpus and two finished runs (one complete and partial, one failed) before this file runs.
test.beforeAll(async () => {
  await installAdministrator();
  await createApprovedMember();
});

async function asAdmin(page: Page) {
  await page.context().addCookies((await sessionFor(ADMIN)).cookies);
}

/** Gives a voice of a clip the Person of another one, through the API (the first one names her). */
async function validateAs(speakerId: number, personId: number) {
  const { api, post } = await adminApi();
  await post(`/api/local-speakers/${speakerId}/validation`, {
    person_id: personId,
  });
  await api.dispose();
}

async function personOf(page: Page, clipId: number, index: number) {
  const voice = await (
    await page.request.get(`/api/clips/${clipId}/voice`)
  ).json();
  return voice.speakers[index].person.id as number;
}

test("seuls les administrateurs accèdent au benchmark", async ({ browser }) => {
  const context = await browser.newContext({
    storageState: await sessionFor(MEMBER),
  });
  const page = await context.newPage();
  await page.goto("/admin/benchmark");
  await expect(
    page.getByRole("heading", { name: "Administrators only" }),
  ).toBeVisible();
  for (const path of [
    "/api/admin/benchmarks/datasets",
    "/api/admin/benchmarks/clips",
    "/api/admin/benchmarks/runs",
    "/api/admin/benchmarks/models?stage=fingerprint",
  ]) {
    expect((await page.request.get(path)).status()).toBe(403);
  }
  await context.close();
});

const STM =
  ";; synthetic reference\nclip 1 alice 0.0 1.0 <o,f0,female> Bonjour tout le monde\nclip 1 bob 1.0 2.0 salut Alice\n";
const RTTM =
  "SPEAKER clip 1 0.0 1.0 <NA> <NA> alice <NA> <NA>\nSPEAKER clip 1 1.0 1.0 <NA> <NA> bob <NA> <NA>\n";

function referenceFile(name: string, text: string) {
  return { name, mimeType: "text/plain", buffer: Buffer.from(text) };
}

test("un administrateur crée un corpus de clips annotés", async ({ page }) => {
  await asAdmin(page);
  // Two reference clips enrol Alice and Bob; two test clips are measured.
  const clips: Record<string, number> = {};
  const voices: Record<string, number[]> = {};
  for (const name of ["ref-a", "ref-b", "test-c", "test-d"]) {
    clips[name] = await uploadClip(`fixture-bench-${name}.mp4`, "shared");
    voices[name] = await transcribeAndDiarize(clips[name]);
  }
  await nameVoice(voices["ref-a"][0], "Alice");
  await nameVoice(voices["ref-a"][1], "Bob");
  const alice = await personOf(page, clips["ref-a"], 0);
  const bob = await personOf(page, clips["ref-a"], 1);
  for (const name of ["ref-b", "test-c", "test-d"]) {
    await validateAs(voices[name][0], alice);
    await validateAs(voices[name][1], bob);
  }

  await page.goto("/admin/benchmark");
  await expect(
    page.getByRole("navigation", { name: "Administration sections" }),
  ).toContainText("Benchmark");
  await expect(
    page.getByRole("link", { name: "Benchmarks", exact: true }),
  ).toBeVisible();
  // The seeded corpus is offered; nothing else is, and a stage that does not run yet is not a button to press.
  await expect(
    page.getByRole("radio", { name: "Corpus de démonstration" }),
  ).toBeVisible();
  // The three other stages run, but this corpus has no certified clip.
  for (const stage of [/Transcription/, /Diarization/, /End to end/]) {
    await expect(page.getByRole("button", { name: stage })).toBeDisabled();
    await expect(page.getByRole("button", { name: stage })).toContainText(
      "cannot measure",
    );
  }

  await page.getByRole("button", { name: "Create a corpus" }).click();
  const dialog = page.getByRole("dialog", {
    name: "Create a corpus of annotated clips",
  });
  const group = (name: string) =>
    dialog.getByRole("group", { name: `Role · fixture-bench-${name}.mp4` });
  for (const name of ["ref-a", "ref-b"])
    await group(name).getByRole("button", { name: "Reference" }).click();
  for (const name of ["test-c", "test-d"])
    await group(name).getByRole("button", { name: "Test" }).click();
  await expect(
    group("ref-a").getByRole("button", { name: "Reference" }),
  ).toHaveAttribute("aria-pressed", "true");

  // Without certification only the voice fingerprint can be measured.
  const stages = dialog.getByRole("list").last();
  await expect(
    stages.getByRole("listitem").filter({ hasText: "Voice fingerprint" }),
  ).toContainText("Measurable");
  await expect(
    stages.getByRole("listitem").filter({ hasText: "Transcription" }),
  ).toContainText("Not measurable");
  await expect(dialog.getByRole("note")).toHaveCount(0);
  // Certifying a test clip opens the other stages and says the reference comes from the production pipeline.
  await dialog
    .getByRole("checkbox", { name: /Fully reviewed/ })
    .first()
    .check();
  await expect(
    stages.getByRole("listitem").filter({ hasText: "End to end" }),
  ).toContainText("Measurable");
  await expect(dialog.getByRole("note")).toContainText(
    "the production model has an advantage",
  );

  // An imported reference replaces the production one: the clip counts as certified and the bias warning goes.
  const testD = "fixture-bench-test-d.mp4";
  const importArea = dialog.getByRole("group", {
    name: new RegExp(`Import a reference · ${testD}`),
  });
  await importArea
    .getByLabel(`STM file (text) · ${testD}`)
    .setInputFiles(referenceFile("reference.stm", STM));
  await importArea
    .getByLabel(`RTTM file (speaker turns) · ${testD}`)
    .setInputFiles(referenceFile("reference.rttm", RTTM));
  await expect(importArea).toContainText("reference.stm");
  await expect(importArea).toContainText("reference.rttm");
  await expect(dialog.getByRole("note")).toHaveCount(0);
  await expect(
    stages.getByRole("listitem").filter({ hasText: "End to end" }),
  ).toContainText("Measurable");
  // Removing the files brings the production reference, and its warning, back.
  await importArea
    .getByRole("button", { name: "Remove reference.stm" })
    .click();
  await importArea
    .getByRole("button", { name: "Remove reference.rttm" })
    .click();
  await expect(dialog.getByRole("note")).toContainText(
    "the production model has an advantage",
  );
  await expectNoViolations(page);

  await dialog.getByLabel("Name").fill("Corpus e2e");
  await dialog.getByRole("button", { name: "Create the corpus" }).click();
  await expect(dialog).toBeHidden();
  const card = page.getByRole("radio", { name: "Corpus e2e" });
  await expect(card).toBeChecked();
  const label = page.locator("label").filter({ has: card });
  await expect(label).toContainText("2 test clips · 2 reference clips");
  await expect(label).toContainText("1 certified as fully reviewed");
  // The unexecutable stages stay disabled; the executable one waits for two models ready on a Worker.
  await expect(
    page.getByRole("button", { name: /Voice fingerprint/ }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(
    page.getByRole("checkbox", { name: /wespeaker/i }),
  ).toBeDisabled();
  await expect(page.getByText("Ready on no Worker")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Start the benchmark" }),
  ).toBeDisabled();
  await expect(page.getByText("Pick at least 2 models.")).toBeVisible();
  await expectNoViolations(page);
});

test("un corpus à référence importée n'annonce pas le biais de production", async ({
  page,
}) => {
  await asAdmin(page);
  await page.goto("/admin/benchmark");
  await page.getByRole("button", { name: "Create a corpus" }).click();
  const dialog = page.getByRole("dialog", {
    name: "Create a corpus of annotated clips",
  });
  const group = (name: string) =>
    dialog.getByRole("group", { name: `Role · fixture-bench-${name}.mp4` });
  for (const name of ["ref-a", "ref-b"])
    await group(name).getByRole("button", { name: "Reference" }).click();
  await group("test-c").getByRole("button", { name: "Test" }).click();
  const importArea = dialog.getByRole("group", {
    name: /Import a reference · fixture-bench-test-c.mp4/,
  });
  await importArea
    .getByLabel("STM file (text) · fixture-bench-test-c.mp4")
    .setInputFiles(referenceFile("reference.stm", STM));
  await importArea
    .getByLabel("RTTM file (speaker turns) · fixture-bench-test-c.mp4")
    .setInputFiles(referenceFile("reference.rttm", RTTM));
  await expect(dialog.getByRole("note")).toHaveCount(0);
  await dialog.getByLabel("Name").fill("Corpus importé UI");
  await dialog.getByRole("button", { name: "Create the corpus" }).click();
  await expect(dialog).toBeHidden();
  const label = page
    .locator("label")
    .filter({ has: page.getByRole("radio", { name: "Corpus importé UI" }) });
  await expect(label).toContainText("Reference: imported (STM/RTTM)");
  await expect(label).not.toContainText("production");
  // The corpus built from the production reference says so.
  await expect(
    page
      .locator("label")
      .filter({ has: page.getByRole("radio", { name: "Corpus e2e" }) }),
  ).toContainText("Reference: production, corrected");
  await expectNoViolations(page);
});

test("la diarisation propose les modèles du catalogue et attend des Workers prêts", async ({
  page,
}) => {
  await asAdmin(page);
  await page.goto("/admin/benchmark");
  await page.getByRole("radio", { name: "Corpus e2e" }).check();
  const diarization = page.getByRole("button", { name: /Diarization/ });
  await expect(diarization).toBeEnabled();
  await diarization.click();
  await expect(diarization).toHaveAttribute("aria-pressed", "true");
  // The catalogue models are offered. Only the community one is ready (on the seeded Worker): a single model cannot start.
  await expect(
    page.getByRole("checkbox", { name: /sortformer/i }),
  ).toBeDisabled();
  await expect(
    page.getByRole("checkbox", { name: /community/i }),
  ).toBeEnabled();
  await expect(page.getByText("Ready on no Worker").first()).toBeVisible();
  await expect(page.getByText("Pick at least 2 models.")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Start the benchmark" }),
  ).toBeDisabled();
  await expectNoViolations(page);
});

test("un benchmark de transcription se lance, se suit et s'annule", async ({
  page,
}) => {
  await asAdmin(page);
  await page.goto("/admin/benchmark");
  // The corpus created by the previous journey has a certified clip, so it can measure the transcription.
  await page.getByRole("radio", { name: "Corpus e2e" }).check();
  const transcription = page.getByRole("button", { name: /Transcription/ });
  await expect(transcription).toBeEnabled();
  await transcription.click();
  await expect(transcription).toHaveAttribute("aria-pressed", "true");
  await expect(
    page.getByRole("button", { name: "Start the benchmark" }),
  ).toBeDisabled();

  // Only a Worker reaches the service: the models are named by the administrator.
  await page.getByLabel("Add a model by name").fill("vendor/model-a");
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await page.getByLabel("Add a model by name").fill("vendor/model-b");
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await expect(
    page.getByRole("checkbox", { name: "vendor/model-a" }),
  ).toBeChecked();
  await expect(
    page.getByRole("checkbox", { name: "vendor/model-b" }),
  ).toBeChecked();
  await expect(
    page.getByText("Checked by a Worker when the task starts."),
  ).toHaveCount(2);
  await expectNoViolations(page);

  await page.getByRole("button", { name: "Start the benchmark" }).click();
  const running = page.getByRole("region", { name: /Benchmark running/ });
  await expect(running).toContainText("Corpus e2e");
  const progress = running.getByRole("progressbar", {
    name: "Benchmark progress",
  });
  await expect(progress).toHaveAttribute("aria-valuenow", "0");
  await expect(progress).toHaveAttribute("aria-valuemax", "2");
  await expect(
    page.getByRole("button", { name: "Start the benchmark" }),
  ).toBeDisabled();
  await expect(page.getByText("A benchmark is already running")).toBeVisible();
  await expectNoViolations(page);

  // No Worker takes the tasks here: the administrator cancels, and the screen says nothing is left running.
  await running.getByRole("button", { name: "Cancel the benchmark" }).click();
  const confirm = page.getByRole("alertdialog", {
    name: "Cancel the benchmark?",
  });
  await expect(confirm).toContainText("Production is not touched");
  await confirm.getByRole("button", { name: "Cancel the benchmark" }).click();
  await expect(running).toBeHidden();
  await expect(page).toHaveURL(/\/admin\/benchmark$/);
  await expect(
    page.getByRole("button", { name: "Start the benchmark" }),
  ).toBeEnabled();
});

test("un benchmark de bout en bout se compose en chaînes, se lance et s'annule", async ({
  page,
}) => {
  await asAdmin(page);
  await page.goto("/admin/benchmark");
  await page.getByRole("radio", { name: "Corpus e2e" }).check();
  const e2e = page.getByRole("button", { name: /End to end/ });
  await expect(e2e).toBeEnabled();
  await e2e.click();
  await expect(e2e).toHaveAttribute("aria-pressed", "true");
  const start = page.getByRole("button", { name: "Start the benchmark" });

  // Two empty chains to begin with: nothing can start, and a chain can only be removed above the minimum.
  const chain = (n: number) => page.getByRole("group", { name: `Chain ${n}` });
  await expect(chain(1)).toBeVisible();
  await expect(chain(2)).toBeVisible();
  await expect(chain(3)).toHaveCount(0);
  await expect(chain(1).getByRole("button", { name: /Remove/ })).toBeDisabled();
  await expect(start).toBeDisabled();
  // Only a diarization model ready on a Worker can be picked.
  await expect(
    chain(1)
      .getByLabel(/Diarization model/)
      .getByRole("option", { name: /sortformer/i }),
  ).toHaveAttribute("disabled", "");

  // A transcription model typed by name goes to the first chain still waiting for one.
  await page.getByLabel("Add a model by name").fill("vendor/model-a");
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await page.getByLabel("Add a model by name").fill("vendor/model-b");
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await expect(chain(1).getByLabel(/Transcription model/)).toHaveValue(
    "vendor/model-a",
  );
  await expect(chain(2).getByLabel(/Transcription model/)).toHaveValue(
    "vendor/model-b",
  );
  for (const n of [1, 2])
    await chain(n)
      .getByLabel(/Diarization model/)
      .selectOption("pyannote-community-1");
  await expect(page.getByText("2 chains on 4 clips")).toBeVisible();
  await expect(start).toBeEnabled();

  // A pair is measured once.
  await chain(2)
    .getByLabel(/Transcription model/)
    .selectOption("vendor/model-a");
  await expect(chain(2).getByRole("alert")).toContainText(
    "Same pair as chain 1",
  );
  await expect(start).toBeDisabled();
  await chain(2)
    .getByLabel(/Transcription model/)
    .selectOption("vendor/model-b");
  await expect(chain(2).getByRole("alert")).toHaveCount(0);

  // Chains are added up to four and removed down to two.
  const add = page.getByRole("button", { name: "Add a chain" });
  await add.click();
  await add.click();
  await expect(chain(4)).toBeVisible();
  await expect(add).toBeDisabled();
  await expect(start).toBeDisabled(); // two chains are still empty
  await chain(4)
    .getByRole("button", { name: /Remove/ })
    .click();
  await chain(3)
    .getByRole("button", { name: /Remove/ })
    .click();
  await expect(chain(3)).toHaveCount(0);
  await expect(page.getByText("2 chains on 4 clips")).toBeVisible();
  await expectNoViolations(page);

  await start.click();
  const running = page.getByRole("region", { name: /Benchmark running/ });
  await expect(running).toContainText("Corpus e2e");
  await expect(
    running.getByRole("progressbar", { name: "Benchmark progress" }),
  ).toHaveAttribute("aria-valuenow", "0");
  await expect(page.getByText("A benchmark is already running")).toBeVisible();
  await expectNoViolations(page);

  await running.getByRole("button", { name: "Cancel the benchmark" }).click();
  const confirm = page.getByRole("alertdialog", {
    name: "Cancel the benchmark?",
  });
  await confirm.getByRole("button", { name: "Cancel the benchmark" }).click();
  await expect(running).toBeHidden();
  await expect(page).toHaveURL(/\/admin\/benchmark$/);
});

test("l'administrateur lit les résultats d'un benchmark", async ({ page }) => {
  await asAdmin(page);
  await page.goto("/admin/benchmark/results");

  // The best model of the stage comes from the finished run; a partial result is never crowned.
  const best = page.getByRole("region", { name: "Best model per stage" });
  await expect(best).toContainText("EER 5.0%");
  await expect(best).toContainText("Already in production");
  // Diarization: the Sortformer model has the lower DER (the fixtures run a fake diarizer, so no catalogue model is
  // the one in production and no comparison with it is claimed).
  await expect(best).toContainText("DER 7.0%");
  await expect(best).toContainText(/sortformer/i);
  // Transcription: the more accurate model is crowned and differs from the production one.
  await expect(best).toContainText("seed-asr-b");
  await expect(best).toContainText("WER 8.1%");
  await expect(best).toContainText("Differs from production");
  // End to end: the chain with the lowest cpWER is crowned, with its two models.
  await expect(best).toContainText("cpWER 10.5%");
  await expect(best).toContainText("community-1 + seed-asr-b");
  const history = page.getByRole("region", { name: "History" });
  const row = (text: string) =>
    history.getByRole("row").filter({ hasText: text });
  // A model name can appear in several stages (the chains reuse them): tell the rows apart by their stage cell.
  const stageRow = (stage: string, text: string) =>
    history
      .getByRole("row")
      .filter({ has: page.getByRole("cell", { name: stage, exact: true }) })
      .filter({ hasText: text });
  await expect(row("Failed")).toHaveCount(1);
  await expect(row("Cancelled")).toHaveCount(2); // the runs cancelled by the two previous journeys
  await expect(row("Partial")).toHaveCount(1);
  await expectNoViolations(page);

  await row("Partial").getByRole("link", { name: "Details" }).click();
  await expect(page).toHaveURL(/\/admin\/benchmark\/results\?run=\d+/);
  await expect(page.getByRole("note")).toContainText(
    "raw scores of different models cannot be compared",
  );
  await expect(page.getByRole("status")).toContainText("14 of 18 measured");
  const table = page.getByRole("table");
  await expect(table.getByRole("row")).toHaveCount(3);
  // A measure is explained by its "i": on hover and from the keyboard.
  const info = table.getByRole("button", {
    name: "Definition: EER",
    exact: true,
  });
  await info.hover();
  await expect(page.getByRole("tooltip")).toContainText("Equal Error Rate");
  await page.mouse.move(0, 0);
  await expect(page.getByRole("tooltip")).toHaveCount(0);
  // Reached with the keyboard: a programmatic focus after a pointer is not a keyboard focus.
  await info.focus();
  await page.keyboard.press("Shift+Tab");
  await page.keyboard.press("Tab");
  await expect(page.getByRole("tooltip")).toContainText("Lower is better");
  await page.keyboard.press("Escape");
  await expectNoViolations(page);

  await page.getByRole("link", { name: "Back to history" }).click();
  await row("Failed").getByRole("link", { name: "Details" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "No result was produced.",
  );
  await expect(
    page.getByRole("link", { name: "Open Voice processing" }),
  ).toBeVisible();

  // The diarization run: DER and its three parts, the matching note, and the bias warning.
  await page.getByRole("link", { name: "Back to history" }).click();
  await stageRow("Diarization", "Sortformer")
    .getByRole("link", { name: "Details" })
    .click();
  await expect(
    page.getByRole("heading", { name: /^Diarization ·/ }),
  ).toBeVisible();
  await expect(page.getByRole("note").first()).toContainText("anonymous");
  const derTable = page.getByRole("table");
  await expect(derTable.getByRole("row")).toHaveCount(3);
  await expect(derTable.getByRole("row").nth(1)).toContainText("7.0%"); // rows follow the model id
  await expect(derTable.getByRole("row").nth(2)).toContainText("11.4%");
  await expect(
    page.getByText("Diarization error rate (lower is better)"),
  ).toBeVisible();
  await derTable
    .getByRole("button", { name: "Definition: Confusion", exact: true })
    .hover();
  await expect(page.getByRole("tooltip")).toContainText("wrong speaker");
  await page.mouse.move(0, 0);
  await expectNoViolations(page);

  // A run on a corpus whose reference was imported has no production bias to announce.
  await page.getByRole("link", { name: "Back to history" }).click();
  await row("seed-asr-c").getByRole("link", { name: "Details" }).click();
  await expect(page.getByRole("table").getByRole("row")).toHaveCount(3);
  await expect(
    page.getByText("the production model has an advantage"),
  ).toHaveCount(0);

  // The transcription run: WER and CER, the bias warning, and no separation chart (a voice-only reading).
  await page.getByRole("link", { name: "Back to history" }).click();
  await stageRow("Transcription", "seed-asr-a")
    .getByRole("link", { name: "Details" })
    .click();
  await expect(
    page.getByRole("heading", { name: /Transcription/ }),
  ).toBeVisible();
  await expect(
    page.getByText("the production model has an advantage"),
  ).toBeVisible();
  const werTable = page.getByRole("table");
  await expect(werTable.getByRole("row")).toHaveCount(3);
  await expect(werTable.getByRole("row").nth(1)).toContainText("12.0%");
  await expect(werTable.getByRole("row").nth(2)).toContainText("8.1%");
  await expect(
    page.getByText("Word error rate (lower is better)"),
  ).toBeVisible();
  await expect(page.getByText("Separation of the mean scores")).toHaveCount(0);
  const wer = werTable.getByRole("button", {
    name: "Definition: WER",
    exact: true,
  });
  await wer.hover();
  await expect(page.getByRole("tooltip")).toContainText("Word Error Rate");
  await page.mouse.move(0, 0);
  await expectNoViolations(page);

  // The end-to-end run: cpWER first, then WER and DER; a chain is named after its two models.
  await page.getByRole("link", { name: "Back to history" }).click();
  await stageRow("End to end", "seed-asr-a")
    .getByRole("link", { name: "Details" })
    .click();
  await expect(
    page.getByRole("heading", { name: /^End to end ·/ }),
  ).toBeVisible();
  await expect(page.getByRole("note").first()).toContainText("wrong speaker");
  await expect(
    page.getByText("the production model has an advantage"),
  ).toBeVisible();
  const cpTable = page.getByRole("table");
  await expect(cpTable.getByRole("columnheader").nth(1)).toContainText("cpWER");
  await expect(cpTable.getByRole("row")).toHaveCount(3);
  const first = cpTable.getByRole("row").nth(1);
  await expect(first).toContainText("community-1 + seed-asr-a");
  await expect(first).toContainText("19.0%"); // cpWER
  await expect(first).toContainText("12.0%"); // WER
  await expect(first).toContainText("11.4%"); // DER
  await expect(first).toContainText("3 / 3");
  await expect(cpTable.getByRole("row").nth(2)).toContainText("10.5%");
  await expect(
    page.getByText("Speaker-attributed word error rate, cpWER"),
  ).toBeVisible();
  const cp = cpTable.getByRole("button", {
    name: "Definition: cpWER",
    exact: true,
  });
  await cp.hover();
  await expect(page.getByRole("tooltip")).toContainText("wrong speaker");
  await page.mouse.move(0, 0);
  await expectNoViolations(page);
});

// Last: it changes the model in production, which the summary of the previous journey compares with.
test("l'administrateur utilise un modèle mesuré en production, sans rien réanalyser", async ({
  page,
}) => {
  await asAdmin(page);
  await page.goto("/admin/benchmark/results");
  const history = page.getByRole("region", { name: "History" });
  await history
    .getByRole("row")
    .filter({ hasText: "seed-asr-a" })
    .filter({ hasNotText: "End to end" }) // the end-to-end run names the same models
    .first()
    .getByRole("link", { name: "Details" })
    .click();
  const model = async () =>
    (await (await page.request.get("/api/admin/transcription-settings")).json())
      .model as string;
  const before = await model();

  await page
    .getByRole("button", { name: /Use in production/ })
    .filter({ hasText: "seed-asr-b" })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "Use seed-asr-b in production?",
  });
  await expect(dialog).toContainText(
    "Only the work to come uses the new model",
  );
  await expect(dialog).toContainText("Nothing is reanalysed");
  const confirm = dialog.getByRole("button", { name: "Use in production" });
  await expect(confirm).toBeDisabled(); // never without the explicit acknowledgement
  await expectNoViolations(page);
  await dialog.getByRole("checkbox").check();
  await expect(confirm).toBeEnabled();
  await confirm.click();
  await expect(dialog).toContainText("The model in production has changed");
  await expect(
    dialog.getByRole("link", { name: "Open the reanalysis screen" }),
  ).toBeVisible();
  expect(await model()).toBe("seed-asr-b");
  expect(before).not.toBe("seed-asr-b");
  // Nothing was reanalysed: no job was created by the change.
  const jobs = await (
    await page.request.get("/api/admin/jobs?status=pending")
  ).json();
  expect(JSON.stringify(jobs)).not.toContain("transcription.audio");

  // Reopened, the model is reported as already the one in production and cannot be confirmed again.
  await dialog.getByRole("button", { name: "Close" }).first().click();
  await page
    .getByRole("button", { name: /Use in production/ })
    .filter({ hasText: "seed-asr-b" })
    .click();
  const again = page.getByRole("dialog", {
    name: "Use seed-asr-b in production?",
  });
  await expect(again).toContainText("already the one in production");
  await expect(
    again.getByRole("button", { name: "Use in production" }),
  ).toBeDisabled();
});
