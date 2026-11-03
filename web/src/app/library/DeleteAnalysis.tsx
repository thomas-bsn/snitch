import { useEffect, useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { useSession } from "../session";

type Run = components["schemas"]["ClipAnalysisRun"];
type Stake = components["schemas"]["ClipAnalysisDeletionStake"];
type Choice = "transcription" | "diarization" | "fingerprints" | "matching";

const CHOICES: Choice[] = [
  "transcription",
  "diarization",
  "fingerprints",
  "matching",
];

/**
 * Super-administrator only: deletes some steps of an analysis (or all of it). The losses are
 * computed before anything is deleted and listed; the deletion starts on the explicit button.
 */
export function DeleteAnalysis({
  clipId,
  open,
  onOpenChange,
  onDeleted,
}: {
  clipId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted: (run: Run | null) => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const [all, setAll] = useState(false);
  const [picked, setPicked] = useState<Choice[]>([]);
  const [stake, setStake] = useState<Stake>();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);

  // The steps in the server's order, so that the preview and the deletion read the same list.
  const steps = all ? CHOICES : CHOICES.filter((c) => picked.includes(c));
  const choice = steps.join(",");

  useEffect(() => {
    if (!open || steps.length === 0) return;
    let current = true;
    setStake(undefined);
    void api
      .GET("/api/admin/clips/{id}/analysis/deletion-preview", {
        params: {
          path: { id: clipId },
          query: all ? { all: true } : { steps: choice },
        },
      })
      .then((r) => {
        if (!current) return;
        if (r.data) setStake(r.data.stake);
        else setFailure(errorCode(r.error) ?? "unknown");
      })
      .catch(() => current && setFailure("unavailable"));
    return () => {
      current = false;
    };
  }, [open, clipId, choice]);

  function close(next: boolean) {
    if (!next) {
      setAll(false);
      setPicked([]);
      setStake(undefined);
      setFailure(undefined);
    }
    onOpenChange(next);
  }

  async function remove() {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.DELETE("/api/admin/clips/{id}/analysis", {
        params: {
          path: { id: clipId },
          header: mutation(session?.csrf_token),
        },
        body: all
          ? { all: true, confirm_loss: true }
          : { steps, confirm_loss: true },
      });
      if (r.data) {
        onDeleted(r.data.run ?? null);
        close(false);
      } else setFailure(errorCode(r.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  const losses = stake
    ? [
        stake.segments > 0 &&
          t("clip.analysis.delete.lossSegments", { count: stake.segments }),
        stake.corrections &&
          (stake.corrections.text_segments > 0 ||
            stake.corrections.speaker_segments > 0) &&
          t("clip.analysis.delete.lossCorrections", {
            text: stake.corrections.text_segments,
            speaker: stake.corrections.speaker_segments,
          }),
        stake.voices > 0 &&
          t("clip.analysis.delete.lossVoices", {
            voices: stake.voices,
            validated: stake.validated_voices,
          }),
        stake.voice_samples > 0 &&
          t("clip.analysis.delete.lossSamples", {
            samples: stake.voice_samples,
          }),
        stake.persons > 0 &&
          t("clip.analysis.delete.lossPersons", { persons: stake.persons }),
      ].filter((line): line is string => typeof line === "string")
    : [];

  return (
    <Modal
      open={open}
      onOpenChange={close}
      title={t("clip.analysis.delete.title")}
      closeLabel={t("clip.reanalyze.close")}
    >
      <p className="m-0 text-sm text-mute">{t("clip.analysis.delete.intro")}</p>
      <fieldset className="m-0 flex flex-col gap-1 border-0 p-0">
        <legend className="mb-1 text-sm font-medium">
          {t("clip.analysis.delete.what")}
        </legend>
        {CHOICES.map((step) => (
          <label
            key={step}
            className="flex min-h-11 items-center gap-2 aria-disabled:text-mute"
            aria-disabled={all}
          >
            <input
              type="checkbox"
              disabled={all}
              checked={all || picked.includes(step)}
              onChange={(e) =>
                setPicked(
                  e.target.checked
                    ? [...picked, step]
                    : picked.filter((s) => s !== step),
                )
              }
            />
            {t(`clip.analysis.step.${step}`)}
          </label>
        ))}
        <label className="flex min-h-11 items-center gap-2">
          <input
            type="checkbox"
            checked={all}
            onChange={(e) => setAll(e.target.checked)}
          />
          {t("clip.analysis.delete.all")}
        </label>
      </fieldset>
      {steps.length > 0 && stake && (
        <div role="note" className="flex flex-col gap-1 text-sm">
          <p className="m-0 font-medium">
            {losses.length > 0
              ? t("clip.analysis.delete.losses")
              : t("clip.analysis.delete.none")}
          </p>
          {losses.length > 0 && (
            <ul className="m-0 flex list-disc flex-col gap-1 pl-5">
              {losses.map((line) => (
                <li key={line}>{line}</li>
              ))}
            </ul>
          )}
          <p className="m-0 text-mute">{t("clip.analysis.delete.kept")}</p>
        </div>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      <div className="flex justify-end gap-2">
        <button
          type="button"
          onClick={() => close(false)}
          className={buttonClass("secondary")}
        >
          {t("clip.analysis.delete.cancel")}
        </button>
        <button
          type="button"
          disabled={busy || steps.length === 0 || !stake}
          onClick={() => void remove()}
          className={buttonClass("danger")}
        >
          {t("clip.analysis.delete.confirm")}
        </button>
      </div>
    </Modal>
  );
}
