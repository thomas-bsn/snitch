import { request, type APIRequestContext } from "@playwright/test";
import { ADMIN, sessionFor } from "./shell-fixture";

// Synthetic library for the journeys of the new library and clip screens. Everything goes
// through the API: uploads, a simulated worker (transcription, diarization, fingerprints),
// a validated voice, and corrections. No real media: the videos are a few bytes.

const origin = () => process.env.SNITCH_TEST_URL!;

export async function adminApi(): Promise<{
  api: APIRequestContext;
  post: (path: string, data?: unknown) => Promise<unknown>;
  put: (path: string, data: unknown) => Promise<unknown>;
}> {
  const api = await request.newContext({
    baseURL: origin(),
    extraHTTPHeaders: { Origin: origin() },
    storageState: await sessionFor(ADMIN),
  });
  const me = await (await api.get("/api/auth/me")).json();
  const headers = { "X-CSRF-Token": me.csrf_token };
  const settle = async (response: Awaited<ReturnType<typeof api.post>>) => {
    if (!response.ok()) {
      throw new Error(`${response.url()} answered ${response.status()}`);
    }
    return response.json().catch(() => null);
  };
  return {
    api,
    post: async (path, data) =>
      settle(await api.post(path, { data: data ?? {}, headers })),
    put: async (path, data) => settle(await api.put(path, { data, headers })),
  };
}

/** Uploads a synthetic video and returns the id of the clip. */
export async function uploadClip(
  name: string,
  visibility: "shared" | "private",
  content = name,
): Promise<number> {
  const { api, post } = await adminApi();
  const body = Buffer.from(`synthetic-video-${content}`);
  const upload = (await post("/api/uploads", {
    filename: name,
    size_bytes: body.length,
    visibility,
  })) as { id: string };
  const me = await (await api.get("/api/auth/me")).json();
  const chunk = await api.put(`/api/uploads/${upload.id}/chunks`, {
    data: body,
    headers: {
      "Content-Type": "application/octet-stream",
      "Content-Range": `bytes 0-${body.length - 1}/${body.length}`,
      "X-CSRF-Token": me.csrf_token,
    },
  });
  if (!chunk.ok()) throw new Error(`chunk refused: ${chunk.status()}`);
  const done = (await post(`/api/uploads/${upload.id}/complete`)) as {
    clip_id: number;
  };
  await api.dispose();
  return done.clip_id;
}

type Segment = { start: number; end: number; text: string };

export const SEGMENTS: Segment[] = [
  { start: 1.25, end: 2.1, text: "Bien joué !" },
  { start: 2.5, end: 3.4, text: "Merci" },
  { start: 3.5, end: 4.1, text: "À bientôt" },
];

/**
 * Transcribes the clip with a simulated worker, then diarizes it into two voices
 * (SPEAKER_00 until 2.2 s, SPEAKER_01 after) and gives them fingerprints. Returns the ids
 * of the two local speakers.
 */
/** Amplitude peaks sent with the simulated results when a test wants a drawn waveform. */
const PEAKS = Array.from({ length: 400 }, (_, index) =>
  Math.round(40 + 180 * Math.abs(Math.sin(index / 9))),
);

