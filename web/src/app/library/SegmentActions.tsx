import { useState, type FormEvent } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Field } from "../components/Field";
import { useI18n } from "../i18n";
import { useSession } from "../session";
import type { TranscriptSegment } from "./libraryApi";

type Speaker = NonNullable<
  components["schemas"]["Transcription"]["local_speakers"]
>[number];
type Conflict = components["schemas"]["SegmentCorrection"];
export function SegmentActions({
  clipId,
  jobId,
  segment,
  speakers,
  onChanged,
}: {
  clipId: number;
  jobId?: number;
  segment: TranscriptSegment;
  speakers: Speaker[];
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const [mode, setMode] = useState<"text" | "speaker">();
  const [draft, setDraft] = useState("");
  const [reason, setReason] = useState("");
  const [speaker, setSpeaker] = useState("");
  const [version, setVersion] = useState(segment.correction_version ?? 0);
  const [conflict, setConflict] = useState<Conflict>();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  const index = segment.index;
  if (index === undefined) return null;
  function startText() {
    setDraft(segment.text);
    setReason("");
    setVersion(segment.correction_version ?? 0);
    setConflict(undefined);
    setFailure(undefined);
    setMode("text");
  }
  function startSpeaker() {
    setSpeaker(
      segment.local_speaker_corrected
        ? String(segment.local_speaker_id ?? "")
        : "",
    );
    setVersion(segment.correction_version ?? 0);
    setConflict(undefined);
    setFailure(undefined);
    setMode("speaker");
  }
  async function save(e: FormEvent) {
    e.preventDefault();
    if (index === undefined || !mode) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const params = {
        path: { id: clipId, index },
        header: mutation(session?.csrf_token),
      };
      const r =
        mode === "text"
          ? await api.PUT("/api/clips/{id}/segments/{index}/text", {
              params,
              body: {
                text: draft,
                expected_version: version,
                reason: reason.trim() || undefined,
                transcription_job_id: jobId,
              },
            })
          : await api.PUT("/api/clips/{id}/segments/{index}/speaker", {
              params,
              body: {
                local_speaker_id: speaker ? Number(speaker) : null,
                expected_version: version,
                transcription_job_id: jobId,
              },
            });
      if (r.error) {
        if (
          "current" in r.error &&
          r.error.error === "segment_conflict" &&
          r.error.current
        ) {
          setConflict(r.error.current);
          onChanged();
        } else setFailure(errorCode(r.error) ?? "unknown");
        return;
      }
      setMode(undefined);
      setConflict(undefined);
      onChanged();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="mt-2 flex flex-col gap-2">
      {!mode && (
        <div className="flex flex-wrap gap-2">
          <button type="button" className={buttonClass()} onClick={startText}>
            {t("segmentEdit.editText")}
          </button>
          {speakers.length > 0 && (
            <button
              type="button"
              className={buttonClass()}
              onClick={startSpeaker}
            >
              {t("segmentEdit.editSpeaker")}
            </button>
          )}
        </div>
      )}
      {mode && (
        <form onSubmit={save} className="flex flex-col gap-2">
          {mode === "text" ? (
            <>
              <label className="flex flex-col gap-1">
                {t("segmentEdit.text")}
                <textarea
                  required
                  maxLength={4000}
                  rows={3}
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  className="rounded-lg border border-line-strong bg-surface p-2"
                />
              </label>
              <Field
                label={t("segmentEdit.reason")}
                value={reason}
                maxLength={500}
                onChange={(e) => setReason(e.target.value)}
              />
            </>
          ) : (
            <label className="flex flex-col gap-1">
              {t("segmentEdit.speaker")}
              <select
                value={speaker}
                onChange={(e) => setSpeaker(e.target.value)}
                className="h-11 rounded-lg border border-line-strong bg-surface px-2"
              >
                <option value="">{t("segmentEdit.inferred")}</option>
                {speakers.map((s) => (
                  <option value={s.id} key={s.id}>
                    {s.label}
                  </option>
                ))}
              </select>
            </label>
          )}
          {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          {conflict && (
            <div
              role="group"
              aria-label={t("segmentEdit.conflict")}
              className="rounded-lg border border-line-strong p-3"
            >
              <p>
                {t("segmentEdit.conflictText", {
                  author: conflict.author?.username ?? t("segmentEdit.unknown"),
                  text: conflict.text,
                })}
              </p>
              <button
                type="button"
                className={buttonClass()}
                onClick={() => {
                  setVersion(conflict.version);
                  setConflict(undefined);
                }}
              >
                {t("segmentEdit.keepDraft")}
              </button>
              <button
                type="button"
                className={buttonClass()}
                onClick={() => {
                  setConflict(undefined);
                  setMode(undefined);
                  onChanged();
                }}
              >
                {t("segmentEdit.discard")}
              </button>
            </div>
          )}
          <div className="flex flex-wrap gap-2">
            <button
              disabled={busy || !!conflict}
              className={buttonClass("primary")}
            >
              {mode === "text"
                ? t("segmentEdit.saveText")
                : t("segmentEdit.saveSpeaker")}
            </button>
            <button
              type="button"
              className={buttonClass()}
              onClick={() => setMode(undefined)}
            >
              {t("segmentEdit.cancel")}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
