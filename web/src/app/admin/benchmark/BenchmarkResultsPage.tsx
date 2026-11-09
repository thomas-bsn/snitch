import { useEffect, type ReactNode } from "react";
import { Link, useSearchParams } from "react-router";
import { ArrowLeft, CircleDashed, Loader } from "lucide-react";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { StatusPill } from "../../components/StatusPill";
import { useDocumentTitle } from "../../components/useDocumentTitle";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";
import {
  isRunning,
  loadModels,
  loadRun,
  loadRuns,
  type Model,
  type Run,
  type RunResult,
} from "./api";
import { dateTime, decimal, elapsed, percent } from "./format";
import { InfoTip, type Metric } from "./InfoTip";

const card = "flex flex-col gap-3 rounded-xl border border-line bg-surface p-4";
/** Static class names, so the stylesheet keeps them: a series is always drawn with a label too. */
const SERIES = ["bg-series-1", "bg-series-2", "bg-series-3", "bg-series-4"];

function labelOf(models: Model[], id: string, repo?: string) {
  return models.find((m) => m.id === id)?.label ?? repo ?? id;
}

function RunState({ status }: { status: Run["status"] }) {
  const { t } = useI18n();
  const label = t(`benchmark.results.states.${status}` as MessageKey);
  const tone = {
    completed: "ok",
    failed: "bad",
    running: "busy",
    cancelled: "idle",
  } as const;
  const icon = { running: Loader, cancelled: CircleDashed }[
    status as "running" | "cancelled"
  ];
  return (
    <StatusPill tone={tone[status]} icon={icon}>
      {label}
    </StatusPill>
  );
}

/** The results of the benchmarks: the best model of a stage, the history, and the detail of one run. */
export function BenchmarkResultsPage() {
  const [params] = useSearchParams();
  const run = Number(params.get("run"));
  return Number.isInteger(run) && run > 0 ? (
    <RunDetail id={run} />
  ) : (
    <Overview />
  );
}

