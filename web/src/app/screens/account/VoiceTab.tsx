import { useState } from "react";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import type { Person } from "../../api/client";
import { Section } from "./Section";
import { useMyVoice } from "./useMyVoice";

/** The "My voice" tab: where your voice is identified, and the deletion of your voice profile. */
export function VoiceTab() {
  const { t, error, language } = useI18n();
  const { voice, erase } = useMyVoice();
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [done, setDone] = useState<string>();

  if (voice.phase === "loading") {
    return <p className="text-mute">{t("account.voice.loading")}</p>;
  }
  if (voice.phase === "failed") {
    return <ErrorAlert>{error(voice.code)}</ErrorAlert>;
  }
  if (voice.phase === "none") {
    return (
      <Section title={t("account.voice.title")}>
        <p className="text-mute">{t("account.voice.noPerson")}</p>
      </Section>
    );
  }

  const { person, clips } = voice;
  const deleted = person.voice_profile_deleted_at !== null;

  async function confirm(target: Person) {
    setBusy(true);
    setFailure(undefined);
    const outcome = await erase(target);
    setBusy(false);
    setConfirming(false);
    if (outcome.ok) {
      setDone(
        t("account.voice.done", {
          samples: outcome.value.samples,
          fingerprints: outcome.value.fingerprints,
          attributions: outcome.value.attributions,
        }),
      );
    } else {
      setFailure(outcome.code);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Section title={t("account.voice.title")}>
        <p>{t("account.voice.identity", { name: person.display_name })}</p>
        <p className="text-mute">
          {deleted
            ? t("account.voice.deleted")
            : t("account.voice.samples", {
                count: person.validated_sample_count,
              })}
        </p>
        <h3 className="mt-1 font-medium">{t("account.voice.clipsTitle")}</h3>
        {clips.length === 0 ? (
          <p className="text-mute">{t("account.voice.noClips")}</p>
        ) : (
          <ul
            aria-label={t("account.voice.clipsLabel")}
            className="flex flex-col gap-1"
          >
            {clips.map((clip) => (
              <li key={clip.id}>
                {clip.original_name}
                <span className="text-mute">
                  {" · "}
                  {new Date(clip.recorded_at).toLocaleDateString(language)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Section>
      <Section title={t("account.voice.deleteTitle")}>
        <p className="text-mute">{t("account.voice.deleteEffects")}</p>
        {done && <StatusNotice>{done}</StatusNotice>}
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {!deleted && (
          <div>
            <button
              type="button"
              onClick={() => setConfirming(true)}
              className={buttonClass("danger")}
            >
              {t("account.voice.deleteButton")}
            </button>
          </div>
        )}
      </Section>
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={t("account.voice.confirmTitle")}
        cancelLabel={t("account.voice.keep")}
        confirmLabel={
          busy ? t("account.voice.deleting") : t("account.voice.confirm")
        }
        onConfirm={() => void confirm(person)}
        busy={busy}
        danger
      >
        <p>{t("account.voice.confirmBody")}</p>
      </ConfirmDialog>
    </div>
  );
}
