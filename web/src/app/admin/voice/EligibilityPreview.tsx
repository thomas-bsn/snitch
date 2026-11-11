import type { components } from "../../../api.gen";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";

type Preview = components["schemas"]["VoiceEligibilityPreview"];
type Group = components["schemas"]["VoiceEligibilityGroupPreview"];
type Outcome = components["schemas"]["VoiceEligibilityOutcome"];
type Scenario = components["schemas"]["VoiceEligibilityScenario"];
type Requalification = components["schemas"]["VoiceEligibilityRequalification"];
type Rule = components["schemas"]["VoiceEligibilityInput"];

const COLUMNS = [
  "ruleColumn",
  "comparableColumn",
  "contributiveColumn",
  "rejectedColumn",
  "referenceColumn",
] as const;

/**
 * The figures of the eligibility preview (#394): what a requalification under the examined rule, or under each
 * minimum duration, would decide for the fingerprints already stored. Nothing is written until the
 * administrator requalifies.
 */
export function EligibilityPreview({
  preview,
  rule,
  busy,
  onRequalify,
}: {
  preview: Preview;
  rule: Rule;
  busy: boolean;
  onRequalify: () => void;
}) {
  const { t } = useI18n();
  const seconds = useSeconds();
  return (
    <div className="flex flex-col gap-4 rounded-lg border border-line p-3">
      <p id="adv-eligibility-preview" className="font-medium">
        {t("voiceAdmin.eligibility.previewTitle", {
          comparable: seconds(rule.comparable_min_seconds),
          contributive: seconds(rule.contributive_min_seconds),
        })}
      </p>
      {preview.groups.length === 0 && (
        <p className="text-mute">{t("voiceAdmin.eligibility.previewEmpty")}</p>
      )}
      {preview.groups.map((group) => (
        <GroupPreview
          key={`${group.model}|${group.model_version}|${group.dimensions}`}
          group={group}
          personsWithSamples={preview.persons_with_samples}
        />
      ))}
      <p className="text-mute">{t("voiceAdmin.eligibility.requalifyHint")}</p>
      <div>
        <button
          type="button"
          disabled={busy}
          className={buttonClass("primary")}
          onClick={onRequalify}
        >
          {t("voiceAdmin.eligibility.requalify")}
        </button>
      </div>
    </div>
  );
}

function useSeconds() {
  const { language } = useI18n();
  const format = new Intl.NumberFormat(language, { maximumFractionDigits: 2 });
  return (value: number) => `${format.format(value)} s`;
}

