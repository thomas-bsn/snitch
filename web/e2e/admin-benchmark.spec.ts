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
  // Transcription and diarization run, but this corpus has no certified clip; the end-to-end stage does not run yet.
  for (const stage of [/Transcription/, /Diarization/]) {
    await expect(page.getByRole("button", { name: stage })).toBeDisabled();
    await expect(page.getByRole("button", { name: stage })).toContainText(
      "cannot measure",
    );
  }
  await expect(page.getByRole("button", { name: /End to end/ })).toBeDisabled();
  await expect(page.getByRole("button", { name: /End to end/ })).toContainText(
    "Coming soon",
  );

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
  // The catalogue models are offered, none ready on a Worker here: nothing can start.
  await expect(
    page.getByRole("checkbox", { name: /community/i }),
  ).toBeDisabled();
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
  const history = page.getByRole("region", { name: "History" });
  const row = (text: string) =>
    history.getByRole("row").filter({ hasText: text });
  await expect(row("Failed")).toHaveCount(1);
  await expect(row("Cancelled")).toHaveCount(1); // the run cancelled by the previous journey
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
  await row("Sortformer").getByRole("link", { name: "Details" }).click();
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

  // The transcription run: WER and CER, the bias warning, and no separation chart (a voice-only reading).
  await page.getByRole("link", { name: "Back to history" }).click();
  await row("seed-asr-a").getByRole("link", { name: "Details" }).click();
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
});
