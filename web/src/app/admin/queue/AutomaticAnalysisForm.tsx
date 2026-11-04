import { useState, type FormEvent } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

type Run = components["schemas"]["ClipAnalysisRun"];
type Stake = components["schemas"]["ClipReanalysisStake"];
type Mode = Run["mode"];
type ForcedStep = Run["forced_steps"][number];

const MODES: Mode[] = ["missing", "full", "steps"];
const FORCEABLE: ForcedStep[] = ["transcription", "indexing", "diarization"];

/**
 * Automatic analysis of one clip, the default way to schedule work: the whole analysis (scope
 * full) with the reanalysis modes of the clip page. When the choice would make human work
 * obsolete the server answers with the consequences, they are shown, and nothing starts before the
 * explicit confirmation. The single task of the advanced form stays apart.
 */
export function AutomaticAnalysisForm({
  onStarted,
}: {
  onStarted: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const [clip, setClip] = useState("");
  const [mode, setMode] = useState<Mode>("missing");
  const [steps, setSteps] = useState<ForcedStep[]>([]);
  const [stake, setStake] = useState<Stake>();
  // When the server read the stake it showed: only the work that existed then is replaced.
  const [stakeAt, setStakeAt] = useState<string>();
  const [failure, setFailure] = useState<string>();
  const [started, setStarted] = useState(false);
  const [busy, setBusy] = useState(false);

  const missingStep = mode === "steps" && steps.length === 0;

  async function submit(confirm: boolean) {
    setBusy(true);
    setFailure(undefined);
    setStarted(false);
    try {
      const response = await api.POST("/api/admin/clips/{id}/analysis", {
        params: {
          path: { id: Number(clip) },
          header: mutation(session?.csrf_token),
        },
        body: {
          scope: "full",
          mode,
          ...(mode === "steps" ? { steps } : {}),
          ...(confirm
            ? {
                confirm_replace_corrections: true,
                confirm_replace_validations: true,
                ...(stakeAt ? { confirmed_stake_at: stakeAt } : {}),
              }
            : {}),
        },
      });
      if (response.data) {
        setStake(undefined);
        setStarted(true);
        onStarted();
        return;
      }
      const refusal = response.error as
        | { error?: string; stake?: Stake; stake_at?: string }
        | undefined;
      if (
        refusal?.error === "reanalysis_confirmation_required" &&
        refusal.stake
      ) {
        setStake(refusal.stake);
        setStakeAt(refusal.stake_at);
      } else setFailure(errorCode(response.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  function onSubmit(event: FormEvent) {
    event.preventDefault();
    if (!missingStep) void submit(false);
  }

  return (
    <div data-automatic-schedule className="flex flex-col gap-3">
      <ConfirmDialog
        open={!!stake}
        onOpenChange={(open) => {
          if (!open) setStake(undefined);
        }}
        title={t("clip.reanalyze.confirmIntro")}
        cancelLabel={t("clip.reanalyze.back")}
        confirmLabel={t("clip.reanalyze.confirm")}
        busy={busy}
        onConfirm={() => void submit(true)}
      >
        {stake && (
          <>
            <ul className="m-0 flex list-disc flex-col gap-2 pl-5 text-sm">
              {stake.corrections && (
                <li>
                  {t("clip.reanalyze.stakeCorrections", {
                    text: stake.corrections.text_segments,
                    speaker: stake.corrections.speaker_segments,
                  })}
                </li>
              )}
              {stake.voice && (
                <li>
                  {t("clip.reanalyze.stakeVoice", {
                    speakers: stake.voice.validated_speakers,
                    samples: stake.voice.voice_samples,
                    corrections: stake.voice.speaker_corrections,
                  })}
                </li>
              )}
            </ul>
            <p className="m-0 mt-2 text-sm text-mute">
              {t("clip.reanalyze.kept")}
            </p>
          </>
        )}
      </ConfirmDialog>
      <div>
        <h2 className="m-0 text-base font-semibold">
          {t("queue.automatic.title")}
        </h2>
        <p className="m-0 mt-1 text-sm text-mute">
          {t("queue.automatic.hint")}
        </p>
      </div>
      <form onSubmit={onSubmit} className="flex flex-col gap-3">
        <label className="flex flex-col gap-1 text-sm font-medium">
          {t("queue.automatic.clip")}
          <input
            type="number"
            min={1}
            required
            value={clip}
            onChange={(e) => setClip(e.target.value)}
            className="h-11 w-40 rounded-lg border border-line-strong bg-surface px-3 text-sm text-ink md:h-9"
          />
        </label>
        <fieldset className="m-0 flex flex-col gap-1 border-0 p-0">
          <legend className="mb-1 text-sm font-medium">
            {t("clip.reanalyze.what")}
          </legend>
          {MODES.map((value) => (
            <label key={value} className="flex min-h-11 items-center gap-2">
              <input
                type="radio"
                name="automatic-mode"
                checked={mode === value}
                onChange={() => setMode(value)}
              />
              {t(`clip.reanalyze.mode.${value}`)}
            </label>
          ))}
          {mode === "steps" && (
            <div className="ml-6 flex flex-col gap-1">
              {FORCEABLE.map((step) => (
                <label key={step} className="flex min-h-11 items-center gap-2">
                  <input
                    type="checkbox"
                    checked={steps.includes(step)}
                    onChange={(e) =>
                      setSteps(
                        e.target.checked
                          ? [...steps, step]
                          : steps.filter((s) => s !== step),
                      )
                    }
                  />
                  {t(`clip.analysis.step.${step}`)}
                </label>
              ))}
            </div>
          )}
        </fieldset>
        {missingStep && (
          <p role="note" className="m-0 text-sm text-warn-fg">
            {t("queue.automatic.stepRequired")}
          </p>
        )}
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {started && <StatusNotice>{t("queue.automatic.started")}</StatusNotice>}
        <div>
          <button
            disabled={busy || missingStep}
            className={buttonClass("primary")}
          >
            {t("queue.automatic.start")}
          </button>
        </div>
      </form>
    </div>
  );
}
