import { Upload } from "lucide-react";
import { useId, useRef, useState } from "react";
import { useI18n } from "../i18n";

/** Accepted by the file picker; the server stays the judge of what is a readable video. */
export const VIDEO_ACCEPT = "video/*,.mkv,.mp4,.mov,.webm";
const VIDEO_EXTENSION = /\.(mkv|mp4|mov|webm|m4v|avi)$/i;

export const looksLikeVideo = (file: File) =>
  file.type.startsWith("video/") || VIDEO_EXTENSION.test(file.name);

/** Where videos are dropped, or chosen with the keyboard through the button. */
export function DropZone({ onFiles }: { onFiles: (files: File[]) => void }) {
  const { t } = useI18n();
  const [over, setOver] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  const hintId = useId();

  function receive(list: FileList | null) {
    const files = list ? Array.from(list) : [];
    if (files.length > 0) onFiles(files);
  }

  return (
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
      className={`flex flex-col items-center gap-2 rounded-xl border-2 border-dashed px-4 py-10 text-center transition-colors ${
        over
          ? "border-accent-solid bg-accent-soft"
          : "border-line-strong bg-surface"
      }`}
    >
      <Upload className="size-7 text-accent-ink" aria-hidden="true" />
      <p className="text-base font-medium">
        {over ? t("import.drop.active") : t("import.drop.title")}
      </p>
      <p className="text-mute">
        {t("import.drop.or")}{" "}
        <button
          type="button"
          onClick={() => input.current?.click()}
          aria-describedby={hintId}
          className="font-medium text-accent-ink underline underline-offset-2"
        >
          {t("import.drop.choose")}
        </button>
      </p>
      <p id={hintId} className="text-xs text-mute">
        {t("import.drop.hint")}
      </p>
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
  );
}
