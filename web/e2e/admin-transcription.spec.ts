import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

const tags = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];

test("le profil de bibliothèque et le nom propre du Worker", async ({
  browser,
}) => {
  await installAdministrator();
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/transcription");
  await expect(
    page.getByText("Transcription needs configuration."),
  ).toBeVisible();
  await page
    .getByLabel("Whisper/Speaches server URL")
    .fill("http://fixture:8000");
  await page.getByLabel("Library model").fill("library-model");
  await page.getByLabel("Language").fill("fr");
  await page.getByRole("button", { name: "Save and version" }).click();
  await expect(
    page.getByText("Saved, not yet tested from a Worker."),
  ).toBeVisible();
  expect(
    (await new AxeBuilder({ page }).withTags(tags).analyze()).violations,
  ).toEqual([]);
  await page.goto("/admin/workers/new");
  await page.getByLabel("Worker name").fill("fixture-worker");
  await expect(page.getByLabel("Whisper server address")).toHaveValue(
    "http://fixture:8000",
  );
  await page.getByRole("button", { name: "Create the Worker" }).click();
  await page.getByRole("button", { name: "Done" }).click();
  await page.goto("/admin/transcription");
  await expect(page.getByLabel("Library model")).toHaveValue("library-model");
  await page.getByRole("button", { name: "fixture-worker" }).click();
  await page.getByLabel("Worker model name").fill("server-model");
  await page.getByRole("button", { name: "Save" }).click();
  // The table does not list the model any more: reopen the Worker to see what was kept.
  await page.getByRole("button", { name: "fixture-worker" }).click();
  await expect(page.getByLabel("Worker model name")).toHaveValue(
    "server-model",
  );
  await page.getByRole("button", { name: "Close" }).click();
  await page.getByLabel("Library model").fill("new-library-model");
  await page.getByRole("button", { name: "Save and version" }).click();
  await expect(page.getByLabel("Library model")).toHaveValue(
    "new-library-model",
  );
  expect(
    (await new AxeBuilder({ page }).withTags(tags).analyze()).violations,
  ).toEqual([]);
  await page.getByRole("link", { name: "Vocabulary" }).click();
  await page.getByLabel("Term", { exact: true }).fill("Final Fantasy");
  await page.getByLabel("Variants").fill("FF, Final");
  await page.getByLabel("Preferred spelling").fill("Final Fantasy");
  await page.getByRole("button", { name: "Add term" }).click();
  await expect(page.getByText("FF · Final")).toBeVisible();
  expect(
    (await new AxeBuilder({ page }).withTags(tags).analyze()).violations,
  ).toEqual([]);
  const term = page.getByRole("listitem").filter({ hasText: "FF · Final" });
  await term.getByRole("button", { name: "Edit" }).click();
  await page.getByLabel("Preferred spelling").fill("FFXIV");
  await page.getByRole("button", { name: "Save term" }).click();
  await expect(page.getByText("FFXIV", { exact: true })).toBeVisible();
  await page
    .getByRole("listitem")
    .filter({ hasText: "FFXIV" })
    .getByRole("button", { name: "Delete" })
    .click();
  await expect(page.getByText("FFXIV", { exact: true })).toHaveCount(0);
  await page.goto("/admin/voice");
  await expect(page.getByLabel("Diarization model")).toBeVisible();
  await expect(page.getByText("Automatic attribution: disabled")).toBeVisible();
  await page.getByLabel("Suggestion threshold").fill("0.70");
  await page.getByRole("button", { name: "Save thresholds" }).click();
  await expect(page.getByRole("status")).toContainText("saved");
  await page.reload();
  await expect(page.getByLabel("Suggestion threshold")).toHaveValue("0.7");
  await page.getByLabel("Minimum confirmations").fill("1");
  await page.getByRole("button", { name: "Save activation criteria" }).click();
  await expect(page.getByRole("status")).toContainText("saved");
  await expect(
    page.getByRole("button", { name: "Enable automatic attribution" }),
  ).toBeDisabled();
  await page.getByLabel("Enable even though criteria are unmet").check();
  await page
    .getByRole("button", { name: "Enable automatic attribution" })
    .click();
  await expect(page.getByText("Automatic attribution: enabled")).toBeVisible();
  await page
    .getByRole("button", { name: "Disable automatic attribution" })
    .click();
  await expect(page.getByText("Automatic attribution: disabled")).toBeVisible();
  expect(
    (await new AxeBuilder({ page }).withTags(tags).analyze()).violations,
  ).toEqual([]);
  await context.close();
});

test("bouton Test indisponible pour un Worker hors ligne", async ({
  browser,
}) => {
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/transcription");
  // Configure transcription first
  await page
    .getByLabel("Whisper/Speaches server URL")
    .fill("http://fixture:8000");
  await page.getByLabel("Library model").fill("library-model");
  await page.getByLabel("Language").fill("fr");
  await page.getByRole("button", { name: "Save and version" }).click();
  await expect(
    page.getByText("Saved, not yet tested from a Worker."),
  ).toBeVisible();

  // Create a Worker
  await page.goto("/admin/workers/new");
  await page.getByLabel("Worker name").fill("test-worker-1");
  await expect(page.getByLabel("Whisper server address")).toHaveValue(
    "http://fixture:8000",
  );
  await page.getByRole("button", { name: "Create the Worker" }).click();
  await page.getByRole("button", { name: "Done" }).click();

  // Go back to transcription admin - Worker should appear in the table
  await page.goto("/admin/transcription");

  // Check that the Worker row has the Test button
  const workerRow = page.getByRole("row", { name: /test-worker-1/ });
  await expect(workerRow).toBeVisible();

  // A newly created Worker is offline and cannot be asked to test yet.
  await expect(
    workerRow.getByRole("button", { name: "Test from Worker" }),
  ).toHaveCount(0);
  await expect(workerRow.getByText("Configuration received")).toBeVisible();

  expect(
    (await new AxeBuilder({ page }).withTags(tags).analyze()).violations,
  ).toEqual([]);
  await context.close();
});

test("Workers synthétiques hors ligne", async ({ browser }) => {
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/transcription");
  // Configure transcription
  await page
    .getByLabel("Whisper/Speaches server URL")
    .fill("http://fixture:8000");
  await page.getByLabel("Library model").fill("library-model");
  await page.getByLabel("Language").fill("fr");
  await page.getByRole("button", { name: "Save and version" }).click();

  // Create two Workers to test different error states
  await page.goto("/admin/workers/new");
  await page.getByLabel("Worker name").fill("missing-model-worker");
  await expect(page.getByLabel("Whisper server address")).toHaveValue(
    "http://fixture:8000",
  );
  await page.getByRole("button", { name: "Create the Worker" }).click();
  await page.getByRole("button", { name: "Done" }).click();

  await page.goto("/admin/workers/new");
  await page.getByLabel("Worker name").fill("unreachable-worker");
  await page.getByRole("button", { name: "Create the Worker" }).click();
  await page.getByRole("button", { name: "Done" }).click();

  await page.goto("/admin/transcription");

  // Neither synthetic Worker has connected, so neither can run a test.
  for (const name of ["missing-model-worker", "unreachable-worker"]) {
    const row = page.getByRole("row", { name: new RegExp(name) });
    await expect(row.getByText("Configuration received")).toBeVisible();
    await expect(
      row.getByRole("button", { name: "Test from Worker" }),
    ).toHaveCount(0);
  }

  expect(
    (await new AxeBuilder({ page }).withTags(tags).analyze()).violations,
  ).toEqual([]);
  await context.close();
});
