import { useCallback } from "react";
import { useOutletContext, useSearchParams } from "react-router";
import { useLoaded } from "../api/loaded";
import { buttonClass } from "../components/buttons";
import { ErrorAlert } from "../components/Alert";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import type { PeopleOutlet } from "./PeopleLayout";
import { fetchLocalSpeaker } from "./peopleApi";
import { ProcessScreen } from "./ProcessScreen";
import { ToProcessList } from "./ToProcessList";

const positive = (value: string | null) => {
  const number = Number(value);
  return Number.isInteger(number) && number > 0 ? number : undefined;
};

/** Where a direct link to a voice that is missing or hidden lands: the same message for both. */
function DirectLinkNotice({ onContinue }: { onContinue: () => void }) {
  const { t } = useI18n();
  return (
    <section
      aria-label={t("people.verification.directTitle")}
      className="flex flex-col items-start gap-3 rounded-xl border border-line p-4"
    >
      <h2 className="m-0 text-base font-semibold">
        {t("people.verification.unavailableTitle")}
      </h2>
      <p role="status" className="m-0">
        {t("people.verification.unavailable")}
      </p>
      <button
        type="button"
        onClick={onContinue}
        className={buttonClass("secondary")}
      >
        {t("people.verification.continueReview")}
      </button>
    </section>
  );
}

/** The clip of a case address (?case=): the screen opens on that voice. */
function CaseScreen({
  voiceId,
  onLeave,
}: {
  voiceId: number;
  onLeave: () => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const { state } = useLoaded(
    useCallback(() => fetchLocalSpeaker(voiceId), [voiceId]),
  );
  if (state.status === "loading") {
    return (
      <p className="text-mute">{t("people.verification.directLoading")}</p>
    );
  }
  if (state.status === "failed") {
    return state.code === "local_speaker_not_found" ? (
      <DirectLinkNotice onContinue={onLeave} />
    ) : (
      <ErrorAlert>{failureText(state.code)}</ErrorAlert>
    );
  }
  return (
    <ProcessScreen
      key={state.value.clip_id}
      clipId={state.value.clip_id}
      focusId={voiceId}
    />
  );
}

/**
 * The "To process" section: the list of clips, and the processing screen of one clip when
 * the address names it (`?clip=` from the list and the notifications, `?case=` from the
 * former single-case address, which opens the clip of that case).
 */
export function VerificationTab() {
  const outlet = useOutletContext<PeopleOutlet | null>();
  const [params, setParams] = useSearchParams();
  const clip = positive(params.get("clip"));
  const voiceId = positive(params.get("case"));
  if (voiceId !== undefined) {
    return (
      <CaseScreen
        voiceId={voiceId}
        onLeave={() => setParams({}, { replace: true })}
      />
    );
  }
  if (clip !== undefined) return <ProcessScreen key={clip} clipId={clip} />;
  return <ToProcessList version={0} onToDecide={outlet?.reportToDecide} />;
}
