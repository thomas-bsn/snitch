import { useCallback, useState, type FormEvent } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { fetchWorkers, type Worker } from "../workers/workersApi";

type Pipeline = components["schemas"]["TranscriptionPipeline"];
type Settings = components["schemas"]["TranscriptionSettingsState"];
async function load() {
  try {
    const workers = await fetchWorkers();
    if (!workers.ok) return workers;
    const { data, error } = await api.GET("/api/admin/transcription-pipeline");
    if (!data)
      return { ok: false, code: errorCode(error) ?? "unknown" } as const;
    const settings = await api.GET("/api/admin/transcription-settings");
    if (!settings.data)
      return {
        ok: false,
        code: errorCode(settings.error) ?? "unknown",
      } as const;
    return {
      ok: true,
      value: {
        workers: workers.value,
        pipeline: data,
        settings: settings.data,
      },
    } as const;
  } catch {
    return { ok: false, code: "unavailable" } as const;
  }
}

export function TranscriptionTab() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  const [selected, setSelected] = useState<Worker>();
  const [override, setOverride] = useState("");
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState(false);
  const [draft, setDraft] = useState<Pipeline>();
  const [settingsDraft, setSettingsDraft] = useState<Partial<Settings>>();
  const [busy, setBusy] = useState(false);
  const read = useCallback(() => reload(), [reload]);
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  const config = draft ?? {
    ...state.value.pipeline.config,
    model:
      state.value.settings.mode === "unconfigured"
        ? ""
        : state.value.pipeline.config.model,
    language:
      state.value.settings.mode === "unconfigured"
        ? ""
        : state.value.pipeline.config.language,
  };
  const settings = { ...state.value.settings, ...settingsDraft };
  function change(part: Partial<Pipeline>) {
    setDraft({ ...config, ...part });
    setNotice(false);
  }
  function changeSettings(part: Partial<Settings>) {
    setSettingsDraft({ ...settings, ...part });
    setNotice(false);
  }
  async function save(event: FormEvent) {
    event.preventDefault();
    setFailure(undefined);
    setBusy(true);
    try {
      const saved = await api.PUT("/api/admin/transcription-settings", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          mode: settings.mode === "disabled" ? "disabled" : "enabled",
          endpoint: settings.endpoint ?? "",
          model: config.model,
          language: config.language,
          profile: settings.mode === "disabled" ? undefined : config,
        },
      });
      if (saved.error) {
        setFailure(errorCode(saved.error) ?? "unknown");
        return;
      }
      setDraft(undefined);
      setSettingsDraft(undefined);
      setNotice(true);
      read();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function saveOverride(event: FormEvent) {
    event.preventDefault();
    if (!selected) return;
    setFailure(undefined);
    setBusy(true);
    try {
      const result = await api.PUT(
        "/api/admin/workers/{id}/transcription-model",
        {
          params: {
            path: { id: selected.id },
            header: mutation(session?.csrf_token),
          },
          body: { model: override },
        },
      );
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unknown");
        return;
      }
      setSelected(undefined);
      read();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function testWorker(id: string) {
    setFailure(undefined);
    setBusy(true);
    try {
      const result = await api.POST(
        "/api/admin/workers/{id}/transcription-test",
        { params: { path: { id }, header: mutation(session?.csrf_token) } },
      );
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unknown");
        return;
      }
      read();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex flex-col gap-6">
      <form
        onSubmit={(event) => void save(event)}
        className="flex max-w-xl flex-col gap-4"
      >
        <h2 className="text-lg font-semibold">{t("transAdmin.title")}</h2>
        <p role="status">{t(`transAdmin.settingsStatus.${settings.status}`)}</p>
        {state.value.workers.first_worker && (
          <p>{t("transAdmin.configureBeforeWorker")}</p>
        )}
        <label className="flex flex-col gap-2">
          {t("transAdmin.transcriptionMode")}
          <select
            className="h-11 rounded-lg border border-line-strong bg-surface px-3"
            value={settings.mode === "disabled" ? "disabled" : "enabled"}
            onChange={(event) =>
              changeSettings({
                mode: event.target.value as "enabled" | "disabled",
                status:
                  event.target.value === "disabled" ? "disabled" : "untested",
              })
            }
          >
            <option value="enabled">{t("transAdmin.enabled")}</option>
            <option value="disabled">{t("transAdmin.disabled")}</option>
          </select>
        </label>
        {settings.mode !== "disabled" && (
          <Field
            label={t("transAdmin.endpoint")}
            required
            type="url"
            maxLength={300}
            value={settings.endpoint ?? ""}
            onChange={(event) =>
              changeSettings({ endpoint: event.target.value })
            }
          />
        )}
        <p className="text-mute">
          {t("transAdmin.version", {
            version: settings.version || state.value.pipeline.version,
          })}
        </p>
        <Field
          label={t("transAdmin.model")}
          required={settings.mode !== "disabled"}
          maxLength={100}
          value={config.model}
          onChange={(event) => change({ model: event.target.value })}
        />
        <Field
          label={t("transAdmin.language")}
          required={settings.mode !== "disabled"}
          maxLength={32}
          value={config.language}
          onChange={(event) => change({ language: event.target.value })}
        />
        <details className="rounded-lg border border-line p-4">
          <summary className="cursor-pointer font-medium">
            {t("transAdmin.vad")}
          </summary>
          <label className="mt-3 flex flex-col gap-2">
            {t("transAdmin.mode")}
            <select
              className="h-11 rounded-lg border border-line-strong bg-surface px-3"
              value={config.vad.mode}
              onChange={(event) =>
                change({
                  vad: {
                    ...config.vad,
                    mode: event.target.value as "off" | "energy",
                    version:
                      event.target.value === "off"
                        ? "vad-off-v1"
                        : "vad-energy-v1",
                    threshold: config.vad.threshold ?? 0.02,
                    min_speech_ms: config.vad.min_speech_ms ?? 100,
                    min_silence_ms: config.vad.min_silence_ms ?? 300,
                    padding_ms: config.vad.padding_ms ?? 100,
                  },
                })
              }
            >
              <option value="off">{t("transAdmin.off")}</option>
              <option value="energy">{t("transAdmin.energy")}</option>
            </select>
          </label>
          {config.vad.mode === "energy" && (
            <div className="mt-3 flex flex-col gap-3">
              {(
                [
                  ["threshold", "transAdmin.threshold", 0.001, 0.999, 0.001],
                  ["min_speech_ms", "transAdmin.speech", 1, 5000, 1],
                  ["min_silence_ms", "transAdmin.silence", 1, 10000, 1],
                  ["padding_ms", "transAdmin.padding", 0, 2000, 1],
                ] as const
              ).map(([key, label, min, max, step]) => (
                <Field
                  key={key}
                  label={t(label)}
                  type="number"
                  min={min}
                  max={max}
                  step={step}
                  value={config.vad[key] ?? 0}
                  onChange={(event) =>
                    change({
                      vad: { ...config.vad, [key]: Number(event.target.value) },
                    })
                  }
                />
              ))}
            </div>
          )}
        </details>
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {notice && <StatusNotice>{t("transAdmin.saved")}</StatusNotice>}
        <div>
          <button disabled={busy} className={buttonClass("primary")}>
            {t("transAdmin.save")}
          </button>
        </div>
      </form>
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("transAdmin.workers")}</h2>
        <div
          role="region"
          tabIndex={0}
          aria-label={t("transAdmin.workers")}
          className="overflow-x-auto"
        >
          <table className="w-full min-w-[38rem] text-left">
            <thead>
              <tr>
                {(
                  ["name", "server", "sent", "language", "status"] as const
                ).map((key) => (
                  <th key={key} scope="col" className="p-2">
                    {t(`transAdmin.columns.${key}`)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {state.value.workers.items.map((worker) => {
                const sync = settings.workers?.find(
                  (item) => item.id === worker.id,
                );
                const syncState =
                  sync?.error_code === "model_missing"
                    ? "modelMissing"
                    : sync?.error_code === "connection_failed"
                      ? "connectionFailed"
                      : sync?.active_version === settings.version &&
                          settings.version
                        ? "active"
                        : sync?.test_requested || sync?.received_version
                          ? "pending"
                          : "notReceived";
                return (
                  <tr key={worker.id} className="border-t border-line">
                    <td className="p-2">
                      <button
                        type="button"
                        className="text-accent-ink underline"
                        onClick={() => {
                          setSelected(worker);
                          setOverride(worker.transcription_model ?? "");
                        }}
                      >
                        {worker.name}
                      </button>
                    </td>
                    <td className="p-2">
                      {worker.models?.transcription?.endpoint ??
                        t("transAdmin.undeclared")}
                    </td>
                    <td className="p-2">
                      {worker.transcription_model || config.model}
                    </td>
                    <td className="p-2">{config.language}</td>
                    <td className="p-2">
                      <p role="status">{t(`transAdmin.sync.${syncState}`)}</p>
                      <p className="text-mute">
                        {worker.connected
                          ? t("transAdmin.online")
                          : t("transAdmin.offline")}
                      </p>
                      {sync && (
                        <p className="text-mute text-xs">
                          {t("transAdmin.sync.versions", {
                            received: sync.received_version || "—",
                            tested: sync.tested_version || "—",
                            active: sync.active_version || "—",
                          })}
                        </p>
                      )}
                      {settings.mode === "enabled" && (
                        <button
                          type="button"
                          disabled={busy}
                          className="text-accent-ink underline"
                          onClick={() => void testWorker(worker.id)}
                        >
                          {t("transAdmin.sync.test")}
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>
      <Modal
        open={!!selected}
        onOpenChange={(open) => {
          if (!open) setSelected(undefined);
        }}
        title={selected?.name ?? ""}
        closeLabel={t("transAdmin.close")}
      >
        <form
          onSubmit={(event) => void saveOverride(event)}
          className="flex flex-col gap-4"
        >
          <p>{t("transAdmin.overrideHint")}</p>
          <Field
            label={t("transAdmin.override")}
            value={override}
            maxLength={100}
            onChange={(event) => setOverride(event.target.value)}
          />
          {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          <div>
            <button disabled={busy} className={buttonClass("primary")}>
              {t("transAdmin.saveSimple")}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
