import type { components } from "../../../api.gen";
import { api, errorCode } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { StatusPill, type PillTone } from "../../components/StatusPill";
import { useI18n } from "../../i18n";

type Overview = components["schemas"]["VoiceOverview"];
type Row = Overview["workers"][number];
type Model = Row["diarization"];

async function load() {
  try {
    const result = await api.GET("/api/admin/voice-overview");
    if (result.data) return { ok: true as const, value: result.data };
    return { ok: false as const, code: errorCode(result.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

const tones: Record<Row["voice_state"], PillTone> = {
  ready: "ok",
  untested: "idle",
  unavailable: "bad",
  models_missing: "idle",
};

/** What each Worker really reports about the voice models: never typed by an administrator. */
export function VoiceInventory() {
  const { t, error } = useI18n();
  const { state } = useLoaded(load);
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  const overview = state.value;
  const model = (value: Model) =>
    value
      ? `${value.model}${value.version ? ` @ ${value.version}` : ""}`
      : t("voiceInventory.none");
  return (
    <section
      aria-labelledby="voice-inventory"
      className="flex flex-col gap-3 rounded-xl border border-line p-4"
    >
      <h3 id="voice-inventory" className="font-semibold">
        {t("voiceInventory.title")}
      </h3>
      <p className="text-mute">{t("voiceInventory.hint")}</p>
      <div className="flex flex-wrap gap-x-8 gap-y-2">
        <p>
          <span className="text-mute">{t("voiceInventory.ready")} </span>
          <strong className="tabular-nums">
            {overview.workers_voice_ready}/{overview.workers_total}
          </strong>
        </p>
        <p>
          <span className="text-mute">{t("voiceInventory.pending")} </span>
          <strong className="tabular-nums">
            {overview.pending_voice_jobs}
          </strong>
        </p>
      </div>
      {overview.workers_total === 0 ? (
        <p className="text-mute">{t("voiceInventory.empty")}</p>
      ) : (
        <ul className="flex flex-col divide-y divide-line rounded-lg border border-line">
          {overview.workers.map((worker) => (
            <li key={worker.id} className="flex flex-col gap-1.5 px-3 py-2.5">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="font-medium">{worker.name}</span>
                <span className="flex flex-wrap gap-2">
                  {!worker.connected && (
                    <StatusPill tone="idle">
                      {t("voiceInventory.offline")}
                    </StatusPill>
                  )}
                  <StatusPill tone={tones[worker.voice_state]}>
                    {t(`voiceInventory.state.${worker.voice_state}`)}
                  </StatusPill>
                </span>
              </div>
              <dl className="grid gap-x-4 gap-y-1 text-sm sm:grid-cols-[10rem_1fr]">
                <dt className="text-mute">{t("voiceInventory.diarization")}</dt>
                <dd className="font-mono break-all">
                  {model(worker.diarization)}
                </dd>
                <dt className="text-mute">{t("voiceInventory.fingerprint")}</dt>
                <dd className="font-mono break-all">
                  {model(worker.fingerprint)}
                </dd>
              </dl>
              {worker.voice_state === "models_missing" && (
                <p className="text-sm text-mute">
                  {t("voiceInventory.missing", {
                    model: overview.active_models.diarization,
                  })}
                </p>
              )}
              {worker.cause_code && (
                <p className="text-sm text-bad-fg">
                  {t("voiceInventory.cause", { code: worker.cause_code })}
                </p>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
