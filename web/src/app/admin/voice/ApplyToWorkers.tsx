import { useCallback, useEffect, useState } from "react";
import type { components } from "../../../api.gen";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { api, errorCode, mutation } from "../../api/client";
import { buttonClass } from "../../components/buttons";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import type { Catalogue } from "./VoiceModelCards";

type Row = components["schemas"]["WorkerModelsRow"];
type Kind = components["schemas"]["WorkerModelCell"]["kind"];

type Order = { worker: Row; key: string; replaces?: string };
type Preview = components["schemas"]["VoiceModelOrderPreview"];

/**
 * "Apply to several Workers": an explicit action. One order is built per ticked Worker and listed before
 * anything is sent; each order is then sent on its own with its own key, and its outcome is shown line by line.
 */
export function ApplyToWorkers({
  open,
  onClose,
  workers,
  catalogue,
  kinds,
  initialModelId,
  initialWorkerIds = [],
  onChanged,
}: {
  open: boolean;
  onClose: () => void;
  workers: Row[];
  catalogue: Catalogue;
  kinds: Kind[];
  initialModelId?: string;
  initialWorkerIds?: string[];
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const headers = mutation(session?.csrf_token);
  const [kind, setKind] = useState<Kind>(
    catalogue.models.find((m) => m.id === initialModelId)?.kind ??
      "diarization",
  );
  const [modelId, setModelId] = useState(initialModelId ?? "");
  const [ticked, setTicked] = useState<string[]>([]);
  const [orders, setOrders] = useState<Order[]>();
  const [outcome, setOutcome] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [chosenRuntime, setChosenRuntime] = useState("");
  const [previews, setPreviews] = useState<Record<string, Preview>>({});
  const [prepareError, setPrepareError] = useState("");
  const [preparing, setPreparing] = useState(false);
  const { state: registry } = useLoaded(
    useCallback(async () => {
      try {
        const result = await api.GET("/api/admin/voice-engines");
        return result.data
          ? { ok: true as const, value: result.data.engines }
          : {
              ok: false as const,
              code: errorCode(result.error) ?? "unavailable",
            };
      } catch {
        return { ok: false as const, code: "unavailable" };
      }
    }, []),
  );
  const model = catalogue.models.find((m) => m.id === modelId);
  // A model that names an on-demand family never uses a signed release: the engine is ordered with it (ADR 0056).
  const onDemand = !!model?.family;
  const releases =
    registry.status === "ready" && !onDemand
      ? registry.value.filter(
          (r) =>
            r.manifest.engine === model?.engine && r.manifest.kind === kind,
        )
      : [];
  const available = releases.filter((r) => !r.revoked);
  const runtimeDigest = available.some((r) => r.digest === chosenRuntime)
    ? chosenRuntime
    : available[0]?.digest;
  const runtimeRequired =
    !onDemand &&
    (releases.length > 0 ||
      model?.engine === "nemo-titanet" ||
      model?.engine === "speechbrain-ecapa");
  const runtimeBlocked =
    !onDemand &&
    (registry.status !== "ready" || (runtimeRequired && !runtimeDigest));
  const models = catalogue.models.filter(
    (m) => m.kind === kind && m.enabled && m.supported,
  );
  const [confirmed, setConfirmed] = useState(false);
  const [transition, setTransition] =
    useState<components["schemas"]["FingerprintTransition"]>();
  useEffect(() => {
    let current = true;
    setTransition(undefined);
    if (kind !== "fingerprint") return;
    api
      .GET("/api/admin/voice-models/fingerprint-transition")
      .then((r) => {
        if (current && r.data?.open) setTransition(r.data.transition);
      })
      .catch(() => {});
    return () => {
      current = false;
    };
  }, [kind]);
  function replacement(w: Row) {
    const cell = w.models.find((m) => m.kind === kind);
    if (!cell || cell.state === "not_installed") return undefined;
    if (
      (cell.state === "ready" ||
        (cell.state === "failed" && cell.replacement_phase === "preparing")) &&
      (cell.replaces_model_id || cell.model_id) !== modelId &&
      (kind === "diarization" ||
        (transition?.to_model_id === modelId &&
          transition.from_model_id ===
            (cell.replaces_model_id || cell.model_id)))
    )
      return cell.replaces_model_id || cell.model_id;
    return undefined;
  }
  function reason(w: Row) {
    const cell = w.models.find((m) => m.kind === kind);
    if (
      !cell ||
      cell.state === "not_installed" ||
      replacement(w) ||
      (cell.state === "failed" && !cell.replaces_model_id)
    )
      return "";
    if (cell.model_id === modelId && cell.state === "ready")
      return t("voiceWorkspace.alreadyReady");
    return t(
      cell.state === "ready"
        ? "voiceWorkspace.occupied"
        : "voiceWorkspace.busyWorker",
    );
  }
  const eligible = workers.filter((w) => !reason(w));
  useEffect(() => {
    setModelId(
      catalogue.models.find((m) => m.id === initialModelId && m.kind === kind)
        ?.id ??
        models[0]?.id ??
        "",
    );
    setTicked(initialWorkerIds);
    setOrders(undefined);
    setOutcome({});
    setConfirmed(false);
    setPrepareError("");
    // Mounted afresh for each plan; a background catalogue refresh must not discard its preview.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, initialModelId]);
  const label = (id: string) => {
    const m = catalogue.models.find((x) => x.id === id);
    return m?.label || m?.repo || id;
  };
  // The confirmation shows, per Worker, the engine the model needs before anything is ordered. A family that
  // Snitch does not support stops here, with its own message and no order.
  async function prepare() {
    const picked = eligible.filter((w) => ticked.includes(w.worker_id));
    setPrepareError("");
    const found: Record<string, Preview> = {};
    if (onDemand) {
      setPreparing(true);
      try {
        for (const w of picked) {
          const result = await api.GET(
            "/api/admin/workers/{id}/voice-model-orders/preview",
            {
              params: {
                path: { id: w.worker_id },
                query: { model_id: modelId },
              },
            },
          );
          if (result.data) found[w.worker_id] = result.data;
          else {
            setPrepareError(error(errorCode(result.error) ?? "unavailable"));
            return;
          }
        }
      } catch {
        setPrepareError(error("unavailable"));
        return;
      } finally {
        setPreparing(false);
      }
    }
    setPreviews(found);
    setOrders(
      picked.map((worker) => ({
        worker,
        key: crypto.randomUUID(),
        replaces: replacement(worker),
      })),
    );
  }
  async function send() {
    if (!orders || !confirmed) return;
    setBusy(true);
    for (const order of orders) {
      if (outcome[order.worker.worker_id] === "sent") continue;
      try {
        const result = await api.POST(
          "/api/admin/workers/{id}/voice-model-orders",
          {
            params: {
              path: { id: order.worker.worker_id },
              header: headers,
            },
            body: {
              model_id: modelId,
              ...(runtimeDigest ? { runtime_digest: runtimeDigest } : {}),
              idempotency_key: order.key,
              ...(order.replaces ? { replaces_model_id: order.replaces } : {}),
            },
          },
        );
        setOutcome((o) => ({
          ...o,
          [order.worker.worker_id]: result.error
            ? error(errorCode(result.error) ?? "unknown")
            : "sent",
        }));
      } catch {
        setOutcome((o) => ({
          ...o,
          [order.worker.worker_id]: error("unavailable"),
        }));
      }
    }
    setBusy(false);
    onChanged();
  }
  const allSent =
    orders !== undefined &&
    orders.every((o) => outcome[o.worker.worker_id] === "sent");
  return (
    <Modal
      open={open}
      onOpenChange={(next) => !next && !busy && onClose()}
      title={t("voiceModels.byWorker.applyTitle")}
      closeLabel={t("voiceModels.byWorker.close")}
    >
      <p className="text-mute">{t("voiceModels.byWorker.applyHint")}</p>
      {orders === undefined ? (
        <>
          <label className="flex flex-col gap-1 text-sm">
            {t("voiceModels.byWorker.applyType")}
            <select
              value={kind}
              onChange={(e) => setKind(e.target.value as Kind)}
              className="h-11 rounded-md border border-line-strong bg-surface px-2 md:h-9"
            >
              {kinds.map((k) => (
                <option key={k} value={k}>
                  {t(`voiceModels.byWorker.type.${k}`)}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-1 text-sm">
            {t("voiceModels.byWorker.applyModel")}
            <select
              value={modelId}
              onChange={(e) => {
                setModelId(e.target.value);
                setConfirmed(false);
              }}
              className="h-11 rounded-md border border-line-strong bg-surface px-2 md:h-9"
            >
              {models.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.label || m.repo}
                </option>
              ))}
            </select>
          </label>
          {!onDemand && registry.status === "loading" && (
            <p role="status">{t("voiceEngines.loading")}</p>
          )}
          {!onDemand && registry.status === "failed" && (
            <ErrorAlert>{error(registry.code)}</ErrorAlert>
          )}
          {!onDemand &&
            registry.status === "ready" &&
            (available.length > 0 ? (
              <label className="flex flex-col gap-1 text-sm">
                {t("voiceEngines.select")}
                <select
                  value={runtimeDigest}
                  onChange={(e) => setChosenRuntime(e.target.value)}
                  className="h-11 rounded-md border border-line-strong bg-surface px-2 md:h-9"
                >
                  {available.map((r) => (
                    <option key={r.digest} value={r.digest}>
                      {r.manifest.engine} · {r.manifest.version} · CUDA{" "}
                      {r.manifest.cuda}
                    </option>
                  ))}
                </select>
                {available
                  .filter((r) => r.digest === runtimeDigest)
                  .map((r) => (
                    <small key={r.digest}>
                      {t("voiceEngines.constraints", {
                        disk: Math.ceil(r.manifest.disk_bytes / 1024 ** 2),
                        vram: Math.ceil(r.manifest.vram_bytes / 1024 ** 2),
                        license: r.manifest.license,
                      })}
                    </small>
                  ))}
              </label>
            ) : (
              <p>
                {t(
                  runtimeRequired
                    ? "voiceEngines.unavailable"
                    : "voiceEngines.bundled",
                )}
              </p>
            ))}
          <fieldset className="flex flex-col gap-1">
            <legend className="mb-1 text-sm">
              {t("voiceModels.byWorker.applyWorkers")}
            </legend>
            {eligible.length === 0 && (
              <p className="text-mute">{t("voiceModels.byWorker.applyNone")}</p>
            )}
            {workers.map((w) => (
              <label
                key={w.worker_id}
                className="flex min-h-11 items-center gap-2 md:min-h-8"
              >
                <input
                  type="checkbox"
                  className="size-4"
                  disabled={!!reason(w)}
                  checked={!reason(w) && ticked.includes(w.worker_id)}
                  onChange={(e) =>
                    setTicked(
                      e.target.checked
                        ? [...ticked, w.worker_id]
                        : ticked.filter((id) => id !== w.worker_id),
                    )
                  }
                />
                <span>
                  {w.name}
                  {reason(w) && (
                    <small className="block text-mute">{reason(w)}</small>
                  )}
                </span>
              </label>
            ))}
          </fieldset>
          {prepareError && <ErrorAlert>{prepareError}</ErrorAlert>}
          <div className="flex justify-end">
            <button
              type="button"
              disabled={
                preparing ||
                runtimeBlocked ||
                !eligible.some((w) => ticked.includes(w.worker_id)) ||
                !models.some((m) => m.id === modelId)
              }
              onClick={() => void prepare()}
              className={buttonClass("primary")}
            >
              {t("voiceModels.byWorker.applyPrepare")}
            </button>
          </div>
        </>
      ) : (
        <>
          <h4 className="font-medium">
            {t("voiceModels.byWorker.applyPreview", { count: orders.length })}
          </h4>
          {runtimeDigest && (
            <p>
              {t("voiceModels.byWorker.applyEnginePlan", {
                version:
                  available.find((r) => r.digest === runtimeDigest)?.manifest
                    .version ?? "",
                disk: Math.ceil(
                  (available.find((r) => r.digest === runtimeDigest)?.manifest
                    .disk_bytes ?? 0) /
                    1024 ** 2,
                ),
                weights: Math.ceil(
                  (models.find((m) => m.id === modelId)?.size_bytes ?? 0) /
                    1024 ** 2,
                ),
              })}
            </p>
          )}
          <ul className="flex flex-col divide-y divide-line rounded-lg border border-line">
            {orders.map((o) => {
              const result = outcome[o.worker.worker_id];
              return (
                <li key={o.worker.worker_id} className="px-3 py-2">
                  {o.replaces
                    ? t("voiceWorkspace.replacementLine", {
                        old: label(o.replaces),
                        model: label(modelId),
                        worker: o.worker.name,
                      })
                    : t("voiceModels.byWorker.applyLine", {
                        model: label(modelId),
                        worker: o.worker.name,
                      })}
                  {previews[o.worker.worker_id]?.engine_required && (
                    <EnginePlan
                      preview={previews[o.worker.worker_id]}
                      worker={o.worker.name}
                    />
                  )}
                  {result === "sent" && (
                    <span className="block text-ok-fg">
                      {t("voiceModels.byWorker.applySent")}
                    </span>
                  )}
                  {result && result !== "sent" && (
                    <span role="alert" className="block text-bad-fg">
                      {t("voiceModels.byWorker.applyFailed", {
                        reason: result,
                      })}
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
          {!allSent && (
            <label className="flex min-h-11 items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={confirmed}
                disabled={busy}
                onChange={(e) => setConfirmed(e.target.checked)}
              />
              {t("voiceWorkspace.acknowledge")}
            </label>
          )}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => {
                setOrders(undefined);
                setOutcome({});
                setConfirmed(false);
              }}
              className={buttonClass("secondary")}
            >
              {t("voiceModels.byWorker.applyBack")}
            </button>
            {allSent ? (
              <button
                type="button"
                onClick={onClose}
                className={buttonClass("primary")}
              >
                {t("voiceModels.byWorker.close")}
              </button>
            ) : (
              <button
                type="button"
                disabled={busy || !confirmed}
                onClick={() => void send()}
                className={buttonClass("primary")}
              >
                {t("voiceModels.byWorker.applySend", { count: orders.length })}
              </button>
            )}
          </div>
        </>
      )}
    </Modal>
  );
}

/** What the engine of the model costs on one Worker: installed first when missing, reused when it is there. */
function EnginePlan({ preview, worker }: { preview: Preview; worker: string }) {
  const { t, error } = useI18n();
  const params = {
    worker,
    engine: preview.engine ?? "",
    version: preview.version ?? "",
    disk: Math.ceil(preview.disk_bytes / 1024 ** 3),
    minutes: Math.max(1, Math.round(preview.estimated_seconds / 60)),
  };
  if (preview.state === "ready")
    return (
      <small className="block text-mute">
        {t("voiceModels.byWorker.applyEngineReady", params)}
      </small>
    );
  if (preview.state === "in_progress")
    return (
      <small className="block text-mute">
        {t("voiceModels.byWorker.applyEngineProgress", params)}
      </small>
    );
  return (
    <>
      <small className="block text-mute">
        {t("voiceModels.byWorker.applyEngineMissing", params)}
      </small>
      {preview.blocker && (
        <small role="alert" className="block text-bad-fg">
          {error(preview.blocker)}
        </small>
      )}
    </>
  );
}
