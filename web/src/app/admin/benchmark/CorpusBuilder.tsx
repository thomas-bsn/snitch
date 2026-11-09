import { useMemo, useState } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { loadClips, type Clip } from "./api";
import { elapsed } from "./format";

type Role = "test" | "reference" | "none";
type Roles = Record<number, Role>;
type ImportedFile = { name: string; bytes: number; text: string };
type Imports = Record<number, { stm?: ImportedFile; rttm?: ImportedFile }>;

/** The imported texts of a corpus are bounded as on the server (ADR 0061). */
const MAX_IMPORT_BYTES = 2 * 1024 * 1024;

const ROLES: Role[] = ["test", "reference", "none"];
const roleKey = {
  test: "benchmark.corpus.roleTest",
  reference: "benchmark.corpus.roleReference",
  none: "benchmark.corpus.roleNone",
} as const;

/** The stages the choice allows, from the clips alone. Indicative: the server decides at creation. */
function measurable(
  clips: Clip[],
  roles: Roles,
  reviewed: Set<number>,
  imports: Imports,
) {
  const byRole = (role: Role) => clips.filter((c) => roles[c.id] === role);
  // Persons are told apart by id: two different Persons may share a name.
  const persons = (list: Clip[]) =>
    new Set(list.flatMap((c) => c.validated_persons.map((p) => p.id)));
  const refPersons = persons(byRole("reference"));
  const testPersons = persons(byRole("test"));
  // An imported reference replaces the production one and certifies the clip by itself.
  const certified = byRole("test").filter(
    (c) => reviewed.has(c.id) && !imports[c.id],
  );
  const imported = byRole("test").flatMap((c) =>
    imports[c.id] ? [imports[c.id]] : [],
  );
  const fingerprint =
    refPersons.size >= 2 &&
    testPersons.size > 0 &&
    [...testPersons].every((p) => refPersons.has(p));
  const transcription =
    certified.some((c) => c.segments_total > 0) || imported.some((i) => i.stm);
  const diarization =
    certified.some((c) => c.validated_persons.length > 0) ||
    imported.some((i) => i.rttm);
  // The bias of the production reference only concerns the clips that still rely on it.
  const production = certified.some(
    (c) => c.segments_total > 0 || c.validated_persons.length > 0,
  );
  return {
    production,
    fingerprint,
    transcription,
    diarization,
    e2e: fingerprint && transcription && diarization,
  };
}

