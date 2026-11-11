import type { components } from "../../../api.gen";
import { buttonClass } from "../../components/buttons";
import { Disclosure } from "../../components/Disclosure";
import { useI18n } from "../../i18n";
import { useNumberFormats } from "../../useNumberFormats";

type Calibration = components["schemas"]["VoiceCalibration"];
type Mode = components["schemas"]["VoiceCalibrationMode"];
type Proposal = components["schemas"]["VoiceThresholdProposal"];
type ScoreMode = Mode["mode"];

/**
 * The calibration of the thresholds of the active fingerprint model (ADR 0064): both score modes side by side.
 * Scores are cosine similarities, shown as such: never as a percentage nor a probability. Thresholds belong to a
 * score mode: a proposal is applied for the mode it was measured in, and only holds while that mode is active.
 */
export function VoiceCalibration({
  calibration,
  busy,
  onApply,
}: {
  calibration: Calibration;
  busy: boolean;
  onApply: (proposal: Proposal, mode: ScoreMode) => void;
}) {
  const { t, language } = useI18n();
  const { score } = useNumberFormats();
  // A rate of simulated unknowns wrongly recognised is a share of tests, unlike a score.
  const rate = (x: number | null) =>
    x === null
      ? "–"
      : x.toLocaleString(language, {
          style: "percent",
          maximumFractionDigits: 1,
        });
  const modeLabel = (m: ScoreMode) =>
    m === "cumulative"
      ? t("voiceAdmin.score.cumulative")
      : t("voiceAdmin.score.bestSample");
  return (
    <section aria-labelledby="adv-calibration" className="flex flex-col gap-3">
      <h5 id="adv-calibration" className="font-semibold">
        {t("voiceAdmin.calibration.title")}
      </h5>
      <p className="text-mute">{t("voiceAdmin.calibration.hint")}</p>
      <p className="text-sm text-mute">
        {t("voiceAdmin.thresholds.scoreNote")}
      </p>
      {calibration.modes.length === 0 ? (
        <p>{t("voiceAdmin.calibration.noGroup")}</p>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {calibration.modes.map((m) => {
            const active = m.mode === calibration.active_mode;
            const id = `calibration-${m.mode}`;
            return (
              <article
                key={m.mode}
                aria-labelledby={id}
                className="flex min-w-0 flex-col gap-3 rounded-lg border border-line p-3"
              >
                <h6 id={id} className="font-medium">
                  {active
                    ? t("voiceAdmin.calibration.activeMode", {
                        mode: modeLabel(m.mode),
                      })
                    : modeLabel(m.mode)}
                </h6>
                <p className="text-sm">
                  {t("voiceAdmin.calibration.figures", {
                    tests: m.tests,
                    persons: m.persons,
                    same: m.same_person_scores,
                    other: m.other_person_scores,
                  })}
                </p>
                <p className="text-sm">
                  {m.thresholds
                    ? t("voiceAdmin.calibration.applied", {
                        suggestion: score(m.thresholds.suggestion_threshold),
                        automatic: score(m.thresholds.automatic_threshold),
                        margin: score(m.thresholds.ambiguity_margin),
                      })
                    : t("voiceAdmin.calibration.noneApplied")}
                </p>
                <p className="text-sm">
                  {m.current
                    ? t("voiceAdmin.calibration.current", {
                        automatic: m.current.automatic,
                        errors: m.current.errors,
                        toVerify: m.current.to_verify,
                        unknown: m.current.unknown_automatic,
                        tests: m.tests,
                        rate: rate(m.current.unknown_automatic_rate),
                      })
                    : t("voiceAdmin.calibration.noCurrent")}
                </p>
                {m.proposal ? (
                  <div className="flex flex-col gap-2">
                    <p>
                      {t("voiceAdmin.calibration.proposal", {
                        suggestion: score(m.proposal.suggestion_threshold),
                        automatic: score(m.proposal.automatic_threshold),
                        margin: score(m.proposal.ambiguity_margin),
                        automaticCount: m.proposal.figures.automatic,
                        errors: m.proposal.figures.errors,
                        toVerify: m.proposal.figures.to_verify,
                        unknown: m.proposal.figures.unknown_automatic,
                        tests: m.tests,
                        rate: rate(m.proposal.figures.unknown_automatic_rate),
                      })}
                    </p>
                    <p className="text-sm text-mute">
                      {t("voiceAdmin.calibration.rule")}
                    </p>
                    {!active && (
                      <p className="text-sm text-mute">
                        {t("voiceAdmin.calibration.otherMode")}
                      </p>
                    )}
                    <div>
                      <button
                        type="button"
                        disabled={busy}
                        className={buttonClass(
                          active ? "primary" : "secondary",
                        )}
                        onClick={() =>
                          m.proposal && onApply(m.proposal, m.mode)
                        }
                      >
                        {active
                          ? t("voiceAdmin.calibration.apply")
                          : t("voiceAdmin.calibration.applyInactive", {
                              mode: modeLabel(m.mode),
                            })}
                      </button>
                    </div>
                  </div>
                ) : (
                  <p>
                    {m.withheld === "no_safe_threshold"
                      ? t("voiceAdmin.calibration.noSafe")
                      : t("voiceAdmin.calibration.tooFew", {
                          tests: m.tests,
                          persons: m.persons,
                          minTests: calibration.min_tests,
                          minPersons: calibration.min_persons,
                        })}
                  </p>
                )}
                <Disclosure
                  id={`${id}-distribution`}
                  label={t("voiceAdmin.calibration.distribution")}
                >
                  <div
                    role="region"
                    tabIndex={0}
                    aria-label={`${t("voiceAdmin.calibration.distribution")} · ${modeLabel(m.mode)}`}
                    className="overflow-x-auto"
                  >
                    <table className="w-full text-left text-sm tabular-nums">
                      <thead>
                        <tr>
                          <th scope="col" className="p-1">
                            {t("voiceAdmin.calibration.scoreColumn")}
                          </th>
                          <th scope="col" className="p-1">
                            {t("voiceAdmin.calibration.sameColumn")}
                          </th>
                          <th scope="col" className="p-1">
                            {t("voiceAdmin.calibration.otherColumn")}
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {m.distribution
                          .filter((b) => b.same_person + b.other_person > 0)
                          .map((b) => (
                            <tr key={b.from} className="border-t border-line">
                              <td className="p-1">
                                {score(b.from)} – {score(b.to)}
                              </td>
                              <td className="p-1">{b.same_person}</td>
                              <td className="p-1">{b.other_person}</td>
                            </tr>
                          ))}
                      </tbody>
                    </table>
                  </div>
                </Disclosure>
                <Disclosure
                  id={`${id}-sweep`}
                  label={t("voiceAdmin.calibration.sweep")}
                >
                  <div
                    role="region"
                    tabIndex={0}
                    aria-label={`${t("voiceAdmin.calibration.sweep")} · ${modeLabel(m.mode)}`}
                    className="overflow-x-auto"
                  >
                    <table className="w-full min-w-[28rem] text-left text-sm tabular-nums">
                      <thead>
                        <tr>
                          {(
                            [
                              "thresholdColumn",
                              "automaticColumn",
                              "errorsColumn",
                              "toVerifyColumn",
                              "unknownColumn",
                            ] as const
                          ).map((k) => (
                            <th scope="col" key={k} className="p-1">
                              {t(`voiceAdmin.calibration.${k}`)}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {m.sweep.map((row) => (
                          <tr
                            key={row.automatic_threshold}
                            className="border-t border-line"
                          >
                            <td className="p-1">
                              {score(row.automatic_threshold)}
                            </td>
                            <td className="p-1">{row.automatic}</td>
                            <td className="p-1">{row.errors}</td>
                            <td className="p-1">{row.to_verify}</td>
                            <td className="p-1">
                              {row.unknown_automatic} (
                              {rate(row.unknown_automatic_rate)})
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </Disclosure>
              </article>
            );
          })}
        </div>
      )}
    </section>
  );
}