function Overview() {
  const { t, error, language } = useI18n();
  useDocumentTitle(t("admin.tabs.benchmark"));
  const runs = useLoaded(loadRuns);
  const models = useLoaded(loadModels);
  const catalogue =
    models.state.status === "ready" ? models.state.value.models : [];

  if (runs.state.status === "loading")
    return <p role="status">{t("benchmark.results.loading")}</p>;
  if (runs.state.status === "failed")
    return (
      <div className="flex flex-col items-start gap-2">
        <ErrorAlert>
          {t("benchmark.results.loadFailed")} {error(runs.state.code)}
        </ErrorAlert>
        <button type="button" className={buttonClass()} onClick={runs.reload}>
          {t("benchmark.results.retry")}
        </button>
      </div>
    );
  const list = runs.state.value.runs;
  const finished = list.find(
    (r) =>
      r.stage === "fingerprint" &&
      r.status === "completed" &&
      r.results.some((x) => x.state === "complete"),
  );
  const comparable = (finished?.results ?? [])
    .filter((x) => x.state === "complete" && x.metrics.eer != null)
    .sort((a, b) => (a.metrics.eer ?? 1) - (b.metrics.eer ?? 1));
  const best = comparable[0];
  const second = comparable[1];
  const production = catalogue.find((m) => m.active);

  return (
    <div className="flex flex-col gap-4">
      <section aria-labelledby="best-title" className={card}>
        <h2 id="best-title" className="text-base font-semibold">
          {t("benchmark.results.summaryTitle")}
        </h2>
        <p className="text-sm text-mute">
          {t("benchmark.results.summaryLead")}
        </p>
        <div className="rounded-xl border border-line p-3">
          <div className="flex flex-wrap items-center gap-2">
            <strong>{t("benchmark.run.stageNames.fingerprint")}</strong>
            <span className="flex-1" />
            {best && (
              <span className="inline-flex items-center rounded-full bg-accent-soft px-2.5 py-0.5 text-xs font-medium text-accent-ink">
                EER {percent(language, best.metrics.eer)}
                <InfoTip metric="eer" />
              </span>
            )}
          </div>
          {best && finished ? (
            <div className="mt-2 flex flex-col gap-1">
              <p className="text-base font-semibold">
                {labelOf(catalogue, best.model_id, best.model)}
              </p>
              <p className="text-sm text-mute">
                {production?.id === best.model_id ? (
                  <span className="rounded-full bg-ok-bg px-2 py-0.5 text-xs font-medium text-ok-fg">
                    {t("benchmark.results.inProduction")}
                  </span>
                ) : (
                  production && (
                    <span className="rounded-full bg-warn-bg px-2 py-0.5 text-xs font-medium text-warn-fg">
                      {t("benchmark.results.differs", {
                        model: production.label,
                      })}
                    </span>
                  )
                )}{" "}
                {second &&
                  t("benchmark.results.ahead", {
                    gap: (
                      ((second.metrics.eer ?? 0) - (best.metrics.eer ?? 0)) *
                      100
                    ).toFixed(1),
                    other: labelOf(catalogue, second.model_id, second.model),
                  })}
              </p>
              <p className="text-sm text-mute">
                {dateTime(language, finished.created_at)} ·{" "}
                {finished.dataset_name} ·{" "}
                <Link
                  className="text-accent-ink underline"
                  to={`/admin/benchmark/results?run=${finished.id}`}
                >
                  {t("benchmark.results.details")}
                </Link>
              </p>
            </div>
          ) : (
            <p className="mt-2 text-sm text-mute">
              {t("benchmark.results.noBenchmark")}{" "}
              <Link className="text-accent-ink underline" to="/admin/benchmark">
                {t("benchmark.results.run")}
              </Link>
            </p>
          )}
        </div>
      </section>

      <section aria-labelledby="history-title" className={card}>
        <h2 id="history-title" className="text-base font-semibold">
          {t("benchmark.results.history")}
        </h2>
        <p className="text-sm text-mute">
          {t("benchmark.results.historyLead")}
        </p>
        {list.length === 0 ? (
          <p className="rounded-lg bg-side px-3 py-6 text-center text-mute">
            {t("benchmark.results.empty")}
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[720px] border-collapse text-sm">
              <thead>
                <tr className="text-left text-xs text-mute uppercase">
                  {(
                    [
                      "date",
                      "stage",
                      "corpus",
                      "compared",
                      "state",
                      "author",
                    ] as const
                  ).map((c) => (
                    <th key={c} className="px-2 py-2">
                      {t(`benchmark.results.${c}`)}
                    </th>
                  ))}
                  <th className="px-2 py-2">
                    <span className="sr-only">
                      {t("benchmark.results.detail")}
                    </span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {list.map((r) => (
                  <tr key={r.id} className="border-t border-line align-top">
                    <td className="px-2 py-2.5">
                      {dateTime(language, r.created_at)}
                    </td>
                    <td className="px-2 py-2.5">
                      {t(`benchmark.run.stageNames.${r.stage}` as MessageKey)}
                    </td>
                    <td className="px-2 py-2.5">{r.dataset_name}</td>
                    <td className="px-2 py-2.5">
                      {r.model_ids
                        .map((id) => labelOf(catalogue, id))
                        .join(", ")}
                    </td>
                    <td className="px-2 py-2.5">
                      <span className="inline-flex flex-wrap items-center gap-1">
                        <RunState status={r.status} />
                        {r.results.some((x) => x.state === "partial") && (
                          <span className="rounded-full bg-warn-bg px-2 py-0.5 text-xs font-medium text-warn-fg">
                            {t("benchmark.results.partial")}
                          </span>
                        )}
                      </span>
                    </td>
                    <td className="px-2 py-2.5">{r.author_name}</td>
                    <td className="px-2 py-2.5">
                      <Link
                        className={buttonClass()}
                        to={`/admin/benchmark/results?run=${r.id}`}
                      >
                        {t("benchmark.results.detail")}
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}

type Column = {
  metric: Metric;
  render: (x: RunResult, language: "en" | "fr") => ReactNode;
  /** The best value of the column among comparable results: the lowest or the highest. */
  best?: { value: (x: RunResult) => number | null | undefined; low: boolean };
};

const columns: Column[] = [
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
  { metric: "coverage", render: (x) => `${x.items_scored} / ${x.items_total}` },
  { metric: "elapsed", render: (x) => elapsed(x.elapsed_seconds) },
];

function RunDetail({ id }: { id: number }) {
  const { t, error, language } = useI18n();
  useDocumentTitle(t("admin.tabs.benchmark"));
  const loader = useLoaded(() => loadRun(id));
  const models = useLoaded(loadModels);
  const catalogue =
    models.state.status === "ready" ? models.state.value.models : [];
  const run = loader.state.status === "ready" ? loader.state.value : undefined;
  const { reload } = loader;
  useEffect(() => {
    if (!run || !isRunning(run)) return;
    const timer = setInterval(reload, 3000);
    return () => clearInterval(timer);
  }, [run, reload]);

  const back = (
    <Link
      to="/admin/benchmark/results"
      className={`${buttonClass()} self-start`}
    >
      <ArrowLeft className="size-4" aria-hidden="true" />
      {t("benchmark.results.back")}
    </Link>
  );
  if (loader.state.status === "loading")
    return <p role="status">{t("benchmark.results.loading")}</p>;
  if (loader.state.status === "failed")
    return (
      <div className="flex flex-col gap-3">
        {back}
        <ErrorAlert>
          {loader.state.code === "not_found"
            ? t("benchmark.results.notFound")
            : `${t("benchmark.results.loadFailed")} ${error(loader.state.code)}`}
        </ErrorAlert>
      </div>
    );
  const data = loader.state.value;
  const comparable = data.results.filter((x) => x.state === "complete");
  const label = (x: RunResult) => labelOf(catalogue, x.model_id, x.model);
  const bestOf = (c: Column) => {
    const values = comparable
      .map((x) => c.best?.value(x))
      .filter((v): v is number => typeof v === "number");
    if (!c.best || values.length < 2) return undefined;
    return c.best.low ? Math.min(...values) : Math.max(...values);
  };
  const partial = data.results.filter((x) => x.state === "partial");
  const scale = Math.max(...data.results.map((x) => x.metrics.eer ?? 0), 0.01);

  return (
    <div className="flex flex-col gap-4">
      {back}
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="text-lg font-semibold">
          {t(`benchmark.run.stageNames.${data.stage}` as MessageKey)} ·{" "}
          {data.dataset_name}
        </h2>
        <RunState status={data.status} />
      </div>
      <p className="text-sm text-mute">
        {dateTime(language, data.created_at)} · {data.author_name}
      </p>
      {isRunning(data) && (
        <StatusNotice>{t("benchmark.results.waiting")}</StatusNotice>
      )}
      {data.status === "failed" && (
        <ErrorAlert>{t("benchmark.results.failedNote")}</ErrorAlert>
      )}
      {partial.map((x) => (
        <p
          key={x.model_id}
          role="status"
          className="rounded-lg bg-warn-bg px-3 py-2.5 text-warn-fg"
        >
          {t("benchmark.results.partialNote", {
            model: label(x),
            scored: x.items_scored,
            total: x.items_total,
          })}
        </p>
      ))}
      {data.progress.errors.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold">
            {t("benchmark.results.taskErrors")}
          </h3>
          <ul className="m-0 list-none p-0 text-sm text-mute">
            {data.progress.errors.map((e) => (
              <li key={`${e.job_id}-${e.model_id}`}>
                #{e.job_id} · {labelOf(catalogue, e.model_id)} · {e.error}
              </li>
            ))}
          </ul>
        </div>
      )}
      {data.results.length > 0 && (
        <>
          <p
            role="note"
            className="rounded-lg bg-accent-soft px-3 py-2.5 text-accent-ink"
          >
            {t("benchmark.results.note")}
          </p>
          <section aria-labelledby="comparison-title" className={card}>
            <h3 id="comparison-title" className="text-base font-semibold">
              {t("benchmark.results.comparison")}
            </h3>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[760px] border-collapse text-sm">
                <thead>
                  <tr className="text-xs text-mute uppercase">
                    <th scope="col" className="px-2 py-2 text-left">
                      {t("benchmark.results.model")}
                    </th>
                    {columns.map((c) => (
                      <th
                        key={c.metric}
                        scope="col"
                        className="px-2 py-2 text-right whitespace-nowrap"
                      >
                        {t(`benchmark.metrics.${c.metric}.name` as MessageKey)}
                        <InfoTip metric={c.metric} />
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {data.results.map((x, i) => (
                    <tr key={x.model_id} className="border-t border-line">
                      <th
                        scope="row"
                        className="px-2 py-2.5 text-left font-semibold"
                      >
                        <span className="flex items-start gap-2">
                          <span
                            aria-hidden="true"
                            className={`mt-1.5 size-2.5 shrink-0 rounded-sm ${SERIES[i % SERIES.length]}`}
                          />
                          <span>
                            {label(x)}
                            {x.state === "partial" && (
                              <span className="ml-2 rounded-full bg-warn-bg px-2 py-0.5 text-xs font-medium text-warn-fg">
                                {t("benchmark.results.partial")}
                              </span>
                            )}
                          </span>
                        </span>
                      </th>
                      {columns.map((c) => {
                        const value = c.best?.value(x);
                        const isBest =
                          x.state === "complete" &&
                          typeof value === "number" &&
                          value === bestOf(c);
                        return (
                          <td
                            key={c.metric}
                            className={`px-2 py-2.5 text-right tabular-nums ${isBest ? "font-bold" : ""}`}
                          >
                            {c.render(x, language)}
                            {isBest && <span aria-hidden="true"> ▲</span>}
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="text-xs text-mute">
              {t("benchmark.results.bestMark")}{" "}
              {t("benchmark.results.tableNote")}
            </p>
          </section>

          <section aria-labelledby="eer-title" className={card}>
            <h3 id="eer-title" className="text-base font-semibold">
              {t("benchmark.results.eerChart")}
              <InfoTip metric="eer" />
            </h3>
            <ul className="m-0 flex list-none flex-col gap-2 p-0">
              {data.results.map((x, i) => (
                <li
                  key={x.model_id}
                  className="grid grid-cols-[minmax(7rem,14rem)_1fr_4rem] items-center gap-3 text-sm"
                >
                  <span>{label(x)}</span>
                  <span
                    aria-hidden="true"
                    className="h-4 overflow-hidden rounded border border-line bg-side"
                  >
                    <span
                      className={`block h-full ${SERIES[i % SERIES.length]}`}
                      style={{
                        width: `${((x.metrics.eer ?? 0) / scale) * 100}%`,
                      }}
                    />
                  </span>
                  <strong className="text-right tabular-nums">
                    {percent(language, x.metrics.eer)}
                  </strong>
                </li>
              ))}
            </ul>
            <p className="text-xs text-mute">
              {t("benchmark.results.chartText")}
            </p>
          </section>

          <section aria-labelledby="separation-title" className={card}>
            <h3 id="separation-title" className="text-base font-semibold">
              {t("benchmark.results.separation")}
            </h3>
            <p className="text-sm text-mute">
              {t("benchmark.results.separationLead")}
            </p>
            <ul className="m-0 flex list-none flex-col gap-2 p-0">
              {data.results.map((x, i) => {
                const impostor = x.metrics.impostor_mean_score ?? 0;
                const genuine = x.metrics.genuine_mean_score ?? 0;
                return (
                  <li
                    key={x.model_id}
                    className="grid grid-cols-[minmax(7rem,14rem)_1fr_9rem] items-center gap-3 text-sm"
                  >
                    <span>{label(x)}</span>
                    <span
                      aria-hidden="true"
                      className="relative h-4 rounded border border-line bg-side"
                    >
                      <span
                        className={`absolute top-0 h-full ${SERIES[i % SERIES.length]}`}
                        style={{
                          left: `${Math.max(0, impostor) * 100}%`,
                          width: `${Math.max(0, genuine - impostor) * 100}%`,
                        }}
                      />
                    </span>
                    <span className="text-right tabular-nums">
                      {t("benchmark.results.impostor")}{" "}
                      {decimal(language, impostor)} →{" "}
                      {t("benchmark.results.genuine")}{" "}
                      {decimal(language, genuine)}
                    </span>
                  </li>
                );
              })}
            </ul>
          </section>
        </>
      )}
      <section aria-labelledby="next-title" className={card}>
        <h3 id="next-title" className="text-base font-semibold">
          {t("benchmark.results.next")}
        </h3>
        <p className="text-sm text-mute">{t("benchmark.results.nextBody")}</p>
        <Link to="/admin/voice" className={`${buttonClass()} self-start`}>
          {t("benchmark.results.openVoice")}
        </Link>
      </section>
    </div>
  );
}
