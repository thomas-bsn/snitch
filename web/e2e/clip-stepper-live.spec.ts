import { expect, test, type Page } from "@playwright/test";
import { uploadClip } from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

type StepState = "none" | "queued" | "running" | "done" | "failed";
type StepName =
  | "preparation"
  | "transcription"
  | "indexing"
  | "diarization"
  | "fingerprints"
  | "matching";
type RunStatus = "waiting" | "running" | "blocked" | "failed" | "done";

/** What the simulated server says, changed by the test between two events. */
type Server = {
  progress: { transcription: StepState; indexing: StepState };
  run: {
    clipId: number;
    status: "active" | "completed";
    steps: Partial<Record<StepName, RunStatus>>;
  };
  seq: number;
  /** Holds the clip back, to prove the stepper does not wait for it. */
  clipDelayMs?: number;
  /** Refuses the stream, as a network cut would. */
  down?: boolean;
};

function runBody(server: Server) {
  const now = new Date().toISOString();
  const names: StepName[] = [
    "preparation",
    "transcription",
    "indexing",
    "diarization",
    "fingerprints",
    "matching",
  ];
  return {
    id: 1,
    clip_id: server.run.clipId,
    scope: "full",
    status: server.run.status,
    started_by: null,
    created_at: now,
    updated_at: now,
    finished_at: server.run.status === "active" ? null : now,
    mode: "missing",
    forced_steps: [],
    confirmed_replace_corrections: false,
    confirmed_replace_validations: false,
    confirmed_stake_at: null,
    pending_voices: 0,
    steps: names.map((step) => ({
      step,
      status: server.run.steps[step] ?? "waiting",
      reason:
        server.run.steps[step] === "blocked"
          ? "audio_track_selection_required"
          : "awaiting_previous_step",
      job_id: null,
      attempt: 0,
      input_version: null,
      profile: null,
      forced: false,
      started_at: null,
      finished_at: null,
      updated_at: now,
    })),
  };
}

/**
 * The real server holds the clip; the analysis stream and the clip's progress are simulated.
 * Each (re)connection of the stream gets the current state; it ends after its frame and the
 * browser reconnects after 200 ms, which plays the part of an event pushed by the server.
 */
async function simulate(page: Page, clipId: number, server: Server) {
  await page.route("**/api/admin/clips/*/analysis/events*", (route) => {
    if (server.down) return route.abort();
    server.seq += 1;
    return route.fulfill({
      status: 200,
      contentType: "text/event-stream",
      body: `retry: 200\nid: ${server.seq}\nevent: snapshot\ndata: ${JSON.stringify({ seq: server.seq, snapshot: { run: runBody(server) } })}\n\n`,
    });
  });
  await page.route(/\/api\/admin\/clips\/\d+\/analysis$/, (route) =>
    route.fulfill({ json: { run: runBody(server) } }),
  );
  await page.route(new RegExp(`/api/clips/${clipId}$`), async (route) => {
    const real = await route.fetch();
    if (server.clipDelayMs)
      await new Promise((done) => setTimeout(done, server.clipDelayMs));
    await route.fulfill({
      response: real,
      json: { ...(await real.json()), progress: server.progress },
    });
  });
}

const stepper = (page: Page, label: string) =>
  page
    .getByRole("dialog", { name: /^Clip analysis$/ })
    .locator("li[data-state]")
    .filter({ hasText: label });

async function openPanel(page: Page) {
  await page
    .getByRole("button", { name: /^Clip analysis/, includeHidden: true })
    .click();
}

test.beforeAll(async () => {
  await installAdministrator();
});

test("le stepper suit l'analyse en direct, sans rechargement", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-stepper-live.mp4", "shared");
  const server: Server = {
    progress: { transcription: "running", indexing: "none" },
    run: {
      clipId,
      status: "active",
      steps: { preparation: "done", transcription: "running" },
    },
    seq: 0,
  };
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await simulate(page, clipId, server);
  await page.goto(`/clips/${clipId}`);
  await openPanel(page);
  await page.evaluate(() => {
    (window as unknown as { __kept: boolean }).__kept = true;
  });

  // running
  await expect(stepper(page, "Transcribed")).toHaveAttribute(
    "data-state",
    "running",
  );
  await expect(stepper(page, "Indexed")).toHaveAttribute(
    "data-state",
    "queued",
  );

  // transcription done, indexing running
  server.progress = { transcription: "done", indexing: "running" };
  server.run.steps = {
    preparation: "done",
    transcription: "done",
    indexing: "running",
  };
  await expect(stepper(page, "Transcribed")).toHaveAttribute(
    "data-state",
    "done",
  );
  await expect(stepper(page, "Indexed")).toHaveAttribute(
    "data-state",
    "running",
  );

  // failed
  server.progress = { transcription: "done", indexing: "failed" };
  server.run.steps = {
    preparation: "done",
    transcription: "done",
    indexing: "failed",
  };
  await expect(stepper(page, "Indexed")).toHaveAttribute(
    "data-state",
    "failed",
  );

  // the run ends done
  server.progress = { transcription: "done", indexing: "done" };
  server.run.status = "completed";
  server.run.steps = {
    preparation: "done",
    transcription: "done",
    indexing: "done",
  };
  await expect(stepper(page, "Indexed")).toHaveAttribute("data-state", "done");

  // the page was never reloaded
  expect(
    await page.evaluate(
      () => (window as unknown as { __kept?: boolean }).__kept,
    ),
  ).toBe(true);
  await context.close();
});

