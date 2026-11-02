import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Locator, type Page } from "@playwright/test";
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

  // The schedule form sits above the board and is visible without scrolling;
  // the queue bar is always on screen.
  await page.goto("/admin/queue");
  const scheduleForm = page.getByRole("form", { name: "Schedule task" });
  await expect(scheduleForm).toBeVisible();
  const formBox = (await scheduleForm.boundingBox())!;
  const gridBox = (await queue.boundingBox())!;
  expect(formBox.y + formBox.height).toBeLessThanOrEqual(gridBox.y);
  expect(formBox.y + formBox.height).toBeLessThanOrEqual(
    page.viewportSize()!.height,
  );
  const bar = page.getByRole("region", { name: "Queue and concurrency" });
  await expect(bar).toContainText("0 of 1");
  await expect(bar.getByText("Queue paused")).toHaveCount(0);

  // Concurrency controls, in the queue bar.
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
  await expect(bar).toContainText("Queue paused");
  await page.reload();
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
  await expect(bar).toContainText("0 of 3");
  await expect(bar.getByText("Queue paused")).toHaveCount(0);

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
  await expect(page.getByRole("radio")).toHaveCount(0);
  await page.getByLabel("Clip ID", { exact: true }).fill(String(clipA));
  await expect(
    page.getByRole("button", { name: "Schedule", exact: true }),
  ).toBeDisabled();
  await page
    .locator("summary")
    .filter({ hasText: "Advanced option: single task" })
    .click();
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
  await expect(bar).toContainText("Waiting clips");
  await expect(bar.getByTestId("waiting-count")).toHaveText("1");
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
