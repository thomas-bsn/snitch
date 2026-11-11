import { useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import type { Catalogue } from "./VoiceModelCards";
import type { Overview } from "./VoiceInventory";

type Accuracy = components["schemas"]["VoiceAccuracy"];

const pct = (x: number | null) =>
  x === null ? null : `${Math.round(x * 100)}%`;

/** The state of the voice processing at a glance, and the one switch of automatic attribution. */
export function VoiceHeader({
  overview,
  accuracy,
  catalogue,
  onChanged,
}: {
  overview: Overview;
  accuracy: Accuracy;
  catalogue: Catalogue;
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const headers = mutation(session?.csrf_token);
  const [confirming, setConfirming] = useState(false);
  const [forced, setForced] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const enabled = accuracy.auto_enabled;
  const unmet = !accuracy.criteria.met;
  const precision = pct(accuracy.evaluation.precision);
  async function apply(next: boolean, confirmUnmet: boolean) {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.PUT("/api/admin/voice-automation", {
        params: { header: headers },
        body: { enabled: next, confirm_unmet_criteria: confirmUnmet },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else onChanged();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setConfirming(false);
      setForced(false);
    }
  }
  const causes = new Set(
    catalogue.models.flatMap((m) =>
      m.workers.filter((w) => w.state === "failed").map((w) => w.cause_code),
    ),
  );
  const offline = overview.workers.filter((w) => !w.connected).length;
  const notices: string[] = [];
  if (overview.workers_total === 0) notices.push(t("voiceScreen.noWorker"));
  else {
    if (overview.workers_voice_ready === 0)
      notices.push(t("voiceScreen.noneReady"));
    if (offline > 0) notices.push(t("voiceScreen.offline", { count: offline }));
  }
  if (causes.has("hf_token_missing"))
    notices.push(t("voiceScreen.tokenMissing"));
  if (causes.has("hf_access_denied"))
    notices.push(t("voiceScreen.accessDenied"));
  if (overview.pending_voice_jobs > 0)
    notices.push(
      t("voiceScreen.pending", { count: overview.pending_voice_jobs }),
    );
  const active = (kind: "diarization" | "fingerprint") =>
    catalogue.models.find((m) => m.kind === kind && m.active)?.repo ??
    overview.active_models[kind];
  return (
    <section
      aria-labelledby="voice-summary"
      className="flex flex-col gap-4 rounded-xl border border-line p-4"
    >
      <h3 id="voice-summary" className="font-semibold">
        {t("voiceScreen.summary")}
      </h3>
      <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2">
        <div>
          <dt className="text-sm text-mute">{t("voiceInventory.ready")}</dt>
          <dd className="text-lg font-semibold tabular-nums">
            {overview.workers_voice_ready}/{overview.workers_total}
          </dd>
        </div>
        <div>
          <dt className="text-sm text-mute">{t("voiceScreen.activeDiar")}</dt>
          <dd className="font-mono text-sm break-all">
            {active("diarization")}
          </dd>
        </div>
        <div>
          <dt className="text-sm text-mute">
            {t("voiceScreen.activeFingerprint")}
          </dt>
          <dd className="font-mono text-sm break-all">
            {active("fingerprint")}
          </dd>
        </div>
        <div>
          <dt className="text-sm text-mute">{t("voiceScreen.measured")}</dt>
          <dd className="font-semibold tabular-nums">
            {precision ?? t("voiceAdmin.notMeasured")}
          </dd>
        </div>
      </dl>
      {notices.length > 0 && (
        <ul className="flex flex-col gap-2">
          {notices.map((n) => (
            <li key={n}>
              <StatusNotice>{n}</StatusNotice>
            </li>
          ))}
        </ul>
      )}
      <div className="flex flex-col gap-2 border-t border-line pt-4">
        <div className="flex items-center justify-between gap-3">
          <span id="voice-auto-label" className="font-medium">
            {t("voiceAdmin.automation")}
          </span>
          <button
            type="button"
            role="switch"
            aria-checked={enabled}
            aria-labelledby="voice-auto-label"
            aria-describedby="voice-auto-state"
            disabled={busy}
            onClick={() =>
              enabled ? void apply(false, false) : setConfirming(true)
            }
            className={`relative h-11 w-14 shrink-0 rounded-full border transition-colors disabled:opacity-60 md:h-8 md:w-12 ${
              enabled
                ? "border-accent-solid bg-accent-solid"
                : "border-line-strong bg-side"
            }`}
          >
            <span
              aria-hidden="true"
              className={`absolute top-1/2 h-6 w-6 -translate-y-1/2 rounded-full bg-surface shadow transition-all md:h-5 md:w-5 ${
                enabled ? "left-7 md:left-6" : "left-1"
              }`}
            />
          </button>
        </div>
        <p id="voice-auto-state" className="text-sm">
          {enabled ? t("voiceAdmin.enabled") : t("voiceAdmin.disabled")}
        </p>
        <p className="text-sm text-mute">{t("voiceScreen.autoHint")}</p>
        {accuracy.calibration.group && !accuracy.calibration.thresholds && (
          <StatusNotice>{t("voiceScreen.autoSuspended")}</StatusNotice>
        )}
        {accuracy.activation_proposed && (
          <StatusNotice>{t("voiceAdmin.proposed")}</StatusNotice>
        )}
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      </div>
      <ConfirmDialog
        open={confirming}
        onOpenChange={(open) => {
          if (!open) {
            setConfirming(false);
            setForced(false);
          }
        }}
        title={t("voiceScreen.enableTitle")}
        cancelLabel={t("voiceModels.cancelDialog")}
        confirmLabel={t("voiceAdmin.enable")}
        busy={busy}
        confirmDisabled={unmet && !forced}
        onConfirm={() => void apply(true, unmet && forced)}
      >
        <p>{t("voiceScreen.enableText")}</p>
        <p>
          {t("voiceScreen.enableMeasure", {
            value: precision ?? t("voiceAdmin.notMeasured"),
          })}
        </p>
        {unmet && (
          <label className="flex min-h-11 items-center gap-2">
            <input
              type="checkbox"
              checked={forced}
              onChange={(e) => setForced(e.target.checked)}
            />
            {t("voiceAdmin.force")}
          </label>
        )}
      </ConfirmDialog>
    </section>
  );
}
