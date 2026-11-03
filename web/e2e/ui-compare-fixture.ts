import { adminApi, uploadClip } from "./library-fixture";
import { createPersons } from "./people-fixture";

// Jeu de données synthétique de la comparaison visuelle (ticket #251). Il reproduit celui des
// prototypes (web/src/app/library/prototype-fiche/data.ts et people/prototype-people/data.ts
// de la branche prototype/fiche-video) : le clip « Réunion de préparation — tournage S12 »
// avec 5 voix, 10 segments et une étape d'empreintes en échec, l'annuaire, les clips à traiter.
// Aucune donnée réelle : les vidéos font quelques octets.

export const MEETING = "Réunion de préparation — tournage S12";

export const PEOPLE = [
  "Marc Lefèvre",
  "Julie Martin",
  "Claire Dubois",
  "Thomas Roux",
  "Hugo Bernard",
  "Inès Garnier",
];

/** Les autres clips du prototype (liste « À traiter »), 2 voix inconnues chacun. */
export const OTHER_CLIPS = [
  "Rushes parking — prise 3",
  "Interview Claire — making-of",
  "Brief équipe lumière",
  "Visio production — budget",
  "Repérages Studio B",
];

// 10 segments, 5 voix (SPEAKER_00 à SPEAKER_04), comme le prototype.
const SEGMENTS: [number, number, number, string][] = [
  [0, 6.2, 0, "Bon, on commence. Tout le monde m'entend bien ?"],
  [6.2, 9.8, 1, "Oui, c'est bon de mon côté."],
  [
    9.8,
    18.4,
    0,
    "Parfait. Premier point : le planning du tournage de la semaine prochaine.",
  ],
  [18.4, 21.0, 4, "Euh… pardon, mauvais micro."],
  [
    21.0,
    33.5,
    2,
    "Pour jeudi, il faudrait décaler la scène du parking, la lumière n'ira pas.",
  ],
  [
    33.5,
    41.2,
    1,
    "On peut la passer vendredi matin, j'ai vérifié les autorisations.",
  ],
  [41.2, 47.9, 3, "Je peux venir avec le deuxième boîtier si besoin."],
  [
    47.9,
    58.0,
    0,
    "Très bien, on note vendredi. Julie, tu envoies le récap ce soir ?",
  ],
  [58.0, 61.3, 1, "Je m'en occupe."],
  [
    61.3,
    74.8,
    2,
    "Dernière chose : la carte SD du plan large était pleine, on a perdu les dix dernières minutes.",
  ],
];

const PEAKS = Array.from({ length: 400 }, (_, index) =>
  Math.round(40 + 180 * Math.abs(Math.sin(index / 9))),
);

async function worker() {
  const { api, post, put } = await adminApi();
  const registered = (await post("/api/admin/workers", {
    name: `ui-compare-${Date.now()}`,
    max_concurrency: 2,
  })) as { token: string };
  const call = async (path: string, body?: unknown, lease?: string) => {
    const response = await api.post(path, {
      data: body,
      headers: {
        Authorization: `Bearer ${registered.token}`,
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
  return { api, post, put, call, token: registered.token };
}

/**
 * Transcrit (segments donnés), détecte les voix, puis laisse l'étape d'empreintes en échec
 * (cause du catalogue fermé : invalid_input) si `failFingerprints`.
 */
async function analyse(
  clipId: number,
  segments: [number, number, number, string][],
  speakers: number,
  failFingerprints: boolean,
) {
  const { api, post, put, call, token } = await worker();
  await put(`/api/admin/clips/${clipId}/audio-track`, {
    audio_stream_index: 2,
  });
  await post(`/api/admin/clips/${clipId}/transcription`);
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
        peaks: PEAKS,
        segments: segments.map(([start, end, , text]) => ({
          start,
          end,
          text,
          words: [{ word: text, start, end }],
        })),
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
    fake: { speakers, dimensions: 2 },
  });
  await post(`/api/admin/clips/${clipId}/voice`, { audio_stream_index: 2 });
  const diarization = await call("/api/worker/jobs/claim");
  const input = await (
    await api.get(`/api/worker/jobs/${diarization.job.id}/input`, {
      headers: {
        Authorization: `Bearer ${token}`,
        "X-Worker-Lease-Token": diarization.lease_token,
      },
    })
  ).json();
  const config = input.voice.config;
  const turns = new Map<number, { start: number; end: number }[]>();
  for (const [start, end, speaker] of segments) {
    turns.set(speaker, [...(turns.get(speaker) ?? []), { start, end }]);
  }
  await call(
    `/api/worker/jobs/${diarization.job.id}/complete`,
    {
      input_version: diarization.job.input_version,
      result: {
        provider: config.provider,
        model: config.model,
        model_version: config.model_version,
        audio_stream_index: 2,
        peaks: PEAKS,
        speakers: [...turns.entries()].map(([speaker, list]) => ({
          label: `SPEAKER_${String(speaker).padStart(2, "0")}`,
          turns: list,
        })),
      },
    },
    diarization.lease_token,
  );
  const fingerprints = await call("/api/worker/jobs/claim");
  if (failFingerprints) {
    await call(
      `/api/worker/jobs/${fingerprints.job.id}/fail`,
      { error: "invalid_input" },
      fingerprints.lease_token,
    );
  } else {
    const embedding = (speaker: number) => [
      Math.cos(speaker + 1),
      Math.sin(speaker + 1),
    ];
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
          fingerprints: [...turns.entries()].map(([speaker, list]) => ({
            speaker_label: `SPEAKER_${String(speaker).padStart(2, "0")}`,
            scope: "speaker",
            start: list[0].start,
            end: list[0].end,
            quality: 0.9,
            embedding: embedding(speaker),
          })),
        },
      },
      fingerprints.lease_token,
    );
  }
  await api.dispose();
}

/** Ferme le bandeau « Finir la configuration » : il ne fait pas partie de ce qu'on compare. */
async function dismissSetupBanner() {
  const { api, post } = await adminApi();
  await post("/api/admin/setup-progress/dismiss");
  await api.dispose();
}

/** Crée toutes les données et renvoie l'id du clip de la réunion. */
export async function seedUiCompare(): Promise<{ meetingId: number }> {
  await dismissSetupBanner();
  await createPersons(PEOPLE);
  const meetingId = await uploadClip(`${MEETING}.mp4`, "shared");
  await analyse(meetingId, SEGMENTS, 5, true);
  for (const title of OTHER_CLIPS) {
    const id = await uploadClip(`${title}.mp4`, "shared");
    await analyse(
      id,
      [
        [0, 4, 0, "Bonjour à tous."],
        [4, 9, 1, "On reprend où on s'était arrêtés."],
      ],
      2,
      false,
    );
  }
  return { meetingId };
}
