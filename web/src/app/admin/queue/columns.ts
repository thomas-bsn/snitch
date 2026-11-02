import type { components } from "../../../api.gen";

type Run = components["schemas"]["AnalysisQueueRun"];
export type Step = Run["steps"][number];

export const COLUMNS = [
  "transcription",
  "search",
  "voices",
  "recognition",
  "completed",
] as const;
export type Column = (typeof COLUMNS)[number];

/** The steps of an analysis, in the order they advance. */
const ORDER: Step["step"][] = [
  "preparation",
  "transcription",
  "indexing",
  "diarization",
  "fingerprints",
  "matching",
];

const COLUMN_OF_STEP: Record<Step["step"], Exclude<Column, "completed">> = {
  preparation: "transcription",
  transcription: "transcription",
  indexing: "search",
  diarization: "voices",
  fingerprints: "recognition",
  matching: "recognition",
};

export const isKnownStep = (name: string): name is Step["step"] =>
  ORDER.includes(name as Step["step"]);

/** A step that does not apply, or is out of the perimeter, never places the card. */
const withoutEffect = (status: Step["status"]) =>
  status === "not_applicable" || status === "not_yet_supported";

export const needsAttention = (status: Step["status"]) =>
  status === "failed" || status === "blocked" || status === "retry";

/**
 * The step that places an active analysis: the failed or blocked one, else the first applicable
 * step that is not done. An analysis whose applicable steps are all done stays with the last of
 * them; steps the interface does not know never move a card.
 */
export function placingStep(steps: Step[]): Step | undefined {
  const known = steps
    .filter((s) => isKnownStep(s.step))
    .sort((a, b) => ORDER.indexOf(a.step) - ORDER.indexOf(b.step));
  return (
    known.find((s) => s.status === "failed" || s.status === "blocked") ??
    known.find((s) => !withoutEffect(s.status) && s.status !== "done") ??
    known.filter((s) => !withoutEffect(s.status)).at(-1)
  );
}

/** The column of an active analysis. */
export function columnOf(run: Run): Exclude<Column, "completed"> {
  const step = placingStep(run.steps);
  return step ? COLUMN_OF_STEP[step.step] : "transcription";
}
