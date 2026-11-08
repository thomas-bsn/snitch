import AxeBuilder from "@axe-core/playwright";
import { expect, test, type APIRequestContext } from "@playwright/test";
import { adminApi, uploadClip } from "./library-fixture";
import { ADMIN, installAdministrator, sessionFor } from "./shell-fixture";

// The multitrack chain end to end (#274, #344): nothing is simulated in the browser (no page.route). The
// server, its database, the audio plan, the diagnostics, the acoustic comparison, the publication and the voices
// are the real ones; the only double is the Worker, played through the real Worker API with scripted answers
// (the detector and the dedup adapter have their own acoustic suites). The synthetic probe describes two audio
// tracks: 1 "Micro" and 2 "Voix" (titles carry no meaning for the pipeline).

const DETECTOR = {
  detector_version: "silero-v6.2.1-7e30209a3e901f9842f81b225f3e93d8199902b1",
  detector_config_version: "channels-16k-512-hi050-lo020-min096-v1",
};

type Observation = { status: string; cause: string; intervals: unknown[] };
type Script = {
  diagnostics: Record<number, Observation>;
  words: Record<number, { start: number; end: number; text: string }[]>;
  /** Steps to leave queued (prefix match), so a test can act while a plan is half done. */
  hold?: string[];
};

const speech: Observation = {
  status: "speech",
  cause: "neural_speech",
  intervals: [{ start: 0.5, end: 4 }],
};
const silence: Observation = {
  status: "no_speech",
  cause: "no_speech_detected",
  intervals: [],
};

class Worker {
  private constructor(
    private api: APIRequestContext,
    private token: string,
    readonly id: string,
  ) {}

  static async create(): Promise<Worker> {
    const admin = await adminApi();
    const created = (await admin.post("/api/admin/workers", {
      name: `synthetic-activation-${Date.now()}`,
      max_concurrency: 2,
    })) as { id: string; token: string };
    const worker = new Worker(admin.api, created.token, created.id);
    await worker.pulse([]);
    // The pinned detector is installed by an explicit order, never by a claim.
    const order = (await admin.post(
      `/api/admin/workers/${created.id}/voice-model-orders`,
      {
        model_id: "silero-vad-v6.2.1",
        idempotency_key: `activation-${Date.now()}`,
      },
    )) as { order_id: string; model_id: string; revision: string };
    await worker.post(`/api/worker/voice-model-orders/${order.order_id}/ack`, {
      accepted: true,
    });
    await worker.post("/api/worker/voice-models/status", {
      order_id: order.order_id,
      model_id: order.model_id,
      revision: order.revision,
      state: "ready",
      cause_code: "",
      bytes_received: 2272526,
      bytes_total: 2272526,
      occupied_bytes: 2272526,
    });
    return worker;
  }

  private headers(lease?: string) {
    return {
      Authorization: `Bearer ${this.token}`,
      ...(lease ? { "X-Worker-Lease-Token": lease } : {}),
    };
  }

  async post(path: string, body?: unknown, lease?: string) {
    const response = await this.api.post(path, {
      data: body,
      headers: this.headers(lease),
    });
    if (!response.ok())
      throw new Error(`${path}: ${response.status()} ${await response.text()}`);
    return response.json().catch(() => null);
  }

  async pulse(steps: string[]) {
    await this.post("/api/worker/pulse", {
      capacity: 2,
      supported_steps: steps,
    });
  }

  /** Runs every task it can claim until the queue is empty; returns the steps it ran. */
  async drive(script: Script): Promise<string[]> {
    const ran: string[] = [];
    const caps = [
      "audio.diagnostic.*",
      "transcription.audio.*",
      "audio.dedup.*",
      "audio.multitrack.v1",
    ];
    for (let i = 0; i < 30; i++) {
      await this.pulse(caps);
      const claim = await this.post("/api/worker/jobs/claim");
      if (!claim?.job) return ran;
      const job = claim.job as {
        id: number;
        step: string;
        input_version: string;
      };
      const lease = claim.lease_token as string;
      ran.push(job.step);
      const track = Number(job.step.split(".").pop());
      let result: unknown;
      if (job.step.startsWith("audio.diagnostic.")) {
        result = { ...(script.diagnostics[track] ?? speech), ...DETECTOR };
      } else if (job.step.startsWith("transcription.audio.")) {
        const input = await (
          await this.api.get(`/api/worker/jobs/${job.id}/input`, {
            headers: this.headers(lease),
          })
        ).json();
        result = {
          provider: "fixture",
          model: "deterministic-v1",
          version: "1",
          audio_stream_index: track,
          time_base: "clip",
          audio_plan_id: input.audio_plan.plan_id,
          audio_plan_version: input.audio_plan.input_version,
          segments: (script.words[track] ?? []).map((s) => ({
            ...s,
            words: s.text.split(" ").map((word, n, all) => ({
              word,
              start: s.start + ((s.end - s.start) * n) / all.length,
              end: s.start + ((s.end - s.start) * (n + 1)) / all.length,
            })),
          })),
        };
      } else if (job.step.startsWith("audio.dedup.")) {
        result = { schema_version: 1, comparisons: [] };
      } else {
        throw new Error(`unexpected task ${job.step}`);
      }
      await this.post(
        `/api/worker/jobs/${job.id}/complete`,
        { input_version: job.input_version, result },
        lease,
      );
    }
    throw new Error("the Worker never ran out of tasks");
  }
}

