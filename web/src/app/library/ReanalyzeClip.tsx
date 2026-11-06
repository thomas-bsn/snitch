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
type StepName = Run["steps"][number]["step"];
type Choice = { scope: Scope; mode: Mode; steps: ForcedStep[] };
type Added =
  | { step: StepName; reason: "prerequisite" | "dependent"; of: StepName }
  | { step: StepName; reason: "missing" };

const FORCEABLE: ForcedStep[] = ["transcription", "indexing", "diarization"];

// Mirrors the server: which steps a scope runs.
function applicable(scope: Scope, step: StepName) {
  if (step === "preparation" || step === "transcription") return true;
  if (step === "indexing") return scope === "full";
  return scope === "voice" || scope === "full";
}

const VOICE_STEPS: StepName[] = ["diarization", "fingerprints", "matching"];

/**
 * Mirrors the server's closure of chosen steps (it recomputes it, never trusting this one): a
 * missing transcription is a prerequisite of the indexing and of the voice detection, the
 * indexing follows a transcription, the fingerprints and the matching follow the voice
 * detection. The scope then widens to hold them all; a scope widened to the full analysis
 * also runs the voice steps that never ran, listed as missing.
 */
export function closeSteps(
  scope: Scope,
  chosen: ForcedStep[],
  hasTranscription: boolean,
  hasVoices: boolean,
): Added[] {
  const added: Added[] = [];
  const has = (step: StepName) =>
    (chosen as StepName[]).includes(step) || added.some((a) => a.step === step);
  if (!has("transcription") && !hasTranscription) {
    const needer = (["indexing", "diarization"] as const).find(has);
    if (needer)
      added.push({ step: "transcription", reason: "prerequisite", of: needer });
  }
  if (has("transcription") && !has("indexing"))
    added.push({ step: "indexing", reason: "dependent", of: "transcription" });
  if (has("diarization"))
    added.push(
      { step: "fingerprints", reason: "dependent", of: "diarization" },
      { step: "matching", reason: "dependent", of: "fingerprints" },
    );
  // A voice scope never transcribes and only the full scope indexes.
  const transcribes = scope !== "voice" || has("transcription");
  const voices = scope !== "transcription" || has("diarization");
  const widened: Scope =
    scope === "full" || has("indexing") || (transcribes && voices)
      ? "full"
      : voices
        ? "voice"
        : "transcription";
  if (!hasVoices)
    for (const step of VOICE_STEPS)
      if (!applicable(scope, step) && applicable(widened, step) && !has(step))
        added.push({ step, reason: "missing" });
  return added;
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
  hasTranscription,
  hasVoices,
  open,
  onOpenChange,
  onStarted,
}: {
  clipId: number;
  /** Whether the clip has a valid transcription: without it, it is a prerequisite. */
  hasTranscription: boolean;
  /** Whether a voice detection was ever launched on the clip (its voices are not "none"). */
  hasVoices: boolean;
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
  // When the server read the stake it showed: only the work that existed then is replaced.
  const [stakeAt, setStakeAt] = useState<string>();
  // What the stake was read for: a confirmation replaces exactly this choice, even when it
  // came from a quick choice.
  const [pick, setPick] = useState<Choice>();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);

  // Every step can be chosen: what it needs and what follows it are added, and the scope
  // widens on the server to hold them.
  const chosen = FORCEABLE.filter((s) => steps.includes(s));
  const invalid = mode === "steps" && chosen.length === 0;
  const added =
    mode === "steps"
      ? closeSteps(scope, chosen, hasTranscription, hasVoices)
      : [];

  function close(next: boolean) {
    if (!next) {
      setStake(undefined);
      setStakeAt(undefined);
      setFailure(undefined);
    }
    onOpenChange(next);
  }

  async function submit(confirm: boolean, quick?: Choice) {
    const current: Choice = { scope, mode, steps: chosen };
    // A confirmation replays the choice whose consequences were shown; a new request uses the choice made now.
    const next: Choice = confirm ? (pick ?? current) : (quick ?? current);
    if (!confirm) setPick(next);
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/admin/clips/{id}/analysis", {
        params: { path: { id: clipId }, header: mutation(session?.csrf_token) },
        body: {
          scope: next.scope,
          mode: next.mode,
          ...(next.mode === "steps" ? { steps: next.steps } : {}),
          ...(confirm
            ? {
                confirm_replace_corrections: true,
                confirm_replace_validations: true,
                ...(stakeAt ? { confirmed_stake_at: stakeAt } : {}),
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
      const err = r.error as
        | { error?: string; stake?: Stake; stake_at?: string }
        | undefined;
      if (err?.error === "reanalysis_confirmation_required" && err.stake) {
        setStake(err.stake);
        setStakeAt(err.stake_at);
      } else setFailure(errorCode(r.error) ?? "unknown");
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
              onClick={() => void submit(true, pick)}
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
          <div
            role="group"
            aria-label={t("clip.analysis.quick")}
            className="flex flex-col gap-2"
          >
            <button
              type="button"
              disabled={busy}
              onClick={() =>
                void submit(false, { scope: "full", mode: "full", steps: [] })
              }
              className={buttonClass()}
            >
              {t("clip.analysis.launchFull")}
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() =>
                void submit(false, {
                  scope: "full",
                  mode: "missing",
                  steps: [],
                })
              }
              className={buttonClass("secondary")}
            >
              {t("clip.analysis.launchMissing")}
            </button>
          </div>
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
                    className="flex min-h-11 items-center gap-2"
                  >
                    <input
                      type="checkbox"
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
                {added.length > 0 && (
                  <ul
                    aria-label={t("clip.reanalyze.added")}
                    className="m-0 flex list-none flex-col gap-1 p-0 text-sm text-mute"
                  >
                    {added.map((a) => (
                      <li key={a.step}>
                        {a.reason === "missing"
                          ? t("clip.reanalyze.addedBecause.missing", {
                              step: t(`clip.analysis.step.${a.step}`),
                            })
                          : t(`clip.reanalyze.addedBecause.${a.reason}`, {
                              step: t(`clip.analysis.step.${a.step}`),
                              of: t(`clip.analysis.step.${a.of}`),
                            })}
                      </li>
                    ))}
                  </ul>
                )}
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