export async function transcribeAndDiarize(
  clipId: number,
  options: {
    secondFingerprintEnd?: number;
    peaks?: boolean;
    /** The first segment runs on past 2.2 s with a word of SPEAKER_01: two parts. */
    sharedSegment?: boolean;
  } = {},
) {
  const { api, post, put } = await adminApi();
  await put(`/api/admin/clips/${clipId}/audio-track`, {
    audio_stream_index: 2,
  });
  await post(`/api/admin/clips/${clipId}/transcription`);
  const worker = (await post("/api/admin/workers", {
    name: "synthetic-library",
    max_concurrency: 2,
  })) as { token: string };
  const call = async (path: string, body?: unknown, lease?: string) => {
    const response = await api.post(path, {
      data: body,
      headers: {
        Authorization: `Bearer ${worker.token}`,
        ...(lease ? { "X-Worker-Lease-Token": lease } : {}),
      },
    });
    if (!response.ok()) throw new Error(`${path}: ${response.status()}`);
    return response.json().catch(() => null);
  };
  await call("/api/worker/pulse", {
    capacity: 2,
    supported_steps: ["transcription.audio.*", "voice.*"],
  });
  const transcription = await call("/api/worker/jobs/claim");
  await call(
    `/api/worker/jobs/${transcription.job.id}/complete`,
    {
      input_version: transcription.job.input_version,
      result: {
        provider: "fixture",
        model: "deterministic-v1",
        version: "deterministic-v1",
        language: "fr",
        audio_stream_index: 2,
        ...(options.peaks ? { peaks: PEAKS } : {}),
        segments: SEGMENTS.map((segment, index) =>
          options.sharedSegment && index === 0
            ? {
                start: segment.start,
                end: 2.45,
                text: `${segment.text} Salut`,
                words: [
                  {
                    word: segment.text,
                    start: segment.start,
                    end: segment.end,
                  },
                  { word: "Salut", start: 2.25, end: 2.45 },
                ],
              }
            : {
                ...segment,
                words: [
                  {
                    word: segment.text,
                    start: segment.start,
                    end: segment.end,
                  },
                ],
              },
        ),
      },
    },
    transcription.lease_token,
  );

  await put("/api/admin/voice-pipeline", {
    provider: "fake",
    model: "fixture-diarization",
    model_version: "fixture-v1",
    min_fingerprint_seconds: 1,
    min_quality: 0.5,
    fake: { speakers: 2, dimensions: 2 },
  });
  await post(`/api/admin/clips/${clipId}/voice`, { audio_stream_index: 2 });
  const diarization = await call("/api/worker/jobs/claim");
  const input = await (
    await api.get(`/api/worker/jobs/${diarization.job.id}/input`, {
      headers: {
        Authorization: `Bearer ${worker.token}`,
        "X-Worker-Lease-Token": diarization.lease_token,
      },
    })
  ).json();
  const config = input.voice.config;
  await call(
    `/api/worker/jobs/${diarization.job.id}/complete`,
    {
      input_version: diarization.job.input_version,
      result: {
        provider: config.provider,
        model: config.model,
        model_version: config.model_version,
        audio_stream_index: 2,
        ...(options.peaks ? { peaks: PEAKS } : {}),
        speakers: [
          { label: "SPEAKER_00", turns: [{ start: 0, end: 2.2 }] },
          { label: "SPEAKER_01", turns: [{ start: 2.2, end: 4.2 }] },
        ],
      },
    },
    diarization.lease_token,
  );
  const fingerprints = await call("/api/worker/jobs/claim");
  await call(
    `/api/worker/jobs/${fingerprints.job.id}/complete`,
    {
      input_version: fingerprints.job.input_version,
      result: {
        provider: config.provider,
        embedding_model: "fixture-embedding",
        embedding_model_version: "emb-v1",
        audio_stream_index: 2,
        diarization_job_id: diarization.job.id,
        dimensions: 2,
        fingerprints: [
          {
            speaker_label: "SPEAKER_00",
            scope: "speaker",
            start: 0,
            end: 2.2,
            quality: 0.9,
            embedding: [0.6, 0.8],
          },
          {
            speaker_label: "SPEAKER_01",
            scope: "speaker",
            start: 2.2,
            end: options.secondFingerprintEnd ?? 2.5,
            quality: 0.9,
            embedding: [0.8, 0.6],
          },
        ],
      },
    },
    fingerprints.lease_token,
  );
  const voice = await (await api.get(`/api/clips/${clipId}/voice`)).json();
  await api.dispose();
  return voice.speakers.map(
    (speaker: { id: number }) => speaker.id,
  ) as number[];
}

/** Gives a voice of the clip to a new Person (a validated, human identification). */
export async function nameVoice(speakerId: number, personName: string) {
  const { api, post } = await adminApi();
  await post(`/api/local-speakers/${speakerId}/validation`, {
    new_person_name: personName,
  });
  await api.dispose();
}

