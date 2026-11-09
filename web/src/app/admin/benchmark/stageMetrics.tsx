import type { ReactNode } from "react";
import type { Language } from "../../preferences";
import type { Run, RunResult } from "./api";
import { decimal, elapsed, percent } from "./format";
import type { Metric } from "./InfoTip";

export type Column = {
  metric: Metric;
  render: (x: RunResult, language: Language) => ReactNode;
  /** The best value of the column among comparable results: the lowest or the highest. */
  best?: { value: (x: RunResult) => number | null | undefined; low: boolean };
};

const coverage: Column = {
  metric: "coverage",
  render: (x) => `${x.items_scored} / ${x.items_total}`,
};
const elapsedColumn: Column = {
  metric: "elapsed",
  render: (x) => elapsed(x.elapsed_seconds),
};

const fingerprintColumns: Column[] = [
  {
    metric: "eer",
    render: (x, l) => percent(l, x.metrics.eer),
    best: { value: (x) => x.metrics.eer, low: true },
  },
  { metric: "thr", render: (x, l) => decimal(l, x.metrics.eer_threshold) },
  {
    metric: "top1",
    render: (x, l) => percent(l, x.metrics.top1_accuracy),
    best: { value: (x) => x.metrics.top1_accuracy, low: false },
  },
  {
    metric: "precision",
    render: (x, l) => percent(l, x.metrics.precision),
    best: { value: (x) => x.metrics.precision, low: false },
  },
  {
    metric: "recall",
    render: (x, l) => percent(l, x.metrics.recall),
    best: { value: (x) => x.metrics.recall, low: false },
  },
  {
    metric: "fp",
    render: (x) => x.metrics.false_positives,
    best: { value: (x) => x.metrics.false_positives, low: true },
  },
  {
    metric: "fn",
    render: (x) => x.metrics.false_negatives,
    best: { value: (x) => x.metrics.false_negatives, low: true },
  },
  coverage,
  elapsedColumn,
];

const transcriptionColumns: Column[] = [
  {
    metric: "wer",
    render: (x, l) => percent(l, x.metrics.wer),
    best: { value: (x) => x.metrics.wer, low: true },
  },
  {
    metric: "cer",
    render: (x, l) => percent(l, x.metrics.cer),
    best: { value: (x) => x.metrics.cer, low: true },
  },
  coverage,
  elapsedColumn,
];

const diarizationColumns: Column[] = [
  {
    metric: "der",
    render: (x, l) => percent(l, x.metrics.der),
    best: { value: (x) => x.metrics.der, low: true },
  },
  {
    metric: "missed",
    render: (x, l) => percent(l, x.metrics.missed_speech_rate),
    best: { value: (x) => x.metrics.missed_speech_rate, low: true },
  },
  {
    metric: "falseAlarm",
    render: (x, l) => percent(l, x.metrics.false_alarm_rate),
    best: { value: (x) => x.metrics.false_alarm_rate, low: true },
  },
  {
    metric: "confusion",
    render: (x, l) => percent(l, x.metrics.confusion_rate),
    best: { value: (x) => x.metrics.confusion_rate, low: true },
  },
  coverage,
  elapsedColumn,
];

const e2eColumns: Column[] = [
  {
    metric: "cpwer",
    render: (x, l) => percent(l, x.metrics.cpwer),
    best: { value: (x) => x.metrics.cpwer, low: true },
  },
  {
    metric: "wer",
    render: (x, l) => percent(l, x.metrics.wer),
    best: { value: (x) => x.metrics.wer, low: true },
  },
  {
    metric: "der",
    render: (x, l) => percent(l, x.metrics.der),
    best: { value: (x) => x.metrics.der, low: true },
  },
  coverage,
  elapsedColumn,
];

export const columnsOf = (stage: Run["stage"]): Column[] =>
  stage === "transcription"
    ? transcriptionColumns
    : stage === "diarization"
      ? diarizationColumns
      : stage === "e2e"
        ? e2eColumns
        : fingerprintColumns;

/** The measure that ranks the models of a stage, lowest first, and the label of its chart. */
export const primaryOf = (stage: Run["stage"]) =>
  stage === "transcription"
    ? {
        metric: "wer" as const,
        value: (x: RunResult) => x.metrics.wer,
        label: "WER",
      }
    : stage === "diarization"
      ? {
          metric: "der" as const,
          value: (x: RunResult) => x.metrics.der,
          label: "DER",
        }
      : stage === "e2e"
        ? {
            metric: "cpwer" as const,
            value: (x: RunResult) => x.metrics.cpwer,
            label: "cpWER",
          }
        : {
            metric: "eer" as const,
            value: (x: RunResult) => x.metrics.eer,
            label: "EER",
          };
