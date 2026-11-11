import { useState, type FormEvent } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { MatchingStatus } from "../../voices/MatchingStatus";
import {
  EligibilityPreview,
  RequalificationReport,
} from "./EligibilityPreview";
import { FingerprintRecompute } from "./FingerprintRecompute";
import { VoiceCalibration } from "./VoiceCalibration";

type Eligibility = components["schemas"]["VoiceEligibility"];
type EligibilityInput = components["schemas"]["VoiceEligibilityInput"];
type Preview = components["schemas"]["VoiceEligibilityPreview"];
type Matching = components["schemas"]["VoiceMatchingSettings"];
type ScoreMode = components["schemas"]["VoiceScoreMode"];
type ScoreModeName = ScoreMode["mode"];
type Accuracy = components["schemas"]["VoiceAccuracy"];
type Thresholds = components["schemas"]["VoiceGroupThresholds"];

// The values the installation starts with: a reset goes back to them. The eligibility rule's come from the
// server (its provisional version).
const DEFAULTS = {
  automatic_threshold: 0.9,
  suggestion_threshold: 0.75,
  ambiguity_margin: 0.05,
  max_candidates: 5,
  precision_threshold: 0.95,
  min_confirmations: 50,
};

const pct = (x: number | null, none: string) =>
  x === null ? none : `${Math.round(x * 100)}%`;

