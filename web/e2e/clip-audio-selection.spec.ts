import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { uploadClip } from "./library-fixture";
import {
  ADMIN,
  installAdministrator,
  sessionFor,
  setAccountLanguage,
} from "./shell-fixture";

// Contract fixtures: the clip is real, the audio-selection endpoints are simulated with the
// shapes of contracts/openapi.yaml (real indices 2 and 7, never positions 0 and 1). Wiring to
// the real diagnostics and analysis is checked by the final ticket of the spec.

type Diagnostic = {
  track_index: number;
  status: "speech" | "no_speech" | "ambiguous" | "unexamined" | "failed";
  freshness: "current" | "stale" | "missing";
  cause: string;
};

type Sim = {
  version: string;
  diagnostics: Diagnostic[];
  confirmation: boolean;
  workerCompatible: boolean;
  /** Error code the next start answers with (HTTP 409), once. */
  startConflict?: string;
  previews: Record<string, unknown>[];
  starts: Record<string, unknown>[];
};

const inventory = (extra: { index: number }[] = []) => [
  {
    index: 2,
    title: "Desktop",
    codec: "aac",
    language: "fr",
    channels: 2,
    channel_layout: "stereo",
    duration_seconds: 61.5,
    start_seconds: 0,
    evidence: [],
  },
  {
    index: 7,
    title: "Microphone",
    codec: "opus",
    language: "",
    channels: 1,
    channel_layout: "mono",
    duration_seconds: null,
    start_seconds: 1.25,
    evidence: [],
  },
  ...extra,
];

function sim(overrides: Partial<Sim> = {}): Sim {
  return {
    version: "v1",
    diagnostics: [
      { track_index: 2, status: "speech", freshness: "current", cause: "" },
      {
        track_index: 7,
        status: "ambiguous",
        freshness: "current",
        cause: "insufficient_evidence",
      },
    ],
    confirmation: false,
    workerCompatible: true,
    previews: [],
    starts: [],
    ...overrides,
  };
}

async function simulate(
  page: Page,
  state: Sim,
  tracks: { index: number }[] = inventory(),
) {
  await page.route("**/api/admin/clips/*/audio-selection**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === "GET")
      return route.fulfill({
        json: {
          enabled: true,
          version: state.version,
          state_version: "s1",
          inventory: tracks,
          diagnostics: state.diagnostics,
        },
      });
    const body = request.postDataJSON();
    if (url.pathname.endsWith("/preview")) {
      state.previews.push(body);
      const indices: number[] =
        body.mode === "auto"
          ? tracks
              .map((t) => t.index)
              .filter(
                (i) =>
                  !body.decisions.some(
                    (d: { track_index: number; action: string }) =>
                      d.track_index === i && d.action === "exclude",
                  ),
              )
          : body.track_indices;
      return route.fulfill({
        json: {
          input: { mode: body.mode },
          effects: {
            added_steps: indices.flatMap((i) => [
              `audio.diagnostic.${i}`,
              `transcription.audio.${i}`,
            ]),
            invalidated_steps: ["indexing"],
            track_indices: indices,
            worker_compatible: state.workerCompatible,
            stake_version: "stake-1",
            losses: {
              expected_revision_id: "legacy:1",
              previewed_at: new Date().toISOString(),
              correction_count: state.confirmation ? 3 : 0,
              validated_voice_count: state.confirmation ? 1 : 0,
              replaced_track_indices: state.confirmation ? indices : [],
              invalidated_steps: ["indexing"],
              confirmation_required: state.confirmation,
            },
          },
          preview_token: `token-${state.previews.length}`,
          expires_at: new Date(Date.now() + 900000).toISOString(),
        },
      });
    }
    state.starts.push(body);
    if (state.startConflict) {
      const error = state.startConflict;
      state.startConflict = undefined;
      if (error === "audio_selection_stale") state.version = "v2";
      return route.fulfill({ status: 409, json: { error } });
    }
    return route.fulfill({
      json: {
        worker_compatible: state.workerCompatible,
        state: { plan: {}, children: [] },
      },
    });
  });
}

const panelName = /^(Clip analysis|Analyse du clip)$/;

async function openReanalysis(page: Page, launch: RegExp) {
  await page
    .getByRole("button", {
      name: /^(Clip analysis|Analyse du clip)/,
      includeHidden: true,
    })
    .click();
  const panel = page.getByRole("region", { name: panelName });
  await panel.getByRole("button", { name: launch }).click();
  return page.getByRole("dialog", {
    name: /^(Launch analysis|Lancer l’analyse)$/,
  });
}

async function openClip(
  browser: import("@playwright/test").Browser,
  name: string,
) {
  const clipId = await uploadClip(name, "shared");
  const context = await browser.newContext({
    storageState: await sessionFor(ADMIN),
  });
  const page = await context.newPage();
  return { clipId, context, page };
}

test.beforeAll(async () => {
  await installAdministrator();
});

test.afterEach(async () => {
  await setAccountLanguage(ADMIN, "en");
});

