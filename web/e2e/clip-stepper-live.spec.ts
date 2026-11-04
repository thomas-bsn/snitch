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
      reason: "awaiting_previous_step",
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
  await expect(stepper(page, "Indexed")).toHaveAttribute("data-state", "none");

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
