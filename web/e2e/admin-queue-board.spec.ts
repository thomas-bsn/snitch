import AxeBuilder from "@axe-core/playwright";
import {
  expect,
  request,
  test,
  type APIRequestContext,
  type Browser,
  type Page,
} from "@playwright/test";
import { uploadClip } from "./library-fixture";
import {
  ADMIN,
  installAdministrator,
  sessionFor,
  setAccountLanguage,
} from "./shell-fixture";

// The columns of the Task Queue board, the causes of failure and their advised actions, the
// load summary and the reconnection of the live view. Every journey owns its clips (unique
// names, no fixed account, no count shared with another test) and cancels what it started.

const origin = () => process.env.SNITCH_TEST_URL!;

type Admin = {
  api: APIRequestContext;
  csrf: string;
  start: (clip: number) => Promise<void>;
  chooseTrack: (clip: number) => Promise<void>;
  cancel: (clip: number) => Promise<void>;
};

async function admin(): Promise<Admin> {
  await installAdministrator().catch(() => undefined);
  const api = await request.newContext({
    baseURL: origin(),
    extraHTTPHeaders: { Origin: origin() },
    storageState: await sessionFor(ADMIN),
  });
  const csrf = (await (await api.get("/api/auth/me")).json()).csrf_token;
  const headers = { "X-CSRF-Token": csrf };
  return {
    api,
    csrf,
    start: async (clip) => {
      const response = await api.post(`/api/admin/clips/${clip}/analysis`, {
        headers,
        data: { scope: "full" },
      });
      expect(response.ok()).toBe(true);
    },
    chooseTrack: async (clip) => {
      const response = await api.put(`/api/admin/clips/${clip}/audio-track`, {
        headers,
        data: { audio_stream_index: 2 },
      });
      expect(response.ok()).toBe(true);
    },
    cancel: async (clip) => {
      await api.post(`/api/admin/clips/${clip}/analysis/cancel`, { headers });
    },
  };
}

let sequence = 0;
const unique = (label: string) =>
  `synthetic-board-${label}-${Date.now()}-${sequence++}.mp4`;

/** A test Worker that claims the queued transcription of a clip and either fails or finishes it. */
async function runWorker(
  a: Admin,
  clip: number,
  outcome: { fail: string } | { complete: true },
) {
  const created = await a.api.post("/api/admin/workers", {
    headers: { "X-CSRF-Token": a.csrf },
    data: { name: unique("worker").replace(".mp4", ""), max_concurrency: 1 },
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
  try {
    await call("/api/worker/pulse", {
      capacity: 1,
      supported_steps: ["transcription.audio.*"],
    });
    // A task left over by an earlier journey of the shared database is cancelled, not run.
    let claimed = await call("/api/worker/jobs/claim");
    for (
      let n = 0;
      n < 10 && claimed?.job && claimed.job.clip_id !== clip;
      n++
    ) {
      await a.api.delete(`/api/admin/jobs/${claimed.job.id}`, {
        headers: { "X-CSRF-Token": a.csrf },
      });
      claimed = await call("/api/worker/jobs/claim");
    }
    expect(claimed?.job?.clip_id).toBe(clip);
    if ("fail" in outcome)
      await call(
        `/api/worker/jobs/${claimed.job.id}/fail`,
        { error: outcome.fail },
        claimed.lease_token,
      );
    else
      await call(
        `/api/worker/jobs/${claimed.job.id}/complete`,
        {
          input_version: claimed.job.input_version,
          result: {
            provider: "fixture",
            model: "deterministic-v1",
            version: "deterministic-v1",
            language: "fr",
            audio_stream_index: 2,
            segments: [
              {
                start: 0.2,
                end: 0.9,
                text: "Bonjour",
                words: [{ word: "Bonjour", start: 0.2, end: 0.9 }],
              },
            ],
          },
        },
        claimed.lease_token,
      );
  } finally {
    await a.api.delete(`/api/admin/workers/${worker.id}`, {
      headers: { "X-CSRF-Token": a.csrf },
    });
  }
}

async function open(
  browser: Browser,
  options: {
    language?: "fr" | "en";
    theme?: "light" | "dark";
    viewport?: { width: number; height: number };
  } = {},
) {
  // The account's language wins over the browser's: set it, whatever a journey before left.
  await setAccountLanguage(ADMIN, options.language ?? "en");
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
    viewport: options.viewport,
  });
  await context.addInitScript((theme) => {
    if (theme) localStorage.setItem("snitch-ui-theme", theme);
  }, options.theme);
  const page = await context.newPage();
  await page.goto("/queue");
  return { context, page };
}