function GroupPreview({
  group,
  personsWithSamples,
}: {
  group: Group;
  personsWithSamples: number;
}) {
  const { t } = useI18n();
  const seconds = useSeconds();
  const name = `${group.model} ${group.model_version} (${group.dimensions})`;
  const delta = (gained: number, lost: number) =>
    [gained > 0 ? ` +${gained}` : "", lost > 0 ? ` −${lost}` : ""].join("");
  const row = (label: string, o: Outcome, s?: Scenario) => (
    <tr key={label} className="border-t border-line">
      <th scope="row" className="p-2 font-normal">
        {label}
      </th>
      <td className="p-2">
        {o.comparable}
        {s && delta(s.recovered_comparable, s.lost_comparable)}
      </td>
      <td className="p-2">
        {o.contributive}
        {s && delta(s.recovered_contributive, s.lost_contributive)}
      </td>
      <td className="p-2">
        {t("voiceAdmin.eligibility.rejected", {
          tooShort: o.too_short,
          overlap: o.overlap_only,
          clipped: o.clipped + o.low_quality,
        })}
      </td>
      <td className="p-2">{o.persons_with_reference}</td>
    </tr>
  );
  return (
    <section
      aria-label={t("voiceAdmin.eligibility.groupLabel", { name })}
      className="flex flex-col gap-3"
    >
      <p className="font-medium">{name}</p>
      <p>
        {t("voiceAdmin.eligibility.previewFacts", {
          present: group.current.present,
          missing: group.missing_sources,
          estimated: group.estimated,
          original: group.original_vectors,
        })}{" "}
        {group.overlap_share === null
          ? t("voiceAdmin.eligibility.overlapUnknown")
          : t("voiceAdmin.eligibility.overlapShare", {
              value: `${Math.round(group.overlap_share * 1000) / 10} %`,
            })}
      </p>
      <p>
        {t("voiceAdmin.eligibility.personsReference", {
          count: group.candidate.persons_with_reference,
          total: personsWithSamples,
        })}
      </p>
      <div
        role="region"
        tabIndex={0}
        aria-label={t("voiceAdmin.eligibility.scenarios")}
        className="overflow-x-auto"
      >
        <table className="w-full min-w-[36rem] text-left">
          <caption className="text-left font-medium">
            {t("voiceAdmin.eligibility.scenarios")}
          </caption>
          <thead>
            <tr>
              {COLUMNS.map((k) => (
                <th className="p-2" scope="col" key={k}>
                  {t(`voiceAdmin.eligibility.${k}`)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {row(t("voiceAdmin.eligibility.current"), group.current)}
            {row(
              t("voiceAdmin.eligibility.examined", {
                comparable: seconds(
                  group.candidate.rule.comparable_min_seconds,
                ),
                contributive: seconds(
                  group.candidate.rule.contributive_min_seconds,
                ),
              }),
              group.candidate,
              group.candidate,
            )}
            {group.scenarios.map((s) =>
              row(
                t("voiceAdmin.eligibility.duration", {
                  value: seconds(s.rule.comparable_min_seconds),
                }),
                s,
                s,
              ),
            )}
          </tbody>
        </table>
      </div>
      <p className="text-mute">{t("voiceAdmin.eligibility.scenariosHint")}</p>
      {group.persons.length > 0 && (
        <div
          role="region"
          tabIndex={0}
          aria-label={t("voiceAdmin.eligibility.perPerson")}
          className="overflow-x-auto"
        >
          <table className="w-full min-w-[36rem] text-left">
            <caption className="text-left font-medium">
              {t("voiceAdmin.eligibility.perPerson")}
            </caption>
            <thead>
              <tr>
                {(
                  [
                    "personColumn",
                    "presentColumn",
                    "comparableColumn",
                    "contributiveColumn",
                    "rejectedColumn",
                    "missingColumn",
                    "referenceColumn",
                  ] as const
                ).map((k) => (
                  <th className="p-2" scope="col" key={k}>
                    {t(`voiceAdmin.eligibility.${k}`)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {group.persons.map((p) => (
                <tr key={p.person.id} className="border-t border-line">
                  <th scope="row" className="p-2 font-normal">
                    {p.person.display_name}
                  </th>
                  <td className="p-2">{p.present}</td>
                  <td className="p-2">{p.comparable}</td>
                  <td className="p-2">{p.contributive}</td>
                  <td className="p-2">
                    {t("voiceAdmin.eligibility.rejected", {
                      tooShort: p.too_short,
                      overlap: p.overlap_only,
                      clipped: p.clipped,
                    })}
                  </td>
                  <td className="p-2">{p.missing_sources}</td>
                  <td className="p-2">
                    {t(
                      p.reference
                        ? p.reference_now
                          ? "voiceAdmin.eligibility.referenceKept"
                          : "voiceAdmin.eligibility.referenceGained"
                        : p.reference_now
                          ? "voiceAdmin.eligibility.referenceLost"
                          : "voiceAdmin.eligibility.referenceNone",
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

/** The report of a requalification: who applied which rule, when, and what changed. */
export function RequalificationReport({ report }: { report: Requalification }) {
  const { t } = useI18n();
  const outcome = (key: "before" | "after") => {
    const o = report[key];
    return t("voiceAdmin.eligibility.reportOutcome", {
      comparable: o.comparable,
      contributive: o.contributive,
      tooShort: o.too_short,
      overlap: o.overlap_only,
      clipped: o.clipped,
      lowQuality: o.low_quality,
      persons: o.persons_with_reference,
    });
  };
  return (
    <div className="flex flex-col gap-1">
      <p className="font-medium">
        {t("voiceAdmin.eligibility.reportTitle", {
          version: report.rule_version,
          author: report.author ?? t("voiceAdmin.eligibility.snitch"),
          date: new Date(report.created_at).toLocaleString(),
          count: report.changed,
        })}
      </p>
      <ul className="list-disc pl-5">
        <li>
          {t("voiceAdmin.eligibility.reportBefore")} {outcome("before")}
        </li>
        <li>
          {t("voiceAdmin.eligibility.reportAfter")} {outcome("after")}
        </li>
        <li>
          {t("voiceAdmin.eligibility.reportFacts", {
            missing: report.missing_sources,
            estimated: report.estimated,
            original: report.original_vectors,
          })}
        </li>
      </ul>
    </div>
  );
}
