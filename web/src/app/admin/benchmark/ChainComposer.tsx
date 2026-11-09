import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import type { Model } from "./api";

export const MIN_CHAINS = 2;
export const MAX_CHAINS = 4;

/** A chain being composed: the ids of its two models, empty until picked. `key` only keeps the rows apart. */
export type ChainDraft = {
  key: number;
  diarization: string;
  transcription: string;
};

export const emptyChains = (): ChainDraft[] =>
  Array.from({ length: MIN_CHAINS }, (_, key) => ({
    key,
    diarization: "",
    transcription: "",
  }));

/** The chains that can run: both models picked and the diarization model still ready on a Worker. */
export const runnableChains = (chains: ChainDraft[], diarizers: Model[]) =>
  chains.filter(
    (c) =>
      c.transcription !== "" &&
      diarizers.some((m) => m.id === c.diarization && m.ready_workers > 0),
  );

/** The index of the earlier chain a chain repeats, if any: a pair is measured once. */
export const duplicateOf = (chains: ChainDraft[], index: number) => {
  const c = chains[index];
  if (c.diarization === "" || c.transcription === "") return -1;
  return chains.findIndex(
    (o) =>
      o.diarization === c.diarization && o.transcription === c.transcription,
  );
};

const select =
  "h-11 min-w-0 rounded-lg border border-line-strong bg-surface px-2 md:h-9";

/**
 * The composer of the end-to-end stage: two to four chains, each a diarization model of the catalogue and a
 * transcription model (the production one, or a name typed by the administrator).
 */
export function ChainComposer({
  chains,
  onChange,
  diarizers,
  transcribers,
  nameOk,
  onName,
}: {
  chains: ChainDraft[];
  onChange: (next: ChainDraft[]) => void;
  diarizers: Model[];
  transcribers: Model[];
  nameOk: (name: string) => boolean;
  onName: (name: string) => void;
}) {
  const { t } = useI18n();
  const [draft, setDraft] = useState("");
  const update = (key: number, patch: Partial<ChainDraft>) =>
    onChange(chains.map((c) => (c.key === key ? { ...c, ...patch } : c)));
  const add = () =>
    onChange([
      ...chains,
      {
        key: Math.max(-1, ...chains.map((c) => c.key)) + 1,
        diarization: "",
        transcription: "",
      },
    ]);
  const submitName = () => {
    const name = draft.trim();
    if (!nameOk(name)) return;
    onName(name);
    setDraft("");
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col gap-3">
        {chains.map((c, i) => {
          const twin = duplicateOf(chains, i);
          const duplicate = twin !== -1 && twin !== i;
          const number = i + 1;
          return (
            <div
              key={c.key}
              role="group"
              aria-label={t("benchmark.run.chainLabel", { n: number })}
              className="flex flex-col gap-2 rounded-xl border border-line-strong p-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                <strong>{t("benchmark.run.chainLabel", { n: number })}</strong>
                <span className="flex-1" />
                <button
                  type="button"
                  className={buttonClass()}
                  disabled={chains.length <= MIN_CHAINS}
                  aria-label={t("benchmark.run.chainRemove", { n: number })}
                  onClick={() => onChange(chains.filter((o) => o !== c))}
                >
                  <Trash2 className="size-4" aria-hidden="true" />
                  {t("benchmark.run.chainRemoveShort")}
                </button>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="flex flex-col gap-1">
                  <span className="text-sm font-medium">
                    {t("benchmark.run.chainDiarization")}
                    <span className="sr-only"> · {number}</span>
                  </span>
                  <select
                    className={select}
                    value={c.diarization}
                    aria-invalid={duplicate}
                    onChange={(e) =>
                      update(c.key, { diarization: e.target.value })
                    }
                  >
                    <option value="">{t("benchmark.run.chainPick")}</option>
                    {diarizers.map((m) => (
                      <option
                        key={m.id}
                        value={m.id}
                        disabled={m.ready_workers <= 0}
                      >
                        {m.label}
                        {m.ready_workers <= 0
                          ? ` · ${t("benchmark.run.modelNowhereShort")}`
                          : ""}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="flex flex-col gap-1">
                  <span className="text-sm font-medium">
                    {t("benchmark.run.chainTranscription")}
                    <span className="sr-only"> · {number}</span>
                  </span>
                  <select
                    className={select}
                    value={c.transcription}
                    aria-invalid={duplicate}
                    onChange={(e) =>
                      update(c.key, { transcription: e.target.value })
                    }
                  >
                    <option value="">{t("benchmark.run.chainPick")}</option>
                    {transcribers.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.label}
                        {m.active
                          ? ` · ${t("benchmark.run.chainProduction")}`
                          : ""}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
              {duplicate && (
                <p role="alert" className="text-sm text-bad-fg">
                  {t("benchmark.run.chainDuplicate", { n: twin + 1 })}
                </p>
              )}
            </div>
          );
        })}
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          className={buttonClass()}
          disabled={chains.length >= MAX_CHAINS}
          onClick={add}
        >
          <Plus className="size-4" aria-hidden="true" />
          {t("benchmark.run.chainAdd")}
        </button>
        <span className="text-sm text-mute">
          {t("benchmark.run.chainsRange", { min: MIN_CHAINS, max: MAX_CHAINS })}
        </span>
      </div>
      <form
        className="flex flex-col gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          submitName();
        }}
      >
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">
            {t("benchmark.run.addModel")}
          </span>
          <span className="text-xs text-mute">
            {t("benchmark.run.addModelHelp")}
          </span>
          <span className="flex flex-wrap gap-2">
            <input
              value={draft}
              maxLength={64}
              onChange={(event) => setDraft(event.target.value)}
              placeholder="Systran/faster-whisper-large-v3"
              className="h-11 min-w-0 flex-1 rounded-lg border border-line-strong bg-surface px-3 md:h-9"
            />
            <button
              type="submit"
              className={buttonClass()}
              disabled={!nameOk(draft.trim())}
            >
              {t("benchmark.run.addModelButton")}
            </button>
          </span>
        </label>
      </form>
    </div>
  );
}