const card = (page: Page, clip: number) =>
  page.locator(`[data-run-card="${clip}"]`);
const column = (page: Page, name: string) =>
  page.locator(`[data-column="${name}"]`);
/** The schedule form sits above the board: bring the board into the window first. */
const boardIntoView = (page: Page) =>
  page.locator("[data-board]").scrollIntoViewIfNeeded();

test("les colonnes placent chaque analyse selon son étape courante, avec leur compteur", async ({
  browser,
}) => {
  const a = await admin();
  const blocked = await uploadClip(unique("blocked"), "shared");
  const waiting = await uploadClip(unique("waiting"), "shared");
  const { context, page } = await open(browser);
  await expect(page.getByRole("status").first()).toHaveText("Live updates on");
  await a.start(blocked);
  await a.start(waiting);
  await a.chooseTrack(waiting);
  await boardIntoView(page);

  const names = [
    "Transcription",
    "Search",
    "Distinct voices",
    "Voice recognition",
    "Completed",
  ];
  const headings = await page
    .locator("[data-board] [data-column] h3")
    .allTextContents();
  expect(headings.map((h) => h.replace(/\d+$/, "").trim())).toEqual(names);

  // Both analyses are still at their first step, whatever it is waiting for.
  await expect(
    column(page, "transcription").locator(card(page, blocked)),
  ).toBeVisible();
  await expect(
    column(page, "transcription").locator(card(page, waiting)),
  ).toBeVisible();
  // Waiting for a Worker advises to start one.
  await expect(card(page, waiting)).toContainText("install or start a Worker");
  await expect(card(page, blocked)).toContainText(
    "Waiting for a compatible Worker",
  );
  // The counter is the number of cards of the column.
  for (const name of ["transcription", "search", "voices", "recognition"]) {
    const cards = await column(page, name).locator("[data-run-card]").count();
    await expect(column(page, name).locator("[data-column-count]")).toHaveText(
      String(cards),
    );
  }

  // Once the transcription is done the analysis moves on, without a reload.
  await runWorker(a, waiting, { complete: true });
  await expect(
    column(page, "transcription").locator(card(page, waiting)),
  ).toHaveCount(0, { timeout: 20000 });
  await expect(
    page.locator("[data-board]").locator(card(page, waiting)),
  ).toHaveCount(1);
  await expect(card(page, blocked)).toBeVisible();

  await a.cancel(blocked);
  await a.cancel(waiting);
  await context.close();
  await a.api.dispose();
});

test("plus de vingt analyses actives sont toutes affichées, sans pagination", async ({
  browser,
}) => {
  const a = await admin();
  const clips: number[] = [];
  for (let n = 0; n < 22; n++)
    clips.push(await uploadClip(unique(`many-${n}`), "shared"));
  for (const clip of clips) await a.start(clip);
  const { context, page } = await open(browser);
  await boardIntoView(page);
  for (const clip of clips)
    await expect(card(page, clip)).toHaveCount(1, { timeout: 20000 });
  await expect(page.getByRole("button", { name: /next|page/i })).toHaveCount(0);
  await expect(card(page, clips[0])).toHaveCSS("content-visibility", "auto");
  for (const clip of clips) await a.cancel(clip);
  await context.close();
  await a.api.dispose();
});

