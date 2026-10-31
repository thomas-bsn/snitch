import { useState, type FormEvent } from "react";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import type { Person } from "../../api/client";
import { fetchPersons } from "../../people/peopleApi";
import { Section } from "./Section";
import { useMyVoice } from "./useMyVoice";
import { usePersonLink } from "./usePersonLink";

const selectedClass = (active: boolean) =>
  `w-full justify-start rounded-lg border px-3 py-2 text-left text-sm ${
    active ? "border-accent bg-accent-soft text-accent-ink" : "border-line"
  }`;

/**
 * The "My voice" tab: asking to link your account to a person, where your voice is
 * identified, the deletion of your voice profile, and unlinking your account. A link
 * is created by an administrator; the account only asks and can withdraw.
 */
export function VoiceTab() {
  const { t, error, language } = useI18n();
  const { voice, erase, reload } = useMyVoice();
  const { link, request, cancel, unlink } = usePersonLink(reload);
  const persons = useLoaded(fetchPersons);

  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<number>();
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

  const needle = query.trim().toLowerCase();
  const matches =
    persons.state.status === "ready"
      ? persons.state.value
          .filter(
            (p) => !needle || p.display_name.toLowerCase().includes(needle),
          )
          .slice(0, 20)
      : [];

  async function submitRequest(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (selected === undefined) return;
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    const outcome = await request(selected, message);
    setBusy(false);
    if (outcome.ok) {
      setSelected(undefined);
      setMessage("");
      setQuery("");
    } else {
      setFailure(outcome.code);
    }
  }

  async function withdraw() {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    const outcome = await cancel();
    setBusy(false);
    if (outcome.ok) setNotice(t("account.voice.linkCancelled"));
    else setFailure(outcome.code);
  }

  async function confirmUnlink() {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    const outcome = await unlink();
    setBusy(false);
    setConfirmingUnlink(false);
    if (outcome.ok) setNotice(t("account.voice.unlinked"));
    else setFailure(outcome.code);
  }

  async function confirmErase(person: Person) {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    const outcome = await erase(person);
    setBusy(false);
    setConfirmingErase(false);
    if (outcome.ok) {
      setNotice(
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

  if (voice.phase === "none") {
    return (
      <Section title={t("account.voice.linkTitle")}>
        <p className="text-mute">{t("account.voice.noPerson")}</p>
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {notice && <StatusNotice>{notice}</StatusNotice>}
        {link.phase === "pending" ? (
          <div className="flex flex-col gap-3">
            <StatusNotice>
              {t("account.voice.linkPending", {
                name: link.request.person.display_name,
              })}
            </StatusNotice>
            <div>
              <button
                type="button"
                disabled={busy}
                onClick={() => void withdraw()}
                className={buttonClass("secondary")}
              >
                {busy
                  ? t("account.voice.linkCancelling")
                  : t("account.voice.linkCancel")}
              </button>
            </div>
          </div>
        ) : (
          <>
            {link.phase === "refused" && (
              <StatusNotice>
                {t("account.voice.linkRefused", {
                  name: link.request.person.display_name,
                })}
              </StatusNotice>
            )}
            {link.phase === "failed" && (
              <ErrorAlert>{error(link.code)}</ErrorAlert>
            )}
            <form onSubmit={submitRequest} className="flex flex-col gap-3">
              <Field
                label={t("account.voice.linkSearch")}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                maxLength={100}
              />
              {persons.state.status === "failed" && (
                <ErrorAlert>{error(persons.state.code)}</ErrorAlert>
              )}
              {persons.state.status === "ready" && (
                <ul
                  aria-label={t("account.voice.linkSearch")}
                  className="flex max-h-64 list-none flex-col gap-1 overflow-y-auto p-0"
                >
                  {matches.map((p) => (
                    <li key={p.id}>
                      <button
                        type="button"
                        aria-pressed={selected === p.id}
                        className={selectedClass(selected === p.id)}
                        onClick={() => setSelected(p.id)}
                      >
                        {p.display_name}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
              {persons.state.status === "ready" && matches.length === 0 && (
                <p className="text-mute">{t("account.voice.linkNoMatches")}</p>
              )}
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
                  disabled={busy || selected === undefined}
                  className={buttonClass("primary")}
                >
                  {busy
                    ? t("account.voice.linkRequesting")
                    : t("account.voice.linkButton")}
                </button>
              </div>
            </form>
          </>
        )}
      </Section>
    );
  }

  const { person, clips } = voice;
  const deleted = person.voice_profile_deleted_at !== null;

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
                  {clip.recorded_at
                    ? new Date(clip.recorded_at).toLocaleDateString(language)
                    : t("clip.speaker.unknown")}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Section>
      <Section title={t("account.voice.deleteTitle")}>
        <p className="text-mute">{t("account.voice.deleteEffects")}</p>
        {notice && <StatusNotice>{notice}</StatusNotice>}
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {!deleted && (
          <div>
            <button
              type="button"
              onClick={() => setConfirmingErase(true)}
              className={buttonClass("danger")}
            >
              {t("account.voice.deleteButton")}
            </button>
          </div>
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
        confirmLabel={
          busy ? t("account.voice.unlinking") : t("account.voice.unlinkConfirm")
        }
        onConfirm={() => void confirmUnlink()}
        busy={busy}
      >
        <p>
          {t("account.voice.unlinkConfirmBody", { name: person.display_name })}
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
        onConfirm={() => void confirmErase(person)}
        busy={busy}
        danger
      >
        <p>{t("account.voice.confirmBody")}</p>
      </ConfirmDialog>
    </div>
  );
}
