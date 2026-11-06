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
    page.getByRole("heading", { name: "Voice models" }),
  ).toBeVisible();

  // Settings are prominent above the matrix; automation never enables itself.
  await page
    .getByRole("button", { name: "Identification and thresholds" })
    .click();
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
  const session = await sessionFor(ADMIN);
  const origin = process.env.SNITCH_TEST_URL!;
  const admin = await request.newContext({
    baseURL: origin,
    storageState: session,
    extraHTTPHeaders: { Origin: origin },
  });
  const me = await (await admin.get("/api/auth/me")).json();
  const csrf = { "X-CSRF-Token": me.csrf_token };
  const { id } = await (
    await admin.post("/api/admin/workers", {
      headers: csrf,
      data: { name: "mobile-matrix", max_concurrency: 1 },
    })
  ).json();
  const context = await browser.newContext({
    storageState: session,
    viewport: { width: 375, height: 812 },
  });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  await expect(page.getByRole("table")).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth - innerWidth,
    ),
  ).toBeLessThanOrEqual(0);
  await expectNoViolations(page);
  await page
    .getByRole("button", { name: "Identification and thresholds" })
    .click();
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
  await admin.delete(`/api/admin/workers/${id}`, { headers: csrf });
  await context.close();
  await admin.dispose();
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
  await page
    .getByRole("button", { name: "Identification and thresholds" })
    .click();
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
  await page.getByRole("tab", { name: /Catalogue/ }).click();
  const card = page.getByRole("article", {
    name: "pyannote/speaker-diarization-community-1",
    exact: true,
  });
  await card.getByRole("button", { name: "Details", exact: true }).click();
  let details = page.getByRole("dialog");
  await expect(details).toContainText("cc-by-4.0");
  await details.getByRole("button", { name: "Choose as candidate" }).click();
  await expect(details.getByText("Candidate", { exact: true })).toBeVisible();
  await expect(
    details.getByRole("button", { name: "Activate", exact: true }),
  ).toBeDisabled();
  await details.getByRole("button", { name: "Close", exact: true }).click();
  await page.getByRole("tab", { name: /Workers/ }).click();
  const cell = page
    .getByRole("row", { name: /worker-prepare/ })
    .locator("td")
    .nth(1);
  await preparing({
    state: "downloading",
    bytes_received: 12_000_000,
    bytes_total: 34_000_000,
  });
  await page.reload();
  await expect(cell).toContainText("12 / 34 MB");
  await preparing({ state: "failed", cause_code: "hf_token_missing" });
  await page.reload();
  await expect(cell).toContainText("Hugging Face token missing");
  await cell.getByRole("button", { name: "Retry", exact: true }).click();
  await page
    .getByRole("alertdialog")
    .getByRole("button", { name: "Order the installation" })
    .click();
  await preparing({ state: "verifying" });
  await preparing({ state: "ready" });
  await page.reload();
  await expect(cell).toContainText("Ready");
  await page.getByRole("tab", { name: /Catalogue/ }).click();
  await card.getByRole("button", { name: "Details", exact: true }).click();
  details = page.getByRole("dialog");
  await details.getByRole("button", { name: "Activate", exact: true }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog).toContainText("new voice pipeline version");
  await expect(dialog).toContainText("worker-sans-modele");
  await expect(dialog).toContainText("stop taking voice jobs");
  await dialog
    .getByRole("button", { name: "Activate for every Worker" })
    .click();
  await expect(details.getByText("Active", { exact: true })).toBeVisible();
  await expect(details.getByText("Candidate", { exact: true })).toHaveCount(0);

  await context.close();
  await admin.dispose();
});