test("une colonne très chargée n'affiche d'abord qu'une fenêtre de cartes et dit combien en restent", async ({
  browser,
}) => {
  // A synthetic queue of 130 runs at the same step, served in place of the real one: no clip
  // is created, the stream is refused so the page reads the snapshot.
  const now = new Date().toISOString();
  const run = (id: number) => ({
    id,
    clip_id: id,
    clip_name: `synthetic-window-${id}.mp4`,
    scope: "full",
    status: "active",
    started_by: null,
    created_at: now,
    updated_at: now,
    finished_at: null,
    mode: "",
    forced_steps: [],
    confirmed_replace_corrections: false,
    confirmed_replace_validations: false,
    confirmed_stake_at: null,
    pending_voices: 0,
    steps: ["preparation", "transcription"].map((step, i) => ({
      step,
      status: i === 0 ? "done" : "waiting",
      reason: i === 0 ? "completed" : "awaiting_worker",
      job_id: null,
      attempt: i === 0 ? 1 : 0,
      started_at: null,
      finished_at: null,
      updated_at: now,
    })),
  });
  const queue = {
    active: Array.from({ length: 130 }, (_, i) => run(i + 1)),
    completed: [],
    tasks: [],
    summary: {
      running_tasks: 0,
      waiting_clips: 130,
      max_concurrency: 1,
      paused: false,
    },
  };
  await setAccountLanguage(ADMIN, "en");
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.route("**/api/admin/analysis/queue/events*", (route) =>
    route.abort(),
  );
  await page.route("**/api/admin/analysis/queue", (route) =>
    route.fulfill({ json: queue }),
  );
  await page.goto("/queue");
  const transcription = column(page, "transcription");
  await expect(transcription.locator("[data-column-count]")).toHaveText("130", {
    timeout: 20000,
  });
  // Only a window of cards is drawn, and the rest is said, never hidden.
  await expect(transcription.locator("[data-run-card]")).toHaveCount(50);
  const more = transcription.locator("[data-column-more]");
  await expect(more).toContainText("80 more cards in this column");
  await more.getByRole("button", { name: /Show 80 more/ }).click();
  await expect(transcription.locator("[data-run-card]")).toHaveCount(130);
  await expect(more).toHaveCount(0);
  await context.close();
});

for (const language of ["en", "fr"] as const) {
  test(`une cause d'échec montre son texte, son code et l'action conseillée (${language})`, async ({
    browser,
  }) => {
    const a = await admin();
    const clip = await uploadClip(unique("failed"), "shared");
    await a.start(clip);
    await a.chooseTrack(clip);
    await runWorker(a, clip, { fail: "invalid_input" });
    const { context, page } = await open(browser, { language });
    await boardIntoView(page);
    const failure = card(page, clip).locator("[data-failure]");
    await expect(failure).toBeVisible();
    await expect(failure.locator("[data-failure-code]")).toHaveText(
      "invalid_input",
    );
    if (language === "en") {
      await expect(failure).toContainText("Invalid media");
      await expect(failure).toContainText("Check the clip media.");
    } else {
      await expect(failure).toContainText("Média invalide");
      await expect(failure).toContainText("Vérifier le média du clip.");
    }
    // A dead analysis stays on the board, with its state, in the column of the failing step.
    await expect(
      column(page, "transcription").locator(card(page, clip)),
    ).toBeVisible();
    await expect(card(page, clip).locator("[data-run-status]")).toHaveText(
      language === "en" ? "Partly failed" : "Échec partiel",
    );
    await context.close();
    await a.api.dispose();
  });
}

test("la barre de résumé suit la charge du serveur", async ({ browser }) => {
  const a = await admin();
  const clip = await uploadClip(unique("summary"), "shared");
  await a.start(clip);
  const { context, page } = await open(browser);
  const summary = page.locator("[data-queue-summary]");
  await expect(summary).toBeVisible();
  const queue = await (await a.api.get("/api/admin/analysis/queue")).json();
  await expect(summary).toContainText(
    `${queue.summary.running_tasks} of ${queue.summary.max_concurrency}`,
  );
  await expect(summary.locator("[data-summary-waiting]")).toContainText(
    String(queue.summary.waiting_clips),
  );
  await expect(summary.locator("[data-summary-paused]")).toContainText(
    queue.summary.paused ? "paused" : "active",
  );
  // The concurrency and pause popover is still there.
  await page
    .locator("summary")
    .filter({ hasText: "Concurrency settings" })
    .click();
  await expect(page.getByLabel("Global concurrency")).toBeVisible();
  await a.cancel(clip);
  await context.close();
  await a.api.dispose();
});

