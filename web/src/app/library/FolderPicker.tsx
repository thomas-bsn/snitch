import { useCallback, useEffect, useState } from "react";
import { ArrowLeft, Folder } from "lucide-react";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { useSession } from "../session";

type Entry = { path: string; directory: boolean; size_bytes: number };

/**
 * Picks an existing subfolder of the managed video root as the upload
 * destination. Browsing only reads the root: nothing is scanned.
 */
export function FolderPicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (path: string) => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const csrf = session?.csrf_token ?? "";
  const [open, setOpen] = useState(false);
  const [directory, setDirectory] = useState("");
  const [entries, setEntries] = useState<Entry[]>([]);
  const [failure, setFailure] = useState("");

  const load = useCallback(
    async (path: string) => {
      try {
        const response = await fetch(
          `/api/admin/video-root/browse?path=${encodeURIComponent(path)}`,
          { headers: { "X-CSRF-Token": csrf } },
        );
        const data = await response.json();
        if (!response.ok) throw new Error(data.error ?? "unavailable");
        setEntries(data.items.filter((entry: Entry) => entry.directory));
        setFailure("");
      } catch (e) {
        setFailure(e instanceof Error ? e.message : "unavailable");
      }
    },
    [csrf],
  );

  useEffect(() => {
    if (open) void load(directory);
  }, [open, directory, load]);

  return (
    <div className="flex max-w-xl flex-col gap-2">
      <span className="font-medium">{t("import.destination.label")}</span>
      <div className="flex flex-wrap items-center gap-2">
        <span className="break-all text-sm text-mute">
          {value ? `/${value}` : t("import.destination.root")}
        </span>
        <button
          type="button"
          className={buttonClass("secondary")}
          aria-expanded={open}
          onClick={() => setOpen(!open)}
        >
          {t("import.destination.choose")}
        </button>
      </div>
      {open && (
        <div className="flex flex-col gap-2 rounded-lg border border-line p-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="break-all text-sm">/{directory}</span>
            {directory && (
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() =>
                  setDirectory(directory.split("/").slice(0, -1).join("/"))
                }
              >
                <ArrowLeft size={16} />
                {t("import.destination.up")}
              </button>
            )}
            <button
              type="button"
              className={buttonClass("primary")}
              onClick={() => {
                onChange(directory);
                setOpen(false);
              }}
            >
              {t("import.destination.use")}
            </button>
          </div>
          <ul className="divide-y divide-line">
            {entries.map((entry) => (
              <li key={entry.path} className="flex min-h-11 items-center gap-2">
                <Folder size={18} />
                <button
                  type="button"
                  className="min-h-11 break-all text-left text-accent-ink"
                  onClick={() => setDirectory(entry.path)}
                >
                  {entry.path.split("/").pop()}
                </button>
              </li>
            ))}
            {!entries.length && (
              <li className="py-2 text-sm text-mute">
                {t("import.destination.empty")}
              </li>
            )}
          </ul>
          {failure && (
            <p role="alert" className="text-sm text-danger">
              {error(failure)}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
