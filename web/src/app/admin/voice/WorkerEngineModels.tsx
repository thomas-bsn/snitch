import { useCallback, useEffect, useState } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded, type Result } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import type { components } from "../../../api.gen";
import type { Release } from "./engineOrders";

type Binding = components["schemas"]["RuntimeBinding"];

async function fetchBindings(): Promise<Result<Binding[]>> {
  try {
    const { data, error } = await api.GET("/api/admin/voice-engines/bindings");
    return data
      ? { ok: true, value: data.bindings }
      : { ok: false, code: errorCode(error) ?? "unknown" };
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

const causes = [
  "disk_full",
  "runtime_incompatible",
  "inference_failed",
  "invalid_order",
  "hf_network",
] as const;
/** A cause the interface has a text for; anything else reads as an unusable runtime. */
const causeOf = (code: string) =>
  causes.find((c) => c === code) ?? "runtime_incompatible";

/** The other approved versions of the same engine a model can move to. A revoked version is never offered. */
export function updateTargets(binding: Binding, releases: Release[]) {
  return releases.filter(
    (r) =>
      !r.revoked &&
      r.manifest.engine === binding.engine &&
      r.manifest.kind === binding.kind &&
      r.digest !== binding.digest,
  );
}

/**
 * Ready models of the chosen Worker: update their runtime or go back to the previous one. Every change is an
 * explicit order, shown with its plan first; the model keeps serving on its current runtime meanwhile.
 */
export function WorkerEngineModels({
  workerId,
  releases,
}: {
  workerId: string;
  releases: Release[];
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const loaded = useLoaded(useCallback(fetchBindings, []));
  const [target, setTarget] = useState<Record<string, string>>({});
  const [confirming, setConfirming] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const bindings =
    loaded.state.status === "ready"
      ? loaded.state.value.filter((b) => b.worker_id === workerId)
      : [];
  const reload = loaded.reload;
  const waiting = bindings.some((b) => b.pending);
  // An order in progress moves on its own: the view follows it without a click.
  useEffect(() => {
    if (!waiting) return;
    const timer = setInterval(reload, 4000);
    return () => clearInterval(timer);
  }, [waiting, reload]);

  async function order(
    binding: Binding,
    intent: "update" | "rollback",
    digest: string,
  ) {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.POST("/api/admin/voice-model-runtime-orders", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          worker_id: binding.worker_id,
          model_id: binding.model_id,
          digest,
          intent,
          idempotency_key: crypto.randomUUID(),
        },
      });
      if (result.error) {
        const code = errorCode(result.error) ?? "unknown";
        setFailure(
          intent === "rollback" && code === "engine_not_installed"
            ? t("workerEngines.models.notHeld")
            : error(code),
        );
      }
    } catch {
      setFailure(error("unavailable"));
    } finally {
      setBusy(false);
      setConfirming(undefined);
      reload();
    }
  }

  if (loaded.state.status !== "ready") return null;
  return (
    <section className="engine-models" aria-labelledby="engine-models-title">
      <h4 id="engine-models-title">{t("workerEngines.models.title")}</h4>
      <p className="engine-note">{t("workerEngines.models.lead")}</p>
      {failure && <ErrorAlert>{failure}</ErrorAlert>}
      {bindings.length === 0 && (
        <p className="engine-empty">{t("workerEngines.models.none")}</p>
      )}
      {bindings.map((b) => {
        const targets = updateTargets(b, releases);
        const chosen = targets.find((r) => r.digest === target[b.model_id]);
        const key = b.model_id;
        return (
          <div key={key} className="engine-model">
            <strong>{b.model_label || b.model_id}</strong>
            <span className="engine-sub">
              {b.version
                ? t("workerEngines.models.bound", { version: b.version })
                : t("workerEngines.models.unknown")}
            </span>
            {b.revoked && (
              <span className="engine-warn">
                {t("workerEngines.models.revoked")}
              </span>
            )}
            {b.pending && (
              <span className="engine-badge run" aria-live="polite">
                {t("workerEngines.models.pending", {
                  intent: t(`workerEngines.models.intent.${b.pending.intent}`),
                  version: b.pending.version,
                  status: t(`workerEngines.models.status.${b.pending.status}`),
                })}
              </span>
            )}
            {!b.pending && (
              <div className="engine-model-actions">
                <select
                  value={target[key] ?? ""}
                  aria-label={t("workerEngines.models.target", {
                    model: b.model_label || b.model_id,
                  })}
                  onChange={(e) => {
                    setTarget((all) => ({ ...all, [key]: e.target.value }));
                    setConfirming(undefined);
                  }}
                >
                  <option value="">{t("workerEngines.models.choose")}</option>
                  {targets.map((r) => (
                    <option key={r.digest} value={r.digest}>
                      {r.manifest.version}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  className="voice-btn"
                  disabled={busy || !chosen}
                  onClick={() => setConfirming(`update:${key}`)}
                >
                  {t("workerEngines.models.update")}
                </button>
                {b.previous_digest && !b.previous_revoked && (
                  <button
                    type="button"
                    className="voice-btn"
                    disabled={busy}
                    onClick={() => setConfirming(`rollback:${key}`)}
                  >
                    {t("workerEngines.models.rollback", {
                      version: b.previous_version,
                    })}
                  </button>
                )}
              </div>
            )}
            {targets.length === 0 && !b.pending && (
              <span className="engine-note">
                {t("workerEngines.models.noTarget")}
              </span>
            )}
            {b.previous_digest && b.previous_revoked && (
              <span className="engine-warn">
                {t("workerEngines.models.rollbackBlocked", {
                  version: b.previous_version,
                })}
              </span>
            )}
            {confirming === `update:${key}` && chosen && (
              <div className="engine-plan" aria-live="polite">
                <span>
                  {t("workerEngines.models.plan", {
                    version: chosen.manifest.version,
                    disk: Math.ceil(chosen.manifest.disk_bytes / 1024 ** 2),
                  })}
                </span>
                <button
                  type="button"
                  className="voice-btn primary"
                  disabled={busy}
                  onClick={() => void order(b, "update", chosen.digest)}
                >
                  {t("workerEngines.models.confirm")}
                </button>
              </div>
            )}
            {confirming === `rollback:${key}` && (
              <div className="engine-plan" aria-live="polite">
                <span>
                  {t("workerEngines.models.rollbackPlan", {
                    version: b.previous_version,
                  })}
                </span>
                <button
                  type="button"
                  className="voice-btn primary"
                  disabled={busy}
                  onClick={() => void order(b, "rollback", b.previous_digest)}
                >
                  {t("workerEngines.models.rollback", {
                    version: b.previous_version,
                  })}
                </button>
              </div>
            )}
            {b.history.length > 0 && (
              <details>
                <summary>{t("workerEngines.models.history")}</summary>
                <ol className="engine-history">
                  {b.history.map((h, i) => (
                    <li key={i} className={h.ignored ? "ignored" : ""}>
                      {new Date(h.created_at).toLocaleString()} ·{" "}
                      {t(`workerEngines.models.intent.${h.intent}`)} {h.version}{" "}
                      · {t(`workerEngines.models.state.${h.state}`)}
                      {h.cause_code &&
                        ` · ${t(`workerEngines.models.cause.${causeOf(h.cause_code)}`)}`}
                      {h.ignored &&
                        ` · ${t("workerEngines.models.ignored", { reason: h.reason })}`}
                    </li>
                  ))}
                </ol>
              </details>
            )}
          </div>
        );
      })}
    </section>
  );
}