test("réanalyse : Auto, une piste, plusieurs pistes et décision d'une piste ambiguë", async ({
  browser,
}) => {
  const { clipId, context, page } = await openClip(
    browser,
    "fixture-sel-1.mp4",
  );
  const state = sim();
  await simulate(page, state);
  await page.goto(`/clips/${clipId}`);
  const dialog = await openReanalysis(page, /^Launch analysis…$/);

  // Real indices and the metadata we have, no role assumed from the position.
  const choice = dialog.getByRole("group", { name: "Audio tracks" });
  await expect(choice.getByLabel(/Auto — examine all tracks/)).toBeChecked();
  const tracks = choice.getByRole("list", { name: "Tracks to analyze" });
  await expect(tracks.getByRole("listitem")).toHaveCount(2);
  await expect(tracks.getByRole("listitem").nth(0)).toContainText(
    "Track 2 — Desktop",
  );
  await expect(tracks.getByRole("listitem").nth(0)).toContainText(
    "Codec aac · 2 channels · Language fr · Duration 61.5 s",
  );
  await expect(tracks.getByRole("listitem").nth(0)).toContainText(
    "Speech detected · Up to date",
  );
  await expect(tracks.getByRole("listitem").nth(1)).toContainText(
    "Track 7 — Microphone",
  );
  await expect(tracks.getByRole("listitem").nth(1)).toContainText(
    "Starts at 1.25 s",
  );
  await expect(tracks.getByRole("listitem").nth(1)).toContainText(
    "Ambiguous · Up to date · Cause: insufficient_evidence",
  );

  // An ambiguous track needs an explicit decision before anything is previewed.
  const preview = dialog.getByRole("button", { name: "Preview the effects" });
  await expect(preview).toBeDisabled();
  await expect(dialog.locator("p[role=note]")).toContainText(
    "each ambiguous track",
  );
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);
  await dialog
    .getByRole("radiogroup", { name: "Ambiguous track 7: your decision" })
    .getByLabel("Exclude")
    .check();
  await preview.click();
  const result = dialog.getByRole("region", {
    name: "What will be recalculated",
  });
  await expect(result).toBeVisible();
  await expect(result).toContainText("Track 2");
  await expect(result).not.toContainText("Track 7");
  await expect(result.getByRole("list").first()).toContainText(
    "Diagnostic of track 2",
  );
  await expect(result).toContainText("Transcription of track 2");
  await expect(result).toContainText("No correction or validation is replaced");
  expect(state.previews[0]).toMatchObject({
    version: "v1",
    mode: "auto",
    track_indices: [],
    decisions: [{ track_index: 7, action: "exclude", forced: false }],
    analysis: { scope: "full", mode: "missing", steps: [] },
  });

  // Changing the choice drops the preview; one track, then two, each by its real index.
  await result.getByRole("button", { name: "Change the choice" }).click();
  await expect(result).toHaveCount(0);
  await dialog.getByLabel(/Choose the tracks myself/).check();
  await expect(preview).toBeDisabled();
  await expect(dialog.locator("p[role=note]")).toContainText(
    "at least one track",
  );
  await dialog.getByLabel(/Track 7 — Microphone/).check();
  await preview.click();
  await expect(result).toContainText("Transcription of track 7");
  expect(state.previews[1]).toMatchObject({
    mode: "manual",
    track_indices: [7],
    decisions: [{ track_index: 7, action: "include", forced: false }],
  });
  await result.getByRole("button", { name: "Change the choice" }).click();
  await dialog.getByLabel(/Track 2 — Desktop/).check();
  await preview.click();
  expect(state.previews[2]).toMatchObject({ track_indices: [2, 7] });
  await result.getByRole("button", { name: "Start the analysis" }).click();
  await expect(dialog).toHaveCount(0);
  expect(state.starts).toHaveLength(1);
  expect(state.starts[0]).toMatchObject({
    preview_token: "token-3",
    confirm_losses: false,
    selection: { mode: "manual", track_indices: [2, 7] },
  });
  await context.close();
});

test("réanalyse : forcer une piste sans parole garde l'intention explicite", async ({
  browser,
}) => {
  const { clipId, context, page } = await openClip(
    browser,
    "fixture-sel-2.mp4",
  );
  const state = sim({
    diagnostics: [
      {
        track_index: 2,
        status: "no_speech",
        freshness: "current",
        cause: "silence",
      },
      { track_index: 7, status: "speech", freshness: "stale", cause: "" },
    ],
  });
  await simulate(page, state);
  await page.goto(`/clips/${clipId}`);
  const dialog = await openReanalysis(page, /^Launch analysis…$/);
  await expect(dialog.getByText("Out of date")).toBeVisible();
  await dialog.getByLabel(/Choose the tracks myself/).check();
  await dialog.getByLabel(/Track 2 — Desktop/).check();
  const preview = dialog.getByRole("button", { name: "Preview the effects" });
  await expect(preview).toBeDisabled();
  await expect(dialog.locator("p[role=note]")).toContainText(
    "Track 2 shows no speech",
  );
  await dialog.getByLabel("Use this track despite the diagnostic").check();
  await dialog
    .getByLabel("Why (kept with the analysis)")
    .fill("audible game sound");
  await preview.click();
  expect(state.previews[0]).toMatchObject({
    track_indices: [2],
    decisions: [
      {
        track_index: 2,
        action: "include",
        forced: true,
        reason: "audible game sound",
      },
    ],
  });
  await context.close();
});

