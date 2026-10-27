import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { transcribeAndDiarize, uploadClip } from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

test("corriger, résoudre un conflit et restaurer un segment dans son clip", async ({
  browser,
}) => {
  await installAdministrator();
  const clipId = await uploadClip("fixture-corrections.mp4", "shared");
  await transcribeAndDiarize(clipId);
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto(`/clips/${clipId}`);
  const segment = page
    .getByRole("list", { name: "Transcript segments" })
    .getByRole("listitem")
    .first();
  await segment.getByRole("button", { name: "Edit text" }).click();
  await segment.getByLabel("Segment text").fill("Draft correction");
  const me = await (await page.request.get("/api/auth/me")).json();
  const other = await page.request.put(`/api/clips/${clipId}/segments/0/text`, {
    headers: {
      Origin: process.env.SNITCH_TEST_URL!,
      "X-CSRF-Token": me.csrf_token,
    },
    data: { text: "Other correction", expected_version: 0 },
  });
  expect(other.status()).toBe(200);
  await segment.getByRole("button", { name: "Save text" }).click();
  await expect(segment.getByLabel("Segment text")).toHaveValue(
    "Draft correction",
  );
  await expect(
    segment.getByRole("group", { name: "Concurrent correction" }),
  ).toContainText("Other correction");
  await segment.getByRole("button", { name: "Keep my draft" }).click();
  await segment.getByRole("button", { name: "Save text" }).click();
  await expect(segment.getByText("Draft correction")).toBeVisible();
  await expect(segment.getByText("Bien joué !")).toBeVisible();
  await segment.getByRole("button", { name: "Edit speaker" }).click();
  await segment.getByLabel("Speaker").selectOption({ label: "SPEAKER_01" });
  await segment.getByRole("button", { name: "Save speaker" }).click();
  await expect(segment).toContainText("SPEAKER_01");
  await expect(segment).toContainText("Speaker corrected by administrator");
  await page.getByRole("tab", { name: "History" }).click();
  await page
    .getByRole("button", { name: /Show the history/ })
    .first()
    .click();
  await expect(page.getByText("Other correction").first()).toBeVisible();
  await page.getByRole("button", { name: "Restore original" }).click();
  await page.getByRole("tab", { name: "Transcript" }).click();
  await expect(segment.getByText("Bien joué !")).toBeVisible();
  await page.getByRole("tab", { name: "History" }).click();
  await page
    .getByRole("button", { name: /Show the history/ })
    .first()
    .click();
  await page.getByRole("button", { name: "Restore version 2" }).click();
  await page.getByRole("tab", { name: "Transcript" }).click();
  await expect(segment.getByText("Draft correction")).toBeVisible();
  await page.getByRole("button", { name: "Transcribe again" }).click();
  await expect(page.getByRole("alertdialog")).toContainText("correction");
  await page
    .getByRole("alertdialog")
    .getByRole("button", { name: "Cancel" })
    .click();
  await page.getByRole("button", { name: "Transcribe again" }).click();
  await page
    .getByRole("alertdialog")
    .getByRole("button", { name: "Confirm retranscription" })
    .click();
  await expect(page.getByText("Transcription scheduled.")).toBeVisible();
  const headers = {
    Origin: process.env.SNITCH_TEST_URL!,
    "X-CSRF-Token": me.csrf_token,
  };
  const workerResponse = await page.request.post("/api/admin/workers", {
    headers,
    data: { name: "fixture-retranscriber", max_concurrency: 1 },
  });
  expect(workerResponse.status()).toBe(201);
  const worker = await workerResponse.json();
  const authorization = { Authorization: `Bearer ${worker.token}` };
  expect(
    (
      await page.request.post("/api/worker/pulse", {
        headers: authorization,
        data: { capacity: 1, supported_steps: ["transcription.audio.*"] },
      })
    ).ok(),
  ).toBe(true);
  const lease = await (
    await page.request.post("/api/worker/jobs/claim", {
      headers: authorization,
    })
  ).json();
  const completed = await page.request.post(
    `/api/worker/jobs/${lease.job.id}/complete`,
    {
      headers: { ...authorization, "X-Worker-Lease-Token": lease.lease_token },
      data: {
        input_version: lease.job.input_version,
        result: {
          provider: "fixture",
          model: "deterministic-v1",
          version: "deterministic-v1",
          language: "fr",
          audio_stream_index: 2,
          segments: [
            {
              start: 1.25,
              end: 2.1,
              text: "New transcript",
              words: [{ word: "New transcript", start: 1.25, end: 2.1 }],
            },
          ],
        },
      },
    },
  );
  expect(completed.ok()).toBe(true);
  await page.reload();
  await page.getByRole("tab", { name: "History" }).click();
  await page.getByText("Obsolete corrections (1)").click();
  await expect(page.getByText("Draft correction").first()).toBeVisible();
  expect(
    (
      await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze()
    ).violations,
  ).toEqual([]);
  await context.close();
});
