import { ArrowLeft } from "lucide-react";
import { useCallback, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { useLoaded } from "../api/loaded";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { StatusPill } from "../components/StatusPill";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { MergeDialog } from "./PersonDialogs";
import { Avatar, ReferenceBadge, speechText } from "./PersonParts";
import { PersonVoices } from "./PersonVoices";
import {
  fetchPerson,
  fetchPersons,
  referenceLevel,
  usePersonActions,
  type Person,
} from "./peopleApi";

const smallButton = `${buttonClass("secondary")} md:h-8 px-3`;

/** The full page of one Person: linked account, figures, its videos, and its corrections. */
export function PersonPage() {
  const { t } = useI18n();
  const { id: rawId } = useParams();
  const id = Number(rawId);
  const navigate = useNavigate();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const load = useCallback(() => fetchPerson(id), [id]);
  const { state, reload } = useLoaded(load);
  const candidates = useLoaded(fetchPersons);
  const [merging, setMerging] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string>();
  const [failure, setFailure] = useState<string>();

  if (state.status === "loading") {
    return <p className="text-mute">{t("people.page.loading")}</p>;
  }
  if (state.status === "failed") {
    return <ErrorAlert>{failureText(state.code)}</ErrorAlert>;
  }

  const person: Person = state.value;
  const locked = person.user_id !== null;
  const level = referenceLevel(person.validated_sample_count);

  async function remove() {
    setBusy(true);
    const result = await actions.remove(person.id);
    setBusy(false);
    setDeleting(false);
    if (!result.ok) {
      setNotice(undefined);
      setFailure(result.code);
      return;
    }
    navigate("/people", {
      state: {
        notice: t("people.delete.done", {
          name: person.display_name,
          summary: t("people.summary", {
            attributions: result.value.attributions,
            samples: result.value.samples,
            rules: result.value.microphone_rules,
          }),
        }),
      },
    });
  }

  return (
    <div className="flex flex-col gap-6">
      <Link
        to="/people"
        className="inline-flex w-fit items-center gap-1.5 text-mute hover:text-ink"
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
        {t("people.page.back")}
      </Link>

      <header className="flex flex-wrap items-center gap-4">
        <Avatar name={person.display_name} size="lg" />
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <h2 className="m-0 truncate text-[22px] font-semibold">
            {person.display_name}
          </h2>
          <div className="flex flex-wrap items-center gap-2">
            {locked ? (
              <StatusPill tone="ok">{t("people.persons.linked")}</StatusPill>
            ) : (
              <span className="text-mute">
                {t("people.persons.account")}: {t("people.persons.none")}
              </span>
            )}
            {person.voice_profile_deleted_at !== null && (
              <StatusPill tone="idle">
                {t("people.persons.profileDeleted")}
              </StatusPill>
            )}
          </div>
        </div>
        <div
          role="group"
          aria-label={t("people.page.actions", { name: person.display_name })}
          className="flex flex-wrap gap-2"
        >
          <button
            type="button"
            disabled={locked || person.voice_profile_deleted_at !== null}
            onClick={() => setMerging(true)}
            className={smallButton}
          >
            {t("people.persons.merge", { name: person.display_name })}
          </button>
          <button
            type="button"
            disabled={locked}
            onClick={() => setDeleting(true)}
            className={smallButton}
          >
            {t("people.persons.delete", { name: person.display_name })}
          </button>
        </div>
      </header>

      {notice && <StatusNotice>{notice}</StatusNotice>}
      {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}

      <section aria-labelledby="person-figures" className="flex flex-col gap-3">
        <h3 id="person-figures" className="m-0 text-base font-semibold">
          {t("people.page.figuresTitle", { name: person.display_name })}
        </h3>
        <dl className="m-0 grid grid-cols-2 gap-3 md:grid-cols-4">
          <Figure
            label={t("people.figures.videos")}
            value={String(person.clip_count)}
          />
          <Figure
            label={t("people.figures.speech")}
            value={speechText(person.speaking_seconds, t)}
          />
          <Figure
            label={t("people.figures.samples")}
            value={String(person.validated_sample_count)}
          />
          <div className="rounded-lg border border-line bg-surface p-3">
            <dt className="text-xs text-mute">
              {t("people.figures.reference")}
            </dt>
            <dd className="m-0 mt-1">
              <ReferenceBadge samples={person.validated_sample_count} />
            </dd>
          </div>
        </dl>
        {level !== "good" && (
          <p
            role="note"
            className="m-0 rounded-lg bg-warn-bg px-3 py-2 text-sm text-warn-fg"
          >
            {t("people.reference.warning", {
              name: person.display_name,
              level: t(`people.reference.${level}`),
            })}
          </p>
        )}
      </section>

      <section aria-labelledby="person-videos" className="flex flex-col gap-3">
        <h3 id="person-videos" className="m-0 text-base font-semibold">
          {t("people.page.videosTitle", { name: person.display_name })}
        </h3>
        <p className="m-0 text-mute">
          {t("people.page.splitLead", { name: person.display_name })}
        </p>
        <PersonVoices
          person={person}
          onSeparated={(message) => {
            setFailure(undefined);
            setNotice(message);
            reload();
          }}
        />
      </section>

      <MergeDialog
        person={merging ? person : undefined}
        candidates={
          candidates.state.status === "ready" ? candidates.state.value : []
        }
        onClose={() => setMerging(false)}
        onMerged={(source, target, summary) => {
          setMerging(false);
          navigate(`/people`, {
            state: {
              notice: t("people.merge.done", {
                name: source.display_name,
                target: target.display_name,
                summary,
              }),
            },
          });
        }}
      />
      <ConfirmDialog
        open={deleting}
        onOpenChange={(open) => !open && setDeleting(false)}
        title={t("people.delete.title", { name: person.display_name })}
        cancelLabel={t("people.delete.cancel")}
        confirmLabel={t("people.delete.submit")}
        danger
        busy={busy}
        onConfirm={() => void remove()}
      >
        <p>{t("people.delete.body", { name: person.display_name })}</p>
      </ConfirmDialog>
    </div>
  );
}

function Figure({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-line bg-surface p-3">
      <dt className="text-xs text-mute">{label}</dt>
      <dd className="m-0 mt-1 text-base font-semibold">{value}</dd>
    </div>
  );
}
