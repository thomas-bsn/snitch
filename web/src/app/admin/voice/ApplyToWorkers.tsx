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
  const releases =
    registry.status === "ready"
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
    releases.length > 0 ||
    model?.engine === "nemo-titanet" ||
    model?.engine === "speechbrain-ecapa";
  const runtimeBlocked =
    registry.status !== "ready" || (runtimeRequired && !runtimeDigest);
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
    // Mounted afresh for each plan; a background catalogue refresh must not discard its preview.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, initialModelId]);
  const label = (id: string) => {
    const m = catalogue.models.find((x) => x.id === id);
    return m?.label || m?.repo || id;
  };
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
          {registry.status === "loading" && (
            <p role="status">{t("voiceEngines.loading")}</p>
          )}
          {registry.status === "failed" && (
            <ErrorAlert>{error(registry.code)}</ErrorAlert>
          )}
          {registry.status === "ready" &&
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
          <div className="flex justify-end">
            <button
              type="button"
              disabled={
                runtimeBlocked ||
                !eligible.some((w) => ticked.includes(w.worker_id)) ||
                !models.some((m) => m.id === modelId)
              }
              onClick={() =>
                setOrders(
                  eligible
                    .filter((w) => ticked.includes(w.worker_id))
                    .map((worker) => ({
                      worker,
                      key: crypto.randomUUID(),
                      replaces: replacement(worker),
                    })),
                )
              }
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
              {t("voiceEngines.select")}:{" "}
              {
                available.find((r) => r.digest === runtimeDigest)?.manifest
                  .version
              }
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
