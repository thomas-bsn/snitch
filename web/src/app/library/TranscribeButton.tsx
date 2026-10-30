import { useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { useI18n } from "../i18n";
import { useSession } from "../session";

type ClipProgress = components["schemas"]["ClipProgress"];
type AudioTrackSelection = components["schemas"]["AudioTrackSelection"];
type Job = components["schemas"]["Job"];
type MediaTrack = components["schemas"]["MediaTrack"];

export function TranscribeButton({
  clipId,
  progress,
  audioTrackSelection,
  hasCorrections,
  onChanged,
  audioTracks,
}: {
  clipId: number;
  progress: ClipProgress | undefined;
  audioTrackSelection: AudioTrackSelection | undefined;
  hasCorrections: boolean;
  onChanged: () => void;
  audioTracks: MediaTrack[];
}) {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const [warning, setWarning] = useState<{
    text_segments: number;
    speaker_segments: number;
    authors: string[];
  }>();
  const [failure, setFailure] = useState<string>();
  const [done, setDone] = useState(false);
  const [busy, setBusy] = useState(false);
  const [job, setJob] = useState<Job | null>(null);
  const [showTrackSelection, setShowTrackSelection] = useState(false);
  const [selectedStream, setSelectedStream] = useState<number | null>(null);

  // Don't show if transcription is already done or in progress
  const transcriptionState = progress?.transcription ?? "none";
  const isTranscriptionDone = transcriptionState === "done";
  const isTranscriptionRunning =
    transcriptionState === "running" || transcriptionState === "queued";

  // Show button if clip is ready and transcription is not done
  const clipReady = transcriptionState !== "unavailable";
  if (!clipReady || isTranscriptionDone || isTranscriptionRunning) return null;

  // If we have a job, show its status
  if (job) {
    const jobState = job.status;
    const isDone = jobState === "done";
    const isFailed = jobState === "failed" || jobState === "dead";
    const isRunning = jobState === "leased";
    const isQueued =
      jobState === "pending" ||
      jobState === "retry_wait" ||
      jobState === "cancelled";

    return (
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">
            {t("clip.transcribe.scheduled")}
          </span>
          <span
            className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
              isDone
                ? "bg-ok-fg/10 text-ok-fg"
                : isFailed
                  ? "bg-bad-fg/10 text-bad-fg"
                  : isRunning
                    ? "bg-accent-ink/10 text-accent-ink"
                    : "bg-warn-fg/10 text-warn-fg"
            }`}
          >
            {t(`clip.transcribe.job.${jobState}`)}
          </span>
        </div>
        {isFailed && job.last_error && (
          <ErrorAlert>
            {t("clip.transcribe.jobError", { error: error(job.last_error) })}
          </ErrorAlert>
        )}
        {isDone && (
          <StatusNotice>{t("clip.transcribe.scheduled")}</StatusNotice>
        )}
        <button
          type="button"
          disabled={busy}
          onClick={() => {
            setJob(null);
            setDone(false);
          }}
          className={buttonClass("secondary")}
        >
          {t("clip.transcribe.clear")}
        </button>
      </div>
    );
  }

  const needsTrackSelection =
    audioTrackSelection?.status === "selection_required" &&
    audioTrackSelection.audio_stream_index === undefined;

  async function submit(confirm = false) {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/clips/{id}/transcription", {
        params: {
          path: { id: clipId },
          header: mutation(session?.csrf_token),
          query: { confirm_replace_corrections: confirm || undefined },
        },
        body: {
          ...(selectedStream !== null
            ? { audio_stream_index: selectedStream }
            : audioTrackSelection?.audio_stream_index !== undefined
              ? { audio_stream_index: audioTrackSelection.audio_stream_index }
              : {}),
        },
      });
      if (r.error) {
        if (
          "corrections" in r.error &&
          r.error.error === "corrections_would_be_replaced" &&
          r.error.corrections
        )
          setWarning(r.error.corrections);
        else if (r.error.error === "audio_track_selection_required") {
          setShowTrackSelection(true);
          return;
        } else setFailure(errorCode(r.error) ?? "unknown");
        return;
      }
      setWarning(undefined);
      setDone(true);
      if (r.data?.job) setJob(r.data.job);
      onChanged();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  function handleTrackSelect() {
    if (selectedStream !== null) {
      setShowTrackSelection(false);
      void submit();
    }
  }

  if (needsTrackSelection && showTrackSelection) {
    const audioOnlyTracks = audioTracks.filter((t) => t.type === "audio");

    return (
      <div className="flex flex-col gap-2">
        <div>
          <p className="mb-2 text-sm text-mute">
            {t("clip.transcribe.selectTrack")}
          </p>
          <div className="flex gap-2">
            <select
              value={selectedStream ?? ""}
              onChange={(e) =>
                setSelectedStream(
                  e.target.value ? Number(e.target.value) : null,
                )
              }
              className="flex-1 min-h-11 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="">
                {t("clip.transcribe.selectTrackPlaceholder")}
              </option>
              {audioOnlyTracks.map((track) => (
                <option key={track.index} value={track.index}>
                  {track.title ?? `Track ${track.index}`}
                  {track.language && ` (${track.language})`}
                  {track.channels && ` ${track.channels}ch`}
                </option>
              ))}
            </select>
            <button
              type="button"
              disabled={selectedStream === null || busy}
              onClick={handleTrackSelect}
              className={buttonClass()}
            >
              {t("clip.transcribe.confirmTrack")}
            </button>
          </div>
        </div>
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        <button
          type="button"
          onClick={() => setShowTrackSelection(false)}
          className={buttonClass("secondary")}
        >
          {t("segmentEdit.cancel")}
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div>
        <button
          type="button"
          disabled={busy}
          onClick={() => void submit()}
          className={buttonClass()}
        >
          {t("clip.transcribe.button")}
        </button>
      </div>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {done && <StatusNotice>{t("clip.transcribe.scheduled")}</StatusNotice>}
      <ConfirmDialog
        open={!!warning}
        onOpenChange={(open) => {
          if (!open) setWarning(undefined);
        }}
        title={t("clip.transcribe.retranscribeTitle")}
        cancelLabel={t("segmentEdit.cancel")}
        confirmLabel={t("segmentEdit.confirmRetranscribe")}
        danger
        busy={busy}
        onConfirm={() => void submit(true)}
      >
        {warning && (
          <p>
            {t("segmentEdit.retranscribeWarning", {
              texts: warning.text_segments,
              speakers: warning.speaker_segments,
              authors: warning.authors.join(", "),
            })}
          </p>
        )}
      </ConfirmDialog>
    </div>
  );
}
