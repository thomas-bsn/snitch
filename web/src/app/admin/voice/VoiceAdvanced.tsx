import { useState, type FormEvent } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

type Pipeline = components["schemas"]["VoicePipeline"];
type Matching = components["schemas"]["VoiceMatchingSettings"];
type Accuracy = components["schemas"]["VoiceAccuracy"];

// The values the installation starts with: a reset goes back to them.
const DEFAULTS = {
  automatic_threshold: 0.9,
  suggestion_threshold: 0.75,
  ambiguity_margin: 0.05,
  max_candidates: 5,
  min_fingerprint_seconds: 1,
  min_quality: 0.5,
  precision_threshold: 0.95,
  min_confirmations: 50,
};

const pct = (x: number | null, none: string) =>
  x === null ? none : `${Math.round(x * 100)}%`;

/** The thresholds, folded away: nothing here is needed to use the screen. */
export function VoiceAdvanced({
  pipeline,
  matching,
  accuracy,
  onChanged,
}: {
  pipeline: Pipeline;
  matching: Matching;
  accuracy: Accuracy;
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const headers = mutation(session?.csrf_token);
  const [fp, setFp] = useState<Partial<Pipeline>>();
  const [match, setMatch] = useState<Partial<Matching>>();
  const [precision, setPrecision] = useState<string>();
  const [confirmations, setConfirmations] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [summary, setSummary] =
    useState<components["schemas"]["VoiceMatchingSummary"]>();
  const pipe = { ...pipeline, ...fp };
  const m = { ...matching, ...match };
  const none = t("voiceAdmin.notMeasured");
  async function run(
    action: () => Promise<{ error?: unknown }>,
    done?: () => void,
  ) {
    setBusy(true);
    setFailure(undefined);
    setNotice(false);
    try {
      const result = await action();
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        done?.();
        setNotice(true);
        onChanged();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setResetting(false);
    }
  }
  const savePipeline = (e: FormEvent) => {
    e.preventDefault();
    void run(
      () =>
        api.PUT("/api/admin/voice-pipeline", {
          params: { header: headers },
          body: pipe,
        }),
      () => setFp(undefined),
    );
  };
  const saveMatching = (e: FormEvent) => {
    e.preventDefault();
    void run(
      () =>
        api.PUT("/api/admin/voice-matching", {
          params: { header: headers },
          body: {
            automatic_threshold: m.automatic_threshold,
            suggestion_threshold: m.suggestion_threshold,
            ambiguity_margin: m.ambiguity_margin,
            max_candidates: m.max_candidates,
          },
        }),
      () => setMatch(undefined),
    );
  };
  const criteriaBody = (p: number, c: number) =>
    api.PUT("/api/admin/voice-automation/criteria", {
      params: { header: headers },
      body: { precision_threshold: p, min_confirmations: c },
    });
  const saveCriteria = (e: FormEvent) => {
    e.preventDefault();
    void run(
      () =>
        criteriaBody(
          Number(precision ?? accuracy.precision_threshold),
          Number(confirmations ?? accuracy.min_confirmations),
        ),
      () => {
        setPrecision(undefined);
        setConfirmations(undefined);
      },
    );
  };
  function reset() {
    void run(
      async () => {
        const a = await api.PUT("/api/admin/voice-matching", {
          params: { header: headers },
          body: {
            automatic_threshold: DEFAULTS.automatic_threshold,
            suggestion_threshold: DEFAULTS.suggestion_threshold,
            ambiguity_margin: DEFAULTS.ambiguity_margin,
            max_candidates: DEFAULTS.max_candidates,
          },
        });
        if (a.error) return a;
        const b = await criteriaBody(
          DEFAULTS.precision_threshold,
          DEFAULTS.min_confirmations,
        );
        if (b.error) return b;
        if (
          pipeline.min_fingerprint_seconds ===
            DEFAULTS.min_fingerprint_seconds &&
          pipeline.min_quality === DEFAULTS.min_quality
        )
          return b;
        return api.PUT("/api/admin/voice-pipeline", {
          params: { header: headers },
          body: {
            ...pipeline,
            min_fingerprint_seconds: DEFAULTS.min_fingerprint_seconds,
            min_quality: DEFAULTS.min_quality,
          },
        });
      },
      () => {
        setFp(undefined);
        setMatch(undefined);
        setPrecision(undefined);
        setConfirmations(undefined);
      },
    );
  }
  const num = (v: string) => Number(v);
  return (
    <details className="group rounded-xl border border-line">
      <summary className="flex min-h-11 cursor-pointer items-center justify-between gap-2 rounded-xl p-4 font-semibold">
        {t("voiceScreen.advanced")}
        <span className="text-sm font-normal text-mute">
          {t("voiceScreen.advancedHint")}
        </span>
      </summary>
      <div className="flex flex-col gap-6 p-4 pt-0">
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {notice && <StatusNotice>{t("voiceAdmin.saved")}</StatusNotice>}

        <section aria-labelledby="adv-matching" className="flex flex-col gap-3">
          <h4 id="adv-matching" className="font-semibold">
            {t("voiceAdmin.matching")}
          </h4>
          <p className="text-mute">{t("voiceAdmin.matchingHint")}</p>
          <form
            onSubmit={saveMatching}
            className="flex max-w-xl flex-col gap-3"
          >
            <Field
              label={t("voiceAdmin.autoThreshold")}
              hint={t("voiceScreen.autoThresholdHint")}
              type="number"
              min={0}
              max={1}
              step="any"
              value={m.automatic_threshold}
              onChange={(e) =>
                setMatch({ ...match, automatic_threshold: num(e.target.value) })
              }
            />
            <Field
              label={t("voiceAdmin.suggestionThreshold")}
              hint={t("voiceScreen.suggestionHint")}
              type="number"
              min={0}
              max={1}
              step="any"
              value={m.suggestion_threshold}
              onChange={(e) =>
                setMatch({
                  ...match,
                  suggestion_threshold: num(e.target.value),
                })
              }
            />
            <Field
              label={t("voiceAdmin.margin")}
              hint={t("voiceScreen.marginHint")}
              type="number"
              min={0}
              max={1}
              step="any"
              value={m.ambiguity_margin}
              onChange={(e) =>
                setMatch({ ...match, ambiguity_margin: num(e.target.value) })
              }
            />
            <Field
              label={t("voiceAdmin.candidates")}
              hint={t("voiceScreen.candidatesHint")}
              type="number"
              min={2}
              max={20}
              value={m.max_candidates}
              onChange={(e) =>
                setMatch({ ...match, max_candidates: num(e.target.value) })
              }
            />
            <div className="flex flex-wrap gap-2">
              <button disabled={busy} className={buttonClass("primary")}>
                {t("voiceAdmin.saveThresholds")}
              </button>
              <button
                type="button"
                disabled={busy}
                className={buttonClass()}
                onClick={() =>
                  void run(async () => {
                    const r = await api.POST(
                      "/api/admin/voice-matching/rematch",
                      { params: { header: headers } },
                    );
                    if (r.data) setSummary(r.data);
                    return r;
                  })
                }
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

        <section
          aria-labelledby="adv-fingerprint"
          className="flex flex-col gap-3"
        >
          <h4 id="adv-fingerprint" className="font-semibold">
            {t("voiceScreen.fingerprintSettings")}
          </h4>
          <p className="text-mute">{t("voiceScreen.fingerprintHint")}</p>
          <form
            onSubmit={savePipeline}
            className="flex max-w-xl flex-col gap-3"
          >
            <Field
              label={t("voiceAdmin.minDuration")}
              type="number"
              min={0.1}
              max={60}
              step={0.1}
              value={pipe.min_fingerprint_seconds}
              onChange={(e) =>
                setFp({ ...fp, min_fingerprint_seconds: num(e.target.value) })
              }
            />
            <Field
              label={t("voiceAdmin.minQuality")}
              type="number"
              min={0}
              max={1}
              step={0.05}
              value={pipe.min_quality}
              onChange={(e) =>
                setFp({ ...fp, min_quality: num(e.target.value) })
              }
            />
            <div>
              <button disabled={busy} className={buttonClass("primary")}>
                {t("voiceAdmin.savePipeline")}
              </button>
            </div>
          </form>
        </section>

        <section
          aria-labelledby="adv-automation"
          className="flex flex-col gap-3"
        >
          <h4 id="adv-automation" className="font-semibold">
            {t("voiceScreen.automationSettings")}
          </h4>
          <p className="text-mute">{t("voiceAdmin.measureHint")}</p>
          <ul className="list-disc pl-5">
            <li>
              {t("voiceAdmin.precision", {
                value: pct(accuracy.evaluation.precision, none),
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
                value: pct(accuracy.evaluation.coverage, none),
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
                      <td className="p-2">{pct(row.precision, none)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <form
            onSubmit={saveCriteria}
            className="flex max-w-xl flex-col gap-3"
          >
            <Field
              label={t("voiceAdmin.minPrecision")}
              hint={t("voiceScreen.precisionHint")}
              type="number"
              min={0}
              max={1}
              step="any"
              value={precision ?? String(accuracy.precision_threshold)}
              onChange={(e) => setPrecision(e.target.value)}
            />
            <Field
              label={t("voiceAdmin.minConfirmations")}
              hint={t("voiceScreen.confirmationsHint")}
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
          {accuracy.events.length > 0 && (
            <ul className="list-disc pl-5">
              {accuracy.events.map((event, i) => (
                <li key={i}>
                  {event.action === "enable"
                    ? t("voiceAdmin.enabledEvent")
                    : t("voiceAdmin.disabledEvent")}{" "}
                  · {event.author} ·{" "}
                  {new Date(event.created_at).toLocaleString()}
                  {event.forced && ` · ${t("voiceAdmin.forcedEvent")}`}
                </li>
              ))}
            </ul>
          )}
        </section>

        <div>
          <button
            type="button"
            disabled={busy}
            onClick={() => setResetting(true)}
            className={buttonClass("danger")}
          >
            {t("voiceScreen.reset")}
          </button>
        </div>
      </div>
      <ConfirmDialog
        open={resetting}
        onOpenChange={(open) => !open && setResetting(false)}
        title={t("voiceScreen.resetTitle")}
        cancelLabel={t("voiceModels.cancelDialog")}
        confirmLabel={t("voiceScreen.resetConfirm")}
        danger
        busy={busy}
        onConfirm={reset}
      >
        <p>{t("voiceScreen.resetText")}</p>
      </ConfirmDialog>
    </details>
  );
}
