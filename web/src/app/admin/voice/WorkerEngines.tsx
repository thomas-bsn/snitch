import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus } from "lucide-react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded, type Result } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { VoiceEngines } from "./VoiceEngines";
import { WorkerEngineAdd } from "./WorkerEngineAdd";
import { WorkerEngineReplace } from "./WorkerEngineReplace";
import { isSuperAdministrator } from "../../roles";
import { WorkerEngineModels } from "./WorkerEngineModels";
import { WorkerEngineUninstall } from "./WorkerEngineUninstall";
import { WorkerHardware } from "./WorkerHardware";
import {
  engineSteps,
  engineTargets,
  inProgress,
  planOrders,
  type EngineEntry,
  type EngineWorker,
  type Release,
  type Refusal,
} from "./engineOrders";

type Filter = "all" | "todo" | "offline";
type Panel = "installed" | "todo" | "running";

async function fetchView(): Promise<Result<EngineWorker[]>> {
  try {
    const { data, error } = await api.GET("/api/admin/voice-engines/workers");
    return data
      ? { ok: true, value: data.workers }
      : { ok: false, code: errorCode(error) ?? "unknown" };
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

async function fetchReleases(): Promise<Result<Release[]>> {
  try {
    const { data, error } = await api.GET("/api/admin/voice-engines");
    return data
      ? { ok: true, value: data.engines }
      : { ok: false, code: errorCode(error) ?? "unknown" };
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

const mebibytes = (bytes: number) => Math.ceil(bytes / 1024 ** 2);
const counts = (w: EngineWorker) => ({
  failed: w.engines.filter((e) => e.state === "failed").length,
  running: w.engines.filter(inProgress).length,
  ready: w.engines.filter((e) => e.state === "ready").length,
});

/**
 * Engines, Worker first: the Workers on the left, on the right only the engines present on the chosen
 * Worker. Every change is an explicit order; looking at this screen never installs anything.
 */
export function WorkerEngines() {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const owner = user !== null && isSuperAdministrator(user);
  const view = useLoaded(useCallback(fetchView, []));
  const registry = useLoaded(useCallback(fetchReleases, []));
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<Filter>("all");
  const [chosen, setChosen] = useState("");
  const [panel, setPanel] = useState<Panel>("installed");
  const [checked, setChecked] = useState<string[]>([]);
  const [picked, setPicked] = useState<string[]>([]);
  const [scope, setScope] = useState<"this" | "checked">("this");
  const [adding, setAdding] = useState(false);
  const [uninstalling, setUninstalling] = useState<EngineEntry>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [outcome, setOutcome] = useState<string[]>([]);
  const [replacing, setReplacing] = useState<EngineEntry>();
  const [notice, setNotice] = useState<string>();
  const workers = view.state.status === "ready" ? view.state.value : [];
  const releases =
    registry.state.status === "ready" ? registry.state.value : [];
  const reloadView = view.reload;
  const running = workers.some((w) => w.engines.some(inProgress));
  // A running order moves on its own: the view follows it without a click.
  useEffect(() => {
    if (!running) return;
    const timer = setInterval(reloadView, 4000);
    return () => clearInterval(timer);
  }, [running, reloadView]);

  const visible = workers.filter((w) => {
    const c = counts(w);
    return (
      w.name.toLowerCase().includes(query.toLowerCase()) &&
      (filter === "all" ||
        (filter === "todo" && c.failed > 0) ||
        (filter === "offline" && !w.connected))
    );
  });
  visible.sort((a, b) => a.name.localeCompare(b.name));
  const current = workers.find((w) => w.worker_id === chosen) ?? workers[0];
  const entries = useMemo(() => {
    const all = current?.engines ?? [];
    const shown =
      panel === "todo"
        ? all.filter((e) => e.state === "failed")
        : panel === "running"
          ? all.filter(inProgress)
          : all;
    const groups = new Map<string, EngineEntry[]>();
    for (const e of shown)
      groups.set(e.engine, [...(groups.get(e.engine) ?? []), e]);
    return [...groups].sort(([a], [b]) => a.localeCompare(b));
  }, [current, panel]);
  const tabCount = (p: Panel) =>
    !current
      ? 0
      : p === "todo"
        ? counts(current).failed
        : p === "running"
          ? counts(current).running
          : current.engines.length;

  const targets = engineTargets(workers, current, checked, scope);
  const chosenReleases = picked
    .map((d) => releases.find((r) => r.digest === d))
    .filter((r): r is Release => r !== undefined);
  const plan = planOrders(targets, chosenReleases);

  /** One request per release; each Worker is decided on its own and a refusal never hides another result. */
  async function order(items: { worker: EngineWorker; release: Release }[]) {
    setBusy(true);
    setFailure(undefined);
    setOutcome([]);
    const refused: string[] = [];
    try {
      for (const digest of new Set(items.map((i) => i.release.digest))) {
        const result = await api.POST("/api/admin/voice-engine-orders", {
          params: { header: mutation(session?.csrf_token) },
          body: {
            digest,
            intent: "install",
            orders: items
              .filter((i) => i.release.digest === digest)
              .map((i) => ({
                worker_id: i.worker.worker_id,
                idempotency_key: crypto.randomUUID(),
              })),
          },
        });
        if (result.error) {
          setFailure(errorCode(result.error) ?? "unknown");
          break;
        }
        for (const r of result.data.results)
          if (r.status === "refused" && r.cause_code)
            refused.push(
              `${workers.find((w) => w.worker_id === r.worker_id)?.name ?? r.worker_id} : ${error(r.cause_code)}`,
            );
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setOutcome(refused);
      setBusy(false);
      setPicked([]);
      reloadView();
    }
  }

  /** An on-demand engine: one request for the family and version, decided per Worker by the server. */
  async function orderFamily(
    family: string,
    version: string,
    workerIds: string[],
  ) {
    setBusy(true);
    setFailure(undefined);
    setOutcome([]);
    const refused: string[] = [];
    try {
      const result = await api.POST(
        "/api/admin/voice-engines/on-demand-orders",
        {
          params: { header: mutation(session?.csrf_token) },
          body: {
            family,
            version,
            orders: workerIds.map((worker_id) => ({
              worker_id,
              idempotency_key: crypto.randomUUID(),
            })),
          },
        },
      );
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else
        for (const r of result.data.results)
          if (r.status === "refused" && r.cause_code)
            refused.push(
              `${workers.find((w) => w.worker_id === r.worker_id)?.name ?? r.worker_id} : ${error(r.cause_code)}`,
            );
    } catch {
      setFailure("unavailable");
    } finally {
      setOutcome(refused);
      setBusy(false);
      reloadView();
    }
  }

  /** Retry, cancel and clean up are explicit requests; the server decides from the real state and answers the same way when repeated. */
  async function recover(
    entry: EngineEntry,
    verb: "retry" | "cancel" | "cleanup",
  ) {
    if (!current) return;
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    const path = {
      id: current.worker_id,
      digest: entry.digest,
    };
    const header = mutation(session?.csrf_token);
    try {
      const result =
        verb === "retry"
          ? await api.POST(
              "/api/admin/workers/{id}/voice-engines/{digest}/retry",
              {
                params: { path, header },
                body: { idempotency_key: crypto.randomUUID() },
              },
            )
          : verb === "cancel"
            ? await api.POST(
                "/api/admin/workers/{id}/voice-engines/{digest}/cancel",
                { params: { path, header } },
              )
            : await api.POST(
                "/api/admin/workers/{id}/voice-engines/{digest}/cleanup",
                { params: { path, header } },
              );
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else if (verb === "cancel")
        setNotice(
          "status" in result.data && result.data.status === "cancelling"
            ? t("workerEngines.action.cancelling")
            : t("workerEngines.cause.cancelled"),
        );
      else if (verb === "cleanup")
        setNotice(
          "status" in result.data && result.data.status === "done"
            ? t("workerEngines.action.cleanupDone")
            : t("workerEngines.action.cleanupRequested"),
        );
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      reloadView();
    }
  }

  /** An explicit removal of one release on one Worker; the Worker acts once nothing depends on it. */
  async function uninstall(entry: EngineEntry, key: string) {
    if (!current) return;
    setBusy(true);
    setFailure(undefined);
    setOutcome([]);
    try {
      const result = await api.POST("/api/admin/voice-engine-orders", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          digest: entry.digest,
          intent: "uninstall",
          orders: [{ worker_id: current.worker_id, idempotency_key: key }],
        },
      });
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else
        setOutcome(
          result.data.results
            .filter((r) => r.status === "refused" && r.cause_code)
            .map((r) => error(r.cause_code ?? "unknown")),
        );
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setUninstalling(undefined);
      reloadView();
    }
  }

  const refusalText = (r: Refusal) => t(`workerEngines.refusal.${r}`);
  const planSummary = (
    <div className="engine-plan" aria-live="polite">
      <span>
        {t("workerEngines.plan", {
          count: plan.accepted.length,
          disk: mebibytes(plan.disk),
        })}
      </span>
      {plan.refusals.map(([reason, n]) => (
        <span key={reason} className="engine-refused">
          {t("workerEngines.refused", {
            count: n,
            reason: refusalText(reason),
          })}
        </span>
      ))}
    </div>
  );

  return (
    <div className="engine-view">
      {view.state.status === "loading" && (
        <p role="status">{t("app.loading")}</p>
      )}
      {view.state.status === "failed" && (
        <ErrorAlert>{error(view.state.code)}</ErrorAlert>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <p role="status">{notice}</p>}
      {outcome.length > 0 && (
        <ErrorAlert>
          {t("workerEngines.someRefused")}
          <ul>
            {outcome.map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
        </ErrorAlert>
      )}
      {view.state.status === "ready" && (
        <div className="engine-split">
          <aside
            className="engine-workers"
            aria-label={t("workerEngines.workers")}
          >
            <input
              type="search"
              className="engine-search"
              value={query}
              placeholder={t("workerEngines.search")}
              aria-label={t("workerEngines.search")}
              onChange={(e) => setQuery(e.target.value)}
            />
            <div className="engine-pills">
              {(["all", "todo", "offline"] as const).map((f) => (
                <button
                  type="button"
                  key={f}
                  className="engine-pill"
                  aria-pressed={filter === f}
                  onClick={() => setFilter(f)}
                >
                  {t(`workerEngines.filter.${f}`)}
                  {" ("}
                  {f === "all"
                    ? workers.length
                    : f === "todo"
                      ? workers.filter((w) => counts(w).failed > 0).length
                      : workers.filter((w) => !w.connected).length}
                  {")"}
                </button>
              ))}
            </div>
            {visible.length === 0 && (
              <p className="engine-empty">{t("workerEngines.noWorkers")}</p>
            )}
            {visible.map((w) => {
              const c = counts(w);
              return (
                <div
                  key={w.worker_id}
                  className="engine-worker"
                  aria-current={current?.worker_id === w.worker_id}
                >
                  <input
                    type="checkbox"
                    checked={checked.includes(w.worker_id)}
                    aria-label={t("workerEngines.include", {
                      worker: w.name,
                    })}
                    onChange={(e) =>
                      setChecked((list) =>
                        e.target.checked
                          ? [...list, w.worker_id]
                          : list.filter((id) => id !== w.worker_id),
                      )
                    }
                  />
                  <button
                    type="button"
                    onClick={() => {
                      setChosen(w.worker_id);
                      setPicked([]);
                    }}
                  >
                    <strong>{w.name}</strong>
                    <span>
                      {t(
                        w.connected
                          ? "voiceWorkspace.online"
                          : "voiceWorkspace.offline",
                      )}
                    </span>
                  </button>
                  <span className="engine-chips">
                    {c.failed > 0 && (
                      <span className="engine-badge bad">{c.failed}</span>
                    )}
                    {c.running > 0 && (
                      <span className="engine-badge run">{c.running}</span>
                    )}
                    {c.ready > 0 && (
                      <span className="engine-badge ok">{c.ready}</span>
                    )}
                  </span>
                </div>
              );
            })}
          </aside>
          <section className="engine-detail" aria-live="polite">
            {current && (
              <>
                <div className="engine-head">
                  <h3>{current.name}</h3>
                  <button
                    type="button"
                    className="voice-btn primary"
                    onClick={() => setAdding(true)}
                  >
                    <Plus size={16} aria-hidden />
                    {t("workerEngines.add")}
                  </button>
                </div>
                <WorkerHardware worker={current} />
                {!current.connected && (
                  <p className="engine-note">
                    {t("workerEngines.offlineNote")}
                  </p>
                )}
                <div className="engine-tabs" role="tablist">
                  {(["installed", "todo", "running"] as const).map((p) => (
                    <button
                      type="button"
                      key={p}
                      role="tab"
                      aria-selected={panel === p}
                      onClick={() => setPanel(p)}
                    >
                      {t(`workerEngines.tab.${p}`)}
                      <span>{tabCount(p)}</span>
                    </button>
                  ))}
                </div>
                {entries.length === 0 && (
                  <p className="engine-empty">
                    {t(
                      current.engines.length === 0
                        ? "workerEngines.none"
                        : "workerEngines.emptyTab",
                    )}
                  </p>
                )}
                {entries.map(([family, list]) => (
                  <div key={family} className="engine-family">
                    <h4>{family}</h4>
                    {list.map((e) => (
                      <EngineRow
                        key={e.digest}
                        entry={e}
                        checked={picked.includes(e.digest)}
                        selectable={e.state === "failed"}
                        busy={busy}
                        keepsPrevious={current.engines.some(
                          (o) =>
                            o.engine === e.engine &&
                            o.digest !== e.digest &&
                            o.state === "ready",
                        )}
                        onRetry={() => void recover(e, "retry")}
                        onCancel={() => void recover(e, "cancel")}
                        onCleanup={() => void recover(e, "cleanup")}
                        onReplace={() => setReplacing(e)}
                        onUninstall={
                          e.state === "ready" || e.state === "failed"
                            ? () => setUninstalling(e)
                            : undefined
                        }
                        onCheck={(on) =>
                          setPicked((p) =>
                            on
                              ? [...p, e.digest]
                              : p.filter((d) => d !== e.digest),
                          )
                        }
                      />
                    ))}
                  </div>
                ))}
                <WorkerEngineModels
                  workerId={current.worker_id}
                  releases={releases}
                />
                {picked.length > 0 && (
                  <div className="engine-bar">
                    <label>
                      {t("workerEngines.scope")}
                      <select
                        value={scope}
                        onChange={(e) =>
                          setScope(e.target.value as "this" | "checked")
                        }
                      >
                        <option value="this">
                          {t("workerEngines.scopeThis")}
                        </option>
                        <option value="checked">
                          {t("workerEngines.scopeChecked", {
                            count: checked.length,
                          })}
                        </option>
                      </select>
                    </label>
                    {planSummary}
                    <button
                      type="button"
                      className="voice-btn primary"
                      disabled={busy || plan.accepted.length === 0}
                      onClick={() => void order(plan.accepted)}
                    >
                      {t("workerEngines.apply", {
                        count: plan.accepted.length,
                      })}
                    </button>
                  </div>
                )}
              </>
            )}
          </section>
        </div>
      )}
      {uninstalling && current && (
        <WorkerEngineUninstall
          worker={current}
          entry={uninstalling}
          busy={busy}
          onClose={() => setUninstalling(undefined)}
          onConfirm={(key) => uninstall(uninstalling, key)}
        />
      )}
      {adding && !replacing && current && (
        <WorkerEngineAdd
          worker={current}
          workers={workers}
          checked={checked}
          busy={busy}
          onClose={() => setAdding(false)}
          onOrder={async (family, version, ids) => {
            await orderFamily(family, version, ids);
            setAdding(false);
          }}
        />
      )}
      {replacing && current && (
        <WorkerEngineReplace
          replacing={replacing}
          worker={current}
          workers={workers}
          checked={checked}
          releases={releases}
          registryError={
            registry.state.status === "failed" ? registry.state.code : undefined
          }
          busy={busy}
          onClose={() => {
            setAdding(false);
            setReplacing(undefined);
          }}
          onOrder={async (items) => {
            await order(items);
            setAdding(false);
            setReplacing(undefined);
          }}
        />
      )}
      {owner && (
        <details className="engine-advanced">
          <summary>{t("workerEngines.advanced.title")}</summary>
          <p className="text-sm text-ink-soft">
            {t("workerEngines.advanced.lead")}
          </p>
          <VoiceEngines />
        </details>
      )}
    </div>
  );
}

function EngineRow({
  entry,
  checked,
  selectable,
  onCheck,
  onUninstall,
  busy,
  keepsPrevious,
  onRetry,
  onCancel,
  onCleanup,
  onReplace,
}: {
  entry: EngineEntry;
  checked: boolean;
  selectable: boolean;
  onCheck: (on: boolean) => void;
  busy: boolean;
  keepsPrevious: boolean;
  onRetry: () => void;
  onCancel: () => void;
  onCleanup: () => void;
  onReplace: () => void;
  onUninstall?: () => void;
}) {
  const { t } = useI18n();
  const step = engineSteps.indexOf(entry.state as (typeof engineSteps)[number]);
  const name = `${entry.engine} ${entry.version}`;
  return (
    <div className="engine-row">
      <input
        type="checkbox"
        checked={checked}
        disabled={!selectable}
        aria-label={t("workerEngines.select", { engine: name })}
        onChange={(e) => onCheck(e.target.checked)}
      />
      <div>
        <strong>{entry.version}</strong>
        <span className="engine-sub">
          {entry.author_name &&
            t("workerEngines.orderedBy", { name: entry.author_name })}
          {" · "}
          {t("workerEngines.generation", { generation: entry.generation })}
        </span>
        {entry.revoked && (
          <span className="engine-warn">{t("workerEngines.revoked")}</span>
        )}
        {entry.order_intent === "uninstall" && (
          <span className="engine-sub">
            {entry.state === "removing" &&
              (entry.waiting && entry.waiting.models.length > 0
                ? t("workerEngines.uninstall.waiting", {
                    models: entry.waiting.models.map((m) => m.label).join(", "),
                  })
                : t("workerEngines.uninstall.waitingOrder"))}
            {entry.state === "removed" && t("workerEngines.uninstall.done")}
            {entry.order_status === "failed" &&
              entry.order_cause_code !== "" &&
              t("workerEngines.uninstall.refused", {
                cause: t(`workerEngines.cause.${entry.order_cause_code}`),
              })}
            {entry.author_name &&
              ` · ${t("workerEngines.uninstall.author", { name: entry.author_name })}`}
          </span>
        )}
        <details>
          <summary>{t("workerEngines.history")}</summary>
          <ol className="engine-history">
            {entry.history.map((h, i) => (
              <li key={i} className={h.ignored ? "ignored" : ""}>
                {new Date(h.created_at).toLocaleString()} ·{" "}
                {t(`workerEngines.state.${h.state}`)}
                {h.cause_code &&
                  ` · ${t(`workerEngines.cause.${h.cause_code}`)}`}
                {h.ignored &&
                  ` · ${t("workerEngines.ignored", { reason: h.reason })}`}
              </li>
            ))}
          </ol>
        </details>
      </div>
      <div className="engine-state">
        <span
          className={`engine-badge ${entry.state === "ready" ? "ok" : entry.state === "failed" ? "bad" : "run"}`}
        >
          {t(`workerEngines.state.${entry.state}`)}
        </span>
        {step >= 0 && step < engineSteps.length - 1 && (
          <>
            <div className="engine-steps" aria-hidden>
              {engineSteps.slice(0, -1).map((s, i) => (
                <i
                  key={s}
                  className={i < step ? "done" : i === step ? "now" : ""}
                />
              ))}
            </div>
            {entry.state === "downloading_runtime" && entry.bytes_total > 0 && (
              <progress value={entry.bytes_received} max={entry.bytes_total} />
            )}
            {entry.state === "requested" && (
              <span className="engine-sub">{t("workerEngines.delivery")}</span>
            )}
          </>
        )}
        {onUninstall && (
          <button type="button" className="voice-btn" onClick={onUninstall}>
            {t("workerEngines.uninstall.action")}
          </button>
        )}
        {entry.state === "failed" && entry.cause_code && (
          <span className="engine-error">
            {t(`workerEngines.cause.${entry.cause_code}`)}
          </span>
        )}
        {entry.state === "failed" && keepsPrevious && (
          <span className="engine-sub">
            {t("workerEngines.action.keepsPrevious")}
          </span>
        )}
        {entry.cancel_requested && (
          <span className="engine-sub">
            {t("workerEngines.action.cancelling")}
          </span>
        )}
        {entry.cleanup === "requested" && (
          <span className="engine-sub">
            {t("workerEngines.action.cleanupRequested")}
          </span>
        )}
        {entry.cleanup === "done" && (
          <span className="engine-sub">
            {t("workerEngines.action.cleanupDone")}
          </span>
        )}
        <div className="engine-actions">
          {inProgress(entry) &&
            entry.order_intent !== "uninstall" &&
            !entry.cancel_requested && (
              <button
                type="button"
                className="voice-btn"
                disabled={busy}
                aria-label={t("workerEngines.action.cancelLabel", {
                  engine: name,
                })}
                onClick={onCancel}
              >
                {t("workerEngines.action.cancel")}
              </button>
            )}
          {entry.state === "failed" && (
            <>
              {!entry.revoked && (
                <button
                  type="button"
                  className="voice-btn"
                  disabled={busy}
                  aria-label={t("workerEngines.action.retryLabel", {
                    engine: name,
                  })}
                  onClick={onRetry}
                >
                  {t("workerEngines.action.retry")}
                </button>
              )}
              <button
                type="button"
                className="voice-btn"
                disabled={busy}
                aria-label={t("workerEngines.action.replaceLabel", {
                  engine: name,
                })}
                onClick={onReplace}
              >
                {t("workerEngines.action.replace")}
              </button>
              {entry.cleanup === "" && entry.order_intent !== "uninstall" && (
                <button
                  type="button"
                  className="voice-btn"
                  disabled={busy}
                  aria-label={t("workerEngines.action.cleanupLabel", {
                    engine: name,
                  })}
                  onClick={onCleanup}
                >
                  {t("workerEngines.action.cleanup")}
                </button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