test("l'animation d'une étape en cours s'arrête sous mouvement réduit", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-stepper-motion.mp4", "shared");
  const server: Server = {
    progress: { transcription: "running", indexing: "none" },
    run: {
      clipId,
      status: "active",
      steps: { preparation: "done", transcription: "running" },
    },
    seq: 0,
  };
  const animationOf = async (reducedMotion: "reduce" | "no-preference") => {
    const context = await browser.newContext({
      storageState: await sessionFor(ADMIN),
      reducedMotion,
    });
    const page = await context.newPage();
    await simulate(page, clipId, server);
    await page.goto(`/clips/${clipId}`);
    await openPanel(page);
    const icon = stepper(page, "Transcribed").locator("svg");
    await expect(icon).toBeVisible();
    const name = await icon.evaluate(
      (el) => getComputedStyle(el).animationName,
    );
    await context.close();
    return name;
  };
  expect(await animationOf("no-preference")).toBe("spin");
  expect(await animationOf("reduce")).toBe("none");
});

const freshServer = (clipId: number): Server => ({
  progress: { transcription: "running", indexing: "none" },
  run: {
    clipId,
    status: "active",
    steps: { preparation: "done", transcription: "running" },
  },
  seq: 0,
});

test("le stepper suit le même snapshot que le texte, sans attendre le clip", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-stepper-instant.mp4", "shared");
  const server = freshServer(clipId);
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await simulate(page, clipId, server);
  await page.goto(`/clips/${clipId}`);
  await openPanel(page);
  await expect(stepper(page, "Transcribed")).toHaveAttribute(
    "data-state",
    "running",
  );
  // The clip now answers after 5 s, and still says "running": only the run can move the
  // stepper in time. The step list beside it moves with it, in the same frame.
  server.clipDelayMs = 5000;
  server.run.steps = {
    preparation: "done",
    transcription: "done",
    indexing: "running",
  };
  await expect(stepper(page, "Transcribed")).toHaveAttribute(
    "data-state",
    "done",
    { timeout: 2500 },
  );
  await expect(stepper(page, "Indexed")).toHaveAttribute(
    "data-state",
    "running",
    { timeout: 2500 },
  );
  await expect(
    page.getByRole("list", { name: /^Analysis steps$/ }).getByText("Running"),
  ).toBeVisible();
  await context.close();
});

test("une étape bloquée par une action humaine est annoncée comme telle", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-stepper-blocked.mp4", "shared");
  const server = freshServer(clipId);
  server.run.steps = { preparation: "done", transcription: "blocked" };
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await simulate(page, clipId, server);
  await page.goto(`/clips/${clipId}`);
  await openPanel(page);
  const blocked = stepper(page, "Transcribed");
  await expect(blocked).toHaveAttribute("data-state", "blocked");
  await expect(blocked).toContainText("Blocked");
  await expect(
    page
      .getByRole("list", { name: /^Analysis steps$/ })
      .getByText("Choose the audio track to transcribe."),
  ).toBeVisible();
  // The blocked step is not animated.
  expect(
    await blocked
      .locator("svg")
      .evaluate((el) => getComputedStyle(el).animationName),
  ).toBe("none");
  await context.close();
});

test("deux onglets suivent la même analyse et se rattrapent après une coupure", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-stepper-tabs.mp4", "shared");
  const server = freshServer(clipId);
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const first = await context.newPage();
  const second = await context.newPage();
  for (const page of [first, second]) {
    await simulate(page, clipId, server);
    await page.goto(`/clips/${clipId}`);
    await openPanel(page);
    await expect(stepper(page, "Transcribed")).toHaveAttribute(
      "data-state",
      "running",
    );
  }
  // The second tab loses the stream while the analysis moves on: nothing ghostly is left.
  server.down = true;
  server.progress = { transcription: "done", indexing: "running" };
  server.run.steps = {
    preparation: "done",
    transcription: "done",
    indexing: "running",
  };
  await expect(stepper(first, "Indexed")).toHaveCount(1);
  server.down = false;
  for (const page of [first, second]) {
    await expect(stepper(page, "Transcribed")).toHaveAttribute(
      "data-state",
      "done",
    );
    await expect(stepper(page, "Indexed")).toHaveAttribute(
      "data-state",
      "running",
    );
  }
  // A rerun starts again from the first step: the stepper goes back with it, no ghost "done".
  server.progress = { transcription: "running", indexing: "none" };
  server.run.steps = { preparation: "done", transcription: "running" };
  for (const page of [first, second]) {
    await expect(stepper(page, "Transcribed")).toHaveAttribute(
      "data-state",
      "running",
    );
    await expect(stepper(page, "Indexed")).toHaveAttribute(
      "data-state",
      "queued",
    );
  }
  await context.close();
});
