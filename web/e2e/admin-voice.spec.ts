import AxeBuilder from "@axe-core/playwright";
import { expect, request, test, type Page } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];

async function expectNoViolations(page: Page) {
  const { violations } = await new AxeBuilder({ page })
    .withTags(TAGS)
    .analyze();
  expect(violations).toEqual([]);
}

test("un administrateur règle le rapprochement et l'automatisation vocale", async ({
  browser,
}) => {
  await installAdministrator();
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  await expect(
    page.getByRole("heading", { name: "Voice processing" }),
  ).toBeVisible();

  // The summary comes first; the automatic attribution is off and never turns itself on.
  const summary = page.getByRole("region", { name: "State" });
  await expect(summary).toContainText("Active diarization model");
  await expect(summary).toContainText("Active fingerprint model (read only)");
  const toggle = summary.getByRole("switch", {
    name: "Automatic voice attribution",
  });
  await expect(toggle).toHaveAttribute("aria-checked", "false");
  await expect(
    summary.getByText("Automatic attribution: disabled"),
  ).toBeVisible();

  // The thresholds are folded under "Advanced settings".
  const matching = page.getByRole("region", { name: "Voice matching" });
  await expect(matching).toBeHidden();
  await page.locator("summary").click();
  await expect(matching).toContainText(
    "Only human-validated samples are references",
  );
  await matching.getByLabel("Suggestion threshold").fill("0.72");
  await matching.getByRole("button", { name: "Save thresholds" }).click();
  await expect(page.getByText("Settings saved.")).toBeVisible();

  await matching.getByRole("button", { name: "Match voices again" }).click();
  await expect(matching.getByRole("status")).toContainText("voices evaluated:");

  const automation = page.getByRole("region", { name: "Automation criteria" });
  await automation.getByLabel("Minimum confirmations").fill("1");
  await automation.getByLabel("Minimum precision").fill("0.5");
  await automation
    .getByRole("button", { name: "Save activation criteria" })
    .click();
  await expect(page.getByText("Settings saved.")).toBeVisible();

  // Enabling needs an explicit confirmation, and a second one when the criteria are unmet.
  await toggle.click();
  const dialog = page.getByRole("alertdialog");
  const confirm = dialog.getByRole("button", {
    name: "Enable automatic attribution",
  });
  await expect(confirm).toBeDisabled();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(toggle).toHaveAttribute("aria-checked", "false");
  await toggle.click();
  await dialog.getByLabel("Enable even though criteria are unmet").check();
  await confirm.click();
  await expect(
    summary.getByText("Automatic attribution: enabled"),
  ).toBeVisible();
  await toggle.click();
  await expect(
    summary.getByText("Automatic attribution: disabled"),
  ).toBeVisible();

  // A confirmed reset goes back to the installation values.
  await page.getByRole("button", { name: "Reset to default values" }).click();
  await page
    .getByRole("alertdialog")
    .getByRole("button", { name: "Reset", exact: true })
    .click();
  await expect(matching.getByLabel("Suggestion threshold")).toHaveValue("0.75");

  await expectNoViolations(page);
  await context.close();
});

test("l'écran vocal reste utilisable sur téléphone, au clavier", async ({
  browser,
}) => {
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
    viewport: { width: 400, height: 800 },
  });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  await expect(page.getByRole("region", { name: "State" })).toBeVisible();
  await page.locator("summary").focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("region", { name: "Voice matching" }),
  ).toBeVisible();
  const overflow = await page.evaluate(
    () =>
      document.documentElement.scrollWidth -
      document.documentElement.clientWidth,
  );
  expect(overflow).toBeLessThanOrEqual(0);
  await expectNoViolations(page);
  await context.close();
});

