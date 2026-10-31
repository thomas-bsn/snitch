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
  await page.getByLabel("Whisper server address").fill("http://fixture:8000");
  await page.getByLabel("Whisper model name").fill("library-model");
  await page.getByRole("button", { name: "Create the Worker" }).click();
  await page.getByRole("button", { name: "Confirm and create" }).click();
  await page.getByRole("button", { name: "I copied the files" }).click();
  await page.goto("/admin/transcription");
  await expect(page.getByLabel("Library model")).toHaveValue("library-model");
  await page.getByRole("button", { name: "fixture-worker" }).click();
  await page.getByLabel("Worker model name").fill("server-model");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByText("server-model")).toBeVisible();
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