test("la reconnexion garde la dernière vue et le repli est suspendu onglet caché", async ({
  browser,
}) => {
  const a = await admin();
  const clip = await uploadClip(unique("reconnect"), "shared");
  const late = await uploadClip(unique("reconnect-late"), "shared");
  await a.start(clip);
  await setAccountLanguage(ADMIN, "en");
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  // A browser keeps an open stream through a cut, so the page starts with its stream refused
  // (the polling fallback then carries the view); the cut of the polling comes next.
  let streamCut = true;
  let pollCut = false;
  const polls: number[] = [];
  await context.route(
    /\/api\/admin\/analysis\/queue\/events(\?.*)?$/,
    (route) =>
      streamCut ? route.abort("connectionrefused") : route.continue(),
  );
  await context.route(/\/api\/admin\/analysis\/queue$/, (route) => {
    polls.push(Date.now());
    return pollCut ? route.abort("connectionrefused") : route.continue();
  });
  const page = await context.newPage();
  await page.goto("/queue");
  await boardIntoView(page);
  await expect(card(page, clip)).toBeVisible();
  await expect(page.getByRole("status").first()).toHaveText("Reconnecting…");

  // Everything is out of reach: the last view stays, no global error screen.
  pollCut = true;
  await a.start(late);
  await page.waitForTimeout(6000);
  await expect(card(page, clip)).toBeVisible();
  await expect(card(page, late)).toHaveCount(0);
  await expect(page.getByRole("status").first()).toHaveText("Reconnecting…");
  await expect(page.getByRole("alert")).toHaveCount(0);

  // Hidden tab: the polling fallback is suspended.
  await page.evaluate(() => {
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      get: () => "hidden",
    });
    Object.defineProperty(document, "hidden", {
      configurable: true,
      get: () => true,
    });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  const before = polls.length;
  await page.waitForTimeout(7000);
  expect(polls.length).toBe(before);

  // Back in view and back online: it resumes, catches up and the live view returns.
  pollCut = false;
  streamCut = false;
  await page.evaluate(() => {
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      get: () => "visible",
    });
    Object.defineProperty(document, "hidden", {
      configurable: true,
      get: () => false,
    });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await expect
    .poll(() => polls.length, { timeout: 15000 })
    .toBeGreaterThan(before);
  await expect(card(page, late)).toBeVisible({ timeout: 15000 });
  await expect(page.getByRole("status").first()).toHaveText("Live updates on", {
    timeout: 20000,
  });
  await expect(card(page, clip)).toBeVisible();
  await a.cancel(clip);
  await a.cancel(late);
  await context.close();
  await a.api.dispose();
});

test("à 390 px la page ne défile pas horizontalement", async ({ browser }) => {
  const a = await admin();
  const clips = [
    await uploadClip(unique("mobile-a"), "shared"),
    await uploadClip(unique("mobile-b"), "shared"),
  ];
  for (const clip of clips) await a.start(clip);
  const { context, page } = await open(browser, {
    language: "fr",
    viewport: { width: 390, height: 844 },
  });
  await boardIntoView(page);
  for (const clip of clips) await expect(card(page, clip)).toBeVisible();
  const overflow = await page.evaluate(
    () =>
      document.documentElement.scrollWidth -
      document.documentElement.clientWidth,
  );
  expect(overflow).toBeLessThanOrEqual(0);
  for (const clip of clips) await a.cancel(clip);
  await context.close();
  await a.api.dispose();
});

for (const theme of ["light", "dark"] as const) {
  test(`les colonnes, les causes et le résumé passent l'accessibilité, thème ${theme}`, async ({
    browser,
  }) => {
    const a = await admin();
    const failed = await uploadClip(unique("axe-failed"), "shared");
    const waiting = await uploadClip(unique("axe-waiting"), "shared");
    await a.start(failed);
    await a.chooseTrack(failed);
    await runWorker(a, failed, { fail: "invalid_input" });
    await a.start(waiting);
    await a.chooseTrack(waiting);
    const { context, page } = await open(browser, { theme });
    await boardIntoView(page);
    await expect(card(page, failed).locator("[data-failure]")).toBeVisible();
    await expect(card(page, waiting)).toBeVisible();
    const violations = (await new AxeBuilder({ page }).analyze()).violations;
    expect(violations).toEqual([]);
    await a.cancel(waiting);
    await context.close();
    await a.api.dispose();
  });
}

