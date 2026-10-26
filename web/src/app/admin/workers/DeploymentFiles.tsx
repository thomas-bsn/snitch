import { Tabs } from "@base-ui/react/tabs";
import { Check, Copy, Download } from "lucide-react";
import { useState } from "react";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { downloadText } from "./envFile";

type DeploymentFilesProps = {
  env: string;
  compose: string;
};

const FILES = [
  { id: "env", download: "snitch-worker.env", label: "workers.files.env" },
  { id: "compose", download: "compose.yaml", label: "workers.files.compose" },
] as const;

/** The two generated deployment files, to read, copy or download. */
export function DeploymentFiles({ env, compose }: DeploymentFilesProps) {
  const { t } = useI18n();
  const [copied, setCopied] = useState<"env" | "compose" | null>(null);
  const [copyFailed, setCopyFailed] = useState(false);
  const contents = { env, compose };

  async function copy(id: "env" | "compose") {
    try {
      await navigator.clipboard.writeText(contents[id]);
      setCopyFailed(false);
      setCopied(id);
      setTimeout(
        () => setCopied((current) => (current === id ? null : current)),
        2000,
      );
    } catch {
      setCopyFailed(true);
    }
  }

  return (
    <Tabs.Root defaultValue="env" className="flex flex-col gap-3">
      <Tabs.List
        aria-label={t("workers.files.tabsLabel")}
        className="flex gap-1 border-b border-line"
      >
        {FILES.map((file) => (
          <Tabs.Tab
            key={file.id}
            value={file.id}
            className="-mb-px h-11 border-b-2 border-transparent px-3 font-medium text-mute outline-none data-[active]:border-accent data-[active]:text-ink focus-visible:outline-2 md:h-9"
          >
            {t(file.label)}
          </Tabs.Tab>
        ))}
      </Tabs.List>
      {FILES.map((file) => (
        <Tabs.Panel
          key={file.id}
          value={file.id}
          className="flex flex-col gap-3"
        >
          <textarea
            readOnly
            spellCheck={false}
            aria-label={t("workers.files.fileLabel", { file: t(file.label) })}
            value={contents[file.id]}
            rows={14}
            className="w-full resize-y rounded-lg border border-line-strong bg-side p-3 font-mono text-xs text-ink"
          />
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              className={buttonClass("secondary")}
              onClick={() => void copy(file.id)}
            >
              {copied === file.id ? (
                <Check className="size-4" />
              ) : (
                <Copy className="size-4" />
              )}
              {copied === file.id
                ? t("workers.files.copied")
                : t("workers.files.copy")}
            </button>
            <button
              type="button"
              className={buttonClass("secondary")}
              onClick={() => downloadText(file.download, contents[file.id])}
            >
              <Download className="size-4" />
              {t("workers.files.download")} {file.download}
            </button>
          </div>
        </Tabs.Panel>
      ))}
      {copyFailed && (
        <p role="alert" className="text-bad-fg">
          {t("workers.files.copyFailed")}
        </p>
      )}
    </Tabs.Root>
  );
}
