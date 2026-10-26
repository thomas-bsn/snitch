// PROTOTYPE — tiny in-memory store shared by the Workers, Transcription and worker-creation screens. Resets on reload.
// ?empty=1 starts with no worker and no transcription profile, to show the first-worker flow.
import { useSyncExternalStore } from "react";

export type W = {
  id: number;
  name: string;
  online: boolean;
  seen: string;
  created: string;
  conc: number;
  server: string;
  alias: string;
  models: [string, string][];
  machine: [string, string][];
  jobs: [string, string, string][];
};
export type Profile = { model: string; language: string; version: number };
type State = { workers: W[]; profile: Profile | null };

const SEED: W[] = [
  {
    id: 1,
    name: "gaming-pc",
    online: true,
    seen: "12 s ago",
    created: "Oct 3",
    conc: 1,
    server: "pc-gamer:8000",
    alias: "",
    models: [
      ["Transcription", "whisper-large-v3 · openai-compatible · pc-gamer:8000"],
      ["Diarization", "pyannote community-1"],
      ["Voice fingerprint", "wespeaker v1"],
    ],
    machine: [
      ["Hostname", "DESKTOP-4070"],
      ["System", "Windows 11 · Docker Desktop (WSL2)"],
      ["GPU", "NVIDIA RTX 4070 · 12 GB · driver 572.16"],
      ["CPU / RAM", "Ryzen 7 5800X · 32 GB"],
      ["Worker version", "0.6.0"],
      ["Seen from", "192.168.1.24"],
    ],
    jobs: [
      ["Transcription", "Clutch 1v3 sur Bind", "Done · 41 s"],
      ["Voice processing", "Ace en finale", "Running"],
      ["Transcription", "Le bug du siècle", "Failed · model unreachable"],
    ],
  },
  {
    id: 2,
    name: "laptop",
    online: true,
    seen: "3 min ago",
    created: "Sept 21",
    conc: 1,
    server: "laptop.local:9000",
    alias: "Systran/faster-whisper-large-v3",
    models: [
      [
        "Transcription",
        "Systran/faster-whisper-large-v3 · openai-compatible · laptop.local:9000",
      ],
    ],
    machine: [],
    jobs: [],
  },
  {
    id: 3,
    name: "old-desktop",
    online: false,
    seen: "2 days ago",
    created: "Sept 2",
    conc: 1,
    server: "192.168.1.40:8000",
    alias: "",
    models: [],
    machine: [],
    jobs: [],
  },
];

const empty =
  typeof location !== "undefined" &&
  new URLSearchParams(location.search).get("empty") === "1";
let state: State = empty
  ? { workers: [], profile: null }
  : {
      workers: SEED,
      profile: { model: "whisper-large-v3", language: "fr", version: 3 },
    };
const subs = new Set<() => void>();
const set = (s: State) => {
  state = s;
  subs.forEach((f) => f());
};

export const store = {
  addWorker(
    name: string,
    server: string,
    alias: string,
    profile?: { model: string; language: string },
  ) {
    const w: W = {
      id: Date.now(),
      name,
      online: false,
      seen: "never",
      created: "Today",
      conc: 1,
      server,
      alias,
      models: [],
      machine: [],
      jobs: [],
    };
    set({
      workers: [...state.workers, w],
      profile: profile ? { ...profile, version: 1 } : state.profile,
    });
  },
  saveProfile(model: string, language: string) {
    if (!state.profile) return;
    set({
      ...state,
      profile: { model, language, version: state.profile.version + 1 },
    });
  },
  setAlias(id: number, alias: string) {
    set({
      ...state,
      workers: state.workers.map((w) => (w.id === id ? { ...w, alias } : w)),
    });
  },
};

export function useStore<T>(pick: (s: State) => T): T {
  return useSyncExternalStore(
    (cb) => {
      subs.add(cb);
      return () => subs.delete(cb);
    },
    () => pick(state),
  );
}
export const useWorkers = () => useStore((s) => s.workers);
export const useProfile = () => useStore((s) => s.profile);
