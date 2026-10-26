import { useId, useState } from "react";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { tintOf } from "./ExcerptThumb";
import { useFailureText } from "./failureText";
import {
  excerptUrl,
  useVoiceGroupActions,
  type Choice,
  type Person,
  type VoiceGroup,
} from "./peopleApi";

const NEW = "new";
const optionClass =
  "flex min-h-11 cursor-pointer items-center gap-3 rounded-lg border border-line px-3 py-2 has-[:checked]:border-accent has-[:checked]:bg-accent-soft";

/**
 * The "Listen and name" panel of a group: every excerpt side by side, a way to leave a voice
 * out (and put it back) without touching the others, then one action that names the voices
 * that were kept.
 */
export function GroupPanel({
  group,
  persons,
  onClose,
}: {
  group: VoiceGroup | undefined;
  persons: Person[];
  /** Called with the text to announce when the group was named, nothing otherwise. */
  onClose: (message?: string) => void;
}) {
  const { t } = useI18n();
  return (
    <Modal
      open={group !== undefined}
      onOpenChange={(open) => !open && onClose()}
      title={t("people.group.title")}
      closeLabel={t("people.group.close")}
      variant="wide"
    >
      {/* Mounted per group, so a new group starts with every voice kept and no choice. */}
      {group && (
        <GroupForm
          key={group.id}
          group={group}
          persons={persons}
          onClose={onClose}
        />
      )}
    </Modal>
  );
}

function GroupForm({
  group,
  persons,
  onClose,
}: {
  group: VoiceGroup;
  persons: Person[];
  onClose: (message?: string) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = useVoiceGroupActions();
  const radios = useId();
  const [excluded, setExcluded] = useState<number[]>(() =>
    group.members.filter((m) => m.excluded).map((m) => m.local_speaker_id),
  );
  const [pending, setPending] = useState<number>();
  const [picked, setPicked] = useState<string>();
  const [newName, setNewName] = useState("");
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  const kept = group.members.filter(
    (m) => !excluded.includes(m.local_speaker_id),
  );
  const name = newName.trim();
  const choice: Choice | undefined =
    picked === undefined
      ? undefined
      : picked === NEW
        ? name
          ? { newName: name }
          : undefined
        : { personId: Number(picked) };
  const who =
    choice === undefined
      ? ""
      : "newName" in choice
        ? choice.newName
        : (persons.find((p) => p.id === choice.personId)?.display_name ?? "");
  const ready = choice !== undefined && kept.length > 0;
  const available = persons.filter(
    (person) =>
      person.voice_profile_deleted_at === null &&
      person.display_name.toLowerCase().includes(query.trim().toLowerCase()),
  );

  async function toggle(localSpeakerId: number) {
    const leaving = !excluded.includes(localSpeakerId);
    setPending(localSpeakerId);
    setFailure(undefined);
    const result = leaving
      ? await actions.exclude(localSpeakerId)
      : await actions.restore(localSpeakerId);
    setPending(undefined);
    if (!result.ok) {
      setFailure(result.code);
      return;
    }
    setExcluded(
      leaving
        ? [...excluded, localSpeakerId]
        : excluded.filter((id) => id !== localSpeakerId),
    );
  }

  async function confirm() {
    if (!choice || !ready) return;
    setBusy(true);
    setFailure(undefined);
    // A group is known by its smallest voice: the one of what was kept.
    const ids = kept.map((m) => m.local_speaker_id);
    const result = await actions.name(Math.min(...ids), ids, choice);
    setBusy(false);
    if (!result.ok) {
      setFailure(result.code);
      return;
    }
    const person = result.value.person.display_name;
    const count = result.value.validated.length;
    const skipped = result.value.skipped.length;
    const left = excluded.length;
    const text = left
      ? t("people.group.savedLeftOut", { count, person, out: left })
      : t("people.group.saved", { count, person });
    onClose(
      skipped ? `${text} ${t("people.group.savedSkipped", { skipped })}` : text,
    );
  }

  const radio = (value: string, label: string) => (
    <label className={optionClass}>
      <input
        type="radio"
        name={radios}
        value={value}
        checked={picked === value}
        onChange={() => setPicked(value)}
        className="size-4 accent-accent-solid"
      />
      <span className="flex-1">{label}</span>
    </label>
  );

  return (
    <>
      <p className="text-mute">{t("people.group.lead")}</p>
      <p role="status" className="font-medium">
        {t("people.group.selected", {
          kept: kept.length,
          total: group.members.length,
        })}
      </p>
      <ul
        aria-label={t("people.group.voices")}
        className="m-0 grid list-none grid-cols-1 gap-3 p-0 sm:grid-cols-2"
      >
        {group.members.map((member) => {
          const out = excluded.includes(member.local_speaker_id);
          return (
            <li
              key={member.local_speaker_id}
              className="flex flex-col gap-2 rounded-xl border border-line p-3"
            >
              <div className="flex items-center gap-2">
                <span
                  aria-hidden="true"
                  className={`size-8 shrink-0 rounded-full ${tintOf(member.local_speaker_id)} ${out ? "opacity-50" : ""}`}
                />
                <div className="flex min-w-0 flex-col">
                  <b className="truncate">{member.clip_name}</b>
                  <span className="text-mute">
                    {member.label} ·{" "}
                    {t("people.group.speech", {
                      seconds: member.speaking_seconds.toFixed(1),
                    })}
                    {out && <> · {t("people.group.out")}</>}
                  </span>
                </div>
              </div>
              <audio
                controls
                preload="none"
                aria-label={t("people.group.excerpt", {
                  label: member.label,
                  clip: member.clip_name,
                })}
                src={excerptUrl(member.local_speaker_id)}
                className="w-full"
              />
              <button
                type="button"
                disabled={pending === member.local_speaker_id || busy}
                aria-label={t(
                  out ? "people.group.putBackFor" : "people.group.leaveOutFor",
                  { label: member.label, clip: member.clip_name },
                )}
                onClick={() => void toggle(member.local_speaker_id)}
                className={buttonClass("secondary")}
              >
                {t(out ? "people.group.putBack" : "people.group.leaveOut")}
              </button>
            </li>
          );
        })}
      </ul>

      <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
        <legend className="mb-1 font-medium">{t("people.group.who")}</legend>
        <input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          aria-label={t("people.group.searchLabel")}
          placeholder={t("people.group.searchPlaceholder")}
          className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
        />
        {available.map((person) => (
          <div key={person.id}>
            {radio(String(person.id), person.display_name)}
          </div>
        ))}
        {available.length === 0 && (
          <p className="text-mute">{t("people.group.noMatch")}</p>
        )}
        {radio(NEW, t("people.group.someoneNew"))}
        {picked === NEW && (
          <input
            type="text"
            value={newName}
            maxLength={100}
            onChange={(event) => setNewName(event.target.value)}
            aria-label={t("people.group.newName")}
            autoComplete="off"
            autoFocus
            className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
          />
        )}
      </fieldset>

      {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}
      <div className="mt-auto flex justify-end gap-2 pt-2">
        <button
          type="button"
          onClick={() => onClose()}
          className={buttonClass("secondary")}
        >
          {t("people.group.notNow")}
        </button>
        <button
          type="button"
          disabled={!ready || busy || pending !== undefined}
          onClick={() => void confirm()}
          className={buttonClass("primary")}
        >
          {ready
            ? t("people.group.name", { count: kept.length, person: who })
            : t("people.group.nameNone")}
        </button>
      </div>
    </>
  );
}
