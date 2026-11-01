import { useState, type FormEvent } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { VoiceInventory } from "./VoiceInventory";

type Pipeline = components["schemas"]["VoicePipeline"];
type Matching = components["schemas"]["VoiceMatchingSettings"];
async function load() {
  try {
    const [pipeline, matching, accuracy] = await Promise.all([
      api.GET("/api/admin/voice-pipeline"),
      api.GET("/api/voice-matching"),
      api.GET("/api/admin/voice-accuracy"),
    ]);
    if (pipeline.data && matching.data && accuracy.data)
      return {
        ok: true as const,
        value: {
          pipeline: pipeline.data,
          matching: matching.data,
          accuracy: accuracy.data,
        },
      };
    return {
      ok: false as const,
      code:
        errorCode(pipeline.error ?? matching.error ?? accuracy.error) ??
        "unknown",
    };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}
export function VoiceTab() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  const [pipeline, setPipeline] = useState<Pipeline>();
  const [matching, setMatching] = useState<Matching>();
  const [precision, setPrecision] = useState<string>();
  const [confirmations, setConfirmations] = useState<string>();
  const [forced, setForced] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState(false);
  const [summary, setSummary] =
    useState<components["schemas"]["VoiceMatchingSummary"]>();
  const headers = mutation(session?.csrf_token);
  async function run(action: () => Promise<{ error?: unknown }>) {
    setBusy(true);
    setFailure(undefined);
    setNotice(false);
    try {
      const result = await action();
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        setNotice(true);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  const config = pipeline ?? state.value.pipeline.config;
  const match = matching ?? state.value.matching;
  const accuracy = state.value.accuracy;
  // A real model is never typed: switching to pyannote takes the first diarization model of the catalogue.
  async function changeProvider(provider: Pipeline["provider"]) {
    if (provider === "fake") {
      patch({ provider, fake: { speakers: 2, dimensions: 8 } });
      return;
    }
    const models = await api.GET("/api/admin/voice-models");
    const first = models.data?.models.find((m) => m.kind === "diarization");
    patch({
      provider,
      fake: undefined,
      ...(first ? { model: first.repo, model_version: first.revision } : {}),
    });
  }
  function patch(value: Partial<Pipeline>) {
    setPipeline({ ...config, ...value });
  }
  function patchMatch(value: Partial<Matching>) {
    setMatching({ ...match, ...value });
  }
  function savePipeline(e: FormEvent) {
    e.preventDefault();
    void run(async () => {
      const r = await api.PUT("/api/admin/voice-pipeline", {
        params: { header: headers },
        body: config,
      });
      if (r.data) setPipeline(undefined);
      return r;
    });
  }
  function saveMatching(e: FormEvent) {
    e.preventDefault();
    void run(async () => {
      const r = await api.PUT("/api/admin/voice-matching", {
        params: { header: headers },
        body: {
          automatic_threshold: match.automatic_threshold,
          suggestion_threshold: match.suggestion_threshold,
          ambiguity_margin: match.ambiguity_margin,
          max_candidates: match.max_candidates,
        },
      });
      if (r.data) setMatching(undefined);
      return r;
    });
  }
  function saveCriteria(e: FormEvent) {
    e.preventDefault();
    void run(async () => {
      const r = await api.PUT("/api/admin/voice-automation/criteria", {
        params: { header: headers },
        body: {
          precision_threshold: Number(
            precision ?? accuracy.precision_threshold,
          ),
          min_confirmations: Number(
            confirmations ?? accuracy.min_confirmations,
          ),
        },
      });
      if (r.data) {
        setPrecision(undefined);
        setConfirmations(undefined);
      }
      return r;
    });
  }
  function toggle(enabled: boolean) {
    void run(async () => {
      const r = await api.PUT("/api/admin/voice-automation", {
        params: { header: headers },
        body: { enabled, confirm_unmet_criteria: forced },
      });
      if (r.data) {
        setForced(false);
        setMatching(undefined);
      }
      return r;
    });
  }
  function rematch() {
    void run(async () => {
      const r = await api.POST("/api/admin/voice-matching/rematch", {
        params: { header: headers },
      });
      if (r.data) setSummary(r.data);
      return r;
    });
  }
  const pct = (x: number | null) =>
    x === null ? t("voiceAdmin.notMeasured") : `${Math.round(x * 100)}%`;
  return (
    <div className="flex max-w-4xl flex-col gap-6">
      <h2 className="text-lg font-semibold">{t("voiceAdmin.title")}</h2>
      <VoiceInventory />
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <StatusNotice>{t("voiceAdmin.saved")}</StatusNotice>}
      <section className="rounded-xl border border-line p-4">
        <h3 className="font-semibold">{t("voiceAdmin.pipeline")}</h3>
        <p className="text-mute">
          {t("voiceAdmin.version", { version: state.value.pipeline.version })}
        </p>
        <form
          onSubmit={savePipeline}
          className="mt-3 flex max-w-xl flex-col gap-3"
        >
          <label className="flex flex-col gap-1.5">
            {t("voiceAdmin.provider")}
            <select
              value={config.provider}
              onChange={(e) =>
                void changeProvider(e.target.value as Pipeline["provider"])
              }
              className="h-11 rounded-lg border border-line-strong bg-surface px-3"
            >
              <option value="pyannote">pyannote</option>
              <option value="fake">{t("voiceAdmin.fake")}</option>
            </select>
          </label>
          <Field
            label={t("voiceAdmin.model")}
            value={config.model}
            required
            readOnly={config.provider === "pyannote"}
            onChange={(e) => patch({ model: e.target.value })}
          />
          <Field
            label={t("voiceAdmin.modelVersion")}
            value={config.model_version}
            required
            readOnly={config.provider === "pyannote"}
            onChange={(e) => patch({ model_version: e.target.value })}
          />
          <Field
            label={t("voiceAdmin.minDuration")}
            type="number"
            min={0.1}
            max={60}
            step={0.1}
            value={config.min_fingerprint_seconds}
            onChange={(e) =>
              patch({ min_fingerprint_seconds: Number(e.target.value) })
            }
          />
          <Field
            label={t("voiceAdmin.minQuality")}
            type="number"
            min={0}
            max={1}
            step={0.05}
            value={config.min_quality}
            onChange={(e) => patch({ min_quality: Number(e.target.value) })}
          />
          {config.fake && (
            <Field
              label={t("voiceAdmin.fakeSpeakers")}
              type="number"
              min={1}
              max={8}
              value={config.fake.speakers}
              onChange={(e) =>
                patch({
                  fake: { ...config.fake!, speakers: Number(e.target.value) },
                })
              }
            />
          )}
          <div>
            <button disabled={busy} className={buttonClass("primary")}>
              {t("voiceAdmin.savePipeline")}
            </button>
          </div>
        </form>
      </section>
      <section className="rounded-xl border border-line p-4">
        <h3 className="font-semibold">{t("voiceAdmin.matching")}</h3>
        <p className="text-mute">{t("voiceAdmin.matchingHint")}</p>
        <form
          onSubmit={saveMatching}
          className="mt-3 flex max-w-xl flex-col gap-3"
        >
          <Field
            label={t("voiceAdmin.autoThreshold")}
            type="number"
            min={0}
            max={1}
            step="any"
            value={match.automatic_threshold}
            onChange={(e) =>
              patchMatch({ automatic_threshold: Number(e.target.value) })
            }
          />
          <Field
            label={t("voiceAdmin.suggestionThreshold")}
            type="number"
            min={0}
            max={1}
            step="any"
            value={match.suggestion_threshold}
            onChange={(e) =>
              patchMatch({ suggestion_threshold: Number(e.target.value) })
            }
          />
          <Field
            label={t("voiceAdmin.margin")}
            type="number"
            min={0}
            max={1}
            step="any"
            value={match.ambiguity_margin}
            onChange={(e) =>
              patchMatch({ ambiguity_margin: Number(e.target.value) })
            }
          />
          <Field
            label={t("voiceAdmin.candidates")}
            type="number"
            min={2}
            max={20}
            value={match.max_candidates}
            onChange={(e) =>
              patchMatch({ max_candidates: Number(e.target.value) })
            }
          />
          <div className="flex flex-wrap gap-2">
            <button disabled={busy} className={buttonClass("primary")}>
              {t("voiceAdmin.saveThresholds")}
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={rematch}
              className={buttonClass()}
            >
              {t("voiceAdmin.rematch")}
            </button>
          </div>
        </form>
        {summary && (
          <p role="status">
            {t("voiceAdmin.rematched", {
              count: summary.evaluated,
              automatic: summary.automatic,
              suggestion: summary.suggestion,
              ambiguous: summary.ambiguous,
              unknown: summary.unknown,
            })}
          </p>
        )}
      </section>
      <section className="rounded-xl border border-line p-4">
        <h3 className="font-semibold">{t("voiceAdmin.automation")}</h3>
        <p>
          {accuracy.auto_enabled
            ? t("voiceAdmin.enabled")
            : t("voiceAdmin.disabled")}
        </p>
        <p className="text-mute">{t("voiceAdmin.measureHint")}</p>
        <ul className="my-3 list-disc pl-5">
          <li>
            {t("voiceAdmin.precision", {
              value: pct(accuracy.evaluation.precision),
            })}
          </li>
          <li>
            {t("voiceAdmin.confirmed", {
              value: accuracy.evaluation.confirmed,
            })}
          </li>
          <li>
            {t("voiceAdmin.tests", {
              test: accuracy.evaluation.test_samples,
              reference: accuracy.evaluation.reference_samples,
              excluded: accuracy.evaluation.excluded_samples,
            })}
          </li>
          <li>
            {t("voiceAdmin.coverage", {
              value: pct(accuracy.evaluation.coverage),
            })}
          </li>
          <li>
            {t("voiceAdmin.library", {
              automatic: accuracy.library.automatic,
              pending: accuracy.library.to_verify,
              validated: accuracy.library.validated,
              microphone: accuracy.library.microphone_assigned,
            })}
          </li>
        </ul>
        {accuracy.persons.length > 0 && (
          <div
            role="region"
            tabIndex={0}
            aria-label={t("voiceAdmin.perPerson")}
            className="overflow-x-auto"
          >
            <table className="w-full min-w-[36rem] text-left">
              <caption className="text-left font-medium">
                {t("voiceAdmin.perPerson")}
              </caption>
              <thead>
                <tr>
                  {(
                    [
                      "person",
                      "references",
                      "testsColumn",
                      "predictions",
                      "precisionColumn",
                    ] as const
                  ).map((k) => (
                    <th className="p-2" scope="col" key={k}>
                      {t(`voiceAdmin.${k}`)}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {accuracy.persons.map((row) => (
                  <tr key={row.person.id} className="border-t border-line">
                    <td className="p-2">{row.person.display_name}</td>
                    <td className="p-2">{row.reference_samples}</td>
                    <td className="p-2">{row.test_samples}</td>
                    <td className="p-2">{row.automatic_predictions}</td>
                    <td className="p-2">{pct(row.precision)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {accuracy.activation_proposed && (
          <StatusNotice>{t("voiceAdmin.proposed")}</StatusNotice>
        )}
        <form
          onSubmit={saveCriteria}
          className="my-3 flex max-w-xl flex-col gap-3"
        >
          <Field
            label={t("voiceAdmin.minPrecision")}
            type="number"
            min={0}
            max={1}
            step="any"
            value={precision ?? String(accuracy.precision_threshold)}
            onChange={(e) => setPrecision(e.target.value)}
          />
          <Field
            label={t("voiceAdmin.minConfirmations")}
            type="number"
            min={1}
            value={confirmations ?? String(accuracy.min_confirmations)}
            onChange={(e) => setConfirmations(e.target.value)}
          />
          <div>
            <button disabled={busy} className={buttonClass()}>
              {t("voiceAdmin.saveCriteria")}
            </button>
          </div>
        </form>
        {!accuracy.auto_enabled && !accuracy.criteria.met && (
          <label className="flex min-h-11 items-center gap-2">
            <input
              type="checkbox"
              checked={forced}
              onChange={(e) => setForced(e.target.checked)}
            />
            {t("voiceAdmin.force")}
          </label>
        )}
        <button
          type="button"
          disabled={
            busy ||
            (!accuracy.auto_enabled && !accuracy.criteria.met && !forced)
          }
          onClick={() => toggle(!accuracy.auto_enabled)}
          className={buttonClass()}
        >
          {accuracy.auto_enabled
            ? t("voiceAdmin.disable")
            : t("voiceAdmin.enable")}
        </button>
        {accuracy.events.length > 0 && (
          <ul className="mt-3 list-disc pl-5">
            {accuracy.events.map((event, i) => (
              <li key={i}>
                {event.action === "enable"
                  ? t("voiceAdmin.enabledEvent")
                  : t("voiceAdmin.disabledEvent")}{" "}
                · {event.author} · {new Date(event.created_at).toLocaleString()}
                {event.forced && ` · ${t("voiceAdmin.forcedEvent")}`}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
