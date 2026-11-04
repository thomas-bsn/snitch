import type { MessageKey } from "../i18n/messages";
import type { Clip, ClipVoice } from "./libraryApi";

export type StepState =
  | "none"
  | "queued"
  | "running"
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

/** The five steps of a clip, from what the server says of it and of its voices. */
export function clipSteps(clip: Clip, voice: ClipVoice | null): Step[] {
  const processed = voiceProcessing(voice);
  return [
    { key: "imported", label: "clip.progress.imported", state: "done" },
    {
      key: "transcribed",
      label: "clip.progress.transcribed",
      state: clip.progress?.transcription ?? "none",
    },
    {
      key: "indexed",
      label: "clip.progress.indexed",
      state: clip.progress?.indexing ?? "none",
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