test("toutes les colonnes, une étape inconnue et une tâche non suivie restent visibles", async ({
  browser,
}) => {
  await admin().then((a) => a.api.dispose());
  await setAccountLanguage(ADMIN, "en");
  const now = new Date().toISOString();
  const step = (name: string, status: string, extra: object = {}) => ({
    step: name,
    status,
    reason: status === "done" ? "completed" : "awaiting_worker",
    job_id: null,
    attempt: 1,
    input_version: null,
    profile: null,
    forced: false,
    started_at: null,
    finished_at: null,
    updated_at: now,
    ...extra,
  });
  const run = (id: number, status: string, steps: object[]) => ({
    id,
    clip_id: id,
    scope: "full",
    status,
    started_by: null,
    created_at: now,
    updated_at: now,
    finished_at: status === "completed" ? now : null,
    mode: "missing",
    forced_steps: [],
    confirmed_replace_corrections: false,
    confirmed_replace_validations: false,
    confirmed_stake_at: null,
    pending_voices: 0,
    clip_name: `simulated-${id}`,
    steps,
  });
  const done = (name: string) => step(name, "done");
  const snapshot = {
    summary: {
      running_tasks: 1,
      waiting_clips: 3,
      max_concurrency: 2,
      paused: true,
    },
    active: [
      run(9001, "active", [step("preparation", "running")]),
      run(9002, "active", [
        done("preparation"),
        done("transcription"),
        step("indexing", "waiting"),
      ]),
      run(9003, "active", [
        done("preparation"),
        done("transcription"),
        done("indexing"),
        step("diarization", "waiting"),
      ]),
      run(9004, "active", [
        done("preparation"),
        done("transcription"),
        done("indexing"),
        done("diarization"),
        step("fingerprints", "not_applicable"),
        step("matching", "waiting"),
      ]),
      // A step outside the perimeter does not move the card; a failed step does.
      run(9005, "failed", [
        done("preparation"),
        done("transcription"),
        step("indexing", "not_applicable"),
        step("diarization", "failed", {
          reason: "job_dead",
          failure_code: "gpu_out_of_memory",
          recommended_action: "check_gpu",
        }),
        step("fingerprints", "waiting"),
      ]),
      run(9006, "active", [
        step("preparation", "done"),
        step("sorcery", "waiting", { reason: "mystery" }),
      ]),
    ],
    completed: [
      run(9007, "completed", [done("preparation"), done("transcription")]),
    ],
    tasks: [
      {
        id: 77,
        clip_id: 9008,
        clip_name: "orphan-task",
        step: "sorcery.audio.2",
        input_version: "v1",
        status: "dead",
        priority: 0,
        attempts: 3,
        max_attempts: 3,
        lease_owner: null,
        lease_expires_at: null,
        next_attempt_at: null,
        last_error: "worker_failed",
        created_at: now,
        updated_at: now,
        result: null,
        cycle: 1,
        failure_code: "worker_failed",
        recommended_action: "check_worker_logs",
      },
    ],
  };
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  await context.route("**/api/admin/analysis/queue/events*", (route) =>
    route.fulfill({
      status: 200,
      contentType: "text/event-stream",
      body: `retry: 600000\nid: 1\nevent: snapshot\ndata: ${JSON.stringify({ seq: 1, snapshot })}\n\n`,
    }),
  );
  // The stream ends after its frame, so the polling fallback answers the same state.
  await context.route(/\/api\/admin\/analysis\/queue$/, (route) =>
    route.fulfill({ json: snapshot }),
  );
  const page = await context.newPage();
  await page.goto("/queue");
  await boardIntoView(page);
  const where = async (id: number) =>
    page
      .locator(`[data-column] [data-run-card="${id}"]`)
      .evaluate((el) =>
        el.closest("[data-column]")!.getAttribute("data-column"),
      );
  expect(await where(9001)).toBe("transcription");
  expect(await where(9002)).toBe("search");
  expect(await where(9003)).toBe("voices");
  expect(await where(9004)).toBe("recognition");
  expect(await where(9005)).toBe("voices");
  expect(await where(9006)).toBe("transcription");
  expect(await where(9007)).toBe("completed");
  await expect(card(page, 9005)).toContainText("gpu_out_of_memory");
  await expect(card(page, 9005)).toContainText("Check the GPU");
  // An unknown step is an alert on its card, never silently dropped.
  await expect(card(page, 9006).getByRole("alert")).toContainText("sorcery");
  // A task that no run follows and whose step is unknown is an alert too.
  const orphan = page.locator('[data-task-card="77"]');
  await expect(orphan.getByRole("alert")).toBeVisible();
  await expect(orphan).toContainText("worker_failed");
  await expect(orphan).toContainText("Check the Worker logs.");
  const summary = page.locator("[data-queue-summary]");
  await expect(summary).toContainText("1 of 2");
  await expect(summary.locator("[data-summary-waiting]")).toContainText("3");
  await expect(summary.locator("[data-summary-paused]")).toContainText(
    "paused",
  );
  await context.close();
});
