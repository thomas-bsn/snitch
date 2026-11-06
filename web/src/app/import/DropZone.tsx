import { Upload } from "lucide-react";
import { useId, useRef, useState, type ReactNode } from "react";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";

/** Accepted by the file picker; the server stays the judge of what is a readable video. */
export const VIDEO_ACCEPT = "video/*,.mkv,.mp4,.mov,.webm";
const VIDEO_EXTENSION = /\.(mkv|mp4|mov|webm|m4v|avi)$/i;

export const looksLikeVideo = (file: File) =>
  file.type.startsWith("video/") || VIDEO_EXTENSION.test(file.name);

/** Where videos are dropped, or chosen with the keyboard through the button. */
export function DropZone({
  onFiles,
  blocked = false,
  onBlocked,
  children,
}: {
  onFiles: (files: File[]) => void;
  /** No destination yet: files are never taken; trying to add some calls onBlocked. */
  blocked?: boolean;
  onBlocked?: () => void;
  /** The destination line, between the drop area and the hint. */
  children?: ReactNode;
}) {
  const { t } = useI18n();
  const [over, setOver] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  const hintId = useId();

  function receive(list: FileList | null) {
    const files = list ? Array.from(list) : [];
    if (files.length === 0) return;
    if (blocked) onBlocked?.();
    else onFiles(files);
  }

  return (
    <div className="flex flex-col gap-3">
      <div
        onDragOver={(event) => {
          event.preventDefault();
          setOver(true);
        }}
        onDragLeave={() => setOver(false)}
        onDrop={(event) => {
          event.preventDefault();
          setOver(false);
          receive(event.dataTransfer.files);
        }}
        className={`flex flex-col items-center gap-2 rounded-xl border-2 border-dashed px-6 py-10 text-center transition-colors ${
          blocked
            ? "border-line-strong bg-side text-mute"
            : over
              ? "border-accent-solid bg-accent-soft"
              : "border-line-strong bg-surface"
        }`}
      >
        <Upload className="size-6 text-mute" aria-hidden="true" />
        <p className="m-0 font-medium">
          {over ? t("import.drop.active") : t("import.drop.title")}
        </p>
        <button
          type="button"
          onClick={() => (blocked ? onBlocked?.() : input.current?.click())}
          aria-disabled={blocked || undefined}
          aria-describedby={hintId}
          className={buttonClass()}
        >
          {t("import.drop.browse")}
        </button>
        <input
          ref={input}
          type="file"
          multiple
          accept={VIDEO_ACCEPT}
          tabIndex={-1}
          aria-label={t("import.drop.choose")}
          className="sr-only"
          onChange={(event) => {
            receive(event.target.files);
            event.target.value = "";
          }}
        />
      </div>
      {children}
      <p id={hintId} className="m-0 text-xs text-mute">
        {t("import.drop.hint")}
      </p>
    </div>
  );
}
