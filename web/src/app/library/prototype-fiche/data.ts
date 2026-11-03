// PROTOTYPE — fiche vidéo refondue. Données factices en mémoire, jamais fusionné dans main.

export type StepKey =
  | "preparation"
  | "transcription"
  | "indexing"
  | "diarization"
  | "fingerprints"
  | "matching";

export type StepStatus =
  | "absent"
  | "pending"
  | "waiting"
  | "running"
  | "done"
  | "failed"
  | "cancelled";

export type Step = {
  key: StepKey;
  label: string;
  status: StepStatus;
  progress?: number;
  worker?: string;
  durationS?: number;
  error?: { code: string; message: string; at: string };
  waitCause?: string;
};

export const STEP_LABELS: Record<StepKey, string> = {
  preparation: "Préparation audio",
  transcription: "Transcription",
  indexing: "Indexation",
  diarization: "Détection des voix",
  fingerprints: "Empreintes vocales",
  matching: "Identification",
};

export const STEP_ORDER: StepKey[] = [
  "preparation",
  "transcription",
  "indexing",
  "diarization",
  "fingerprints",
  "matching",
];

/** Ce qui disparaît aussi quand on supprime une étape. */
export const DEPENDENTS: Record<StepKey, StepKey[]> = {
  preparation: [
    "transcription",
    "indexing",
    "diarization",
    "fingerprints",
    "matching",
  ],
  transcription: ["indexing"],
  indexing: [],
  diarization: ["fingerprints", "matching"],
  fingerprints: ["matching"],
  matching: [],
};

/** Ce qui doit exister pour lancer une étape. */
export const PREREQUISITES: Record<StepKey, StepKey[]> = {
  preparation: [],
  transcription: ["preparation"],
  indexing: ["transcription"],
  diarization: ["preparation"],
  fingerprints: ["diarization"],
  matching: ["fingerprints"],
};

export function cascade(selected: StepKey[]): StepKey[] {
  const out = new Set<StepKey>(selected);
  let grew = true;
  while (grew) {
    grew = false;
    for (const k of [...out])
      for (const d of DEPENDENTS[k])
        if (!out.has(d)) {
          out.add(d);
          grew = true;
        }
  }
  return STEP_ORDER.filter((k) => out.has(k));
}

export type VoiceState =
  | "validated"
  | "suggested"
  | "ambiguous"
  | "unknown"
  | "ignored"
  | "random";

export type Voice = {
  id: number;
  label: string; // speaker_n
  state: VoiceState;
  person?: string;
  candidates?: string[];
  refused: string[];
  turns: number;
  speakingS: number;
  group?: string;
};

export const initialVoices: Voice[] = [
  {
    id: 1,
    label: "SPEAKER_00",
    state: "validated",
    person: "Marc Lefèvre",
    refused: [],
    turns: 42,
    speakingS: 611,
  },
  {
    id: 2,
    label: "SPEAKER_01",
    state: "suggested",
    person: "Julie Martin",
    candidates: ["Julie Martin", "Claire Dubois"],
    refused: [],
    turns: 31,
    speakingS: 402,
  },
  {
    id: 3,
    label: "SPEAKER_02",
    state: "ambiguous",
    candidates: ["Thomas Roux", "Hugo Bernard"],
    refused: [],
    turns: 12,
    speakingS: 148,
  },
  {
    id: 4,
    label: "SPEAKER_03",
    state: "unknown",
    refused: [],
    turns: 6,
    speakingS: 37,
    group: "Groupe inconnu #7 (3 vidéos)",
  },
  {
    id: 5,
    label: "SPEAKER_04",
    state: "unknown",
    refused: [],
    turns: 2,
    speakingS: 3,
  },
];

export type Segment = {
  id: number;
  start: number;
  end: number;
  voiceId: number;
  text: string;
  edited?: boolean;
};

export const initialSegments: Segment[] = [
  { id: 1, start: 0, end: 6.2, voiceId: 1, text: "Bon, on commence. Tout le monde m'entend bien ?" },
  { id: 2, start: 6.2, end: 9.8, voiceId: 2, text: "Oui, c'est bon de mon côté." },
  { id: 3, start: 9.8, end: 18.4, voiceId: 1, text: "Parfait. Premier point : le planning du tournage de la semaine prochaine." },
  { id: 4, start: 18.4, end: 21.0, voiceId: 5, text: "Euh… pardon, mauvais micro." },
  { id: 5, start: 21.0, end: 33.5, voiceId: 3, text: "Pour jeudi, il faudrait décaler la scène du parking, la lumière n'ira pas." },
  { id: 6, start: 33.5, end: 41.2, voiceId: 2, text: "On peut la passer vendredi matin, j'ai vérifié les autorisations." },
  { id: 7, start: 41.2, end: 47.9, voiceId: 4, text: "Je peux venir avec le deuxième boîtier si besoin." },
  { id: 8, start: 47.9, end: 58.0, voiceId: 1, text: "Très bien, on note vendredi. Julie, tu envoies le récap ce soir ?" },
  { id: 9, start: 58.0, end: 61.3, voiceId: 2, text: "Je m'en occupe." },
  { id: 10, start: 61.3, end: 74.8, voiceId: 3, text: "Dernière chose : la carte SD du plan large était pleine, on a perdu les dix dernières minutes." },
];

