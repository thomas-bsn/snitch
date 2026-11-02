import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Locator, type Page } from "@playwright/test";
import { uploadClip } from "./library-fixture";
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
  ).toHaveCount(2);

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

  // The completed-clip history is its own dialog.
  await page.getByRole("button", { name: "Clip history" }).click();
  await expect(
    page.getByRole("dialog", { name: "Completed clip history" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Close", exact: true }).click();

  // Thumbnails: more cards than the window shows, only the visible ones ask for media.
  const longQueue: number[] = [];
  for (let n = 0; n < 24; n++) {
    const id = await uploadClip(`synthetic-long-${n}.mp4`, "shared");
    longQueue.push(id);
    expect(
      (
        await page.request.post("/api/admin/jobs", {
          headers,
          data: {
            clip_id: id,
            step: "transcription.audio.1",
            input_version: "e2e-thumbs",
            priority: 1,
            max_attempts: 3,
          },
        })
      ).status(),
    ).toBe(201);
  }
  const lost = longQueue[longQueue.length - 1];
  const requested = new Set<number>();
  page.on("request", (r) => {
    const found = r.url().match(/\/api\/clips\/(\d+)\/media/);
    if (found) requested.add(Number(found[1]));
  });
  const consoleErrors: string[] = [];
  page.on("console", (m) => {
    if (m.type() === "error")
      consoleErrors.push(`${m.location().url} ${m.text()}`);
  });
  // The original of the last clip is gone: the clip itself says so.
  await page.route(`**/api/clips/${lost}`, async (route) => {
    const real = await route.fetch();
    await route.fulfill({
      response: real,
      json: { ...(await real.json()), status: "missing" },
    });
  });
  // Slow media keep the loading state observable.
  await page.route(/\/api\/clips\/\d+\/media/, async (route) => {
    await new Promise((done) => setTimeout(done, 1500));
    await route.continue();
  });
  await page.evaluate(() => localStorage.clear());
  await page.goto("/admin/queue");
  const thumbs = queue.locator("[data-thumbnail]");
  await expect(
    queue.locator("[data-column=transcription] details"),
  ).toHaveCount(26);
  const group = page.getByRole("radiogroup", { name: "Thumbnails" });
  await expect(group.getByRole("radio", { name: "Small" })).toBeChecked();
  await expect(thumbs.first()).toHaveAttribute("data-size", "small");
  await expect(queue.locator("[data-thumbnail=loading]").first()).toBeVisible();
  await expect.poll(() => requested.size).toBeGreaterThan(0);
  await page.waitForTimeout(2000);
  const mounted = await queue
    .locator("[data-thumbnail]:not([data-thumbnail=waiting])")
    .count();
  expect(mounted).toBeGreaterThan(0);
  expect(mounted).toBeLessThan(26);
  expect(requested.size).toBeLessThanOrEqual(mounted);
  expect(requested.size).toBeLessThan(26);
  // Scrolling the last card into view loads it, and a missing original shows its state.
  const lastCard = cardOf(lost);
  await lastCard.scrollIntoViewIfNeeded();
  await expect(lastCard.locator("[data-thumbnail=missing]")).toContainText(
    "Original not found",
  );
  expect(requested.has(lost)).toBe(false);
  expect(consoleErrors.filter((e) => e.includes(`/clips/${lost}`))).toEqual([]);

  // Three settings, remembered by the browser.
  await group.getByText("Large", { exact: true }).click();
  await expect(thumbs.first()).toHaveAttribute("data-size", "large");
  await page.reload();
  await expect(
    page.getByRole("radiogroup", { name: "Thumbnails" }).getByRole("radio", {
      name: "Large",
    }),
  ).toBeChecked();
  await expect(thumbs.first()).toHaveAttribute("data-size", "large");
  await page
    .getByRole("radiogroup", { name: "Thumbnails" })
    .getByText("None", { exact: true })
    .click();
  await expect(thumbs).toHaveCount(0);
  const before = requested.size;
  await page.waitForTimeout(500);
  expect(requested.size).toBe(before);

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
