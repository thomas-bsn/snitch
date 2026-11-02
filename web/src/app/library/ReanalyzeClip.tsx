import { useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { useSession } from "../session";

type Run = components["schemas"]["ClipAnalysisRun"];
type Stake = components["schemas"]["ClipReanalysisStake"];
type Scope = Run["scope"];
type Mode = Run["mode"];
type ForcedStep = Run["forced_steps"][number];

const FORCEABLE: ForcedStep[] = ["transcription", "indexing", "diarization"];

// Mirrors the server: which steps a scope can run again.
function applicable(scope: Scope, step: ForcedStep) {
  if (step === "transcription") return true;
  if (step === "indexing") return scope === "full";
  return scope === "voice" || scope === "full";
}

/**
 * Reanalysis of a clip that was already processed: the administrator picks the scope, then
 * what runs again (the missing steps only, everything, or chosen steps). When the choice
 * would make human work obsolete, the server answers with the consequences; they are shown
 * here, and nothing starts before the explicit confirmation. Nothing is deleted: histories,
 * validated samples and previous results stay.
 */
export function ReanalyzeClip({
  clipId,
  open,
  onOpenChange,
  onStarted,
}: {
  clipId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onStarted: (run: Run | null) => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const [scope, setScope] = useState<Scope>("full");
  const [mode, setMode] = useState<Mode>("missing");
  const [steps, setSteps] = useState<ForcedStep[]>([]);
  const [stake, setStake] = useState<Stake>();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);

  const chosen = steps.filter((s) => applicable(scope, s));
  const invalid = mode === "steps" && chosen.length === 0;

  function close(next: boolean) {
    if (!next) {
      setStake(undefined);
      setFailure(undefined);
    }
    onOpenChange(next);
  }

  async function submit(confirm: boolean) {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/admin/clips/{id}/analysis", {
        params: { path: { id: clipId }, header: mutation(session?.csrf_token) },
        body: {
          scope,
          mode,
          ...(mode === "steps" ? { steps: chosen } : {}),
          ...(confirm
            ? {
                confirm_replace_corrections: true,
                confirm_replace_validations: true,
              }
            : {}),
        },
      });
      if (r.data) {
        setStake(undefined);
        onStarted(r.data.run ?? null);
        close(false);
        return;
      }
      const err = r.error as { error?: string; stake?: Stake } | undefined;
      if (err?.error === "reanalysis_confirmation_required" && err.stake)
        setStake(err.stake);
      else setFailure(errorCode(r.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  const forcesTranscription =
    mode === "full" || (mode === "steps" && chosen.includes("transcription"));
  const forcesDiarization =
    (mode === "full" && applicable(scope, "diarization")) ||
    (mode === "steps" && chosen.includes("diarization"));

  return (
    <Modal
      open={open}
      onOpenChange={close}
      title={t("clip.reanalyze.title")}
      closeLabel={t("clip.reanalyze.close")}
    >
      {stake ? (
        <div className="flex flex-col gap-3" role="group">
          <p className="m-0 text-sm font-medium">
            {t("clip.reanalyze.confirmIntro")}
          </p>
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
          <p className="m-0 text-sm text-mute">{t("clip.reanalyze.kept")}</p>
          {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => setStake(undefined)}
              className={buttonClass("secondary")}
            >
              {t("clip.reanalyze.back")}
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => void submit(true)}
              className={buttonClass("danger")}
            >
              {t("clip.reanalyze.confirm")}
            </button>
          </div>
        </div>
      ) : (
        <form
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            void submit(false);
          }}
        >
          <fieldset className="m-0 flex flex-col gap-1 border-0 p-0">
            <legend className="mb-1 text-sm font-medium">
              {t("clip.reanalyze.scopeLabel")}
            </legend>
            {(["full", "transcription", "voice"] as Scope[]).map((value) => (
              <label key={value} className="flex min-h-11 items-center gap-2">
                <input
                  type="radio"
                  name="reanalyze-scope"
                  checked={scope === value}
                  onChange={() => setScope(value)}
                />
                {t(`clip.reanalyze.scope.${value}`)}
              </label>
            ))}
          </fieldset>
          <fieldset className="m-0 flex flex-col gap-1 border-0 p-0">
            <legend className="mb-1 text-sm font-medium">
              {t("clip.reanalyze.what")}
            </legend>
            {(["missing", "full", "steps"] as Mode[]).map((value) => (
              <label key={value} className="flex min-h-11 items-center gap-2">
                <input
                  type="radio"
                  name="reanalyze-mode"
                  checked={mode === value}
                  onChange={() => setMode(value)}
                />
                {t(`clip.reanalyze.mode.${value}`)}
              </label>
            ))}
            {mode === "steps" && (
              <div className="ml-6 flex flex-col gap-1">
                {FORCEABLE.map((step) => (
                  <label
                    key={step}
                    className="flex min-h-11 items-center gap-2 aria-disabled:text-mute"
                    aria-disabled={!applicable(scope, step)}
                  >
                    <input
                      type="checkbox"
                      disabled={!applicable(scope, step)}
                      checked={chosen.includes(step)}
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
          <div className="flex flex-col gap-1 text-sm text-mute" role="note">
            {mode === "missing" && (
              <p className="m-0">{t("clip.reanalyze.noteMissing")}</p>
            )}
            {forcesTranscription && (
              <p className="m-0">{t("clip.reanalyze.noteTranscription")}</p>
            )}
            {forcesDiarization && (
              <p className="m-0">{t("clip.reanalyze.noteDiarization")}</p>
            )}
          </div>
          {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => close(false)}
              className={buttonClass("secondary")}
            >
              {t("clip.reanalyze.cancel")}
            </button>
            <button
              type="submit"
              disabled={busy || invalid}
              className={buttonClass()}
            >
              {t("clip.reanalyze.start")}
            </button>
          </div>
        </form>
      )}
    </Modal>
  );
}