test("l'inventaire vocal montre ce que chaque Worker détient réellement", async ({
  browser,
}) => {
  // The administrator was installed by the previous test of this file (one run, one installation).
  const session = await sessionFor(ADMIN);
  const origin = process.env.SNITCH_TEST_URL!;
  const admin = await request.newContext({
    baseURL: origin,
    extraHTTPHeaders: { Origin: origin },
    storageState: session,
  });
  const me = await (await admin.get("/api/auth/me")).json();
  const csrf = { "X-CSRF-Token": me.csrf_token };
  const announced: string[] = [];
  const announce = async (name: string, models: object) => {
    const created = await admin.post("/api/admin/workers", {
      headers: csrf,
      data: { name, max_concurrency: 1 },
    });
    const { token, id } = await created.json();
    announced.push(id);
    const pulse = await admin.post("/api/worker/pulse", {
      headers: { Authorization: `Bearer ${token}` },
      data: { capacity: 1, supported_steps: ["voice.*"], models },
    });
    expect(pulse.ok()).toBe(true);
    return token as string;
  };
  const completeToken = await announce("inventaire-complet", {
    diarization: {
      provider: "pyannote",
      model: "pyannote/speaker-diarization-community-1",
      version: "3533c8cf8e36",
    },
    fingerprint: {
      provider: "pyannote",
      model: "pyannote/wespeaker-voxceleb-resnet34-LM",
      version: "837717ddb9ff",
    },
  });
  await announce("inventaire-autre-modele", {
    diarization: {
      provider: "pyannote",
      model: "pyannote/speaker-diarization-3.1",
      version: "abcdef012345",
    },
  });

  // Activating the real provider needs a connected Worker that has the model ready.
  const ready = await admin.post("/api/worker/voice-models/status", {
    headers: { Authorization: `Bearer ${completeToken}` },
    data: { model_id: "pyannote-community-1", state: "ready" },
  });
  expect(ready.status()).toBe(204);
  // The real provider must be active for the inventory to require the models.
  const pipeline = await admin.put("/api/admin/voice-pipeline", {
    headers: csrf,
    data: {
      provider: "pyannote",
      model: "pyannote/speaker-diarization-community-1",
      model_version: "3533c8cf8e369892e6b79ff1bf80f7b0286a54ee",
      min_fingerprint_seconds: 1,
      min_quality: 0.5,
    },
  });
  expect(pipeline.ok()).toBe(true);

  const context = await browser.newContext({ storageState: session });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  const inventory = page.getByRole("region", { name: "Worker voice models" });
  await expect(inventory).toBeVisible();

  const complete = inventory.getByRole("listitem").filter({
    hasText: "inventaire-complet",
  });
  await expect(complete).toContainText(
    "pyannote/speaker-diarization-community-1 @ 3533c8cf8e36",
  );
  await expect(complete).toContainText("Not tested");
  await expect(complete).not.toContainText("Offline");

  const other = inventory.getByRole("listitem").filter({
    hasText: "inventaire-autre-modele",
  });
  await expect(other).toContainText("Active model missing");
  await expect(other).toContainText("takes no voice job");
  await expect(other).toContainText("Not installed"); // no embedder declared

  // Leave no Worker behind: the next test starts from a library without any voice model.
  for (const id of announced) {
    const removed = await admin.delete(`/api/admin/workers/${id}`, {
      headers: csrf,
    });
    expect(removed.ok()).toBe(true);
  }
  await context.close();
  await admin.dispose();
});

test("un administrateur choisit, suit puis active un modèle vocal candidat", async ({
  browser,
}) => {
  const session = await sessionFor(ADMIN);
  const origin = process.env.SNITCH_TEST_URL!;
  const admin = await request.newContext({
    baseURL: origin,
    extraHTTPHeaders: { Origin: origin },
    storageState: session,
  });
  const me = await (await admin.get("/api/auth/me")).json();
  const csrf = { "X-CSRF-Token": me.csrf_token };
  // The synthetic provider is active: no catalogue model is active yet, so one can become the candidate.
  const fake = await admin.put("/api/admin/voice-pipeline", {
    headers: csrf,
    data: {
      provider: "fake",
      model: "fixture-diarization",
      model_version: "fixture-v1",
      min_fingerprint_seconds: 1,
      min_quality: 0.5,
      fake: { speakers: 2, dimensions: 2 },
    },
  });
  expect(fake.ok()).toBe(true);
  const worker = async (name: string) => {
    const created = await admin.post("/api/admin/workers", {
      headers: csrf,
      data: { name, max_concurrency: 1 },
    });
    const { token } = await created.json();
    return async (state: object) =>
      expect(
        (
          await admin.post("/api/worker/voice-models/status", {
            headers: { Authorization: `Bearer ${token}` },
            data: { model_id: "pyannote-community-1", ...state },
          })
        ).status(),
      ).toBe(204);
  };
  const preparing = await worker("worker-prepare");
  await worker("worker-sans-modele");

  const context = await browser.newContext({ storageState: session });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  const models = page.getByRole("region", { name: "Diarization model" });
  const card = models.getByRole("listitem", {
    name: "pyannote/speaker-diarization-community-1",
  });
  await expect(card).toContainText("cc-by-4.0");
  await expect(card).toContainText("Restricted access on Hugging Face.");
  await card.getByRole("button", { name: "Choose as candidate" }).click();
  await expect(card.getByText("Candidate", { exact: true })).toBeVisible();
  await expect(card.getByRole("button", { name: "Activate" })).toBeDisabled();

  await preparing({
    state: "downloading",
    bytes_received: 12_000_000,
    bytes_total: 34_000_000,
  });
  await page.reload();
  await expect(card).toContainText("12 / 34 MB");

  await preparing({ state: "failed", cause_code: "hf_token_missing" });
  await page.reload();
  await expect(card).toContainText("Hugging Face token missing");
  await card.getByRole("button", { name: "Retry" }).click();
  // The retry is consumed by the Worker's next report.
  await preparing({ state: "verifying" });
  await preparing({ state: "ready" });
  await page.reload();
  await expect(card).toContainText("Ready");

  await card.getByRole("button", { name: "Activate" }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog).toContainText("new voice pipeline version");
  await expect(dialog).toContainText("worker-sans-modele");
  await expect(dialog).toContainText("stop taking voice jobs");
  await dialog
    .getByRole("button", { name: "Activate for every Worker" })
    .click();
  await expect(card.getByText("Active", { exact: true })).toBeVisible();
  await expect(card.getByText("Candidate", { exact: true })).toHaveCount(0);

  await context.close();
  await admin.dispose();
});
