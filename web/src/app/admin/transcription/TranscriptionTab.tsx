import { useCallback, useState, type FormEvent } from "react";
import { Link } from "react-router";
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
async function load() {
  try {
    const workers = await fetchWorkers();
    if (!workers.ok) return workers;
    const { data, error } = await api.GET("/api/admin/transcription-pipeline");
    if (!data)
      return { ok: false, code: errorCode(error) ?? "unknown" } as const;
    return {
      ok: true,
      value: { workers: workers.value, pipeline: data },
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
  const [busy, setBusy] = useState(false);
  const read = useCallback(() => reload(), [reload]);
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  if (state.value.workers.first_worker)
    return (
      <div className="flex flex-col gap-4">
        <p>{t("transAdmin.empty")}</p>
        <div>
          <Link className={buttonClass("primary")} to="/admin/workers/new">
            {t("transAdmin.create")}
          </Link>
        </div>
      </div>
    );
  const config = draft ?? state.value.pipeline.config;
  function change(part: Partial<Pipeline>) {
    setDraft({ ...config, ...part });
    setNotice(false);
  }
  async function save(event: FormEvent) {
    event.preventDefault();
    setFailure(undefined);
    setBusy(true);
    try {
      const result = await api.PUT("/api/admin/transcription-pipeline", {
        params: { header: mutation(session?.csrf_token) },
        body: config,
      });
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unknown");
        return;
      }
      setDraft(undefined);
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
  return (
    <div className="flex flex-col gap-6">
      <form
        onSubmit={(event) => void save(event)}
        className="flex max-w-xl flex-col gap-4"
      >
        <h2 className="text-lg font-semibold">{t("transAdmin.title")}</h2>
        <p className="text-mute">
          {t("transAdmin.version", { version: state.value.pipeline.version })}
        </p>
        <Field
          label={t("transAdmin.model")}
          required
          maxLength={100}
          value={config.model}
          onChange={(event) => change({ model: event.target.value })}
        />
        <Field
          label={t("transAdmin.language")}
          required
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
              {state.value.workers.items.map((worker) => (
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
                    {worker.connected
                      ? t("transAdmin.online")
                      : t("transAdmin.offline")}
                  </td>
                </tr>
              ))}
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
