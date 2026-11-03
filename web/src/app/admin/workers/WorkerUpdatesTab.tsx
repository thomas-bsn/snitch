import { useEffect, useState, type FormEvent } from "react";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { useDocumentTitle } from "../../components/useDocumentTitle";
import { useLoaded } from "../../api/loaded";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";
import { useFormatDate } from "../format";
import { fetchWorkerUpdates, useWorkerUpdateActions } from "./workerUpdatesApi";

const VERSION = /^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;

/** Text key of a Worker's last reported state. */
function stateKey(state: string): MessageKey {
  switch (state) {
    case "idle":
      return "workerUpdates.states.idle";
    case "pulling":
      return "workerUpdates.states.pulling";
    case "restarting":
      return "workerUpdates.states.restarting";
    case "updated":
      return "workerUpdates.states.updated";
    case "rolled_back":
      return "workerUpdates.states.rolled_back";
    case "failed":
      return "workerUpdates.states.failed";
    default:
      return "workerUpdates.states.none";
  }
}

/** Text key of a stable cause code, when the screen knows it. */
function causeKey(cause: string): MessageKey | undefined {
  switch (cause) {
    case "image_pull_failed":
      return "workerUpdates.causes.image_pull_failed";
    case "digest_unknown":
      return "workerUpdates.causes.digest_unknown";
    case "target_mismatch":
      return "workerUpdates.causes.target_mismatch";
    case "unhealthy":
      return "workerUpdates.causes.unhealthy";
    case "rollback_failed":
      return "workerUpdates.causes.rollback_failed";
    default:
      return undefined;
  }
}

/** The Updates tab: choose the version every Worker runs, and see what each one reported. */
export function WorkerUpdatesTab() {
  const { t, error } = useI18n();
  const formatDate = useFormatDate();
  const { state, reload } = useLoaded(fetchWorkerUpdates);
  const actions = useWorkerUpdateActions();
  useDocumentTitle(t("workerUpdates.title"));

  const [choice, setChoice] = useState("");
  const [manual, setManual] = useState("");
  const [failure, setFailure] = useState<string>();
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (state.status === "ready") {
      setChoice(state.value.target_version ?? "");
      setManual(state.value.target_version ?? "");
    }
  }, [state]);

  async function save(event: FormEvent) {
    event.preventDefault();
    if (state.status !== "ready") return;
    const useManual =
      !state.value.catalog_available || state.value.catalog.length === 0;
    const version = (useManual ? manual : choice).trim();
    setFailure(undefined);
    setSaved(false);
    if (version !== "" && !VERSION.test(version)) {
      setFailure("invalid_worker_version");
      return;
    }
    setBusy(true);
    const result = await actions.setTarget(version === "" ? null : version);
    setBusy(false);
    if (!result.ok) {
      setFailure(result.code);
      return;
    }
    setSaved(true);
    reload();
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="max-w-xl text-mute">{t("workerUpdates.lead")}</p>
      {state.status === "loading" && (
        <p role="status">{t("workers.loading")}</p>
      )}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {state.status === "ready" && (
        <>
          <form
            onSubmit={save}
            className="flex max-w-xl flex-col gap-3 rounded-xl border border-line p-4"
          >
            <label className="flex flex-col gap-1.5">
              <span className="font-medium">
                {t("workerUpdates.targetLabel")}
              </span>
              {state.value.catalog_available &&
              state.value.catalog.length > 0 ? (
                <select
                  value={choice}
                  onChange={(e) => setChoice(e.target.value)}
                  className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
                >
                  <option value="">{t("workerUpdates.none")}</option>
                  {state.value.catalog.map((version) => (
                    <option key={version} value={version}>
                      {version}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  value={manual}
                  onChange={(e) => setManual(e.target.value)}
                  placeholder="v1.2.3"
                  aria-label={t("workerUpdates.manualLabel")}
                  className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
                />
              )}
            </label>
            {!state.value.catalog_available && (
              <p className="text-xs text-mute">
                {t("workerUpdates.catalogUnavailable")}
              </p>
            )}
            {state.value.catalog_available &&
              state.value.catalog.length === 0 && (
                <p className="text-xs text-mute">
                  {t("workerUpdates.noVersions")}
                </p>
              )}
            <p className="text-xs text-mute">
              {t("workerUpdates.compatibility")}
            </p>
            <p className="text-xs text-mute">
              {t("workerUpdates.imageLabel")} :{" "}
              <code>{state.value.target_image ?? "—"}</code>
            </p>
            <div>
              <button
                type="submit"
                disabled={busy}
                className={buttonClass("primary")}
              >
                {t("workerUpdates.save")}
              </button>
            </div>
            {saved && <StatusNotice>{t("workerUpdates.saved")}</StatusNotice>}
            {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          </form>

          {state.value.workers.length === 0 ? (
            <p className="rounded-lg border border-dashed border-line-strong p-6 text-mute">
              {t("workerUpdates.empty")}
            </p>
          ) : (
            <div
              role="region"
              tabIndex={0}
              aria-label={t("workerUpdates.title")}
              className="overflow-x-auto rounded-xl border border-line"
            >
              <table className="w-full min-w-[34rem] border-collapse text-left">
                <thead className="bg-side text-xs text-mute">
                  <tr>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t("workerUpdates.columns.name")}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t("workerUpdates.columns.state")}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t("workerUpdates.columns.version")}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t("workerUpdates.columns.reported")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {state.value.workers.map((worker) => {
                    const cause = causeKey(worker.cause_code);
                    return (
                      <tr key={worker.id} className="border-t border-line">
                        <td className="px-3 py-2">{worker.name}</td>
                        <td className="px-3 py-2">
                          {t(stateKey(worker.state))}
                          {cause && (
                            <span className="block text-xs text-mute">
                              {t(cause)}
                            </span>
                          )}
                        </td>
                        <td className="px-3 py-2">{worker.version || "—"}</td>
                        <td className="px-3 py-2 text-mute">
                          {worker.reported_at
                            ? formatDate(worker.reported_at)
                            : "—"}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}
