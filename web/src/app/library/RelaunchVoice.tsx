import { useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { useI18n } from "../i18n";
import { isAdministrator } from "../roles";
import { useSession } from "../session";
import type { ClipVoice } from "./libraryApi";

type Stake = components["schemas"]["VoiceRediarizationStake"];

/**
 * Relaunches the diarization of a clip already processed, with the active model. It is
 * always an explicit action: when the clip holds human work (validated voices, speaker
 * corrections), the API asks for a confirmation and says what is at stake. The visible
 * result stays until the relaunch succeeds; replaced results are listed with their model.
 */
export function RelaunchVoice({
  clipId,
  voice,
  onScheduled,
}: {
  clipId: number;
  voice: ClipVoice;
  onScheduled: () => void;
}) {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const [stake, setStake] = useState<Stake>();
  const [failure, setFailure] = useState<string>();
  const [done, setDone] = useState(false);
  const [busy, setBusy] = useState(false);
  if (!user || !isAdministrator(user) || voice.status === "none") return null;
  const running = voice.status === "queued" || voice.status === "running";

  async function submit(confirm = false) {
    setBusy(true);
    setFailure(undefined);
    setDone(false);
    try {
      const r = await api.POST("/api/admin/clips/{id}/voice", {
        params: { path: { id: clipId }, header: mutation(session?.csrf_token) },
        body: {
          ...(voice.audio_stream_index != null
            ? { audio_stream_index: voice.audio_stream_index }
            : {}),
          ...(confirm ? { confirm_replace_validations: true } : {}),
        },
      });
      if (r.error) {
        if (
          "stake" in r.error &&
          r.error.stake &&
          r.error.error === "voice_rediarization_confirmation_required"
        )
          setStake(r.error.stake);
        else setFailure(errorCode(r.error) ?? "unknown");
        return;
      }
      setStake(undefined);
      setDone(true);
      onScheduled();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section
      aria-label={t("voiceRelaunch.section")}
      className="flex flex-col gap-3 border-t border-line p-3"
    >
      <div>
        <button
          type="button"
          disabled={busy || running}
          onClick={() => void submit()}
          className={buttonClass()}
        >
          {t("voiceRelaunch.button")}
        </button>
      </div>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {done && <StatusNotice>{t("voiceRelaunch.scheduled")}</StatusNotice>}
      {voice.history.length > 0 && (
        <div>
          <h3 className="m-0 text-sm font-medium">
            {t("voiceRelaunch.history", { count: voice.history.length })}
          </h3>
          <ul className="m-0 mt-1 flex list-none flex-col gap-1 p-0 text-sm text-mute">
            {voice.history.map((entry) => (
              <li key={entry.job_id}>
                {t("voiceRelaunch.entry", {
                  model: entry.model,
                  version: entry.model_version,
                  date: new Date(entry.completed_at).toLocaleString(),
                  speakers: entry.speaker_count,
                  validated: entry.validated_speakers,
                })}
              </li>
            ))}
          </ul>
        </div>
      )}
      <ConfirmDialog
        open={!!stake}
        onOpenChange={(open) => {
          if (!open) setStake(undefined);
        }}
        title={t("voiceRelaunch.title")}
        cancelLabel={t("voiceRelaunch.cancel")}
        confirmLabel={t("voiceRelaunch.confirm")}
        danger
        busy={busy}
        onConfirm={() => void submit(true)}
      >
        {stake && (
          <p>
            {t("voiceRelaunch.warning", {
              corrections: stake.speaker_corrections,
              speakers: stake.validated_speakers,
              samples: stake.voice_samples,
            })}
          </p>
        )}
      </ConfirmDialog>
    </section>
  );
}