const pill = (page: import("@playwright/test").Page) =>
  page.getByRole("button", {
    name: /^(Clip analysis|Analyse du clip)/,
    includeHidden: true,
  });

test.beforeAll(async () => {
  await installAdministrator();
});

test("un clip à deux pistes est examiné seul : seule la parole est transcrite, sans choix", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-activation-1.mp4", "shared");
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto(`/clips/${clipId}`);
  await pill(page).click();
  const panel = page.getByRole("region", {
    name: /^(Clip analysis|Analyse du clip)$/,
  });
  // The real engine offers Auto: every track is examined, no track has to be chosen.
  await panel.getByRole("button", { name: /^Launch analysis…$/ }).click();
  const dialog = page.getByRole("dialog", {
    name: /^(Launch analysis|Lancer l’analyse)$/,
  });
  await expect(dialog.getByLabel(/Auto — examine all tracks/)).toBeChecked();
  await dialog.getByRole("button", { name: "Preview the effects" }).click();
  const result = dialog.getByRole("region", {
    name: "What will be recalculated",
  });
  await expect(result).toContainText("Diagnostic of track 1");
  await expect(result).toContainText("Diagnostic of track 2");
  await expect(result).toContainText("No compatible Worker is connected");
  await expect(result).toContainText("No correction or validation is replaced");
  await result.getByRole("button", { name: "Start the analysis" }).click();
  await expect(dialog).toHaveCount(0);
  // No Worker yet: the wait is named, and no track choice is asked.
  await expect(panel).toContainText("Waiting for a compatible Worker");
  await expect(panel).not.toContainText("Choose the audio track");

  const worker = await Worker.create();
  const ran = await worker.drive({
    diagnostics: { 1: silence, 2: speech },
    words: { 2: [{ start: 1, end: 2.4, text: "bonjour tout le monde" }] },
  });
  expect(ran).toContain("audio.diagnostic.1");
  expect(ran).toContain("audio.diagnostic.2");
  expect(ran).toContain("transcription.audio.2");
  expect(ran).not.toContain("transcription.audio.1");

  // The panel follows the real run: the plan is published, the clip shows the transcript.
  await expect(panel).toContainText(
    "The transcription of the audio tracks is published.",
  );
  await page.reload();
  await expect(page.getByText("bonjour tout le monde").first()).toBeVisible();
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);

  // The voices follow the published plan, track by track (Voices tab).
  await page.getByRole("tab", { name: "Voices" }).click();
  const states = page.getByRole("list", { name: "Voice state per kept track" });
  await expect(states.getByRole("listitem")).toHaveCount(1);
  await expect(states).not.toContainText("Track 1");
  await expect(states).toContainText("Track 2");
  await context.close();
});

