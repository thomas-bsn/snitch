import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { FlaskConical } from "lucide-react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useDocumentTitle } from "../../components/useDocumentTitle";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";
import { useSession } from "../../session";
import {
  EXECUTABLE_STAGES,
  STAGES,
  isRunning,
  loadDatasets,
  loadModels,
  loadRuns,
  loadTranscriptionModels,
  type Dataset,
  type Model,
  type ModelStage,
  type Run,
  type Stage,
} from "./api";
import {
  ChainComposer,
  duplicateOf,
  emptyChains,
  runnableChains,
  type ChainDraft,
} from "./ChainComposer";
import { CorpusBuilder } from "./CorpusBuilder";
import { minutes } from "./format";

const card = "flex flex-col gap-3 rounded-xl border border-line bg-surface p-4";
const MAX_MODELS = 4;

/** Prepare and start a benchmark: a corpus, a stage, the models to compare. Reserved to administrators. */
export function BenchmarkRunPage() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const navigate = useNavigate();
  useDocumentTitle(t("admin.tabs.benchmark"));
  const datasets = useLoaded(loadDatasets);
  const [stage, setStage] = useState<Stage>("fingerprint");
  // The end-to-end stage composes diarization models with transcription models.
  const modelStage: ModelStage =
    stage === "transcription" || stage === "diarization"
      ? stage
      : stage === "e2e"
        ? "diarization"
        : "fingerprint";
  const isE2e = stage === "e2e";
  const loadStageModels = useCallback(
    () => loadModels(modelStage),
    [modelStage],
  );
  const models = useLoaded(loadStageModels);
  const transcriptionModels = useLoaded(loadTranscriptionModels);
  const runs = useLoaded(loadRuns);
  const [chains, setChains] = useState<ChainDraft[]>(emptyChains);
  const [datasetId, setDatasetId] = useState<number>();
  // One choice per stage, so coming back to a stage finds its models ticked.
  const [pickedByStage, setPickedByStage] = useState<
    Partial<Record<ModelStage, Set<string>>>
  >({});
  const picked = pickedByStage[modelStage] ?? new Set<string>();
  // Transcription models the administrator named: only a Worker reaches the service, so it cannot be listed here.
  const [named, setNamed] = useState<string[]>([]);
  const [nameDraft, setNameDraft] = useState("");
  const [building, setBuilding] = useState(false);
  const [confirmCancel, setConfirmCancel] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  const list: Dataset[] =
    datasets.state.status === "ready" ? datasets.state.value.datasets : [];
  const dataset = list.find((d) => d.id === datasetId) ?? list[0];
  const active: Run | undefined =
    runs.state.status === "ready"
      ? runs.state.value.runs.find(isRunning)
      : undefined;

  // While a run waits for its tasks the server settles it when read: poll, then show its result.
  const { reload: reloadRuns } = runs;
  const { reload: reloadModels } = models;
  const { reload: reloadTranscription } = transcriptionModels;
  const watched = useRef<number | undefined>(undefined);
  const allRuns = runs.state.status === "ready" ? runs.state.value.runs : [];
  useEffect(() => {
    if (active) {
      watched.current = active.id;
      const timer = setInterval(() => {
        reloadRuns();
        reloadModels(); // a model can lose its Worker while the run waits
        reloadTranscription();
      }, 3000);
      return () => clearInterval(timer);
    }
    if (watched.current !== undefined) {
      const id = watched.current;
      watched.current = undefined;
      // Only a finished run has something to show: a cancelled or failed one stays here.
      if (allRuns.find((r) => r.id === id)?.status === "completed")
        void navigate(`/admin/benchmark/results?run=${id}`);
    }
  }, [
    active,
    allRuns,
    reloadRuns,
    reloadModels,
    reloadTranscription,
    navigate,
  ]);

  const supported = (s: Stage) => dataset?.stages.includes(s) ?? false;
  const stageState = (s: Stage) =>
    !(EXECUTABLE_STAGES as Stage[]).includes(s)
      ? "soon"
      : !supported(s)
        ? "notInCorpus"
        : "ok";
  const served: Model[] =
    models.state.status === "ready" ? models.state.value.models : [];
  const transcriptionServed: Model[] = isE2e
    ? transcriptionModels.state.status === "ready"
      ? transcriptionModels.state.value.models
      : []
    : served;
  const showsNamed = stage === "transcription" || isE2e;
  const transcribers: Model[] = showsNamed
    ? [
        ...transcriptionServed,
        ...named
          .filter((name) => !transcriptionServed.some((m) => m.id === name))
          .map((name) => ({
            id: name,
            label: name,
            repo: name,
            revision: "",
            engine: "",
            active: false,
            ready_workers: -1, // unknown: the Worker checks the name when the task starts
            total_workers: 0,
          })),
      ]
    : [];
  const modelList: Model[] = stage === "transcription" ? transcribers : served;
  const runnable = runnableChains(chains, served);
  const hasDuplicate = chains.some((_, i) => {
    const twin = duplicateOf(chains, i);
    return twin !== -1 && twin !== i;
  });
  // A fingerprint model that lost its last Worker no longer counts, even if it stays ticked.
  const chosen = modelList.filter(
    (m) =>
      picked.has(m.id) &&
      (modelStage === "transcription" || m.ready_workers > 0),
  );
  const startReason = !dataset
    ? t("benchmark.run.needCorpus")
    : stageState(stage) !== "ok"
      ? t("benchmark.run.stageNotInCorpus")
      : active
        ? t("benchmark.run.alreadyRunning")
        : isE2e
          ? hasDuplicate
            ? t("benchmark.run.needDistinctChains")
            : runnable.length < chains.length
              ? t("benchmark.run.needCompleteChains")
              : undefined
          : chosen.length < 2
            ? t("benchmark.run.needModels")
            : undefined;

  const toggle = (id: string) =>
    setPickedByStage((all) => {
      const next = new Set(all[modelStage] ?? []);
      if (next.has(id)) next.delete(id);
      else if (next.size < MAX_MODELS) next.add(id);
      return { ...all, [modelStage]: next };
    });

  const nameFree = (name: string) =>
    name !== "" &&
    name.length <= 64 &&
    !transcribers.some((m) => m.id === name);
  const nameOk = nameFree(nameDraft.trim());
  const addName = () => {
    const name = nameDraft.trim();
    if (!nameOk) return;
    setNamed((current) => [...current, name]);
    setNameDraft("");
    toggle(name);
  };
  // A model named from the chains goes to the first chain still waiting for a transcription model.
  const nameForChain = (name: string) => {
    setNamed((current) => [...current, name]);
    const open = chains.find((c) => c.transcription === "");
    if (open)
      setChains(
        chains.map((c) => (c === open ? { ...c, transcription: name } : c)),
      );
  };

  const start = async () => {
    if (!dataset) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.POST("/api/admin/benchmarks/runs", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          dataset_id: dataset.id,
          ...(isE2e
            ? {
                stage: "e2e" as const,
                chains: runnable.map((c) => ({
                  diarization_model_id: c.diarization,
                  transcription_model: c.transcription,
                })),
              }
            : {
                stage: modelStage,
                model_ids: chosen.map((m) => m.id),
              }),
        },
      });
      if (result.data) {
        setChains(emptyChains());
        runs.reload();
      } else setFailure(errorCode(result.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  };

  const cancel = async () => {
    if (!active) return;
    setBusy(true);
    try {
      const result = await api.POST("/api/admin/benchmarks/runs/{id}/cancel", {
        params: {
          path: { id: active.id },
          header: mutation(session?.csrf_token),
        },
      });
      if (!result.data) setFailure(errorCode(result.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setConfirmCancel(false);
      watched.current = undefined; // a cancelled run has no result to open
      runs.reload();
    }
  };

  const failed =
    datasets.state.status === "failed"
      ? datasets.state.code
      : models.state.status === "failed"
        ? models.state.code
        : undefined;

  return (
    <div className="flex flex-col gap-4">
      {failed && (
        <div className="flex flex-col items-start gap-2">
          <ErrorAlert>
            {t("benchmark.run.loadFailed")} {error(failed)}
          </ErrorAlert>
          <button
            type="button"
            className={buttonClass()}
            onClick={() => {
              datasets.reload();
              models.reload();
            }}
          >
            {t("benchmark.run.retry")}
          </button>
        </div>
      )}
      {active && (
        <section
          aria-labelledby="running-title"
          className={`${card} border-accent`}
        >
          <div className="flex flex-wrap items-center gap-2">
            <h2 id="running-title" className="text-base font-semibold">
              {t("benchmark.run.runningTitle", { corpus: active.dataset_name })}
            </h2>
            <span className="flex-1" />
            <button
              type="button"
              className={buttonClass("danger")}
              disabled={busy}
              onClick={() => setConfirmCancel(true)}
            >
              {t("benchmark.run.cancel")}
            </button>
          </div>
          <p>
            <strong>
              {t("benchmark.run.progress", {
                done: active.progress.done,
                total: active.progress.jobs,
              })}
            </strong>
          </p>
          <div
            role="progressbar"
            aria-label={t("benchmark.run.progressLabel")}
            aria-valuemin={0}
            aria-valuemax={active.progress.jobs}
            aria-valuenow={active.progress.done}
            className="h-2 overflow-hidden rounded-full bg-line"
          >
            <div
              className="h-full bg-accent-solid motion-safe:transition-[width]"
              style={{
                width: `${active.progress.jobs ? (active.progress.done / active.progress.jobs) * 100 : 0}%`,
              }}
            />
          </div>
          {active.progress.failed > 0 && (
            <StatusNotice>
              {t("benchmark.run.failedTasks", {
                count: active.progress.failed,
              })}
            </StatusNotice>
          )}
          {active.progress.errors.length > 0 && (
            <div>
              <h3 className="text-sm font-semibold">
                {t("benchmark.run.taskErrors")}
              </h3>
              <ul className="m-0 list-none p-0 text-sm text-mute">
                {active.progress.errors.map((e) => (
                  <li key={`${e.job_id}-${e.model_id}`}>
                    #{e.job_id} · {e.model_id} · {e.error}
                  </li>
                ))}
              </ul>
            </div>
          )}
          <p className="text-sm text-mute">{t("benchmark.run.workersWork")}</p>
        </section>
      )}

      <section aria-labelledby="corpus-title" className={card}>
        <div className="flex flex-wrap items-center gap-2">
          <h2 id="corpus-title" className="text-base font-semibold">
            {t("benchmark.run.corpusTitle")}
          </h2>
          <span className="flex-1" />
          <button
            type="button"
            className={buttonClass()}
            onClick={() => setBuilding(true)}
          >
            {t("benchmark.run.createCorpus")}
          </button>
        </div>
        <p className="text-sm text-mute">{t("benchmark.run.corpusLead")}</p>
        {datasets.state.status === "loading" && (
          <p role="status">{t("benchmark.run.loading")}</p>
        )}
        {datasets.state.status === "ready" && list.length === 0 && (
          <p className="rounded-lg bg-side px-3 py-6 text-center text-mute">
            {t("benchmark.run.corpusNone")}
          </p>
        )}
        {list.length > 0 && (
          <div
            role="radiogroup"
            aria-label={t("benchmark.run.corpusLabel")}
            className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
          >
            {list.map((d) => (
              <label
                key={d.id}
                className="relative flex cursor-pointer flex-col gap-1 rounded-xl border border-line-strong p-3 has-[:checked]:border-accent has-[:checked]:bg-accent-soft has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-accent"
              >
                <input
                  type="radio"
                  name="corpus"
                  checked={dataset?.id === d.id}
                  onChange={() => setDatasetId(d.id)}
                  className="absolute top-3 right-3 size-4"
                />
                <strong className="pr-6">{d.name}</strong>
                <span className="text-sm text-mute">
                  {t("benchmark.run.corpusClips", {
                    test: d.test_clips,
                    reference: d.reference_clips,
                  })}
                </span>
                <span className="text-sm text-mute">
                  {t("benchmark.run.corpusPersons", {
                    persons: d.persons,
                    minutes: minutes(d.duration_seconds),
                  })}
                </span>
                {d.reference_sources.map((source) => (
                  <span key={source} className="text-sm text-mute">
                    {source === "import"
                      ? t("benchmark.run.corpusSourcesImport")
                      : t("benchmark.run.corpusSourcesProduction")}
                  </span>
                ))}
                {d.verified_clips > 0 && (
                  <span className="text-sm text-mute">
                    {t("benchmark.run.corpusCertified", {
                      count: d.verified_clips,
                    })}
                  </span>
                )}
              </label>
            ))}
          </div>
        )}
      </section>

      <section aria-labelledby="stage-title" className={card}>
        <h2 id="stage-title" className="text-base font-semibold">
          {t("benchmark.run.stageTitle")}
        </h2>
        <div
          role="group"
          aria-label={t("benchmark.run.stageLabel")}
          className="flex flex-wrap gap-2"
        >
          {STAGES.map((s) => {
            const state = stageState(s);
            return (
              <button
                key={s}
                type="button"
                aria-pressed={stage === s}
                disabled={state !== "ok"}
                onClick={() => setStage(s)}
                className={`flex min-h-11 flex-col items-start rounded-lg border px-3 py-1.5 text-left text-sm md:min-h-9 disabled:opacity-60 ${
                  stage === s
                    ? "border-accent bg-accent-soft text-accent-ink"
                    : "border-line-strong bg-surface"
                }`}
              >
                <strong>
                  {t(`benchmark.run.stageNames.${s}` as MessageKey)}
                </strong>
                {state === "soon" && (
                  <span className="text-xs text-mute">
                    {t("benchmark.run.stageSoon")}
                  </span>
                )}
                {state === "notInCorpus" && (
                  <span className="text-xs text-mute">
                    {t("benchmark.run.stageNotInCorpus")}
                  </span>
                )}
              </button>
            );
          })}
        </div>
        <p className="text-sm text-mute">
          {t(`benchmark.run.stageHints.${stage}` as MessageKey)}
        </p>
        <p className="text-sm text-mute">
          {isE2e
            ? t("benchmark.run.chainsLead")
            : t("benchmark.run.modelsLead")}
        </p>
        {isE2e && (
          <ChainComposer
            chains={chains}
            onChange={setChains}
            diarizers={served}
            transcribers={transcribers}
            nameOk={nameFree}
            onName={nameForChain}
          />
        )}
        {models.state.status === "loading" && (
          <p role="status">{t("benchmark.run.loading")}</p>
        )}
        {models.state.status === "ready" &&
          !isE2e &&
          modelList.length === 0 && (
            <p className="rounded-lg bg-side px-3 py-6 text-center text-mute">
              {t("benchmark.run.modelsEmpty")}
            </p>
          )}
        {!isE2e && modelList.length > 0 && (
          <div
            role="group"
            aria-label={t("benchmark.run.modelsLabel")}
            className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
          >
            {modelList.map((m) => {
              const nowhere = m.ready_workers === 0;
              return (
                <label
                  key={m.id}
                  className={`relative flex flex-col gap-1 rounded-xl border p-3 has-[:checked]:border-accent has-[:checked]:bg-accent-soft has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-accent ${
                    nowhere
                      ? "cursor-not-allowed border-line bg-side opacity-70"
                      : "cursor-pointer border-line-strong"
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={picked.has(m.id)}
                    disabled={nowhere}
                    onChange={() => toggle(m.id)}
                    className="absolute top-3 right-3 size-4"
                  />
                  <strong className="pr-6">{m.label}</strong>
                  {modelStage !== "transcription" && (
                    <span className="text-sm text-mute">
                      {m.engine} · {m.revision.slice(0, 7)}
                    </span>
                  )}
                  <span className="text-sm text-mute">
                    {m.ready_workers < 0
                      ? t("benchmark.run.modelUnchecked")
                      : nowhere
                        ? t("benchmark.run.modelNowhere")
                        : modelStage === "transcription"
                          ? t("benchmark.run.modelServiceReady", {
                              ready: m.ready_workers,
                              total: m.total_workers,
                            })
                          : t("benchmark.run.modelReady", {
                              ready: m.ready_workers,
                              total: m.total_workers,
                            })}
                    {m.active && (
                      <span className="ml-2 rounded-full bg-ok-bg px-2 py-0.5 text-xs font-medium text-ok-fg">
                        {t("benchmark.run.modelActive")}
                      </span>
                    )}
                  </span>
                </label>
              );
            })}
          </div>
        )}
        {stage === "transcription" && (
          <form
            className="flex flex-col gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              addName();
            }}
          >
            <label className="flex flex-col gap-1">
              <span className="text-sm font-medium">
                {t("benchmark.run.addModel")}
              </span>
              <span className="text-xs text-mute">
                {t("benchmark.run.addModelHelp")}
              </span>
              <span className="flex flex-wrap gap-2">
                <input
                  value={nameDraft}
                  maxLength={64}
                  onChange={(event) => setNameDraft(event.target.value)}
                  placeholder="Systran/faster-whisper-large-v3"
                  className="h-11 min-w-0 flex-1 rounded-lg border border-line-strong bg-surface px-3 md:h-9"
                />
                <button
                  type="submit"
                  className={buttonClass()}
                  disabled={!nameOk}
                >
                  {t("benchmark.run.addModelButton")}
                </button>
              </span>
            </label>
          </form>
        )}
      </section>

      <section aria-labelledby="launch-title" className={card}>
        <h2 id="launch-title" className="text-base font-semibold">
          {t("benchmark.run.launchTitle")}
        </h2>
        <p className="text-sm">
          {dataset && (
            <>
              <strong>{dataset.name}</strong> ·{" "}
              {t(`benchmark.run.stageNames.${stage}` as MessageKey)} ·{" "}
              {isE2e
                ? t("benchmark.run.launchSummaryChains", {
                    chains: runnable.length,
                    clips: dataset.test_clips + dataset.reference_clips,
                  })
                : t("benchmark.run.launchSummary", {
                    models: chosen.length,
                    clips: dataset.test_clips + dataset.reference_clips,
                  })}
            </>
          )}
        </p>
        <p className="text-sm text-mute">{t("benchmark.run.noProduction")}</p>
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        <div className="flex flex-wrap items-center gap-3">
          <button
            type="button"
            className={buttonClass("primary")}
            disabled={busy || startReason !== undefined}
            onClick={() => void start()}
          >
            <FlaskConical className="size-4" aria-hidden="true" />
            {busy ? t("benchmark.run.launching") : t("benchmark.run.launch")}
          </button>
          {startReason && (
            <span className="text-sm text-mute">{startReason}</span>
          )}
        </div>
      </section>

      <CorpusBuilder
        key={building ? "open" : "closed"}
        open={building}
        onClose={() => setBuilding(false)}
        onCreated={(id) => {
          setBuilding(false);
          setDatasetId(id);
          datasets.reload();
        }}
      />
      <ConfirmDialog
        open={confirmCancel}
        onOpenChange={setConfirmCancel}
        title={t("benchmark.run.cancelTitle")}
        cancelLabel={t("benchmark.run.cancelKeep")}
        confirmLabel={
          busy ? t("benchmark.run.cancelling") : t("benchmark.run.cancel")
        }
        onConfirm={() => void cancel()}
        danger
        busy={busy}
      >
        <p>{t("benchmark.run.cancelBody")}</p>
      </ConfirmDialog>
    </div>
  );
}
