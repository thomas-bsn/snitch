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
