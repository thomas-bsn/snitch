import { useEffect, useState } from "react";
import type { Result } from "../api/loaded";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { formatSize, formatTimecode } from "./clipFormat";
import {
  fetchExplorerProperties,
  type ExplorerEntry,
  type ExplorerProperties,
} from "./explorerApi";

function formatDate(value: string, language: string) {
  return new Intl.DateTimeFormat(language, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

/** Size, duration, owners and dates of a folder or a video. */
export function ExplorerPropertiesDialog({
  entry,
  open,
  onOpenChange,
}: {
  entry: ExplorerEntry;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t, error, language } = useI18n();
  const [state, setState] = useState<
    | { status: "loading" }
    | { status: "ready"; result: Result<ExplorerProperties> }
  >({ status: "loading" });

  useEffect(() => {
    if (!open) return;
    let live = true;
    setState({ status: "loading" });
    void fetchExplorerProperties(entry.path).then((result) => {
      if (live) setState({ status: "ready", result });
    });
    return () => {
      live = false;
    };
  }, [open, entry.path]);

  const properties =
    state.status === "ready" && state.result.ok ? state.result.value : null;

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title={t("library.explorer.propertiesTitle", { name: entry.name })}
      closeLabel={t("library.explorer.propertiesClose")}
    >
      {state.status === "loading" && (
        <p className="text-mute" role="status">
          {t("library.loading")}
        </p>
      )}
      {state.status === "ready" && !state.result.ok && (
        <p className="text-bad-fg" role="alert">
          {error(state.result.code)}
        </p>
      )}
      {properties && (
        <>
          <p className="m-0 break-all text-sm text-mute">{properties.path}</p>
          <dl className="m-0 grid grid-cols-[minmax(8rem,auto)_1fr] gap-x-4 gap-y-2 text-sm">
            <dt className="text-mute">
              {t("library.explorer.propertiesType")}
            </dt>
            <dd className="m-0 font-medium">
              {properties.directory
                ? t("library.explorer.propertiesFolder")
                : t("library.explorer.propertiesVideo")}
            </dd>

            <dt className="text-mute">
              {t("library.explorer.propertiesSize")}
            </dt>
            <dd className="m-0 tabular-nums">
              {formatSize(properties.size_bytes, language)}
            </dd>

            {properties.directory ? (
              <>
                <dt className="text-mute">
                  {t("library.explorer.propertiesVideos")}
                </dt>
                <dd className="m-0 tabular-nums">{properties.video_count}</dd>
                <dt className="text-mute">
                  {t("library.explorer.propertiesIndexed")}
                </dt>
                <dd className="m-0 tabular-nums">{properties.indexed_count}</dd>
                <dt className="text-mute">
                  {t("library.explorer.propertiesFolders")}
                </dt>
                <dd className="m-0 tabular-nums">{properties.folder_count}</dd>
              </>
            ) : (
              <>
                <dt className="text-mute">
                  {t("library.explorer.propertiesState")}
                </dt>
                <dd className="m-0">
                  {properties.state
                    ? t(`library.explorer.state.${properties.state}`)
                    : "—"}
                </dd>
                {properties.game && (
                  <>
                    <dt className="text-mute">
                      {t("library.explorer.propertiesGame")}
                    </dt>
                    <dd className="m-0">{properties.game}</dd>
                  </>
                )}
              </>
            )}

            <dt className="text-mute">
              {properties.directory
                ? t("library.explorer.propertiesDuration")
                : t("library.explorer.propertiesDuration")}
            </dt>
            <dd className="m-0 tabular-nums">
              {properties.duration_seconds != null
                ? formatTimecode(properties.duration_seconds)
                : "—"}
            </dd>

            <dt className="text-mute">
              {t("library.explorer.propertiesAdded")}
            </dt>
            <dd className="m-0">
              {properties.added_at
                ? formatDate(properties.added_at, language)
                : "—"}
            </dd>

            {!properties.directory && (
              <>
                <dt className="text-mute">
                  {t("library.explorer.propertiesRecorded")}
                </dt>
                <dd className="m-0">
                  {properties.recorded_at
                    ? formatDate(properties.recorded_at, language)
                    : "—"}
                </dd>
              </>
            )}

            <dt className="text-mute">
              {properties.directory
                ? t("library.explorer.propertiesOwners")
                : t("library.explorer.propertiesOwner")}
            </dt>
            <dd className="m-0">
              {properties.owners.length === 0 ? (
                "—"
              ) : (
                <ul className="m-0 list-none p-0">
                  {properties.owners.map((owner) => (
                    <li key={owner.id}>
                      {properties.directory
                        ? t("library.explorer.propertiesOwnerVideos", {
                            name: owner.name,
                            count: owner.videos,
                          })
                        : owner.name}
                    </li>
                  ))}
                </ul>
              )}
            </dd>
          </dl>

          {properties.video_count === 0 && properties.directory && (
            <p className="m-0 text-sm text-mute">
              {t("library.explorer.propertiesNoVideo")}
            </p>
          )}
          {!properties.directory && properties.indexed_count === 0 && (
            <p className="m-0 text-sm text-mute">
              {t("library.explorer.propertiesNotIndexed")}
            </p>
          )}
        </>
      )}
    </Modal>
  );
}
