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

const ROLES: Role[] = ["test", "reference", "none"];
const roleKey = {
  test: "benchmark.corpus.roleTest",
  reference: "benchmark.corpus.roleReference",
  none: "benchmark.corpus.roleNone",
} as const;

/** The stages the choice allows, from the clips alone. Indicative: the server decides at creation. */
function measurable(clips: Clip[], roles: Roles, reviewed: Set<number>) {
  const byRole = (role: Role) => clips.filter((c) => roles[c.id] === role);
  const persons = (list: Clip[]) =>
    new Set(list.flatMap((c) => c.validated_persons));
  const refPersons = persons(byRole("reference"));
  const testPersons = persons(byRole("test"));
  const certified = byRole("test").filter((c) => reviewed.has(c.id));
  const fingerprint =
    refPersons.size >= 2 &&
    testPersons.size > 0 &&
    [...testPersons].every((p) => refPersons.has(p));
  const transcription = certified.some((c) => c.segments_total > 0);
  const diarization = certified.some((c) => c.validated_persons.length > 0);
  return {
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
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const clips = useMemo(
    () => (state.status === "ready" ? state.value.clips : []),
    [state],
  );
  const stages = measurable(clips, roles, reviewed);
  const tests = clips.filter((c) => roles[c.id] === "test");
  const usable = Object.values(stages).some(Boolean);
  const canCreate = name.trim() !== "" && tests.length > 0 && usable && !busy;

  const setRole = (id: number, role: Role) => {
    setRoles((current) => ({ ...current, [id]: role }));
    // A clip is certified as a test clip only.
    if (role !== "test")
      setReviewed((current) => {
        const next = new Set(current);
        next.delete(id);
        return next;
      });
  };

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
                            ? clip.validated_persons.join(", ")
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
                          {role === "test" && (
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
              {(stages.transcription || stages.diarization) && (
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
