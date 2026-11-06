import type { MessageKey } from "../i18n/messages";
import type { Clip, ClipVoice } from "./libraryApi";

export type StepState =
  | "none"
  | "queued"
  | "running"
  | "blocked"
  | "done"
  | "failed"
  | "unavailable"
  | "partial";

export type Step = {
  key: string;
  label: MessageKey;
  state: StepState;
  /** "2 of 3", when the state alone does not say enough. */
  detail?: { done: number; total: number };
};

function voiceProcessing(voice: ClipVoice | null): StepState {
  if (!voice) return "none";
  // A microphone track is attributed without any diarization: its voice is there, processed.
  if (voice.status === "none")
    return voice.speakers.length > 0 ? "done" : "none";
  return voice.status;
}

function voiceIdentification(
  voice: ClipVoice | null,
  processed: StepState,
): Pick<Step, "state" | "detail"> {
  const speakers = voice?.speakers ?? [];
  if (processed !== "done" || speakers.length === 0) return { state: "none" };
  const done = speakers.filter(
    (speaker) => speaker.state === "identified",
  ).length;
  const detail = { done, total: speakers.length };
  if (done === speakers.length) return { state: "done", detail };
  return { state: done === 0 ? "none" : "partial", detail };
}

/** A run, reduced to what the steps need: the state the server gives each of its steps. */
export type LiveRun = {
  status: string;
  steps: readonly { step: string; status: string }[];
};

/** The state of a clip step as the run in progress says it; null when the run does not decide. */
function runStepState(run: LiveRun | null, name: string): StepState | null {
  const status = run?.steps.find((s) => s.step === name)?.status;
  switch (status) {
    case "running":
      return "running";
    case "waiting":
    case "retry":
      return "queued";
    case "blocked":
      return "blocked";
    case "failed":
      return "failed";
    case "done":
      return "done";
    default:
      return null; // not applicable, or not in the run: the clip's own state stands.
  }
}

/**
 * The five steps of a clip, from what the server says of it and of its voices. While an analysis
 * runs, its steps are read from the run itself, the same snapshot as the text beside them,
 * rather than from a clip read a moment ago.
 */
export function clipSteps(
  clip: Clip,
  voice: ClipVoice | null,
  run: LiveRun | null = null,
): Step[] {
  const processed = runStepState(run, "diarization") ?? voiceProcessing(voice);
  return [
    { key: "imported", label: "clip.progress.imported", state: "done" },
    {
      key: "transcribed",
      label: "clip.progress.transcribed",
      state:
        runStepState(run, "transcription") ??
        clip.progress?.transcription ??
        "none",
    },
    {
      key: "indexed",
      label: "clip.progress.indexed",
      state: runStepState(run, "indexing") ?? clip.progress?.indexing ?? "none",
    },
    {
      key: "voicesProcessed",
      label: "clip.progress.voicesProcessed",
      state: processed,
    },
    {
      key: "voicesIdentified",
      label: "clip.progress.voicesIdentified",
      ...voiceIdentification(voice, processed),
    },
  ];
}

/**
 * Where the analysis stands when the server has no run for the clip (steps launched one by
 * one): read from the steps themselves, so that the header never says "never analysed" over
 * a failed step. Null when nothing was ever started.
 */
export function legacyAnalysisState(
  steps: Step[],
): "failed" | "busy" | "complete" | null {
  const analysis = steps.filter(
    (s) => s.key !== "imported" && s.key !== "voicesIdentified",
  );
  if (analysis.some((s) => s.state === "failed")) return "failed";
  if (analysis.some((s) => s.state === "running" || s.state === "queued"))
    return "busy";
  if (analysis.every((s) => s.state === "done")) return "complete";
  return null;
}
