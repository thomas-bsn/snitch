import { useCallback, useEffect, useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

type Preview = components["schemas"]["VoiceRecomputePreview"];
type Run = components["schemas"]["VoiceRecomputeRun"];
const endpoint = "/api/admin/voice-fingerprint-recompute";

async function load() {
  try {
    const result = await api.GET(endpoint);
    return result.data
      ? { ok: true as const, value: result.data }
      : { ok: false as const, code: errorCode(result.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

const FACTS = [
  "fingerprints",
  "recomputable",
  "jobs",
  "clips",
  "missing",
  "archived",
  "references",
] as const;

/**
 * The recalculation of the stored fingerprints on the net speech (#395): its extent, then its progress and its
 * final report. Every figure comes from the server; the UI never simulates a job.
 */
export function FingerprintRecompute() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  const [preview, setPreview] = useState<Preview>();
  const [confirmed, setConfirmed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [cancelling, setCancelling] = useState(false);
  const run = state.status === "ready" ? state.value.run : undefined;
  const running = run?.status === "running";
  const rematching =
    !!run && !running && run.report.recomputed > 0 && run.rematched_at === null;
  useEffect(() => {
    if (!running && !rematching) return;
    const timer = window.setInterval(reload, 3000);
    return () => window.clearInterval(timer);
  }, [running, rematching, reload]);

  const loadPreview = useCallback(async () => {
    setBusy(true);
    setFailure(undefined);
    setConfirmed(false);
    try {
      const r = await api.GET(`${endpoint}/preview`);
      if (r.data) setPreview(r.data);
      else setFailure(errorCode(r.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }, []);

  async function act(action: "start" | "resume" | "cancel") {
    if (action === "start" && (!preview || !confirmed)) return;
    setBusy(true);
    setFailure(undefined);
    const params = { header: mutation(session?.csrf_token) };
    try {
      const r =
        action === "start"
          ? await api.POST(endpoint, {
              params,
              body: { confirm_fingerprints: preview!.fingerprints },
            })
          : action === "resume"
            ? await api.POST(`${endpoint}/resume`, { params })
            : await api.DELETE(endpoint, { params });
      if (r.error) {
        const code = errorCode(r.error) ?? "unknown";
        if (code === "voice_recompute_preview_stale") await loadPreview();
        setFailure(code);
      } else {
        setPreview(undefined);
        setConfirmed(false);
        setCancelling(false);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section
      aria-labelledby="adv-recompute"
      className="flex flex-col gap-3 rounded-lg border border-line p-3"
    >
      <h5 id="adv-recompute" className="font-medium">
        {t("voiceAdmin.recompute.title")}
      </h5>
      <p className="text-mute">{t("voiceAdmin.recompute.hint")}</p>
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {run && <RunState run={run} />}
      <div className="flex flex-wrap gap-2">
        {!running && (
          <button
            type="button"
            disabled={busy}
            className={buttonClass()}
            onClick={() => void loadPreview()}
          >
            {t("voiceAdmin.recompute.preview")}
          </button>
        )}
        {run &&
          !running &&
          run.report.errors + run.report.cancelled + run.report.pending > 0 && (
            <button
              type="button"
              disabled={busy}
              className={buttonClass()}
              onClick={() => void act("resume")}
            >
              {t("voiceAdmin.recompute.resume")}
            </button>
          )}
        {running && (
          <button
            type="button"
            disabled={busy}
            className={buttonClass("danger")}
            onClick={() => setCancelling(true)}
          >
            {t("voiceAdmin.recompute.cancel")}
          </button>
        )}
      </div>
      {preview && !running && (
        <div className="flex flex-col gap-3">
          <dl className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            {FACTS.map((key) => (
              <div key={key}>
                <dt className="text-sm text-mute">
                  {t(`voiceAdmin.recompute.${key}`)}
                </dt>
                <dd className="text-xl font-semibold">
                  {preview[key === "missing" ? "missing_sources" : key]}
                </dd>
              </div>
            ))}
          </dl>
          {preview.transition_open ? (
            <p role="status">{t("voiceAdmin.recompute.transitionOpen")}</p>
          ) : (
            <>
              <label className="flex min-h-11 items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={confirmed}
                  onChange={(e) => setConfirmed(e.target.checked)}
                />
                {t("voiceAdmin.recompute.confirm", {
                  count: preview.fingerprints,
                })}
              </label>
              <div>
                <button
                  type="button"
                  disabled={busy || !confirmed || preview.recomputable === 0}
                  className={buttonClass("primary")}
                  onClick={() => void act("start")}
                >
                  {t("voiceAdmin.recompute.start")}
                </button>
              </div>
            </>
          )}
        </div>
      )}
      <ConfirmDialog
        open={cancelling}
        onOpenChange={setCancelling}
        title={t("voiceAdmin.recompute.cancel")}
        cancelLabel={t("voiceAdmin.recompute.keep")}
        confirmLabel={t("voiceAdmin.recompute.cancel")}
        danger
        busy={busy}
        onConfirm={() => void act("cancel")}
      >
        <p>{t("voiceAdmin.recompute.cancelText")}</p>
      </ConfirmDialog>
    </section>
  );
}

function RunState({ run }: { run: Run }) {
  const { t } = useI18n();
  const r = run.report;
  const date = (d: string | null) => (d ? new Date(d).toLocaleString() : "");
  return (
    <div className="flex flex-col gap-1">
      <p className="text-sm text-mute">
        {t("voiceAdmin.recompute.started", {
          author: run.author ?? "",
          date: date(run.created_at),
          count: run.announced,
        })}
      </p>
      <p role="status">
        {run.status === "running"
          ? t("voiceAdmin.recompute.running", {
              done: r.fingerprints - r.pending,
              total: r.fingerprints,
              pending: r.pending,
            })
          : run.status === "completed"
            ? t("voiceAdmin.recompute.completed", {
                date: date(run.finished_at),
              })
            : t("voiceAdmin.recompute.cancelled", {
                date: date(run.finished_at),
              })}
      </p>
      <ul className="list-disc pl-5">
        <li>
          {t("voiceAdmin.recompute.report", {
            recomputed: r.recomputed,
            recovered: r.recovered,
            recoveredContributive: r.recovered_contributive,
            stillRejected: r.still_rejected,
            lost: r.lost,
          })}
        </li>
        <li>
          {t("voiceAdmin.recompute.reportKept", {
            errors: r.errors,
            cancelled: r.cancelled,
            missing: r.missing_sources,
            archived: r.archived,
            obsolete: r.obsolete,
          })}
        </li>
      </ul>
      {run.status !== "running" &&
        r.recomputed > 0 &&
        (run.rematched_at ? (
          <p>
            {t("voiceAdmin.recompute.rematched", {
              date: date(run.rematched_at),
            })}
          </p>
        ) : (
          <p className="text-mute">{t("voiceAdmin.recompute.rematching")}</p>
        ))}
    </div>
  );
}
