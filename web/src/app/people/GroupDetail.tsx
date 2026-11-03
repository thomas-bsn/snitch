import { ArrowLeft } from "lucide-react";
import { useCallback, useId, useState } from "react";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { ExcerptPlayer } from "./ExcerptPlayer";
import { LevelBadge } from "./LevelBadge";
import { useFailureText } from "./failureText";
import {
  fetchPersonPage,
  fetchVoiceGroupOf,
  useVoiceGroupActions,
  type Choice,
  type VoiceGroup,
} from "./peopleApi";

const NEW = "new";
const SUGGESTIONS = 8;
const optionClass =
  "flex min-h-11 cursor-pointer items-center gap-3 rounded-lg border border-line px-3 py-2 has-[:checked]:border-accent has-[:checked]:bg-accent-soft";
const inputClass =
  "h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10";

/**
 * The group that holds a voice, opened from the address of that voice. A group has no stable
 * id: when the voice is in no group any more, the screen says so and opens nothing else.
 */
export function GroupDetail({
  voice,
  onBack,
  onNamed,
}: {
  voice: number;
  onBack: () => void;
  /** Called with the text to announce once the voices were named. */
  onNamed: (message: string) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const [anchor, setAnchor] = useState(voice);
  const [round, setRound] = useState(0);
  const load = useCallback(
    () => fetchVoiceGroupOf(anchor),
    // A new round reads the groups again, even for the same voice.
    [anchor, round],
  );
  const { state } = useLoaded(load);

  const back = (
    <button
      type="button"
      onClick={onBack}
      className={`${buttonClass("secondary")} w-fit gap-2`}
    >
      <ArrowLeft className="size-4" aria-hidden="true" />
      {t("people.group.back")}
    </button>
  );

  const group = state.status === "ready" ? state.value : undefined;
  return (
    <div className="flex flex-col gap-4">
      {back}
      {state.status === "loading" && (
        <p className="text-mute">{t("people.group.loading")}</p>
      )}
      {state.status === "failed" && (
        <ErrorAlert>{failureText(state.code)}</ErrorAlert>
      )}
      {state.status === "ready" && !group && (
        <div className="rounded-xl border border-dashed border-line-strong p-6">
          <h2 className="text-lg font-semibold">{t("people.group.gone")}</h2>
          <p className="mt-1 text-mute">{t("people.group.goneBody")}</p>
        </div>
      )}
      {group && (
        <GroupForm
          key={`${group.id}:${round}`}
          group={group}
          onNamed={onNamed}
          onReload={(next) => {
            setAnchor(next);
            setRound((n) => n + 1);
          }}
        />
      )}
    </div>
  );
}

function GroupForm({
  group,
  onNamed,
  onReload,
}: {
  group: VoiceGroup;
  onNamed: (message: string) => void;
  /** Reads the group again, from a voice that is still kept. */
  onReload: (voice: number) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = useVoiceGroupActions();
  const radios = useId();
  const [excluded, setExcluded] = useState<number[]>(() =>
    group.members.filter((m) => m.excluded).map((m) => m.local_speaker_id),
  );
  const [pending, setPending] = useState<number>();
  const [picked, setPicked] = useState<{ id: number; name: string } | "new">();
  const [newName, setNewName] = useState("");
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  const search = query.trim();
  const loadPersons = useCallback(
    () => fetchPersonPage(search, 1, SUGGESTIONS),
    [search],
  );
  const persons = useLoaded(loadPersons);
  const available =
    persons.state.status === "ready"
      ? persons.state.value.items.filter(
          (person) => person.voice_profile_deleted_at === null,
        )
      : [];

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
        : { personId: picked.id };
  const who =
    choice === undefined
      ? ""
      : "newName" in choice
        ? choice.newName
        : (picked as { name: string }).name;
  const ready = choice !== undefined && kept.length > 0;
  const conflict = failure === "group_changed";

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

  async function setAside(disposition: "ignored" | "random") {
    setBusy(true);
    setFailure(undefined);
    const result = await actions.setDisposition(group.id, disposition);
    setBusy(false);
    if (!result.ok) {
      setFailure(result.code);
      return;
    }
    onNamed(
      t(
        disposition === "ignored"
          ? "people.group.savedIgnored"
          : "people.group.savedRandom",
        { count: result.value },
      ),
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
    onNamed(
      skipped ? `${text} ${t("people.group.savedSkipped", { skipped })}` : text,
    );
  }

  return (
    <>
      <h2 className="text-lg font-semibold">
        {group.members.length > 1
          ? t("people.group.titleProposed", { count: group.members.length })
          : t("people.group.titleIsolated")}
      </h2>
      <p className="max-w-2xl text-mute">{t("people.group.lead")}</p>
      <p className="m-0 text-mute">
        {t("people.group.videos", {
          count: new Set(group.members.map((m) => m.clip_name)).size,
        })}
      </p>
      <ul
        aria-label={t("people.group.voices")}
        className="m-0 grid list-none grid-cols-1 gap-3 p-0 md:grid-cols-2"
      >
        {group.members.map((member) => {
          const out = excluded.includes(member.local_speaker_id);
          return (
            <li
              key={member.local_speaker_id}
              className="flex flex-col gap-2 rounded-xl border border-line p-3"
            >
              <label className="flex min-h-11 cursor-pointer items-center gap-3">
                <input
                  type="checkbox"
                  checked={!out}
                  disabled={pending !== undefined || busy}
                  aria-label={t("people.group.keep", {
                    label: member.label,
                    clip: member.clip_name,
                  })}
                  onChange={() => void toggle(member.local_speaker_id)}
                  className="size-5 shrink-0 accent-accent-solid"
                />
                <span className="flex min-w-0 flex-col">
                  <b className="truncate">{member.clip_name}</b>
                  <span className="text-mute">{member.label}</span>
                </span>
              </label>
              <div className="flex flex-wrap items-center gap-2">
                <ExcerptPlayer
                  localSpeakerId={member.local_speaker_id}
                  label={member.label}
                  clip={member.clip_name}
                />
                <LevelBadge level="unknown" />
                <span className="text-mute">
                  {t("people.group.speech", {
                    seconds: member.speaking_seconds.toFixed(1),
                  })}
                </span>
                {out && <b>{t("people.group.out")}</b>}
              </div>
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
          className={inputClass}
        />
        {available.map((person) => (
          <label key={person.id} className={optionClass}>
            <input
              type="radio"
              name={radios}
              checked={picked !== NEW && picked?.id === person.id}
              onChange={() =>
                setPicked({ id: person.id, name: person.display_name })
              }
              className="size-4 accent-accent-solid"
            />
            <span className="flex-1">{person.display_name}</span>
          </label>
        ))}
        {persons.state.status === "ready" && available.length === 0 && (
          <p className="text-mute">{t("people.group.noMatch")}</p>
        )}
        <label className={optionClass}>
          <input
            type="radio"
            name={radios}
            checked={picked === NEW}
            onChange={() => setPicked(NEW)}
            className="size-4 accent-accent-solid"
          />
          <span className="flex-1">{t("people.group.someoneNew")}</span>
        </label>
        {picked === NEW && (
          <input
            type="text"
            value={newName}
            maxLength={100}
            onChange={(event) => setNewName(event.target.value)}
            aria-label={t("people.group.newName")}
            autoComplete="off"
            autoFocus
            className={inputClass}
          />
        )}
      </fieldset>

      {conflict && (
        <div
          role="alert"
          className="flex flex-col items-start gap-2 rounded-xl bg-bad-bg p-3 text-bad-fg"
        >
          <p className="m-0">{t("people.group.conflict")}</p>
          <button
            type="button"
            onClick={() =>
              onReload(
                kept[0]?.local_speaker_id ?? group.members[0].local_speaker_id,
              )
            }
            className={buttonClass("primary")}
          >
            {t("people.group.reload")}
          </button>
        </div>
      )}
      {failure && !conflict && <ErrorAlert>{failureText(failure)}</ErrorAlert>}

      <div
        role="region"
        aria-label={t("people.group.bar")}
        className="sticky bottom-0 z-10 -mx-4 flex flex-wrap items-center justify-between gap-3 border-t border-line bg-surface px-4 py-3 md:mx-0 md:rounded-xl md:border"
      >
        <p role="status" className="m-0 font-medium">
          {t("people.group.selected", {
            kept: kept.length,
            total: group.members.length,
          })}
        </p>
        <div className="flex flex-wrap items-center gap-2">
          {(["random", "ignored"] as const).map((disposition) => (
            <button
              key={disposition}
              type="button"
              disabled={busy || pending !== undefined || conflict}
              onClick={() => void setAside(disposition)}
              className={buttonClass()}
            >
              {t(
                disposition === "random"
                  ? "people.group.random"
                  : "people.group.ignore",
              )}
            </button>
          ))}
          <button
            type="button"
            disabled={!ready || busy || pending !== undefined || conflict}
            onClick={() => void confirm()}
            className={buttonClass("primary")}
          >
            {ready
              ? t("people.group.name", { count: kept.length, person: who })
              : t("people.group.nameNone")}
          </button>
        </div>
      </div>
    </>
  );
}