export function CorpusBuilder({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: (id: number) => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(loadClips);
  const [name, setName] = useState("");
  const [roles, setRoles] = useState<Roles>({});
  const [reviewed, setReviewed] = useState<Set<number>>(new Set());
  const [imports, setImports] = useState<Imports>({});
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [importFailure, setImportFailure] = useState<string>();
  const clips = useMemo(
    () => (state.status === "ready" ? state.value.clips : []),
    [state],
  );
  const { production, ...stages } = measurable(clips, roles, reviewed, imports);
  const importedBytes = Object.values(imports).reduce(
    (sum, i) => sum + (i.stm?.bytes ?? 0) + (i.rttm?.bytes ?? 0),
    0,
  );
  const tooBig = importedBytes > MAX_IMPORT_BYTES;
  const tests = clips.filter((c) => roles[c.id] === "test");
  const usable = Object.values(stages).some(Boolean);
  const canCreate =
    name.trim() !== "" && tests.length > 0 && usable && !tooBig && !busy;

  const setRole = (id: number, role: Role) => {
    setRoles((current) => ({ ...current, [id]: role }));
    // A clip is certified as a test clip only.
    if (role !== "test") {
      setReviewed((current) => {
        const next = new Set(current);
        next.delete(id);
        return next;
      });
      setImports((current) => {
        const { [id]: _dropped, ...rest } = current;
        return rest;
      });
    }
  };

  const importFile = async (
    id: number,
    kind: "stm" | "rttm",
    file: File | undefined,
  ) => {
    if (!file) return;
    try {
      const text = await file.text();
      setImportFailure(undefined);
      setImports((current) => ({
        ...current,
        [id]: {
          ...current[id],
          [kind]: { name: file.name, bytes: file.size, text },
        },
      }));
    } catch {
      setImportFailure("benchmark.corpus.importUnreadable");
    }
  };

  const removeImport = (id: number, kind: "stm" | "rttm") =>
    setImports((current) => {
      const kept: Imports[number] = { ...current[id] };
      delete kept[kind];
      const { [id]: _clip, ...others } = current;
      return kept.stm || kept.rttm ? { ...others, [id]: kept } : others;
    });

  const create = async () => {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.POST("/api/admin/benchmarks/datasets", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          name: name.trim(),
          test_clip_ids: tests.map((c) => c.id),
          reference_clip_ids: clips
            .filter((c) => roles[c.id] === "reference")
            .map((c) => c.id),
          verified_clip_ids: tests
            .filter((c) => reviewed.has(c.id))
            .map((c) => c.id),
          reference_imports: tests.flatMap((c) =>
            imports[c.id]
              ? [
                  {
                    clip_id: c.id,
                    stm: imports[c.id].stm?.text,
                    rttm: imports[c.id].rttm?.text,
                  },
                ]
              : [],
          ),
        },
      });
      if (result.data) onCreated(result.data.id);
      else setFailure(errorCode(result.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  };

  const stageRows: [keyof typeof stages, string, string][] = [
    [
      "fingerprint",
      "benchmark.run.stageNames.fingerprint",
      "benchmark.corpus.needFingerprint",
    ],
    [
      "transcription",
      "benchmark.run.stageNames.transcription",
      "benchmark.corpus.needTranscription",
    ],
    [
      "diarization",
      "benchmark.run.stageNames.diarization",
      "benchmark.corpus.needDiarization",
    ],
    ["e2e", "benchmark.run.stageNames.e2e", "benchmark.corpus.needE2e"],
  ];

  return (
    <Modal
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={t("benchmark.corpus.title")}
      closeLabel={t("benchmark.corpus.close")}
      variant="wide"
    >
      <div className="flex flex-col gap-4">
        <p className="text-mute">{t("benchmark.corpus.lead")}</p>
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">
            {t("benchmark.corpus.name")}
          </span>
          <input
            value={name}
            maxLength={100}
            onChange={(event) => setName(event.target.value)}
            className="h-11 rounded-lg border border-line-strong bg-surface px-3 md:h-9"
          />
        </label>
        {state.status === "loading" && (
          <p role="status">{t("benchmark.run.loading")}</p>
        )}
        {state.status === "failed" && (
          <div className="flex flex-col gap-2">
            <ErrorAlert>{error(state.code)}</ErrorAlert>
            <button type="button" className={buttonClass()} onClick={reload}>
              {t("benchmark.run.retry")}
            </button>
          </div>
        )}
        {state.status === "ready" && clips.length === 0 && (
          <p className="rounded-lg bg-side px-3 py-6 text-center text-mute">
            {t("benchmark.corpus.empty")}
          </p>
        )}
        {clips.length > 0 && (
          <>
            <p className="text-sm text-mute">
              {t("benchmark.corpus.roleHelp")}
            </p>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[640px] border-collapse text-sm">
                <thead>
                  <tr className="text-left text-xs text-mute uppercase">
                    <th className="px-2 py-2">{t("benchmark.corpus.clip")}</th>
                    <th className="px-2 py-2">
                      {t("benchmark.corpus.duration")}
                    </th>
                    <th className="px-2 py-2">
                      {t("benchmark.corpus.segments")}
                    </th>
                    <th className="px-2 py-2">
                      {t("benchmark.corpus.voices")}
                    </th>
                    <th className="px-2 py-2">{t("benchmark.corpus.role")}</th>
                  </tr>
                </thead>
                <tbody>
                  {clips.map((clip) => {
                    const role = roles[clip.id] ?? "none";
                    return (
                      <tr
                        key={clip.id}
                        className="border-t border-line align-top"
                      >
                        <th
                          scope="row"
                          className="px-2 py-2.5 text-left font-semibold"
                        >
                          {clip.name}
                        </th>
                        <td className="px-2 py-2.5">
                          {elapsed(clip.duration_seconds)}
                        </td>
                        <td className="px-2 py-2.5">
                          {clip.segments_total > 0
                            ? `${clip.segments_corrected} / ${clip.segments_total}`
                            : "–"}
                        </td>
                        <td className="px-2 py-2.5">
                          {clip.validated_persons.length > 0
                            ? clip.validated_persons
                                .map((p) => p.name)
                                .join(", ")
                            : t("benchmark.corpus.none")}
                        </td>
                        <td className="px-2 py-2.5">
                          <div
                            role="group"
                            aria-label={`${t("benchmark.corpus.role")} · ${clip.name}`}
                            className="inline-flex overflow-hidden rounded-lg border border-line-strong"
                          >
                            {ROLES.map((value) => (
                              <button
                                key={value}
                                type="button"
                                aria-pressed={role === value}
                                onClick={() => setRole(clip.id, value)}
                                className={`h-11 border-r border-line px-3 text-xs last:border-r-0 md:h-9 ${
                                  role === value
                                    ? "bg-accent-soft font-bold text-accent-ink"
                                    : "bg-surface"
                                }`}
                              >
                                {t(roleKey[value])}
                              </button>
                            ))}
                          </div>
                          {role === "test" && !imports[clip.id] && (
                            <label className="mt-2 flex items-start gap-2 text-xs">
                              <input
                                type="checkbox"
                                checked={reviewed.has(clip.id)}
                                onChange={(event) =>
                                  setReviewed((current) => {
                                    const next = new Set(current);
                                    if (event.target.checked) next.add(clip.id);
                                    else next.delete(clip.id);
                                    return next;
                                  })
                                }
                                className="mt-0.5 size-4"
                              />
                              <span>
                                <strong>
                                  {t("benchmark.corpus.reviewed")}
                                </strong>
                                <br />
                                <span className="text-mute">
                                  {t("benchmark.corpus.reviewedHelp")}
                                </span>
                              </span>
                            </label>
                          )}
                          {role === "test" && (
                            <ReferenceImport
                              clipName={clip.name}
                              value={imports[clip.id]}
                              onPick={(kind, file) =>
                                void importFile(clip.id, kind, file)
                              }
                              onRemove={(kind) => removeImport(clip.id, kind)}
                            />
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
            <section
              aria-labelledby="corpus-stages"
              className="flex flex-col gap-2"
            >
              <h3 id="corpus-stages" className="font-semibold">
                {t("benchmark.corpus.stagesTitle")}
              </h3>
              <p className="text-xs text-mute">
                {t("benchmark.corpus.stagesNote")}
              </p>
              <ul
                className="m-0 flex list-none flex-col gap-1.5 p-0 text-sm"
                aria-live="polite"
              >
                {stageRows.map(([key, name, need]) => (
                  <li
                    key={key}
                    className="flex flex-wrap items-baseline gap-x-2"
                  >
                    <span
                      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                        stages[key]
                          ? "bg-ok-bg text-ok-fg"
                          : "bg-side text-mute"
                      }`}
                    >
                      <span aria-hidden="true">{stages[key] ? "✓" : "–"}</span>
                      {stages[key]
                        ? t("benchmark.corpus.stageOk")
                        : t("benchmark.corpus.stageNo")}
                    </span>
                    <strong>{t(name as never)}</strong>
                    <span className="text-mute">{t(need as never)}</span>
                  </li>
                ))}
              </ul>
              {tooBig && (
                <ErrorAlert>{t("benchmark.corpus.importTooBig")}</ErrorAlert>
              )}
              {importFailure && (
                <ErrorAlert>{t(importFailure as never)}</ErrorAlert>
              )}
              {production && (stages.transcription || stages.diarization) && (
                <p
                  role="note"
                  className="rounded-lg bg-warn-bg px-3 py-2 text-sm text-warn-fg"
                >
                  {t("benchmark.corpus.bias")}
                </p>
              )}
            </section>
          </>
        )}
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        <div className="flex flex-wrap justify-end gap-2">
          <button type="button" className={buttonClass()} onClick={onClose}>
            {t("benchmark.corpus.cancel")}
          </button>
          <button
            type="button"
            className={buttonClass("primary")}
            disabled={!canCreate}
            onClick={() => void create()}
          >
            {busy
              ? t("benchmark.corpus.creating")
              : t("benchmark.corpus.create")}
          </button>
        </div>
      </div>
    </Modal>
  );
}

/** The "Import a reference" area of a test clip: an STM (text) and an RTTM (speaker turns), read in the browser. */
function ReferenceImport({
  clipName,
  value,
  onPick,
  onRemove,
}: {
  clipName: string;
  value: Imports[number] | undefined;
  onPick: (kind: "stm" | "rttm", file: File | undefined) => void;
  onRemove: (kind: "stm" | "rttm") => void;
}) {
  const { t } = useI18n();
  const kinds = [
    ["stm", "benchmark.corpus.importStm", ".stm,.txt"],
    ["rttm", "benchmark.corpus.importRttm", ".rttm,.txt"],
  ] as const;
  return (
    <fieldset className="mt-2 flex min-w-0 flex-col gap-1.5 rounded-lg border border-line p-2 text-xs">
      <legend className="px-1 font-semibold">
        {t("benchmark.corpus.importTitle")}
        <span className="sr-only"> · {clipName}</span>
      </legend>
      <p className="m-0 text-mute">{t("benchmark.corpus.importHelp")}</p>
      {kinds.map(([kind, label, accept]) => {
        const file = value?.[kind];
        return (
          <div key={kind} className="flex flex-wrap items-center gap-2">
            {file ? (
              <>
                <span className="font-medium">{t(label)}</span>
                <span className="text-mute">
                  {file.name} ·{" "}
                  {t("benchmark.corpus.importSize", {
                    kb: Math.max(1, Math.round(file.bytes / 1024)),
                  })}
                </span>
                <button
                  type="button"
                  className="h-11 rounded-lg border border-line-strong px-2 md:h-8"
                  onClick={() => onRemove(kind)}
                >
                  {t("benchmark.corpus.importRemove", { file: file.name })}
                </button>
              </>
            ) : (
              <label className="flex flex-col gap-0.5">
                <span className="font-medium">{t(label)}</span>
                <input
                  type="file"
                  accept={accept}
                  aria-label={`${t(label)} · ${clipName}`}
                  onChange={(event) => onPick(kind, event.target.files?.[0])}
                />
              </label>
            )}
          </div>
        );
      })}
    </fieldset>
  );
}