export const clipMeta = {
  title: "Réunion de préparation — tournage S12",
  file: "2026/novembre/reunion-prepa-s12.mp4",
  folder: "/videos/2026/novembre",
  owner: "Thomas",
  recordedAt: "28 oct. 2026, 14:02",
  addedAt: "29 oct. 2026",
  duration: "38:12",
  durationS: 2292,
  visibility: "Partagée",
  size: "1,8 Go",
  format: "H.264 · 1920×1080 · 25 i/s",
  audio: "2 pistes (piste 1 : micro salle · piste 2 : micro HF)",
  fields: [
    ["Projet", "Court-métrage « Parking »"],
    ["Lieu", "Studio B"],
    ["Tags", "réunion, planning"],
  ] as [string, string][],
};

const LOG_BASE = [
  "INFO  job_claimed step=voice.fingerprint.1 lease=120s",
  "INFO  audio_prepared stream=1 sample_rate=16000 channels=1 duration_s=2292.04",
  "INFO  diarization_loaded segments=93 speakers=5",
  "INFO  model_loaded name=wespeaker-voxceleb-resnet34 device=cuda:0",
  "DEBUG embedding_batch index=1/12 segments=8",
  "DEBUG embedding_batch index=2/12 segments=8",
  "DEBUG embedding_batch index=3/12 segments=8",
  "DEBUG embedding_batch index=4/12 segments=8",
  "DEBUG embedding_batch index=5/12 segments=8",
  "DEBUG embedding_batch index=6/12 segments=8",
  "DEBUG embedding_batch index=7/12 segments=8",
  "DEBUG embedding_batch index=8/12 segments=8",
  "DEBUG embedding_batch index=9/12 segments=8",
  "DEBUG embedding_batch index=10/12 segments=8",
  "DEBUG embedding_batch index=11/12 segments=8",
  "WARN  torchcodec_fallback reason=ffmpeg_libs_missing (sans effet sur le décodage)",
  "DEBUG embedding_batch index=12/12 segments=5",
  "WARN  voice_fingerprint_failed stage=segment_extraction exception=ValueError segment_end=2292.06 audio_duration_s=2292.04",
  "WARN  voice_fingerprint_failed detail=\"requested chunk with end time (t=2292.060s) greater than waveform file duration (2292.040s)\"",
  "ERROR job_failed step=voice.fingerprint.1 code=inference_failed",
];

export function failureLogs(): string[] {
  const start = new Date("2026-11-03T14:21:07");
  const pre: string[] = [];
  for (let i = 0; i < 30; i++)
    pre.push(
      `INFO  heartbeat lease_renewed step=voice.diarization.1 elapsed_s=${(i + 1) * 10}`,
    );
  return [...pre, ...LOG_BASE].slice(-50).map((line, i) => {
    const t = new Date(start.getTime() + i * 1700);
    return `${t.toTimeString().slice(0, 8)}  ${line}`;
  });
}

export const whisperErrorLogs = [
  "14:02:11  INFO  POST http://whisper.lan:9000/v1/audio/transcriptions model=large-v3 stream=1",
  "14:02:16  WARN  whisper_http status=503 latency_ms=5012 body=\"model is loading\"",
  "14:02:31  INFO  retry attempt=2/5 backoff_s=15",
  "14:02:36  WARN  whisper_http status=503 latency_ms=4987 body=\"model is loading\"",
];

export function initialSteps(): Step[] {
  return [
    { key: "preparation", label: STEP_LABELS.preparation, status: "done", worker: "gpu-salon", durationS: 21 },
    { key: "transcription", label: STEP_LABELS.transcription, status: "done", worker: "gpu-salon", durationS: 412 },
    { key: "indexing", label: STEP_LABELS.indexing, status: "done", durationS: 9 },
    { key: "diarization", label: STEP_LABELS.diarization, status: "done", worker: "gpu-salon", durationS: 188 },
    {
      key: "fingerprints",
      label: STEP_LABELS.fingerprints,
      status: "failed",
      worker: "gpu-salon",
      durationS: 34,
      error: {
        code: "inference_failed",
        message:
          "Le modèle d'empreintes n'a pas pu traiter un segment (étape : extraction du segment). Le segment dépasse la fin de l'audio.",
        at: "3 nov. 2026, 14:21",
      },
    },
    { key: "matching", label: STEP_LABELS.matching, status: "absent" },
  ];
}