/** Corrects the text of a segment as the administrator. */
export async function correctText(
  clipId: number,
  index: number,
  text: string,
  reason: string,
) {
  const { api, put } = await adminApi();
  const clip = await (await api.get(`/api/clips/${clipId}`)).json();
  await put(`/api/clips/${clipId}/segments/${index}/text`, {
    text,
    reason,
    expected_version: 0,
    transcription_job_id: clip.transcription.transcription_job_id,
  });
  await api.dispose();
}

/** Designates a voice for a segment as the administrator. */
export async function correctSpeaker(
  clipId: number,
  index: number,
  speakerId: number,
) {
  const { api, put } = await adminApi();
  const clip = await (await api.get(`/api/clips/${clipId}`)).json();
  await put(`/api/clips/${clipId}/segments/${index}/speaker`, {
    local_speaker_id: speakerId,
    expected_version: 0,
    transcription_job_id: clip.transcription.transcription_job_id,
  });
  await api.dispose();
}

/** A simulated Worker claims the queued transcription of a clip and completes it. */
export async function completeQueuedTranscription(text = "Bonjour") {
  const { api, post } = await adminApi();
  const worker = (await post("/api/admin/workers", {
    name: `synthetic-analysis-${Date.now()}`,
    max_concurrency: 1,
  })) as { token: string };
  const call = async (path: string, body?: unknown, lease?: string) => {
    const response = await api.post(path, {
      data: body,
      headers: {
        Authorization: `Bearer ${worker.token}`,
        ...(lease ? { "X-Worker-Lease-Token": lease } : {}),
      },
    });
    if (!response.ok()) throw new Error(`${path}: ${response.status()}`);
    return response.json().catch(() => null);
  };
  await call("/api/worker/pulse", {
    capacity: 1,
    supported_steps: ["transcription.audio.*"],
  });
  const claimed = await call("/api/worker/jobs/claim");
  if (!claimed?.job) throw new Error("no transcription task to claim");
  await call(
    `/api/worker/jobs/${claimed.job.id}/complete`,
    {
      input_version: claimed.job.input_version,
      result: {
        provider: "fixture",
        model: "deterministic-v1",
        version: "deterministic-v1",
        language: "fr",
        audio_stream_index: Number(claimed.job.step.split(".").pop()),
        segments: [
          {
            start: 0.2,
            end: 0.9,
            text,
            words: [{ word: text, start: 0.2, end: 0.9 }],
          },
        ],
      },
    },
    claimed.lease_token,
  );
  await api.dispose();
}

/**
 * A simulated Worker claims the queued transcription of a clip and reports it failed with a
 * cause of the closed catalogue. `invalid_input` is final at once, so the step ends "failed".
 */
export async function failQueuedTranscription(code = "invalid_input") {
  const { api, post } = await adminApi();
  const worker = (await post("/api/admin/workers", {
    name: `synthetic-failure-${Date.now()}`,
    max_concurrency: 1,
  })) as { token: string };
  const call = async (path: string, body?: unknown, lease?: string) => {
    const response = await api.post(path, {
      data: body,
      headers: {
        Authorization: `Bearer ${worker.token}`,
        ...(lease ? { "X-Worker-Lease-Token": lease } : {}),
      },
    });
    if (!response.ok()) throw new Error(`${path}: ${response.status()}`);
    return response.json().catch(() => null);
  };
  await call("/api/worker/pulse", {
    capacity: 1,
    supported_steps: ["transcription.audio.*"],
  });
  // Tasks left by earlier journeys of the run fail too: the queue holds nothing else to claim.
  let failed = 0;
  for (;;) {
    const claimed = await call("/api/worker/jobs/claim");
    if (!claimed?.job) break;
    await call(
      `/api/worker/jobs/${claimed.job.id}/fail`,
      { error: code },
      claimed.lease_token,
    );
    failed++;
  }
  if (!failed) throw new Error("no transcription task to claim");
  await api.dispose();
}
