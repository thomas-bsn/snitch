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
  await expect(
    page.getByRole("button", { name: /Transcription/ }),
  ).toBeDisabled();
  await expect(
    page.getByRole("button", { name: /Transcription/ }),
  ).toContainText("Coming soon");

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

test("l'administrateur lit les résultats d'un benchmark", async ({ page }) => {
  await asAdmin(page);
  await page.goto("/admin/benchmark/results");

  // The best model of the stage comes from the finished run; a partial result is never crowned.
  const best = page.getByRole("region", { name: "Best model per stage" });
  await expect(best).toContainText("EER 5.0%");
  await expect(best).toContainText("Already in production");
  const history = page.getByRole("region", { name: "History" });
  await expect(history.getByRole("row")).toHaveCount(3);
  await expect(history.getByRole("row").nth(1)).toContainText("Failed");
  await expect(history.getByRole("row").nth(2)).toContainText("Partial");
  await expectNoViolations(page);

  await history
    .getByRole("row")
    .nth(2)
    .getByRole("link", { name: "Details" })
    .click();
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
  await history
    .getByRole("row")
    .nth(1)
    .getByRole("link", { name: "Details" })
    .click();
  await expect(page.getByRole("alert")).toContainText(
    "No result was produced.",
  );
  await expect(
    page.getByRole("link", { name: "Open Voice processing" }),
  ).toBeVisible();
});
