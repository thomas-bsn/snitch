import { useState, type FormEvent } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

type Source = components["schemas"]["Source"];
async function load() {
  try {
    const r = await api.GET("/api/admin/sources");
    return r.data
      ? { ok: true as const, value: r.data }
      : { ok: false as const, code: errorCode(r.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}
export function FoldersTab() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState(false);
  const [busy, setBusy] = useState(false);
  async function add(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = new FormData(form);
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/admin/sources", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          name: String(data.get("name")),
          path: String(data.get("path")),
        },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else {
        form.reset();
        setNotice(true);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function action(source: Source, kind: "scan" | "disable") {
    setBusy(true);
    setFailure(undefined);
    try {
      const params = {
        path: { id: source.id },
        header: mutation(session?.csrf_token),
      };
      const r =
        kind === "scan"
          ? await api.POST("/api/admin/sources/{id}/scan", { params, body: {} })
          : await api.DELETE("/api/admin/sources/{id}", { params });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else {
        setNotice(true);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex max-w-4xl flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("folders.title")}</h2>
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <StatusNotice>{t("folders.saved")}</StatusNotice>}
      {state.status === "ready" && (
        <>
          {state.value.allowed_roots.length === 0 ? (
            <p>{t("folders.noMount")}</p>
          ) : (
            <>
              <p>
                {t("folders.roots", {
                  roots: state.value.allowed_roots.join(", "),
                })}
              </p>
              <form onSubmit={add} className="flex max-w-md flex-col gap-3">
                <Field
                  label={t("folders.name")}
                  name="name"
                  required
                  maxLength={80}
                />
                <Field
                  label={t("folders.path")}
                  name="path"
                  required
                  maxLength={4096}
                />
                <div>
                  <button disabled={busy} className={buttonClass("primary")}>
                    {t("folders.add")}
                  </button>
                </div>
              </form>
            </>
          )}
          {state.value.items.length === 0 ? (
            <p>{t("folders.empty")}</p>
          ) : (
            <ul className="flex list-none flex-col gap-3 p-0">
              {state.value.items.map((source) => (
                <li
                  key={source.id}
                  className="flex flex-col gap-3 rounded-xl border border-line p-4"
                >
                  <strong>{source.name}</strong>
                  <p>
                    {source.path} ·{" "}
                    {source.enabled
                      ? t(`folders.health.${source.health}`)
                      : t("folders.disabled")}
                  </p>
                  {source.health_message && (
                    <p role="status">{source.health_message}</p>
                  )}
                  <p>
                    {t("folders.lastScan", {
                      date: source.last_scan_at
                        ? new Date(source.last_scan_at).toLocaleString()
                        : t("folders.never"),
                    })}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      disabled={busy || !source.enabled}
                      onClick={() => void action(source, "scan")}
                      className={buttonClass()}
                    >
                      {t("folders.scan")}
                    </button>
                    {source.enabled && (
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => void action(source, "disable")}
                        className={buttonClass()}
                      >
                        {t("folders.disable")}
                      </button>
                    )}
                  </div>
                  <SourceRules id={source.id} />
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </div>
  );
}
function SourceRules({ id }: { id: number }) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const [failure, setFailure] = useState<string>();
  const [invalidSelection, setInvalidSelection] = useState(false);
  const [busy, setBusy] = useState(false);
  const [correctedSkipped, setCorrectedSkipped] = useState(0);
  const [notice, setNotice] = useState(false);
  const [rule, setRule] =
    useState<components["schemas"]["MicrophoneSourceRule"]>();
  const [people, setPeople] = useState<components["schemas"]["Person"][]>([]);
  const [track, setTrack] = useState("");
  const [person, setPerson] = useState("");
  const [newPerson, setNewPerson] = useState("");
  const [criterion, setCriterion] = useState<"index" | "title" | "both">(
    "index",
  );
  async function loadMicrophone() {
    try {
      const [a, b] = await Promise.all([
        api.GET("/api/admin/sources/{id}/microphone-track", {
          params: { path: { id } },
        }),
        api.GET("/api/persons"),
      ]);
      if (a.data && b.data) {
        setRule(a.data);
        setPeople(b.data.items);
      } else setFailure(errorCode(a.error ?? b.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    }
  }
  async function audio(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const num = (key: string) => {
      const v = String(f.get(key) ?? "").trim();
      return v ? Number(v) : undefined;
    };
    try {
      const r = await api.PUT("/api/admin/sources/{id}/audio-track-rule", {
        params: { path: { id }, header: mutation(session?.csrf_token) },
        body: {
          title: String(f.get("title") ?? "").trim() || undefined,
          language: String(f.get("language") ?? "").trim() || undefined,
          channel_layout: String(f.get("layout") ?? "").trim() || undefined,
          channels: num("channels"),
          fallback_audio_stream_index: num("fallback"),
        },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else if (r.data) {
        setCorrectedSkipped(r.data.corrected_clips_skipped);
        setNotice(true);
      }
    } catch {
      setFailure("unavailable");
    }
  }
  async function microphone(e: FormEvent) {
    e.preventDefault();
    if (busy) return;
    setInvalidSelection(false);
    const selected = rule?.detected_tracks.find(
      (item) => String(item.audio_stream_index) === track,
    );
    if (!selected || (criterion !== "index" && !selected.title)) {
      setInvalidSelection(true);
      return;
    }
    setBusy(true);
    try {
      let personId = Number(person);
      if (newPerson.trim()) {
        const created = await api.POST("/api/persons", {
          params: { header: mutation(session?.csrf_token) },
          body: { display_name: newPerson.trim() },
        });
        if (!created.data) {
          setFailure(errorCode(created.error) ?? "unknown");
          return;
        }
        personId = created.data.id;
        setPerson(String(personId));
        setPeople((old) => [...old, created.data]);
        setNewPerson("");
      }
      if (!personId) {
        setInvalidSelection(true);
        return;
      }
      const r = await api.PUT("/api/admin/sources/{id}/microphone-track", {
        params: { path: { id }, header: mutation(session?.csrf_token) },
        body: {
          ...(criterion !== "title" && {
            audio_stream_index: selected.audio_stream_index,
          }),
          ...(criterion !== "index" && { title: selected.title }),
          person_id: personId,
        },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else if (r.data) {
        setRule(r.data);
        setNotice(true);
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function remove() {
    try {
      const r = await api.DELETE("/api/admin/sources/{id}/microphone-track", {
        params: { path: { id }, header: mutation(session?.csrf_token) },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else if (r.data) {
        setRule(r.data);
        setNotice(true);
      }
    } catch {
      setFailure("unavailable");
    }
  }
  return (
    <>
      <details>
        <summary className="cursor-pointer">{t("folders.audio")}</summary>
        <form onSubmit={audio} className="flex max-w-md flex-col gap-3 py-3">
          <Field name="title" label={t("folders.trackTitle")} />
          <Field name="language" label={t("folders.language")} />
          <Field name="layout" label={t("folders.layout")} />
          <Field
            name="channels"
            label={t("folders.channels")}
            type="number"
            min={1}
          />
          <Field
            name="fallback"
            label={t("folders.fallback")}
            type="number"
            min={0}
          />
          <div>
            <button className={buttonClass()}>{t("folders.saveRule")}</button>
          </div>
        </form>
      </details>
      <details
        onToggle={(e) => {
          if (e.currentTarget.open && !rule) void loadMicrophone();
        }}
      >
        <summary className="cursor-pointer">{t("folders.microphone")}</summary>
        {rule && (
          <div className="flex flex-col gap-3 py-3">
            <p>
              {rule.rule
                ? t("folders.designated", {
                    person: rule.rule.person.display_name,
                  })
                : t("folders.noMicrophone")}
            </p>
            {rule.detected_tracks.length === 0 ? (
              <p>{t("folders.noTracks")}</p>
            ) : (
              <form
                onSubmit={microphone}
                className="flex max-w-md flex-col gap-3"
              >
                <label>
                  {t("folders.track")}
                  <select
                    required
                    value={track}
                    onChange={(e) => setTrack(e.target.value)}
                    className="block w-full rounded-lg border border-line-strong bg-surface p-2"
                  >
                    <option value="">{t("folders.choose")}</option>
                    {rule.detected_tracks.map((x, i) => (
                      <option key={i} value={x.audio_stream_index}>
                        {x.audio_stream_index} · {x.title} · {x.language}
                      </option>
                    ))}
                  </select>
                </label>
                <fieldset className="flex flex-col gap-2">
                  <legend>{t("folders.matchBy")}</legend>
                  {(["index", "title", "both"] as const).map((value) => (
                    <label
                      key={value}
                      className="flex min-h-11 items-center gap-2"
                    >
                      <input
                        type="radio"
                        name={`criterion-${id}`}
                        checked={criterion === value}
                        onChange={() => setCriterion(value)}
                      />
                      {t(`folders.match.${value}`)}
                    </label>
                  ))}
                </fieldset>
                <label>
                  {t("folders.person")}
                  <select
                    value={person}
                    onChange={(e) => {
                      setPerson(e.target.value);
                      setNewPerson("");
                    }}
                    className="block w-full rounded-lg border border-line-strong bg-surface p-2"
                  >
                    <option value="">{t("folders.choose")}</option>
                    {people.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.display_name}
                      </option>
                    ))}
                  </select>
                </label>
                <Field
                  label={t("folders.newPerson")}
                  value={newPerson}
                  maxLength={120}
                  onChange={(e) => {
                    setNewPerson(e.target.value);
                    if (e.target.value) setPerson("");
                  }}
                />
                <div>
                  <button disabled={busy} className={buttonClass()}>
                    {t("folders.saveRule")}
                  </button>
                </div>
              </form>
            )}
            {rule.rule && (
              <div>
                <button
                  type="button"
                  onClick={() => void remove()}
                  className={buttonClass()}
                >
                  {t("folders.remove")}
                </button>
              </div>
            )}
          </div>
        )}
      </details>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {invalidSelection && (
        <ErrorAlert>{t("folders.invalidSelection")}</ErrorAlert>
      )}
      {correctedSkipped > 0 && (
        <StatusNotice>
          {t("folders.correctedSkipped", { count: correctedSkipped })}
        </StatusNotice>
      )}
      {notice && <StatusNotice>{t("folders.saved")}</StatusNotice>}
    </>
  );
}
