import AxeBuilder from "@axe-core/playwright";
import {
  expect,
  request,
  test,
  type APIRequestContext,
  type Browser,
} from "@playwright/test";
import { uploadClip } from "./library-fixture";
import {
  ADMIN,
  installAdministrator,
  sessionFor,
  setAccountLanguage,
} from "./shell-fixture";

// The event timeline of the Clip History and the automatic mode of the Schedule form, on the real
// API. Every journey owns its clips (unique names, no fixed account, no count shared with another
// test) and leaves nothing active behind.

const origin = () => process.env.SNITCH_TEST_URL!;
let sequence = 0;
const unique = (label: string) =>
  `synthetic-frise-${label}-${Date.now()}-${sequence++}`;

type Admin = { api: APIRequestContext; csrf: string };

async function admin(): Promise<Admin> {
  await installAdministrator().catch(() => undefined);
  const api = await request.newContext({
    baseURL: origin(),
    extraHTTPHeaders: { Origin: origin() },
    storageState: await sessionFor(ADMIN),
  });
  return {
    api,
    csrf: (await (await api.get("/api/auth/me")).json()).csrf_token,
  };
}

const post = (a: Admin, path: string, data?: unknown) =>
  a.api.post(path, { headers: { "X-CSRF-Token": a.csrf }, data });

async function chooseTrack(a: Admin, clip: number) {
  const response = await a.api.put(`/api/admin/clips/${clip}/audio-track`, {
    headers: { "X-CSRF-Token": a.csrf },
    data: { audio_stream_index: 2 },
  });
  expect(response.ok()).toBe(true);
}

/** A test Worker claims the transcription of the clip and fails it once, with a model and a cause. */
async function failOnce(a: Admin, clip: number, workerName: string) {
  const created = await post(a, "/api/admin/workers", {
    name: workerName,
    max_concurrency: 1,
  });
  const worker = (await created.json()) as { id: number; token: string };
  const call = async (path: string, body?: unknown, lease?: string) => {
    const response = await a.api.post(path, {
      data: body,
      headers: {
        Authorization: `Bearer ${worker.token}`,
        ...(lease ? { "X-Worker-Lease-Token": lease } : {}),
      },
    });
    expect(response.ok(), path).toBe(true);
    return response.json().catch(() => null);
  };
  await call("/api/worker/pulse", {
    capacity: 1,
    supported_steps: ["transcription.audio.*"],
  });
  // A task left over by an earlier journey of the shared database is cancelled, not run.
  let claimed = await call("/api/worker/jobs/claim");
  for (let n = 0; n < 10 && claimed?.job && claimed.job.clip_id !== clip; n++) {
    await a.api.delete(`/api/admin/jobs/${claimed.job.id}`, {
      headers: { "X-CSRF-Token": a.csrf },
    });
    claimed = await call("/api/worker/jobs/claim");
  }
  expect(claimed?.job?.clip_id).toBe(clip);
  await call(
    `/api/worker/jobs/${claimed.job.id}/fail`,
    {
      error: "whisper_model_missing",
      provider: "whisper",
      model: "synthetic-model-v1",
    },
    claimed.lease_token,
  );
}

async function open(browser: Browser, language: "fr" | "en") {
  // The account's language wins over the browser's: set it, whatever a journey before left.
  await setAccountLanguage(ADMIN, language);
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/queue");
  return { context, page };
}

const words = {
  en: {
    history: "Clip history",
    search: "Search a clip",
    attempt: "Attempt 1",
    worker: "Worker",
    started: "Attempt started",
    retry: "Retry scheduled",
    cause: "Cause",
    action: "Advised action",
  },
  fr: {
    history: "Historique des clips",
    search: "Rechercher un clip",
    attempt: "Tentative 1",
    worker: "Worker",
    started: "Tentative démarrée",
    retry: "Nouvel essai prévu",
    cause: "Cause",
    action: "Action conseillée",
  },
} as const;

for (const language of ["en", "fr"] as const) {
  const w = words[language];
  test(`la frise de l'historique montre les événements de chaque étape (${language})`, async ({
    browser,
  }) => {
    const a = await admin();
    const name = unique("timeline");
    const clip = await uploadClip(`${name}.mp4`, "shared");
    const workerName = unique("worker");
    expect(
      (
        await post(a, `/api/admin/clips/${clip}/analysis`, { scope: "full" })
      ).ok(),
    ).toBe(true);
    await chooseTrack(a, clip);
    await failOnce(a, clip, workerName);
    expect(
      (await post(a, `/api/admin/clips/${clip}/analysis/cancel`)).ok(),
    ).toBe(true);

    const { context, page } = await open(browser, language);
    await page.getByRole("button", { name: w.history }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel(w.search).fill(name);
    await dialog.getByRole("button", { name: /^(Search|Rechercher)$/ }).click();
    const run = dialog.locator("[data-history-run]").filter({ hasText: name });
    await expect(run).toHaveCount(1);

    const step = run.locator('[data-history-step="transcription"]');
    const events = step.locator("[data-timeline] [data-timeline-event]");
    await expect(events).toHaveCount(2);
    const started = step.locator('[data-timeline-event="attempt_started"]');
    await expect(started).toContainText(w.started);
    await expect(started).toContainText(w.attempt);
    await expect(started).toContainText(workerName);
    const retry = step.locator('[data-timeline-event="retry_scheduled"]');
    await expect(retry).toContainText(w.retry);
    await expect(retry).toContainText("synthetic-model-v1");
    await expect(retry).toContainText(workerName);
    await expect(retry).toContainText(w.cause);
    await expect(retry.locator("[data-failure-code]")).toHaveText(
      "whisper_model_missing",
    );
    await expect(retry).toContainText(w.action);
    await expect(retry.locator("time")).toHaveAttribute(
      "datetime",
      /\d{4}-\d{2}-\d{2}T/,
    );
    // Never a raw error text nor a path.
    await expect(dialog).not.toContainText("last_error");
    await expect(dialog).not.toContainText("/tmp");

    const violations = (await new AxeBuilder({ page }).analyze()).violations;
    expect(violations).toEqual([]);
    await context.close();
  });
}
