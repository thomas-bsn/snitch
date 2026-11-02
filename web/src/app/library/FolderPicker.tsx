import { Popover } from "@base-ui/react/popover";
import { ChevronDown, ChevronRight, Folder } from "lucide-react";
import { useCallback, useEffect, useId, useState } from "react";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { useSession } from "../session";

type Entry = { path: string; directory: boolean; size_bytes: number };

/**
 * Picks the upload destination among the folders of the managed video root, in a compact menu: a filter,
 * the path to the current folder, and a list that scrolls inside the menu. `value` is undefined until a
 * folder was chosen on purpose; the empty string is the video root, chosen explicitly.
 */
export function FolderPicker({
  value,
  onChange,
}: {
  value: string | undefined;
  onChange: (path: string) => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const csrf = session?.csrf_token ?? "";
  const labelId = useId();
  const valueId = useId();
  const [open, setOpen] = useState(false);
  const [directory, setDirectory] = useState("");
  const [entries, setEntries] = useState<Entry[]>([]);
  const [filter, setFilter] = useState("");
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

  function openChange(next: boolean) {
    setOpen(next);
    if (next) {
      setDirectory(value ?? "");
      setFilter("");
    }
  }

  function choose(path: string) {
    onChange(path);
    setOpen(false);
  }

  const parts = directory ? directory.split("/") : [];
  const shown = entries.filter((entry) =>
    (entry.path.split("/").pop() ?? "")
      .toLowerCase()
      .includes(filter.trim().toLowerCase()),
  );
  const label =
    value === undefined
      ? t("import.destination.choose")
      : value === ""
        ? t("import.destination.rootChosen")
        : `/${value}`;

  return (
    <div className="flex max-w-xl flex-col gap-1.5">
      <span className="font-medium" id={labelId}>
        {t("import.destination.label")}
      </span>
      <Popover.Root open={open} onOpenChange={openChange}>
        <Popover.Trigger
          aria-labelledby={`${labelId} ${valueId}`}
          className={`${buttonClass("secondary")} justify-between ${value === undefined ? "text-mute" : ""}`}
        >
          <span id={valueId} className="break-all text-left">
            {label}
          </span>
          <ChevronDown className="size-4 shrink-0" aria-hidden="true" />
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Positioner align="start" sideOffset={6} className="z-50">
            <Popover.Popup
              aria-label={t("import.destination.menu")}
              className="flex max-h-[min(30rem,var(--available-height))] w-[min(28rem,calc(100vw-2rem))] flex-col gap-2 rounded-xl border border-line bg-surface p-3 text-ink shadow-pop outline-none"
            >
              <input
                type="search"
                value={filter}
                onChange={(event) => setFilter(event.target.value)}
                aria-label={t("import.destination.filter")}
                placeholder={t("import.destination.filter")}
                className="h-11 rounded-lg border border-line-strong bg-canvas px-3 md:h-9"
              />
              <nav
                aria-label={t("import.destination.path")}
                className="flex flex-wrap items-center gap-1 text-xs text-mute"
              >
                <button
                  type="button"
                  className="min-h-8 text-accent-ink underline underline-offset-2"
                  onClick={() => setDirectory("")}
                >
                  {t("import.destination.rootName")}
                </button>
                {parts.map((part, index) => (
                  <span key={parts.slice(0, index + 1).join("/")}>
                    <ChevronRight
                      className="inline size-3"
                      aria-hidden="true"
                    />{" "}
                    <button
                      type="button"
                      className="min-h-8 text-accent-ink underline underline-offset-2"
                      onClick={() =>
                        setDirectory(parts.slice(0, index + 1).join("/"))
                      }
                    >
                      {part}
                    </button>
                  </span>
                ))}
              </nav>
              <button
                type="button"
                className={buttonClass("primary")}
                onClick={() => choose(directory)}
              >
                {directory
                  ? t("import.destination.use")
                  : t("import.destination.useRoot")}
              </button>
              <ul className="min-h-24 flex-1 divide-y divide-line overflow-y-auto">
                {shown.map((entry) => (
                  <li key={entry.path}>
                    <button
                      type="button"
                      className="flex min-h-11 w-full items-center gap-2 break-all text-left text-accent-ink"
                      onClick={() => {
                        setDirectory(entry.path);
                        setFilter("");
                      }}
                    >
                      <Folder
                        size={18}
                        className="shrink-0"
                        aria-hidden="true"
                      />
                      {entry.path.split("/").pop()}
                    </button>
                  </li>
                ))}
                {!shown.length && (
                  <li className="py-2 text-sm text-mute">
                    {t("import.destination.empty")}
                  </li>
                )}
              </ul>
              {failure && (
                <p role="alert" className="text-sm text-bad-fg">
                  {error(failure)}
                </p>
              )}
            </Popover.Popup>
          </Popover.Positioner>
        </Popover.Portal>
      </Popover.Root>
      <p className="break-all text-sm text-mute">
        {value === undefined
          ? t("import.destination.none")
          : t("import.destination.selected", {
              path: value === "" ? "/" : `/${value}`,
            })}
      </p>
    </div>
  );
}