test("réanalyse : pertes à confirmer, conflit concurrent et Worker absent", async ({
  browser,
}) => {
  const { clipId, context, page } = await openClip(
    browser,
    "fixture-sel-3.mp4",
  );
  const state = sim({
    diagnostics: [],
    confirmation: true,
    workerCompatible: false,
  });
  await simulate(page, state);
  await page.goto(`/clips/${clipId}`);
  const dialog = await openReanalysis(page, /^Launch analysis…$/);
  const preview = dialog.getByRole("button", { name: "Preview the effects" });
  await preview.click();
  const result = dialog.getByRole("region", {
    name: "What will be recalculated",
  });
  await expect(result).toContainText(
    "3 corrections to the transcript will become obsolete",
  );
  await expect(result).toContainText("1 validated voices will be detached");
  await expect(result).toContainText("No compatible Worker is connected");
  const start = result.getByRole("button", { name: "Start the analysis" });
  await expect(start).toBeDisabled();
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);

  // The confirmation covers this preview only: going back forgets it.
  await result.getByLabel(/I confirm these consequences/).check();
  await expect(start).toBeEnabled();
  await result.getByRole("button", { name: "Change the choice" }).click();
  await dialog.getByLabel(/Choose the tracks myself/).check();
  await dialog.getByLabel(/Track 2 — Desktop/).check();
  await preview.click();
  await expect(
    result.getByLabel(/I confirm these consequences/),
  ).not.toBeChecked();
  await expect(start).toBeDisabled();

  // Someone else changed the human work meanwhile: the preview is refused and dropped.
  await result.getByLabel(/I confirm these consequences/).check();
  state.startConflict = "audio_preview_changed";
  await start.click();
  await expect(dialog.getByRole("alert")).toContainText(
    "changed since the preview",
  );
  await expect(result).toHaveCount(0);
  expect(state.starts[0]).toMatchObject({ confirm_losses: true });

  // A stale choice reloads the tracks and asks for a new look.
  await preview.click();
  await result.getByLabel(/I confirm these consequences/).check();
  state.startConflict = "audio_selection_stale";
  await start.click();
  await expect(dialog.getByRole("alert")).toContainText("Review it again");
  await preview.click();
  expect(state.previews.at(-1)).toMatchObject({ version: "v2" });
  await context.close();
});

test("préparation : choisir les pistes quand l'analyse l'attend, en français", async ({
  browser,
}) => {
  await setAccountLanguage(ADMIN, "fr");
  const { clipId, context, page } = await openClip(
    browser,
    "fixture-sel-4.mp4",
  );
  const state = sim();
  await simulate(page, state);
  const now = new Date().toISOString();
  const names = [
    "preparation",
    "transcription",
    "indexing",
    "diarization",
    "fingerprints",
    "matching",
  ];
  const run = {
    id: 1,
    clip_id: clipId,
    scope: "full",
    status: "active",
    started_by: null,
    created_at: now,
    updated_at: now,
    finished_at: null,
    mode: "missing",
    forced_steps: [],
    confirmed_replace_corrections: false,
    confirmed_replace_validations: false,
    confirmed_stake_at: null,
    pending_voices: 0,
    steps: names.map((step) => ({
      step,
      status: step === "preparation" ? "blocked" : "waiting",
      reason:
        step === "preparation"
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
  await page.route("**/api/admin/clips/*/analysis/events*", (route) =>
    route.abort(),
  );
  await page.route(/\/api\/admin\/clips\/\d+\/analysis$/, (route) =>
    route.fulfill({ json: { run } }),
  );
  await page.goto(`/clips/${clipId}`);
  await page
    .getByRole("button", { name: /^Analyse du clip/, includeHidden: true })
    .click();
  const panel = page.getByRole("region", { name: "Analyse du clip" });
  const choice = panel.getByRole("group", { name: "Pistes audio" });
  await expect(
    choice.getByLabel(/Auto — examiner toutes les pistes/),
  ).toBeChecked();
  await expect(choice).toContainText("Piste 7 — Microphone");
  await expect(choice).toContainText("Ambiguë · À jour");
  await choice.getByLabel(/Choisir les pistes moi-même/).check();
  await choice.getByLabel(/Piste 7 — Microphone/).check();
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  expect(violations).toEqual([]);
  await panel.getByRole("button", { name: "Aperçu des effets" }).click();
  const result = panel.getByRole("region", { name: "Ce qui sera recalculé" });
  await expect(result).toContainText("Transcription de la piste 7");
  await expect(result).toContainText("Aucune correction ni validation");
  await result.getByRole("button", { name: "Lancer l’analyse" }).click();
  await expect(result).toHaveCount(0);
  expect(state.starts[0]).toMatchObject({
    selection: { mode: "manual", track_indices: [7] },
  });
  await context.close();
});
