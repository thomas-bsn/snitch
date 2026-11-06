import { useState, type FormEvent } from "react";
import { Link } from "react-router";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import type { Person } from "../../people/peopleApi";
import { PersonSearch } from "./PersonSearch";
import { Section } from "./Section";
import { useMyVoice } from "./useMyVoice";
import { usePersonLink, type PersonLinkRequest } from "./usePersonLink";

const STEPS = ["stepSent", "stepReview", "stepIdentified"] as const;

/** The three steps of a request: sent, then validated by an administrator, then identified. */
function Progress({ request }: { request: PersonLinkRequest }) {
  const { t, language } = useI18n();
  // A pending request has been sent and waits for the administrator.
  const current = 1;
  return (
    <ol
      aria-label={t("account.voice.progressLabel")}
      className="flex list-none flex-col gap-2 p-0 md:flex-row md:gap-3"
    >
      {STEPS.map((step, index) => {
        const state =
          index < current ? "Done" : index === current ? "Current" : "Todo";
        return (
          <li
            key={step}
            aria-current={index === current ? "step" : undefined}
            className={`flex min-h-11 flex-1 flex-col justify-center rounded-lg border px-3 py-2 text-sm ${
              index === current
                ? "border-accent bg-accent-soft text-accent-ink"
                : "border-line"
            }`}
          >
            <span className="font-medium">
              {index + 1}. {t(`account.voice.${step}`)}
            </span>
            <span className="text-mute">
              {t(`account.voice.step${state}`)}
              {index === 0 &&
                ` · ${new Date(request.created_at).toLocaleDateString(language)}`}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

/**
 * The "My voice" tab: one card whose content follows the state of the link request (no
 * Person, pending, refused, linked), and, once a Person is linked, the sensitive zone
 * (delete my voice profile, unlink my account). A link is created by an administrator;
 * the account only asks and can withdraw.
 */
export function VoiceTab() {
  const { t, error } = useI18n();
  const { voice, erase, reload } = useMyVoice();
  const { link, request, cancel, unlink } = usePersonLink(reload);

  const [picked, setPicked] = useState<Person>();
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [confirmingUnlink, setConfirmingUnlink] = useState(false);
  const [confirmingErase, setConfirmingErase] = useState(false);

  if (voice.phase === "loading" || link.phase === "loading") {
    return <p className="text-mute">{t("account.voice.loading")}</p>;
  }
  if (voice.phase === "failed") {
    return <ErrorAlert>{error(voice.code)}</ErrorAlert>;
  }

  async function run<T>(
    action: () => Promise<{ ok: true; value: T } | { ok: false; code: string }>,
    done: (value: T) => void,
  ) {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    const outcome = await action();
    setBusy(false);
    if (outcome.ok) done(outcome.value);
    else setFailure(outcome.code);
  }

  function submitRequest(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!picked) return;
    void run(
      () => request(picked.id, message),
      () => {
        setPicked(undefined);
        setMessage("");
      },
    );
  }

  const feedback = (
    <>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <StatusNotice>{notice}</StatusNotice>}
    </>
  );

  if (voice.phase === "none") {
    return (
      <Section title={t("account.voice.title")}>
        {feedback}
        {link.phase === "pending" ? (
          <div className="flex flex-col gap-3">
            <p>
              {t("account.voice.linkPending", {
                name: link.request.person.display_name,
              })}
            </p>
            <Progress request={link.request} />
            <div>
              <button
                type="button"
                disabled={busy}
                onClick={() =>
                  void run(cancel, () =>
                    setNotice(t("account.voice.linkCancelled")),
                  )
                }
                className={buttonClass("secondary")}
              >
                {busy
                  ? t("account.voice.linkCancelling")
                  : t("account.voice.linkCancel")}
              </button>
            </div>
          </div>
        ) : (
          <form onSubmit={submitRequest} className="flex flex-col gap-3">
            <h3 className="font-medium">{t("account.voice.linkTitle")}</h3>
            {link.phase === "refused" ? (
              <p
                role="alert"
                className="rounded-lg bg-bad-bg px-3 py-2.5 text-bad-fg"
              >
                {t("account.voice.linkRefused", {
                  name: link.request.person.display_name,
                })}
              </p>
            ) : (
              <p className="text-mute">{t("account.voice.noPerson")}</p>
            )}
            {link.phase === "failed" && (
              <ErrorAlert>{error(link.code)}</ErrorAlert>
            )}
            <PersonSearch selected={picked?.id} onSelect={setPicked} />
            <label className="flex flex-col gap-1.5 font-medium">
              {t("account.voice.linkMessage")}
              <textarea
                rows={3}
                maxLength={500}
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                className="rounded-lg border border-line-strong bg-surface p-3 text-ink"
              />
            </label>
            <p className="text-xs text-mute">
              {t("account.voice.linkMessageHint")}
            </p>
            <div>
              <button
                type="submit"
                disabled={busy || !picked}
                className={buttonClass("primary")}
              >
                {busy
                  ? t("account.voice.linkRequesting")
                  : t("account.voice.linkButton")}
              </button>
            </div>
          </form>
        )}
      </Section>
    );
  }

  const { person, clipTotal, impact, impactFailed } = voice;
  const deleted = person.voice_profile_deleted_at !== null;
  const counts = [
    ["countSamples", person.validated_sample_count],
    ["countVoices", person.local_speaker_count],
    ["countClips", clipTotal],
  ] as const;
  const effects = impact && {
    samples: impact.voice_samples,
    fingerprints: impact.fingerprints,
    attributions: impact.attributions,
  };

  return (
    <div className="flex flex-col gap-4">
      <Section title={t("account.voice.title")}>
        <p>{t("account.voice.identity", { name: person.display_name })}</p>
        <ul
          aria-label={t("account.voice.countsLabel")}
          className="m-0 grid list-none grid-cols-3 gap-2 p-0"
        >
          {counts.map(([key, value]) => (
            <li key={key} className="rounded-lg border border-line px-3 py-2">
              <span className="block text-xl font-semibold">{value}</span>
              <span className="text-sm text-mute">
                {t(`account.voice.${key}`)}
              </span>
            </li>
          ))}
        </ul>
        <p className="text-sm text-mute">{t("account.voice.clipsHint")}</p>
        <div>
          <Link
            to={`/library?person=${person.id}`}
            className={buttonClass("secondary")}
          >
            {t("account.voice.clipsLink")}
          </Link>
        </div>
        {feedback}
      </Section>
      <Section title={t("account.voice.deleteTitle")}>
        {deleted ? (
          <p className="text-mute">{t("account.voice.deleted")}</p>
        ) : effects ? (
          <>
            <p className="text-mute">
              {t("account.voice.deleteEffects", effects)}
            </p>
            <div>
              <button
                type="button"
                disabled={busy}
                onClick={() => setConfirmingErase(true)}
                className={buttonClass("danger")}
              >
                {t("account.voice.deleteButton")}
              </button>
            </div>
          </>
        ) : (
          impactFailed && (
            <ErrorAlert>{t("account.voice.impactFailed")}</ErrorAlert>
          )
        )}
      </Section>
      <Section title={t("account.voice.unlinkTitle")}>
        <p className="text-mute">{t("account.voice.unlinkBody")}</p>
        <div>
          <button
            type="button"
            disabled={busy}
            onClick={() => setConfirmingUnlink(true)}
            className={buttonClass("secondary")}
          >
            {busy
              ? t("account.voice.unlinking")
              : t("account.voice.unlinkButton")}
          </button>
        </div>
      </Section>
      <ConfirmDialog
        open={confirmingUnlink}
        onOpenChange={setConfirmingUnlink}
        title={t("account.voice.unlinkConfirmTitle", {
          name: person.display_name,
        })}
        cancelLabel={t("account.voice.keep")}
        confirmLabel={t("account.voice.unlinkConfirm")}
        onConfirm={() =>
          void run(unlink, () => setNotice(t("account.voice.unlinked"))).then(
            () => setConfirmingUnlink(false),
          )
        }
        busy={busy}
      >
        <h3 className="font-medium">{t("account.voice.unlinkLoseTitle")}</h3>
        <p>{t("account.voice.unlinkLose", { name: person.display_name })}</p>
        <h3 className="mt-2 font-medium">
          {t("account.voice.unlinkKeepTitle")}
        </h3>
        <p>
          {t("account.voice.unlinkKeep", {
            name: person.display_name,
            voices: person.local_speaker_count,
            clips: clipTotal,
          })}
        </p>
      </ConfirmDialog>
      <ConfirmDialog
        open={confirmingErase}
        onOpenChange={setConfirmingErase}
        title={t("account.voice.confirmTitle")}
        cancelLabel={t("account.voice.keep")}
        confirmLabel={
          busy ? t("account.voice.deleting") : t("account.voice.confirm")
        }
        onConfirm={() =>
          void run(
            () => erase(person),
            (done) => {
              setNotice(
                t("account.voice.done", {
                  samples: done.samples,
                  fingerprints: done.fingerprints,
                  attributions: done.attributions,
                }),
              );
            },
          ).then(() => setConfirmingErase(false))
        }
        busy={busy}
        danger
      >
        {effects && <p>{t("account.voice.confirmBody", effects)}</p>}
      </ConfirmDialog>
    </div>
  );
}
