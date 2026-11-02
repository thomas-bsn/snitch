import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { uploadClip } from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

test("la file montre une carte par analyse, en direct", async ({ browser }) => {
  await installAdministrator();
  const first = await uploadClip("fixture-queue-a.mp4", "shared");
  const second = await uploadClip("fixture-queue-b.mp4", "shared");
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/queue");
  await expect(page.getByRole("status").first()).toHaveText("Live updates on");
  await expect(page.getByText("No analysis in progress.")).toBeVisible();

  const me = await (await page.request.get("/api/auth/me")).json();
  const headers = {
    Origin: process.env.SNITCH_TEST_URL!,
    "X-CSRF-Token": me.csrf_token,
  };
  // Starting analyses elsewhere (here the API, in practice a clip page) shows the cards at once.
  for (const id of [first, second]) {
    expect(
      (
        await page.request.post(`/api/admin/clips/${id}/analysis`, {
          headers,
          data: { scope: "full" },
        })
      ).ok(),
    ).toBe(true);
  }
  const cardA = page.locator(`[data-run-card="${first}"]`);
  const cardB = page.locator(`[data-run-card="${second}"]`);
  await expect(cardA).toBeVisible();
  await expect(cardB).toBeVisible();
  await expect(page.getByText("Analyses in progress · 2")).toBeVisible();
  // Branches and the diagnosis of each step are visible, a blocked step says why.
  await expect(cardA).toContainText("Search branch");
  await expect(cardA).toContainText("Voice branch");
  await expect(cardA).toContainText("Choose the audio track to transcribe.");

  // Choosing the track elsewhere moves the card on, without reloading the page.
  expect(
    (
      await page.request.put(`/api/admin/clips/${first}/audio-track`, {
        headers,
        data: { audio_stream_index: 2 },
      })
    ).ok(),
  ).toBe(true);
  await expect(cardA).toContainText("Waiting for a compatible Worker", {
    timeout: 15000,
  });
  await expect(cardB).toContainText("Choose the audio track to transcribe.");
  await expect(page.getByRole("button", { name: /refresh/i })).toHaveCount(0);
  await expect(page.getByText("Other tasks")).toHaveCount(0);

  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);

  // A cancelled analysis leaves the board.
  for (const id of [first, second]) {
    expect(
      (
        await page.request.post(`/api/admin/clips/${id}/analysis/cancel`, {
          headers,
        })
      ).ok(),
    ).toBe(true);
  }
  await expect(cardA).toHaveCount(0);
  await expect(page.getByText("No analysis in progress.")).toBeVisible();
  await context.close();
});

test("la planification avancée est en haut et l'historique garde les analyses", async ({
  browser,
}) => {
  const clip = await uploadClip("fixture-queue-c.mp4", "shared");
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto("/admin/queue");
  const form = page.locator("[data-advanced-schedule]");
  await expect(form).toBeVisible();
  const formBox = await form.boundingBox();
  const boardBox = await page
    .getByRole("heading", { name: "Analyses in progress" })
    .boundingBox();
  expect(formBox!.y).toBeLessThan(boardBox!.y);

  const me = await (await page.request.get("/api/auth/me")).json();
  const headers = {
    Origin: process.env.SNITCH_TEST_URL!,
    "X-CSRF-Token": me.csrf_token,
  };
  expect(
    (
      await page.request.post(`/api/admin/clips/${clip}/analysis`, {
        headers,
        data: { scope: "full" },
      })
    ).ok(),
  ).toBe(true);
  // An isolated step is refused while the analysis is active.
  const refused = await page.request.post(
    `/api/admin/clips/${clip}/transcription`,
    { headers, data: {} },
  );
  expect(refused.status()).toBe(409);
  expect((await refused.json()).error).toBe("analysis_active");

  expect(
    (
      await page.request.post(`/api/admin/clips/${clip}/analysis/cancel`, {
        headers,
      })
    ).ok(),
  ).toBe(true);
  await expect(page.locator(`[data-run-card="${clip}"]`)).toHaveCount(0);
  await page.getByRole("button", { name: "Clip history" }).click();
  const dialog = page.getByRole("dialog");
  await expect(
    dialog.locator("[data-history-run]").filter({ hasText: "fixture-queue-c" }),
  ).toHaveCount(1);
  await expect(dialog.getByRole("search")).toBeVisible();
  await dialog.getByLabel("Search a clip").fill("no-such-clip");
  await dialog.getByRole("button", { name: "Search" }).click();
  await expect(dialog.getByText("No finished analysis.")).toBeVisible();
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);
  await context.close();
});

test("les miniatures des cartes se chargent à l'apparition et se règlent en trois tailles", async ({
  browser,
}) => {
  // The administrator already exists when the walkthrough above ran first.
  await installAdministrator().catch(() => undefined);
  const clips: number[] = [];
  for (let n = 0; n < 12; n++)
    clips.push(await uploadClip(`synthetic-thumb-${n}.mp4`, "shared"));
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  const me = await (await page.request.get("/api/auth/me")).json();
  const headers = {
    Origin: process.env.SNITCH_TEST_URL!,
    "X-CSRF-Token": me.csrf_token,
  };
  for (const id of clips)
    expect(
      (
        await page.request.post(`/api/admin/clips/${id}/analysis`, {
          headers,
          data: { scope: "full" },
        })
      ).ok(),
    ).toBe(true);

  const mine = new Set(clips);
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
  // The original of the last card is gone: the clip itself says so.
  let lost = 0;
  await page.route(/\/api\/clips\/\d+$/, async (route) => {
    const real = await route.fetch();
    const id = Number(route.request().url().split("/").pop());
    if (id !== lost) return route.fulfill({ response: real });
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
  await page.goto("/admin/queue");
  await expect(page.getByRole("status").first()).toHaveText("Live updates on");
  const cards = page.locator("[data-run-card]");
  await expect.poll(() => cards.count()).toBeGreaterThanOrEqual(12);
  const order = await cards.evaluateAll((els) =>
    els.map((el) => Number(el.getAttribute("data-run-card"))),
  );
  lost = order.filter((id) => mine.has(id)).pop()!;
  const thumbs = page.locator("[data-run-card] [data-thumbnail]");

  const group = page.getByRole("radiogroup", { name: "Thumbnails" });
  await expect(group.getByRole("radio", { name: "Small" })).toBeChecked();
  await expect(thumbs.first()).toHaveAttribute("data-size", "small");
  // The schedule form sits above the board: bring the board into the window first.
  await cards.first().scrollIntoViewIfNeeded();
  await expect(
    page.locator("[data-run-card] [data-thumbnail=loading]").first(),
  ).toBeVisible();
  await expect.poll(() => requested.size).toBeGreaterThan(0);
  await page.waitForTimeout(2000);
  const everyThumbnail = await thumbs.count();
  const mounted = await page
    .locator("[data-run-card] [data-thumbnail]:not([data-thumbnail=waiting])")
    .count();
  expect(mounted).toBeGreaterThan(0);
  expect(mounted).toBeLessThan(everyThumbnail);
  expect(requested.size).toBeLessThan(everyThumbnail);
  expect(requested.has(lost)).toBe(false);

  // Scrolling the last card into view loads it; a missing original says so, quietly.
  const lastCard = page.locator(`[data-run-card="${lost}"]`);
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
    page
      .getByRole("radiogroup", { name: "Thumbnails" })
      .getByRole("radio", { name: "Large" }),
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

  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);

  for (const id of clips)
    await page.request.post(`/api/admin/clips/${id}/analysis/cancel`, {
      headers,
    });
  await context.close();
});