test("réanalyse réelle : pertes montrées, confirmation au départ, publication seule", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-activation-2.mp4", "shared");
  await (
    await adminApi()
  ).post(`/api/admin/clips/${clipId}/analysis`, {
    scope: "transcription",
  });
  const worker = await Worker.create();
  await worker.drive({
    diagnostics: { 1: speech, 2: speech },
    words: {
      1: [{ start: 1, end: 2, text: "premier texte" }],
      2: [{ start: 3, end: 4, text: "second texte" }],
    },
  });
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  const admin = await adminApi();
  const revision = async () => {
    const clip = await (await admin.api.get(`/api/clips/${clipId}`)).json();
    return clip.transcription.revision.id as string;
  };
  const correct = async (index: number, text: string, version: number) =>
    admin.put(`/api/clips/${clipId}/segments/${index}/text`, {
      text,
      expected_version: version,
      expected_revision_id: await revision(),
    });
  await correct(0, "texte corrigé à la main", 0);

  await page.goto(`/clips/${clipId}`);
  await pill(page).click();
  await page
    .getByRole("region", { name: /^(Clip analysis|Analyse du clip)$/ })
    .getByRole("button", { name: /^Launch analysis…$/ })
    .click();
  const dialog = page.getByRole("dialog", {
    name: /^(Launch analysis|Lancer l’analyse)$/,
  });
  // This journey deliberately replaces the transcript; missing mode preserves corrections.
  await dialog
    .getByLabel("Everything again, even where a valid result exists")
    .check();
  // The real tracks, with their real indices and the diagnoses of the previous plan.
  const tracks = dialog.getByRole("list", { name: "Tracks to analyze" });
  await expect(tracks.getByRole("listitem")).toHaveCount(2);
  await expect(tracks).toContainText("Track 1");
  await expect(tracks).toContainText("Track 2");
  await expect(tracks).toContainText("Speech detected");
  await dialog.getByRole("button", { name: "Preview the effects" }).click();
  const result = dialog.getByRole("region", {
    name: "What will be recalculated",
  });
  await expect(result).toContainText(
    "1 corrections to the transcript will become obsolete",
  );
  const start = result.getByRole("button", { name: "Start the analysis" });
  await expect(start).toBeDisabled();
  await result.getByLabel(/I confirm these consequences/).check();
  await start.click();
  await expect(dialog).toHaveCount(0);

  // The confirmation given at the start covers exactly what was previewed: the plan publishes by itself.
  await worker.drive({
    diagnostics: { 1: speech, 2: speech },
    words: {
      1: [{ start: 1, end: 2, text: "nouveau premier" }],
      2: [{ start: 3, end: 4, text: "nouveau second" }],
    },
  });
  await page.reload();
  await expect(page.getByText("nouveau premier").first()).toBeVisible();
  await expect(page.getByText("texte corrigé à la main")).toHaveCount(0);
  await context.close();
});

test("publication suspendue : une correction pendant l'analyse exige une nouvelle confirmation, vue dans le panneau", async ({
  browser,
}) => {
  const clipId = await uploadClip("fixture-activation-3.mp4", "shared");
  await (
    await adminApi()
  ).post(`/api/admin/clips/${clipId}/analysis`, {
    scope: "transcription",
  });
  const worker = await Worker.create();
  await worker.drive({
    diagnostics: { 1: speech, 2: silence },
    words: { 1: [{ start: 1, end: 2, text: "première version" }] },
  });
  const admin = await adminApi();
  const clipJson = async () =>
    (await (await admin.api.get(`/api/clips/${clipId}`)).json()) as {
      transcription: { revision: { id: string } };
    };
  await admin.put(`/api/clips/${clipId}/segments/0/text`, {
    text: "correction humaine",
    expected_version: 0,
    expected_revision_id: (await clipJson()).transcription.revision.id,
  });

  // Start the reanalysis by the API of the dialog (preview + confirmed start), then edit during the run.
  const selection = (await (
    await admin.api.get(`/api/admin/clips/${clipId}/audio-selection`)
  ).json()) as { version: string };
  const request = {
    version: selection.version,
    mode: "auto",
    track_indices: [],
    decisions: [],
    analysis: { scope: "transcription", mode: "full", steps: [] },
  };
  const preview = (await admin.post(
    `/api/admin/clips/${clipId}/audio-selection/preview`,
    request,
  )) as { preview_token: string };
  await admin.post(`/api/admin/clips/${clipId}/audio-selection`, {
    selection: request,
    preview_token: preview.preview_token,
    confirm_losses: true,
  });
  await admin.put(`/api/clips/${clipId}/segments/0/text`, {
    text: "correction pendant l'analyse",
    expected_version: 1,
    expected_revision_id: (await clipJson()).transcription.revision.id,
  });
  await worker.drive({
    diagnostics: { 1: speech, 2: silence },
    words: { 1: [{ start: 1, end: 2, text: "seconde version" }] },
  });

  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  await page.goto(`/clips/${clipId}`);
  await pill(page).click();
  const panel = page.getByRole("region", {
    name: /^(Clip analysis|Analyse du clip)$/,
  });
  // The current transcription and its correction stay visible; the hold is explained and actionable.
  await expect(panel).toContainText("corrections or voice validations changed");
  const hold = panel.getByRole("group", { name: "Publication on hold" });
  await expect(hold).toContainText("replaces 1 correction(s)");
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);
  await hold.getByRole("button", { name: "Replace and publish" }).click();
  await expect(panel).toContainText(
    "The transcription of the audio tracks is published.",
  );
  await page.reload();
  await expect(page.getByText("seconde version").first()).toBeVisible();
  await expect(page.getByText("correction pendant l'analyse")).toHaveCount(0);
  await context.close();
});
