import { expect, request, test } from "@playwright/test";
import { ADMIN, sessionFor } from "./shell-fixture";
import { prepareDiarizedClip } from "./voice-fixture";

// The diarization trial on the page of a clip: run the candidate model with a number of speakers,
// read the outcome beside the current result, delete it. A simulated Worker holds the model ready
// and answers the trial; the clip's own voice result must stay what it was.

const REPO = "pyannote/speaker-diarization-community-1";
const REVISION = "3533c8cf8e369892e6b79ff1bf80f7b0286a54ee";

test("un administrateur essaie un modèle sur un clip et compare au résultat courant", async ({
  browser,
}) => {
  const context = await browser.newContext();
  const page = await context.newPage();
  const clipId = await prepareDiarizedClip(page);

  const origin = process.env.SNITCH_TEST_URL!;
  const admin = await request.newContext({
    baseURL: origin,
    extraHTTPHeaders: { Origin: origin },
    storageState: await sessionFor(ADMIN),
  });
  const me = await (await admin.get("/api/auth/me")).json();
  const csrf = { "X-CSRF-Token": me.csrf_token };
  const created = await admin.post("/api/admin/workers", {
    headers: csrf,
    data: { name: "worker-essai", max_concurrency: 2 },
  });
  const { token } = await created.json();
  const bearer = { Authorization: `Bearer ${token}` };
  // The synthetic provider is active: the community model becomes the candidate, and the Worker holds it ready.
  expect(
    (
      await admin.put("/api/admin/voice-models/candidate", {
        headers: csrf,
        data: { model_id: "pyannote-community-1" },
      })
    ).ok(),
  ).toBe(true);
  expect(
    (
      await admin.post("/api/worker/voice-models/status", {
        headers: bearer,
        data: {
          model_id: "pyannote-community-1",
          state: "ready",
          bytes_received: 1,
          bytes_total: 1,
        },
      })
    ).status(),
  ).toBe(204);

  await page.reload();
  const section = page.getByRole("region", { name: "Diarization trial" });
  await expect(section).toBeVisible();
  await expect(
    section.getByRole("heading", { name: "Current result" }),
  ).toBeVisible();
  await expect(section.getByText("2 speakers").first()).toBeVisible();
  await expect(section.getByText("No trial yet on this clip.")).toBeVisible();

  await section.getByLabel("Number of speakers").fill("3");
  await section.getByRole("button", { name: "Run the trial" }).click();
  const trial = section.getByTestId("voice-trial");
  await expect(trial).toContainText("3 speakers imposed");
  await expect(trial).toContainText("waiting for a Worker");

  // The Worker takes the trial job, with the model and the option it asked for, and answers.
  await admin.post("/api/worker/pulse", {
    headers: bearer,
    data: { capacity: 2, supported_steps: ["voicetrial.*"] },
  });
  const claim = await (
    await admin.post("/api/worker/jobs/claim", { headers: bearer })
  ).json();
  expect(claim.job.step).toMatch(/^voicetrial\.diarization\.\d+$/);
  const lease = { ...bearer, "X-Worker-Lease-Token": claim.lease_token };
  const input = await (
    await admin.get(`/api/worker/jobs/${claim.job.id}/input`, {
      headers: lease,
    })
  ).json();
  expect(input.voice_trial.config.num_speakers).toBe(3);
  expect(input.voice_trial.config.model).toBe(REPO);
  const done = await admin.post(`/api/worker/jobs/${claim.job.id}/complete`, {
    headers: lease,
    data: {
      input_version: claim.job.input_version,
      result: {
        provider: "pyannote",
        model: REPO,
        model_version: REVISION,
        audio_stream_index: 2,
        speakers: [
          { label: "SPEAKER_00", turns: [{ start: 0, end: 1.5 }] },
          { label: "SPEAKER_01", turns: [{ start: 1.5, end: 3 }] },
          { label: "SPEAKER_02", turns: [{ start: 3, end: 4.2 }] },
        ],
      },
    },
  });
  expect(done.ok()).toBe(true);

  // The trial shows up beside the current result, which has not moved.
  await expect(trial).toContainText("done", { timeout: 15_000 });
  await expect(trial).toContainText("3 speakers");
  await expect(trial).toContainText("SPEAKER_02");
  await expect(section.getByText("2 speakers", { exact: true })).toBeVisible();
  const voice = await (await admin.get(`/api/clips/${clipId}/voice`)).json();
  expect(voice.speakers).toHaveLength(2);

  await trial.getByRole("button", { name: "Delete this trial" }).click();
  await expect(section.getByText("No trial yet on this clip.")).toBeVisible();
  await context.close();
  await admin.dispose();
});
