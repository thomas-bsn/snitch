import { expect, request, test } from "@playwright/test";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

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

  const matching = page
    .getByRole("heading", { name: "Voice matching" })
    .locator("..");
  await expect(matching).toContainText(
    "Only human-validated samples are references",
  );
  await matching.getByLabel("Suggestion threshold").fill("0.72");
  await matching.getByRole("button", { name: "Save thresholds" }).click();
  await expect(page.getByRole("status")).toContainText("Settings saved.");

  await matching.getByRole("button", { name: "Match voices again" }).click();
  await expect(matching.getByRole("status")).toContainText("voices evaluated:");
  await expect(page.getByText("Automatic attribution: disabled")).toBeVisible();

  const automation = page
    .getByRole("heading", { name: "Automatic voice attribution" })
    .locator("..");
  await automation.getByLabel("Minimum confirmations").fill("1");
  await automation.getByLabel("Minimum precision").fill("0.5");
  await automation
    .getByRole("button", { name: "Save activation criteria" })
    .click();
  await expect(
    page.getByText("Settings saved.", { exact: true }),
  ).toBeVisible();

  const enable = automation.getByRole("button", {
    name: "Enable automatic attribution",
  });
  await expect(enable).toBeDisabled();
  await automation.getByLabel("Enable even though criteria are unmet").check();
  await enable.click();
  await expect(page.getByText("Automatic attribution: enabled")).toBeVisible();
  await automation
    .getByRole("button", { name: "Disable automatic attribution" })
    .click();
  await expect(page.getByText("Automatic attribution: disabled")).toBeVisible();

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
  // The real provider must be active for the inventory to require the models.
  const pipeline = await admin.put("/api/admin/voice-pipeline", {
    headers: csrf,
    data: {
      provider: "pyannote",
      model: "pyannote/speaker-diarization-community-1",
      model_version: "community-1",
      access_confirmed: true,
      min_fingerprint_seconds: 1,
      min_quality: 0.5,
    },
  });
  expect(pipeline.ok()).toBe(true);
  const announce = async (name: string, models: object) => {
    const created = await admin.post("/api/admin/workers", {
      headers: csrf,
      data: { name, max_concurrency: 1 },
    });
    const { token } = await created.json();
    const pulse = await admin.post("/api/worker/pulse", {
      headers: { Authorization: `Bearer ${token}` },
      data: { capacity: 1, supported_steps: ["voice.*"], models },
    });
    expect(pulse.ok()).toBe(true);
  };
  await announce("inventaire-complet", {
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

  await context.close();
  await admin.dispose();
});
