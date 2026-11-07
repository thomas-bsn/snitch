import { ImageUp, RotateCcw } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import type { Result } from "../api/loaded";
import { buttonClass } from "../components/buttons";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import type { ExplorerEntry } from "./explorerApi";
import { FolderGlyph } from "./FolderGlyph";
import {
  fetchFolderLogoState,
  resetFolderLogo,
  restoreFolderLogo,
  uploadFolderLogo,
  type FolderLogoState,
} from "./folderLogoApi";

const MAX_BYTES = 2 << 20;

/**
 * The logo of a folder (#323): the automatic Steam cover, an uploaded image, and the last four versions.
 * Restoring an older image creates a new version, as Discord does; going back to the automatic cover keeps
 * the history.
 */
export function FolderLogoDialog({
  entry,
  csrf,
  open,
  onOpenChange,
  onChanged,
}: {
  entry: ExplorerEntry;
  csrf: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Called after a change, so the listing and the tree read the new version. */
  onChanged: () => void;
}) {
  const { t, error, language } = useI18n();
  const [state, setState] = useState<FolderLogoState | null>(null);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  const input = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    setLoading(true);
    const result = await fetchFolderLogoState(entry.path);
    setLoading(false);
    if (result.ok) {
      setState(result.value);
      setFailed(false);
    } else {
      setFailed(true);
      setMessage(error(result.code));
    }
  }, [entry.path, error]);

  useEffect(() => {
    if (!open) return;
    setState(null);
    setMessage(null);
    setFailed(false);
    void load();
  }, [open, load]);

  async function run<T>(
    action: () => Promise<Result<T>>,
    success: string,
    apply?: (value: T) => void,
  ) {
    setBusy(true);
    setMessage(null);
    const result = await action();
    setBusy(false);
    if (!result.ok) {
      setMessage(error(result.code));
      return;
    }
    apply?.(result.value);
    setMessage(success);
    onChanged();
  }

  function choose(file: File | undefined) {
    if (input.current) input.current.value = "";
    if (!file) return;
    if (file.size > MAX_BYTES) {
      setMessage(t("library.explorer.logoTooLarge"));
      return;
    }
    if (file.type !== "image/png" && file.type !== "image/jpeg") {
      setMessage(t("library.explorer.logoInvalid"));
      return;
    }
    void run(
      () => uploadFolderLogo(entry.path, file, csrf),
      t("library.explorer.logoChanged"),
      setState,
    );
  }

  const current = state?.custom ? state.versions[0] : undefined;
  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title={t("library.explorer.logoTitle", { name: entry.name })}
      closeLabel={t("library.explorer.propertiesClose")}
    >
      {loading && !state && (
        <p className="text-mute" role="status">
          {t("library.loading")}
        </p>
      )}
      {failed && !state && (
        <p className="text-mute" role="alert">
          {t("library.explorer.logoHistoryFailed")}
        </p>
      )}
      {state && (
        <>
          <div className="flex items-start gap-3">
            <FolderGlyph
              name={entry.name}
              path={current ? entry.path : undefined}
              version={current?.version}
              size={56}
            />
            <div className="min-w-0">
              <p className="m-0 font-medium">
                {current
                  ? t("library.explorer.logoCurrent")
                  : t("library.explorer.logoAutomatic")}
              </p>
              <p className="m-0 text-sm text-mute">
                {current
                  ? t("library.explorer.logoCurrentBody")
                  : t("library.explorer.logoAutomaticBody")}
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <label className={`${buttonClass("secondary")} cursor-pointer`}>
              <ImageUp size={16} aria-hidden="true" />
              {t("library.explorer.logoUpload")}
              <input
                ref={input}
                type="file"
                accept="image/png,image/jpeg"
                className="sr-only"
                disabled={busy}
                onChange={(event) => choose(event.target.files?.[0])}
              />
            </label>
            {state.custom && (
              <button
                type="button"
                className={buttonClass("secondary")}
                disabled={busy}
                onClick={() =>
                  void run(
                    () => resetFolderLogo(entry.path, csrf),
                    t("library.explorer.logoResetDone"),
                    () => {
                      setState((previous) =>
                        previous ? { ...previous, custom: false } : previous,
                      );
                    },
                  )
                }
              >
                <RotateCcw size={16} aria-hidden="true" />
                {t("library.explorer.logoReset")}
              </button>
            )}
          </div>
          <p className="m-0 text-sm text-mute">
            {t("library.explorer.logoUploadHint")}
          </p>

          <section aria-label={t("library.explorer.logoHistory")}>
            <h3 className="m-0 text-sm font-semibold">
              {t("library.explorer.logoHistory")}
            </h3>
            {state.versions.length === 0 ? (
              <p className="mt-1 text-sm text-mute">
                {t("library.explorer.logoEmpty")}
              </p>
            ) : (
              <ul className="m-0 mt-2 grid list-none grid-cols-2 gap-2 p-0 sm:grid-cols-4">
                {state.versions.map((version, index) => {
                  const isCurrent = state.custom && index === 0;
                  const date = new Intl.DateTimeFormat(language, {
                    dateStyle: "medium",
                  }).format(new Date(version.created_at));
                  return (
                    <li key={version.version}>
                      <button
                        type="button"
                        className={`flex w-full flex-col items-center gap-1 rounded-lg border p-2 text-center text-xs disabled:cursor-default ${
                          isCurrent
                            ? "border-accent bg-accent-soft text-accent-ink"
                            : "border-line hover:bg-side"
                        }`}
                        aria-pressed={isCurrent}
                        aria-label={t("library.explorer.logoUseVersion", {
                          date,
                        })}
                        disabled={busy || isCurrent}
                        onClick={() =>
                          void run(
                            () =>
                              restoreFolderLogo(
                                entry.path,
                                version.version,
                                csrf,
                              ),
                            t("library.explorer.logoRestored"),
                            setState,
                          )
                        }
                      >
                        <FolderGlyph
                          name={entry.name}
                          path={entry.path}
                          version={version.version}
                          size={56}
                        />
                        <span>
                          {isCurrent ? t("library.explorer.logoCurrent") : date}
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
            <p className="mt-2 text-sm text-mute">
              {t("library.explorer.logoKept")}
            </p>
          </section>

          {message && (
            <p role="status" className="m-0 text-sm text-ink">
              {message}
            </p>
          )}
        </>
      )}
    </Modal>
  );
}
