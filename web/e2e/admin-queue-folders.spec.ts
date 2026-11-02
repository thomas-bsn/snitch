import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Locator, type Page } from "@playwright/test";
import { transcribeAndDiarize, uploadClip } from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];

/** The queue shows one card per clip and step; its actions live inside the collapsed card. */
async function openCard(card: Locator) {
  if (!(await card.getByRole("button", { name: "History" }).isVisible())) {
    await card.locator("summary").click();
  }
}

async function upload(page: Page, name: string, seed: number) {
  await page.goto("/import");
  await page.getByLabel("choose files").setInputFiles({
    name,
    mimeType: "video/mp4",
    buffer: Buffer.alloc(20000 + seed, seed),
  });
  await expect(
    page.getByRole("listitem").filter({ hasText: name }),
  ).toContainText("Added to the library", { timeout: 30000 });
}

test("l'administrateur pilote la file et voit les dossiers disponibles", async ({
  browser,
}) => {
  await installAdministrator();
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  const queue = page.locator("[data-workflow-grid]");
  const cardOf = (clipId: number) =>
    queue.locator("details").filter({ hasText: `#${clipId}` });

  // Concurrency controls, collapsed behind a disclosure panel.
  await page.goto("/admin/queue");
  await page
    .locator("summary")
    .filter({ hasText: "Concurrency settings" })
    .click();
  await page.getByLabel("Global concurrency").fill("3");
  await page.getByLabel("Pause new acquisitions").check();
  const firstSave = page.waitForResponse(
    (r) =>
      r.url().endsWith("/api/admin/job-controls") &&
      r.request().method() === "PUT" &&
      r.ok(),
  );
  await page.getByRole("button", { name: "Save controls" }).click();
  await firstSave;
  await page.reload();
  await page
    .locator("summary")
    .filter({ hasText: "Concurrency settings" })
    .click();
  await expect(page.getByLabel("Global concurrency")).toHaveValue("3");
  await expect(page.getByLabel("Pause new acquisitions")).toBeChecked();
  await page.getByLabel("Pause new acquisitions").uncheck();
  const saved = page.waitForResponse(
    (r) =>
      r.url().endsWith("/api/admin/job-controls") &&
      r.request().method() === "PUT" &&
      r.ok(),
  );
  await page.getByRole("button", { name: "Save controls" }).click();
  await saved;

  // A worker that can transcribe makes the transcription step available in the form.
  await upload(page, "synthetic-a.mp4", 1);
  await upload(page, "synthetic-b.mp4", 2);
  await upload(page, "synthetic-c.mp4", 3);
  const me = await (await page.request.get("/api/auth/me")).json();
  const headers = {
    Origin: process.env.SNITCH_TEST_URL!,
    "X-CSRF-Token": me.csrf_token,
  };
  const clips = await (await page.request.get("/api/clips")).json();
  const clipA = clips.items.find(
    (clip: { original_name: string }) =>
      clip.original_name === "synthetic-a.mp4",
  ).id;
  const clipC = clips.items.find(
    (clip: { original_name: string }) =>
      clip.original_name === "synthetic-c.mp4",
  ).id;
  const clipB = clips.items.find(
    (clip: { original_name: string }) =>
      clip.original_name === "synthetic-b.mp4",
  ).id;
  const workerResponse = await page.request.post("/api/admin/workers", {
    headers,
    data: { name: "fixture-worker", max_concurrency: 1 },
  });
  expect(workerResponse.status()).toBe(201);
  const worker = await workerResponse.json();
  const authorization = { Authorization: `Bearer ${worker.token}` };
  expect(
    (
      await page.request.post("/api/worker/pulse", {
        headers: authorization,
        data: { capacity: 1, supported_steps: ["transcription.*"] },
      })
    ).ok(),
  ).toBe(true);

  // Scheduling from the queue page: the clip and the step, nothing else.
  expect(
    (
      await page.request.put(`/api/admin/clips/${clipA}/audio-track`, {
        headers,
        data: { audio_stream_index: 2 },
      })
    ).ok(),
  ).toBe(true);
  await page.goto("/admin/queue");
  await page.locator("summary").filter({ hasText: "Schedule task" }).click();
  await page.getByLabel("Clip ID", { exact: true }).fill(String(clipA));
  await page
    .locator("label")
    .filter({ hasText: "Step 0 · Transcription" })
    .click();
  await expect(
    page.getByRole("radio", { name: /Step 0 · Transcription/ }),
  ).toBeChecked();
  const scheduled = page.waitForResponse(
    (r) =>
      r.url().endsWith(`/api/admin/clips/${clipA}/transcription`) &&
      r.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Schedule", exact: true }).click();
  const response = await scheduled;
  expect([200, 201], await response.text()).toContain(response.status());
  const cardA = cardOf(clipA);
  await expect(cardA).toContainText("Queued");
  await openCard(cardA);
  await cardA.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(cardA).toContainText("Cancelled");
  await expect(cardA).toContainText("Schedule the clip again");

  // A task that runs out of attempts is retried, read in its history, then cancelled.
  const jobResponse = await page.request.post("/api/admin/jobs", {
    headers,
    data: {
      clip_id: clipB,
      step: "transcription.audio.9",
      input_version: "e2e-v1",
      priority: 1,
      max_attempts: 1,
    },
  });
  expect(jobResponse.status()).toBe(201);
  const job = await jobResponse.json();
  const claim = await (
    await page.request.post("/api/worker/jobs/claim", {
      headers: authorization,
    })
  ).json();
  expect(claim.job.id).toBe(job.id);
  expect(
    (
      await page.request.post(`/api/worker/jobs/${claim.job.id}/fail`, {
        headers: {
          ...authorization,
          "X-Worker-Lease-Token": claim.lease_token,
        },
        data: { error: "invalid_input" },
      })
    ).ok(),
  ).toBe(true);

  // The board refreshes by itself: no reload needed to see the exhausted task.
  const cardB = cardOf(clipB);
  await expect(cardB).toContainText("Exhausted", { timeout: 15000 });
  await openCard(cardB);
  // The dead task stays in its column with its persisted cause and advice.
  await expect(cardB).toContainText("Invalid media");
  await expect(cardB).toContainText("invalid_input");
  await expect(cardB).toContainText("Check the clip media");
  await cardB.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(cardB).toContainText("Queued");
  await openCard(cardB);
  await cardB.getByRole("button", { name: "History", exact: true }).click();
  await expect(cardB).toContainText("Relaunched by administrator");
  await cardB.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(cardB).toContainText("Cancelled");
  // Semantic indexing is not configured here: the Search step is not applicable.
  await openCard(cardB);
  await expect(cardB).toContainText("Search · Not applicable");

  // Without any Worker seen recently, a pending task waits for a Worker and an
  // unknown step stays attached to the card with an alert.
  expect(
    (
      await page.request.delete(`/api/admin/workers/${worker.id}`, { headers })
    ).ok(),
  ).toBe(true);
  for (const step of ["transcription.audio.1", "mystery.step.1"]) {
    expect(
      (
        await page.request.post("/api/admin/jobs", {
          headers,
          data: { clip_id: clipC, step, input_version: "e2e-v1" },
        })
      ).status(),
    ).toBe(201);
  }
  const cardC = cardOf(clipC);
  await expect(cardC).toContainText("Waiting for a Worker", {
    timeout: 15000,
  });
  await openCard(cardC);
  await expect(cardC).toContainText("install or start a Worker");
  await expect(cardC.locator("[data-unknown-step]")).toContainText(
    "mystery.step.1",
  );

  // One card per clip in the columns of the cards contract; no "other tasks"
  // column and no pagination of the active cards.
  await expect(queue.locator("[data-column]")).toHaveCount(5);
  await expect(
    queue.locator("[data-column=search]").getByRole("heading"),
  ).toContainText("Search · 0");
  await expect(queue.getByText("Other tasks")).toHaveCount(0);
  await expect(
    page.getByRole("navigation", { name: /pagination/i }),
  ).toHaveCount(0);
  await expect(
    queue.locator("[data-column=transcription] details"),
  ).toHaveCount(3);

  // The polling stops while the tab is hidden and resumes when it is back.
  let polls = 0;
  page.on("request", (r) => {
    if (r.url().endsWith("/api/admin/task-cards")) polls++;
  });
  const setVisibility = (state: "hidden" | "visible") =>
    page.evaluate((value) => {
      Object.defineProperty(document, "visibilityState", {
        configurable: true,
        get: () => value,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    }, state);
  await setVisibility("hidden");
  const hiddenFrom = polls;
  await page.waitForTimeout(6500);
  expect(polls).toBe(hiddenFrom);
  await setVisibility("visible");
  await expect.poll(() => polls).toBeGreaterThan(hiddenFrom);

  // Completed holds the three most recently finished distinct clips; a clip with
  // voices left to settle carries the badge. The oldest of four stays out.
  const finished: number[] = [];
  for (const name of ["done-1.mp4", "done-2.mp4", "done-3.mp4", "done-4.mp4"]) {
    const id = await uploadClip(name, "shared");
    await transcribeAndDiarize(id);
    finished.push(id);
  }
  await page.goto("/admin/queue");
  const completed = queue.locator("[data-column=completed]");
  await expect(completed.getByRole("heading")).toContainText("Completed · 3");
  await expect(completed.locator("details")).toHaveCount(3);
  await expect(completed.locator(`#clip-${finished[0]}`)).toHaveCount(0);
  await expect(completed.locator(`#clip-${finished[3]}`)).toHaveCount(1);
  await expect(completed.locator("[data-voices-to-review]")).toHaveCount(3);
  await expect(completed.locator("[data-voices-to-review]").first()).toHaveText(
    "Voices to review",
  );
  expect(
    (await new AxeBuilder({ page }).withTags(TAGS).analyze()).violations,
  ).toEqual([]);

  // The clip history replaces the old list of ten completed clips: events of
  // each step per clip, searchable and paginated.
  await page.getByRole("button", { name: "Clip history" }).click();
  const history = page.getByRole("dialog", { name: "Clip history" });
  await expect(history).toBeVisible();
  await history.getByLabel("Search a clip").fill("synthetic-b");
  await history.getByRole("button", { name: "Search", exact: true }).click();
  const entry = history.locator(`[data-history-clip="${clipB}"]`);
  await expect(entry).toContainText("transcription.audio.9");
  await expect(entry).toContainText("Relaunched by administrator");
  await expect(entry).toContainText("cause invalid_input");
  await expect(history.locator("[data-history-clip]")).toHaveCount(1);
  await history.getByLabel("Search a clip").fill("no-such-clip-anywhere");
  await history.getByRole("button", { name: "Search", exact: true }).click();
  await expect(history).toContainText("No clip found.");
  await history.getByLabel("Search a clip").fill("");
  await history.getByRole("button", { name: "Search", exact: true }).click();
  await expect(history.locator("[data-history-clip]")).toHaveCount(6);
  await expect(history).toContainText("6 clips");
  expect(
    (await new AxeBuilder({ page }).withTags(TAGS).analyze()).violations,
  ).toEqual([]);
  await page.getByRole("button", { name: "Close", exact: true }).click();

  expect(
    (await new AxeBuilder({ page }).withTags(TAGS).analyze()).violations,
  ).toEqual([]);

  await page.goto("/admin/folders");
  await expect(
    page.getByLabel("Root folder accessible to Snitch"),
  ).toBeVisible();
  await expect(page.getByText("No watched folders configured.")).toBeVisible();
  expect(
    (await new AxeBuilder({ page }).withTags(TAGS).analyze()).violations,
  ).toEqual([]);
  await context.close();
});
