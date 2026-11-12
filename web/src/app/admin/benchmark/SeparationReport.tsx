import { useI18n } from "../../i18n";
import { useNumberFormats } from "../../useNumberFormats";
import type { RunResult } from "./api";
import { percent } from "./format";

const card = "flex flex-col gap-3 rounded-xl border border-line bg-surface p-4";
const cell = "px-2 py-2 text-right tabular-nums";

/**
 * What separating the Chevauchements of an end-to-end chain changes (#403): the same clips without and with the words
 * it adds, what it recovers and what it costs in added errors. Too few clips: it says so instead of concluding.
 */
export function SeparationReport({
  results,
  label,
}: {
  results: RunResult[];
  label: (x: RunResult) => string;
}) {
  const { t, language } = useI18n();
  const separated = results.filter((x) => x.metrics.separation);
  if (separated.length === 0) return null;
  return (
    <section aria-labelledby="voice-separation-title" className={card}>
      <h3 id="voice-separation-title" className="text-base font-semibold">
        {t("benchmark.results.voiceSeparation.title")}
      </h3>
      <p className="text-sm text-mute">
        {t("benchmark.results.voiceSeparation.lead")}
      </p>
      {separated.map((x) => {
        const s = x.metrics.separation!;
        const without = s.without_separation;
        const withSep = s.with_separation;
        return (
          <div key={x.model_id} className="flex flex-col gap-2">
            <h4 className="text-sm font-semibold">{label(x)}</h4>
            {without.clips < s.min_clips ? (
              <p
                role="note"
                className="rounded-lg bg-warn-bg px-3 py-2.5 text-warn-fg"
              >
                {t("benchmark.results.voiceSeparation.tooFew", {
                  clips: without.clips,
                  min: s.min_clips,
                })}
              </p>
            ) : (
              <p className="text-sm">
                {t("benchmark.results.voiceSeparation.verdict", {
                  recovered:
                    s.speakers_without_words_before -
                    s.speakers_without_words_after,
                  silent: s.speakers_without_words_before,
                  before: percent(language, without.cpwer),
                  after: percent(language, withSep.cpwer),
                  added: s.words_added,
                  absent: s.words_added_absent_from_reference,
                })}
              </p>
            )}
            <div className="overflow-x-auto">
              <table className="w-full border-collapse text-sm">
                <thead>
                  <tr className="text-xs text-mute uppercase">
                    <th scope="col" className="px-2 py-2 text-left">
                      {t("benchmark.results.voiceSeparation.measure")}
                    </th>
                    <th scope="col" className="px-2 py-2 text-right">
                      {t("benchmark.results.voiceSeparation.without")}
                    </th>
                    <th scope="col" className="px-2 py-2 text-right">
                      {t("benchmark.results.voiceSeparation.with")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-t border-line">
                    <th scope="row" className="px-2 py-2 text-left">
                      cpWER
                    </th>
                    <td className={cell}>{percent(language, without.cpwer)}</td>
                    <td className={cell}>{percent(language, withSep.cpwer)}</td>
                  </tr>
                  <tr className="border-t border-line">
                    <th scope="row" className="px-2 py-2 text-left">
                      WER
                    </th>
                    <td className={cell}>{percent(language, without.wer)}</td>
                    <td className={cell}>{percent(language, withSep.wer)}</td>
                  </tr>
                  <tr className="border-t border-line">
                    <th scope="row" className="px-2 py-2 text-left">
                      {t("benchmark.results.voiceSeparation.silent")}
                    </th>
                    <td className={cell}>{s.speakers_without_words_before}</td>
                    <td className={cell}>{s.speakers_without_words_after}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p className="text-sm text-mute">
              {t("benchmark.results.voiceSeparation.counts", {
                clips: without.clips,
                zones: s.zones,
                added: s.words_added,
                absent: s.words_added_absent_from_reference,
                duplicated: s.words_duplicated_between_sources,
                unattributed: s.sources_without_speaker,
                sources: s.sources,
              })}
            </p>
          </div>
        );
      })}
    </section>
  );
}

/** The cost a Worker reported per model of the run: a value it did not report reads as such, never as zero. */
export function ModelCosts({
  results,
  label,
}: {
  results: RunResult[];
  label: (id: string) => string;
}) {
  const { t, language } = useI18n();
  const { seconds } = useNumberFormats();
  const costs = [
    ...new Map(
      results.flatMap((x) => x.resources ?? []).map((c) => [c.model_id, c]),
    ).values(),
  ];
  if (costs.length === 0) return null;
  const megabytes = new Intl.NumberFormat(language, {
    style: "unit",
    unit: "megabyte",
    maximumFractionDigits: 0,
  });
  const absent = t("benchmark.results.costs.absent");
  return (
    <section aria-labelledby="costs-title" className={card}>
      <h3 id="costs-title" className="text-base font-semibold">
        {t("benchmark.results.costs.title")}
      </h3>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="text-xs text-mute uppercase">
              <th scope="col" className="px-2 py-2 text-left">
                {t("benchmark.results.model")}
              </th>
              <th scope="col" className="px-2 py-2 text-right">
                {t("benchmark.results.costs.tasks")}
              </th>
              <th scope="col" className="px-2 py-2 text-right">
                {t("benchmark.results.costs.inference")}
              </th>
              <th scope="col" className="px-2 py-2 text-right">
                {t("benchmark.results.costs.gpu")}
              </th>
            </tr>
          </thead>
          <tbody>
            {costs.map((c) => (
              <tr key={c.model_id} className="border-t border-line">
                <th scope="row" className="px-2 py-2 text-left">
                  {label(c.model_id)}
                </th>
                <td className={cell}>{c.tasks}</td>
                <td className={cell}>
                  {c.inference_seconds == null
                    ? absent
                    : seconds(c.inference_seconds)}
                </td>
                <td className={cell}>
                  {c.gpu_peak_bytes == null
                    ? absent
                    : megabytes.format(c.gpu_peak_bytes / 1e6)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="text-xs text-mute">{t("benchmark.results.costs.note")}</p>
    </section>
  );
}