/** The thresholds, folded away: nothing here is needed to use the screen. */
export function VoiceAdvanced({
  eligibility,
  scoreMode,
  matching,
  accuracy,
  onChanged,
}: {
  eligibility: Eligibility;
  scoreMode: ScoreMode;
  matching: Matching;
  accuracy: Accuracy;
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const headers = mutation(session?.csrf_token);
  const [rule, setRule] = useState<Partial<EligibilityInput>>();
  const [requalified, setRequalified] = useState<number>();
  const [preview, setPreview] = useState<{
    figures: Preview;
    rule: EligibilityInput;
  }>();
  const [match, setMatch] = useState<Partial<Matching>>();
  const [mode, setMode] = useState<ScoreModeName>();
  const [precision, setPrecision] = useState<string>();
  const [confirmations, setConfirmations] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [summary, setSummary] =
    useState<components["schemas"]["VoiceMatchingSummary"]>();
  const current: EligibilityInput = {
    comparable_min_seconds: eligibility.rule.comparable_min_seconds,
    contributive_min_seconds: eligibility.rule.contributive_min_seconds,
    max_clipped_ratio: eligibility.rule.max_clipped_ratio,
  };
  const r = { ...current, ...rule };
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
  const putRule = async (body: EligibilityInput) => {
    const result = await api.PUT("/api/admin/voice-eligibility", {
      params: { header: headers },
      body,
    });
    if (result.data) {
      setRequalified(result.data.requalified);
      setPreview(undefined);
    }
    return result;
  };
  // The preview writes nothing: it is not a saved setting.
  async function loadPreview() {
    const examined = r;
    setBusy(true);
    setFailure(undefined);
    setNotice(false);
    try {
      const result = await api.GET("/api/admin/voice-eligibility/preview", {
        params: { query: examined },
      });
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else setPreview({ figures: result.data, rule: examined });
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  const saveRule = (e: FormEvent) => {
    e.preventDefault();
    void run(
      () => putRule(r),
      () => setRule(undefined),
    );
  };
  // Thresholds go to the fingerprint group the screen shows: if the model changed meanwhile, the server refuses.
  const group = matching.group ?? undefined;
  const putThresholds = (
    values: Pick<
      Matching,
      "automatic_threshold" | "suggestion_threshold" | "ambiguity_margin"
    >,
    source: "administrator" | "proposal" = "administrator",
  ) =>
    api.PUT("/api/admin/voice-matching", {
      params: { header: headers },
      body: {
        automatic_threshold: values.automatic_threshold,
        suggestion_threshold: values.suggestion_threshold,
        ambiguity_margin: values.ambiguity_margin,
        max_candidates: m.max_candidates,
        ...(group ? { group, source } : {}),
      },
    });
  const saveMatching = (e: FormEvent) => {
    e.preventDefault();
    void run(
      () => putThresholds(m),
      () => setMatch(undefined),
    );
  };
  const thresholds = accuracy.calibration.thresholds;
  const sourceLabel = (v: Thresholds) => t(`voiceAdmin.thresholds.${v.source}`);
  const saveScoreMode = (e: FormEvent) => {
    e.preventDefault();
    void run(
      () =>
        api.PUT("/api/admin/voice-score-mode", {
          params: { header: headers },
          body: { mode: mode ?? scoreMode.mode },
        }),
      () => setMode(undefined),
    );
  };
  const modeLabel = (m: ScoreModeName) =>
    m === "cumulative"
      ? t("voiceAdmin.score.cumulative")
      : t("voiceAdmin.score.bestSample");
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
            ...(group ? { group } : {}),
          },
        });
        if (a.error) return a;
        const b = await criteriaBody(
          DEFAULTS.precision_threshold,
          DEFAULTS.min_confirmations,
        );
        if (b.error) return b;
        if (scoreMode.mode !== "cumulative") {
          const c = await api.PUT("/api/admin/voice-score-mode", {
            params: { header: headers },
            body: { mode: "cumulative" },
          });
          if (c.error) return c;
        }
        const d = eligibility.defaults;
        if (
          d.comparable_min_seconds === current.comparable_min_seconds &&
          d.contributive_min_seconds === current.contributive_min_seconds &&
          d.max_clipped_ratio === current.max_clipped_ratio
        )
          return b;
        return putRule(d);
      },
      () => {
        setRule(undefined);
        setMatch(undefined);
        setMode(undefined);
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
          <MatchingStatus />
          <p className="font-medium">
            {group
              ? t("voiceAdmin.thresholds.group", {
                  model: group.model,
                  version: group.model_version,
                  dimensions: group.dimensions,
                })
              : t("voiceAdmin.thresholds.noGroup")}
          </p>
          {group && (
            <p>
              {thresholds
                ? t("voiceAdmin.thresholds.applied", {
                    version: thresholds.version,
                    source: sourceLabel(thresholds),
                    author:
                      thresholds.author ?? t("voiceAdmin.eligibility.snitch"),
                    date: new Date(thresholds.created_at).toLocaleString(),
                  })
                : t("voiceAdmin.thresholds.none")}
            </p>
          )}
          <p className="text-sm text-mute">
            {t("voiceAdmin.thresholds.scoreNote")}
          </p>
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
          {accuracy.calibration.history.length > 1 && (
            <div>
              <p id="adv-thresholds-history" className="font-medium">
                {t("voiceAdmin.thresholds.history")}
              </p>
              <ul
                aria-labelledby="adv-thresholds-history"
                className="list-disc pl-5"
              >
                {accuracy.calibration.history.slice(1).map((v) => (
                  <li key={v.version}>
                    {t("voiceAdmin.thresholds.historyItem", {
                      version: v.version,
                      automatic: v.automatic_threshold,
                      suggestion: v.suggestion_threshold,
                      margin: v.ambiguity_margin,
                      source: sourceLabel(v),
                      author: v.author ?? t("voiceAdmin.eligibility.snitch"),
                      date: new Date(v.created_at).toLocaleString(),
                    })}
                  </li>
                ))}
              </ul>
            </div>
          )}
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

        <section aria-labelledby="adv-score" className="flex flex-col gap-3">
          <h4 id="adv-score" className="font-semibold">
            {t("voiceAdmin.score.title")}
          </h4>
          <p className="text-mute">{t("voiceAdmin.score.hint")}</p>
          <p>
            {scoreMode.author === null
              ? t("voiceAdmin.score.defaultMode", {
                  version: scoreMode.version,
                })
              : t("voiceAdmin.score.current", {
                  version: scoreMode.version,
                  author: scoreMode.author,
                  date: new Date(scoreMode.created_at).toLocaleString(),
                })}
          </p>
          <form
            onSubmit={saveScoreMode}
            className="flex max-w-xl flex-col gap-3"
          >
            <fieldset className="flex flex-col gap-2">
              <legend className="font-medium">
                {t("voiceAdmin.score.legend")}
              </legend>
              {(["cumulative", "best_sample"] as const).map((value) => (
                <label key={value} className="flex min-h-11 gap-2">
                  <input
                    type="radio"
                    name="score-mode"
                    value={value}
                    checked={(mode ?? scoreMode.mode) === value}
                    onChange={() => setMode(value)}
                    aria-describedby={`score-mode-${value}-hint`}
                    className="mt-1"
                  />
                  <span className="flex flex-col">
                    <span>{modeLabel(value)}</span>
                    <span
                      id={`score-mode-${value}-hint`}
                      className="text-sm text-mute"
                    >
                      {value === "cumulative"
                        ? t("voiceAdmin.score.cumulativeHint")
                        : t("voiceAdmin.score.bestSampleHint")}
                    </span>
                  </span>
                </label>
              ))}
            </fieldset>
            <div>
              <button
                disabled={busy || (mode ?? scoreMode.mode) === scoreMode.mode}
                className={buttonClass("primary")}
              >
                {t("voiceAdmin.score.save")}
              </button>
            </div>
          </form>
          {scoreMode.history.length > 1 && (
            <div>
              <p id="adv-score-history" className="font-medium">
                {t("voiceAdmin.score.history")}
              </p>
              <ul
                aria-labelledby="adv-score-history"
                className="list-disc pl-5"
              >
                {scoreMode.history.slice(1).map((v) => (
                  <li key={v.version}>
                    {t("voiceAdmin.score.historyItem", {
                      version: v.version,
                      mode: modeLabel(v.mode),
                      author: v.author ?? t("voiceAdmin.eligibility.snitch"),
                      date: new Date(v.created_at).toLocaleString(),
                    })}
                  </li>
                ))}
              </ul>
            </div>
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
          <p>
            {eligibility.rule.provisional
              ? t("voiceAdmin.eligibility.provisional", {
                  version: eligibility.rule.version,
                })
              : t("voiceAdmin.eligibility.rule", {
                  version: eligibility.rule.version,
                  author: eligibility.rule.author ?? "",
                  date: new Date(eligibility.rule.created_at).toLocaleString(),
                })}
          </p>
          <p>
            {t("voiceAdmin.eligibility.summary", {
              comparable: eligibility.summary.comparable,
              contributive: eligibility.summary.contributive_only,
              tooShort: eligibility.summary.too_short,
              overlap: eligibility.summary.overlap_only,
              clipped: eligibility.summary.clipped,
              lowQuality: eligibility.summary.low_quality,
            })}
          </p>
          {eligibility.summary.stale > 0 && (
            <p className="text-mute">
              {t("voiceAdmin.eligibility.stale", {
                count: eligibility.summary.stale,
              })}
            </p>
          )}
          <form onSubmit={saveRule} className="flex max-w-xl flex-col gap-3">
            <Field
              label={t("voiceAdmin.eligibility.comparableMin")}
              hint={t("voiceAdmin.eligibility.comparableHint")}
              type="number"
              min={0.5}
              max={60}
              step={0.1}
              value={r.comparable_min_seconds}
              onChange={(e) =>
                setRule({
                  ...rule,
                  comparable_min_seconds: num(e.target.value),
                })
              }
            />
            <Field
              label={t("voiceAdmin.eligibility.contributiveMin")}
              hint={t("voiceAdmin.eligibility.contributiveHint")}
              type="number"
              min={0.5}
              max={60}
              step={0.1}
              value={r.contributive_min_seconds}
              onChange={(e) =>
                setRule({
                  ...rule,
                  contributive_min_seconds: num(e.target.value),
                })
              }
            />
            <Field
              label={t("voiceAdmin.eligibility.maxClipped")}
              hint={t("voiceAdmin.eligibility.maxClippedHint")}
              type="number"
              min={0}
              max={1}
              step="any"
              value={r.max_clipped_ratio}
              onChange={(e) =>
                setRule({ ...rule, max_clipped_ratio: num(e.target.value) })
              }
            />
            <div className="flex flex-wrap gap-2">
              <button disabled={busy} className={buttonClass("primary")}>
                {t("voiceAdmin.eligibility.save")}
              </button>
              <button
                type="button"
                disabled={busy}
                className={buttonClass()}
                onClick={() => void loadPreview()}
              >
                {t("voiceAdmin.eligibility.preview")}
              </button>
            </div>
          </form>
          {preview && (
            <EligibilityPreview
              preview={preview.figures}
              rule={preview.rule}
              busy={busy}
              onRequalify={() =>
                void run(
                  () => putRule(preview.rule),
                  () => setRule(undefined),
                )
              }
            />
          )}
          {requalified !== undefined && (
            <p role="status">
              {t("voiceAdmin.eligibility.requalified", { count: requalified })}
            </p>
          )}
          {eligibility.requalifications.length > 0 && (
            <RequalificationReport report={eligibility.requalifications[0]} />
          )}
          <FingerprintRecompute />
          {eligibility.history.length > 1 && (
            <div>
              <p id="adv-eligibility-history" className="font-medium">
                {t("voiceAdmin.eligibility.history")}
              </p>
              <ul
                aria-labelledby="adv-eligibility-history"
                className="list-disc pl-5"
              >
                {eligibility.history.slice(1).map((v) => (
                  <li key={v.version}>
                    {t("voiceAdmin.eligibility.historyItem", {
                      version: v.version,
                      comparable: v.comparable_min_seconds,
                      contributive: v.contributive_min_seconds,
                      clipped: v.max_clipped_ratio,
                      author: v.author ?? t("voiceAdmin.eligibility.snitch"),
                      date: new Date(v.created_at).toLocaleString(),
                    })}
                  </li>
                ))}
              </ul>
            </div>
          )}
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
          <VoiceCalibration
            calibration={accuracy.calibration}
            busy={busy}
            onApply={(p) =>
              void run(
                () => putThresholds(p, "proposal"),
                () => setMatch(undefined),
              )
            }
          />
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
