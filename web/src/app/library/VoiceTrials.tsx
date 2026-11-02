import { Trash2 } from "lucide-react";
import { useCallback, useEffect, useId, useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { useLoaded, type Result } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { isAdministrator } from "../roles";
import { useSession } from "../session";
import type { ClipVoice } from "./libraryApi";

type Trial = components["schemas"]["VoiceTrial"];
type Model = components["schemas"]["VoiceModels"]["models"][number];

async function loadModels(): Promise<Result<Model[]>> {
  try {
    const r = await api.GET("/api/admin/voice-models");
    return r.data
      ? { ok: true, value: r.data.models }
      : { ok: false, code: errorCode(r.error) ?? "unknown" };
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

async function loadTrials(clipId: number): Promise<Result<Trial[]>> {
  try {
    const r = await api.GET("/api/admin/clips/{id}/voice-trials", {
      params: { path: { id: clipId } },
    });
    return r.data
      ? { ok: true, value: r.data.trials }
      : { ok: false, code: errorCode(r.error) ?? "unknown" };
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

/**
 * The diarization trial of a clip (administrators): run the candidate or active model, optionally with a
 * number of speakers, and read the outcome beside the current result. A trial is never adopted: it changes
 * neither the speakers nor the attributions nor the corrections of the clip.
 */
export function VoiceTrials({
  clipId,
  voice,
}: {
  clipId: number;
  voice: ClipVoice | null;
}) {
  const { user } = useSession();
  if (!user || !isAdministrator(user)) return null;
  return <TrialsSection clipId={clipId} voice={voice} />;
}

function TrialsSection({
  clipId,
  voice,
}: {
  clipId: number;
  voice: ClipVoice | null;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const fieldId = useId();
  const models = useLoaded(loadModels);
  const trials = useLoaded(useCallback(() => loadTrials(clipId), [clipId]));
  const [modelId, setModelId] = useState("");
  const [count, setCount] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  const list = trials.state.status === "ready" ? trials.state.value : [];
  const waiting = list.some(
    (trial) => trial.status === "pending" || trial.status === "running",
  );
  const reloadTrials = trials.reload;
  useEffect(() => {
    if (!waiting) return;
    const timer = window.setInterval(reloadTrials, 4000);
    return () => window.clearInterval(timer);
  }, [waiting, reloadTrials]);

  // Candidate or active, and held by a Worker: the only models a trial can run.
  const usable =
    models.state.status === "ready"
      ? models.state.value.filter(
          (m) =>
            m.kind === "diarization" &&
            (m.active || m.candidate) &&
            m.installed,
        )
      : [];
  const selected = usable.find((m) => m.id === modelId) ?? usable[0];
  const names = new Map(
    (models.state.status === "ready" ? models.state.value : []).map((m) => [
      m.id,
      m.repo,
    ]),
  );

  async function run() {
    if (!selected) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const wanted = selected.supports_num_speakers ? Number(count) : 0;
      const r = await api.POST("/api/admin/clips/{id}/voice-trials", {
        params: { path: { id: clipId }, header: mutation(session?.csrf_token) },
        body: {
          model_id: selected.id,
          ...(wanted > 0 ? { num_speakers: wanted } : {}),
        },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else trials.reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  async function remove(trial: Trial) {
    setFailure(undefined);
    try {
      const r = await api.DELETE("/api/admin/clips/{id}/voice-trials/{trial}", {
        params: {
          path: { id: clipId, trial: trial.id },
          header: mutation(session?.csrf_token),
        },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else trials.reload();
    } catch {
      setFailure("unavailable");
    }
  }

  const current = voice?.speakers ?? [];
  return (
    <section
      aria-labelledby={`${fieldId}-title`}
      className="flex flex-col gap-3 rounded-xl border border-line p-4"
    >
      <div>
        <h2 id={`${fieldId}-title`} className="m-0 text-base font-semibold">
          {t("voiceTrial.title")}
        </h2>
        <p className="mt-1 mb-0 text-xs text-mute">{t("voiceTrial.hint")}</p>
      </div>

      {models.state.status === "ready" && usable.length === 0 && (
        <p className="m-0 text-mute">{t("voiceTrial.noModel")}</p>
      )}
      {selected && (
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(event) => {
            event.preventDefault();
            void run();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <label htmlFor={`${fieldId}-model`} className="font-medium">
              {t("voiceTrial.model")}
            </label>
            <select
              id={`${fieldId}-model`}
              value={selected.id}
              onChange={(event) => setModelId(event.target.value)}
              className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
            >
              {usable.map((m) => (
                <option key={m.id} value={m.id}>
                  {`${m.repo} (${t(m.active ? "voiceTrial.active" : "voiceTrial.candidate")})`}
                </option>
              ))}
            </select>
          </div>
          {selected.supports_num_speakers ? (
            <div className="flex flex-col gap-1.5">
              <label htmlFor={`${fieldId}-count`} className="font-medium">
                {t("voiceTrial.numSpeakers")}
              </label>
              <input
                id={`${fieldId}-count`}
                type="number"
                min={1}
                max={selected.max_speakers ?? 32}
                value={count}
                placeholder={t("voiceTrial.automatic")}
                onChange={(event) => setCount(event.target.value)}
                className="h-11 w-32 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
              />
            </div>
          ) : (
            <p className="m-0 text-xs text-mute">
              {t("voiceTrial.noNumSpeakers")}
            </p>
          )}
          <button type="submit" disabled={busy} className={buttonClass()}>
            {t(busy ? "voiceTrial.starting" : "voiceTrial.run")}
          </button>
        </form>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}

      <div className="grid gap-3 md:grid-cols-2">
        <div className="flex flex-col gap-1 rounded-lg border border-line p-3">
          <h3 className="m-0 text-sm font-semibold">
            {t("voiceTrial.current")}
          </h3>
          <p className="m-0 text-xs text-mute">
            {t("voiceTrial.count", { count: current.length })}
          </p>
          <ul className="m-0 flex list-none flex-col gap-0.5 p-0 text-xs">
            {current.map((speaker) => (
              <li key={speaker.id}>
                {t("voiceTrial.speaker", {
                  label: speaker.label,
                  seconds: Math.round(speaker.speaking_seconds),
                  turns: speaker.turn_count,
                })}
              </li>
            ))}
          </ul>
        </div>
        <div className="flex flex-col gap-2">
          {list.length === 0 && (
            <p className="m-0 text-mute">{t("voiceTrial.none")}</p>
          )}
          {list.map((trial) => (
            <article
              key={trial.id}
              data-testid="voice-trial"
              className="flex flex-col gap-1 rounded-lg border border-line p-3"
            >
              <div className="flex items-start justify-between gap-2">
                <h3 className="m-0 text-sm font-semibold break-all">
                  {names.get(trial.model_id) ?? trial.model_id}
                </h3>
                <button
                  type="button"
                  onClick={() => void remove(trial)}
                  aria-label={t("voiceTrial.delete")}
                  className={`${buttonClass()} !h-8 !px-2`}
                >
                  <Trash2 className="size-4" aria-hidden="true" />
                </button>
              </div>
              <p className="m-0 text-xs text-mute">
                {trial.num_speakers
                  ? t("voiceTrial.imposed", { count: trial.num_speakers })
                  : t("voiceTrial.automatic")}
                {` · ${t(`voiceTrial.status.${trial.status}`)}`}
              </p>
              {trial.status === "done" && (
                <>
                  <p className="m-0 text-xs font-medium">
                    {t("voiceTrial.count", { count: trial.speakers.length })}
                  </p>
                  <ul className="m-0 flex list-none flex-col gap-0.5 p-0 text-xs">
                    {trial.speakers.map((speaker) => (
                      <li key={speaker.label}>
                        {t("voiceTrial.speaker", {
                          label: speaker.label,
                          seconds: Math.round(speaker.speaking_seconds),
                          turns: speaker.turn_count,
                        })}
                      </li>
                    ))}
                  </ul>
                </>
              )}
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