test("un administrateur ordonne l'installation d'un modèle à un Worker et suit son état réel", async ({
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
  const created = await admin.post("/api/admin/workers", {
    headers: csrf,
    data: { name: "worker-ordres", max_concurrency: 1 },
  });
  const { token, id } = await created.json();
  const bearer = { Authorization: `Bearer ${token}` };
  const pendingOrder = async () => {
    const pulled = await (
      await admin.get("/api/worker/voice-models", { headers: bearer })
    ).json();
    return pulled.orders[0] as { order_id: string; revision: string };
  };
  const report = async (order: object, state: object) =>
    expect(
      (
        await admin.post("/api/worker/voice-models/status", {
          headers: bearer,
          data: { model_id: "pyannote-wespeaker-resnet34", ...order, ...state },
        })
      ).status(),
    ).toBe(204);

  const context = await browser.newContext({ storageState: session });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  const section = page.getByRole("region", { name: "Models per Worker" });
  const row = section.getByRole("row", { name: /worker-ordres/ });
  const fingerprint = row.locator("td").nth(2);
  await expect(section).toContainText("Coming · #268");
  await expect(fingerprint).toContainText("Not installed");

  // Ordering needs an explicit confirmation.
  await fingerprint.getByRole("button", { name: /Choose a model/ }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog).toContainText("test inference");
  await dialog.getByRole("button", { name: "Order the installation" }).click();
  await expect(fingerprint).toContainText("Requested");

  // The Worker acknowledges, downloads, and fails with an actionable cause.
  const first = await pendingOrder();
  expect(
    (
      await admin.post(`/api/worker/voice-model-orders/${first.order_id}/ack`, {
        headers: bearer,
        data: { accepted: true },
      })
    ).status(),
  ).toBe(204);
  const answers = { order_id: first.order_id, revision: first.revision };
  await report(answers, {
    state: "downloading",
    bytes_received: 5_000_000,
    bytes_total: 26_000_000,
    occupied_bytes: 5_000_000,
  });
  await page.reload();
  await expect(fingerprint).toContainText("Downloading");
  await expect(fingerprint).toContainText("5 / 26 MB");
  await report(answers, { state: "failed", cause_code: "hf_network" });
  await page.reload();
  await expect(fingerprint).toContainText("Failed");
  await expect(fingerprint).toContainText("cannot be reached");

  // Retry orders again; ready comes only from the Worker's own report.
  await fingerprint.getByRole("button", { name: "Retry" }).click();
  await dialog.getByRole("button", { name: "Order the installation" }).click();
  await expect(fingerprint).toContainText("Requested");
  const second = await pendingOrder();
  expect(second.order_id).not.toBe(first.order_id);
  const next = { order_id: second.order_id, revision: second.revision };
  await report(next, { state: "verifying" });
  await report(next, { state: "ready", occupied_bytes: 26_000_000 });
  await page.reload();
  await expect(fingerprint).toContainText("Ready");
  await expect(fingerprint).toContainText("26 MB on disk");

  // Uninstalling is explicit: the confirmation says that no active model will remain, and the order is kept in
  // the history of the Worker.
  await fingerprint.getByRole("button", { name: "Uninstall" }).click();
  await expect(dialog).toContainText("No active model will remain");
  await dialog.getByRole("button", { name: "Order the removal" }).click();
  await expect(fingerprint).toContainText("Removing");
  await row
    .getByRole("button", { name: /Order history of worker-ordres/ })
    .click();
  const history = page.getByRole("dialog", { name: "Order history" });
  await expect(history).toContainText("Removal");
  await expect(history).toContainText("Installation");
  await expect(history).toContainText("Waiting for the Worker");
  await history.getByRole("button", { name: "Close", exact: true }).click();

  await admin.delete(`/api/admin/workers/${id}`, { headers: csrf });
  await context.close();
  await admin.dispose();
});

test("« Appliquer à plusieurs Workers » crée un ordre par Worker coché et l'affiche avant l'envoi", async ({
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
  const made: { id: string; token: string }[] = [];
  for (const name of ["lot-a", "lot-b", "lot-c"])
    made.push(
      await (
        await admin.post("/api/admin/workers", {
          headers: csrf,
          data: { name, max_concurrency: 1 },
        })
      ).json(),
    );
  const pulled = async (token: string) =>
    (
      await (
        await admin.get("/api/worker/voice-models", {
          headers: { Authorization: `Bearer ${token}` },
        })
      ).json()
    ).orders as unknown[];

  const context = await browser.newContext({ storageState: session });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  await page
    .getByRole("button", { name: "Apply to several Workers", exact: true })
    .first()
    .click();
  const dialog = page.getByRole("dialog", {
    name: "Apply a model to several Workers",
  });
  await dialog.getByLabel("Type").selectOption("fingerprint");
  await dialog.getByLabel("lot-a").check();
  await dialog.getByLabel("lot-c").check();
  await dialog.getByRole("button", { name: "Prepare the orders" }).click();
  await expect(dialog).toContainText("Orders to send (2)");
  await expect(dialog).toContainText("on lot-a");
  await expect(dialog).toContainText("on lot-c");
  await expect(dialog).not.toContainText("on lot-b");
  // Listed, not sent yet.
  expect(await pulled(made[0].token)).toHaveLength(0);
  await expect(
    dialog.getByRole("button", { name: "Send 2 orders" }),
  ).toBeDisabled();
  await dialog
    .getByLabel("I confirm sending one order to each listed Worker.")
    .check();
  await dialog.getByRole("button", { name: "Send 2 orders" }).click();
  await expect(dialog.getByText("Order sent")).toHaveCount(2);
  expect(await pulled(made[0].token)).toHaveLength(1);
  expect(await pulled(made[1].token)).toHaveLength(0);
  expect(await pulled(made[2].token)).toHaveLength(1);

  for (const w of made)
    await admin.delete(`/api/admin/workers/${w.id}`, { headers: csrf });
  await context.close();
  await admin.dispose();
});

test("la matrice B conserve la sélection, le catalogue est éditable et les onglets sont accessibles au clavier", async ({
  browser,
}) => {
  const session = await sessionFor(ADMIN);
  const origin = process.env.SNITCH_TEST_URL!;
  const admin = await request.newContext({
    baseURL: origin,
    storageState: session,
    extraHTTPHeaders: { Origin: origin },
  });
  const me = await (await admin.get("/api/auth/me")).json();
  const csrf = { "X-CSRF-Token": me.csrf_token };
  const { id } = await (
    await admin.post("/api/admin/workers", {
      headers: csrf,
      data: { name: "selection-B", max_concurrency: 1 },
    })
  ).json();
  const context = await browser.newContext({ storageState: session });
  const page = await context.newPage();
  await page.goto("/admin/voice");
  await page
    .getByRole("checkbox", { name: "Select selection-B", exact: true })
    .check();
  const workersTab = page.getByRole("tab", { name: /Workers/ });
  await workersTab.focus();
  await page.keyboard.press("ArrowRight");
  await expect(page.getByRole("tab", { name: /Catalogue/ })).toBeFocused();
  await expect(page).toHaveURL(/view=catalogue/);
  await expect(page.getByRole("tabpanel")).toContainText("pyannote");
  const card = page.getByRole("article", {
    name: "pyannote/wespeaker-voxceleb-resnet34-LM",
    exact: true,
  });
  await card.getByRole("button", { name: "Details", exact: true }).click();
  const details = page.getByRole("dialog");
  await details
    .getByRole("button", { name: "Edit reference", exact: true })
    .click();
  await details
    .getByLabel("Display name", { exact: true })
    .fill("Fingerprint catalogue B");
  await details.getByLabel("Required VRAM (MB)").fill("123.456789");
  await details
    .getByRole("button", { name: "Save reference", exact: true })
    .click();
  await expect(
    details.getByRole("button", { name: "Edit reference", exact: true }),
  ).toBeVisible();
  await details.getByRole("button", { name: "Close", exact: true }).click();
  await page.reload();
  await expect(card).toContainText("Fingerprint catalogue B");
  const catalogue = await (await admin.get("/api/admin/voice-models")).json();
  expect(
    catalogue.models.find(
      (m: { id: string }) => m.id === "pyannote-wespeaker-resnet34",
    ).vram_bytes,
  ).toBe(123456789);
  await page.getByRole("tab", { name: /Workers/ }).click();
  await page
    .getByRole("checkbox", { name: "Select selection-B", exact: true })
    .check();
  await page.getByRole("tab", { name: /Catalogue/ }).click();
  await card.getByRole("button", { name: "Install", exact: true }).click();
  const plan = page.getByRole("dialog");
  await expect(
    plan.getByRole("combobox", { name: "Model", exact: true }),
  ).toHaveValue("pyannote-wespeaker-resnet34");
  await expect(plan.getByLabel("selection-B", { exact: true })).toBeChecked();
  await plan.getByRole("button", { name: "Prepare the orders" }).click();
  await expect(plan).toContainText("on selection-B");
  await expect(
    plan.getByRole("button", { name: "Send 1 orders" }),
  ).toBeDisabled();
  await plan.getByRole("button", { name: "Close", exact: true }).click();
  // Search and category filters operate on references, without ordering downloads.
  await page
    .getByRole("searchbox", { name: "Search references", exact: true })
    .fill("not-a-model");
  await expect(
    page.getByText("No reference matches this filter."),
  ).toBeVisible();
  await page
    .getByRole("searchbox", { name: "Search references", exact: true })
    .fill("");
  await page.getByRole("button", { name: /^Separation/ }).click();
  await expect(page.getByRole("article")).toHaveCount(0);
  await expectNoViolations(page);
  await admin.delete(`/api/admin/workers/${id}`, { headers: csrf });
  await context.close();
  await admin.dispose();
});
