import {
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import {
  MAX_KEPT_LINES,
  appendLines,
  parseLogLine,
  prependLines,
} from "./workerLogs";
import { fetchWorkerLogs, type WorkerLogLine } from "./workersApi";

/** How often the live end of the log is asked for. */
export const LIVE_INTERVAL_MS = 2000;
const FIRST_PAGE = 500;
const OLDER_PAGE = 500;
const AT_BOTTOM_PX = 24;

const LEVEL_CLASS = {
  DEBUG: "text-mute",
  INFO: "text-ink",
  WARN: "text-warn-fg",
  ERROR: "text-bad-fg",
} as const;

function Line({ line }: { line: WorkerLogLine }) {
  const { t } = useI18n();
  const parsed = parseLogLine(line.line);
  // Everything is shown as text: a line is never interpreted as markup.
  if (!parsed.event && !parsed.level) {
    return <li className="whitespace-pre-wrap break-all">{parsed.raw}</li>;
  }
  return (
    <li className="whitespace-pre-wrap break-all">
      {parsed.at && <span className="text-mute">{parsed.at} </span>}
      {parsed.level && (
        <span className={`font-semibold ${LEVEL_CLASS[parsed.level]}`}>
          {t(`workers.logs.level.${parsed.level}`)}{" "}
        </span>
      )}
      {parsed.event && <span className="font-medium">{parsed.event}</span>}
      {parsed.fields.map(([key, value]) => (
        <span key={key} className="text-mute">
          {" "}
          {key}={value}
        </span>
      ))}
    </li>
  );
}

/** The log of the Worker's current run: its start to now, followed live while the box is checked. */
export function WorkerLogsPanel({ id }: { id: string }) {
  const { t, error } = useI18n();
  const liveId = useId();
  const [lines, setLines] = useState<WorkerLogLine[]>([]);
  const [session, setSession] = useState("");
  const [hasMore, setHasMore] = useState(false);
  const [trimmed, setTrimmed] = useState(false);
  const [status, setStatus] = useState<"loading" | "ready" | "failed">(
    "loading",
  );
  const [failure, setFailure] = useState<string>();
  const [live, setLive] = useState(true);
  const [loadingOlder, setLoadingOlder] = useState(false);
  const box = useRef<HTMLDivElement>(null);
  // Whether the reader is at the end of the log: only then does a new line scroll the view.
  const stick = useRef(true);
  // Height before older lines were put in front, to keep the reader on the same line.
  const before = useRef<number | null>(null);
  const linesRef = useRef(lines);
  const sessionRef = useRef(session);
  const busy = useRef(false);
  linesRef.current = lines;
  sessionRef.current = session;

  const loadFirst = useCallback(async () => {
    const result = await fetchWorkerLogs(id, { limit: FIRST_PAGE });
    if (!result.ok) {
      setStatus("failed");
      setFailure(result.code);
      return;
    }
    stick.current = true;
    setLines(result.value.lines);
    setSession(result.value.session_id);
    setHasMore(result.value.has_more);
    setTrimmed(false);
    setFailure(undefined);
    setStatus("ready");
  }, [id]);

  useEffect(() => {
    void loadFirst();
  }, [loadFirst]);

  // Live: one timer at most, cleared when the box is unchecked, the tab is hidden or the page is left.
  useEffect(() => {
    if (!live || status !== "ready") return;
    let timer: ReturnType<typeof setInterval> | undefined;
    let current = true;
    async function poll() {
      if (busy.current || document.hidden) return;
      busy.current = true;
      try {
        const held = linesRef.current;
        const after = held.length ? held[held.length - 1].id : undefined;
        const result = await fetchWorkerLogs(id, after ? { after } : {});
        if (!current || !result.ok) return;
        const page = result.value;
        if (sessionRef.current && page.session_id !== sessionRef.current) {
          // The Worker started again: its new run replaces the old one.
          await loadFirst();
          return;
        }
        if (page.lines.length === 0 && !page.session_id) return;
        if (!after) {
          stick.current = true;
          setLines(page.lines);
          setHasMore(page.has_more);
          setSession(page.session_id);
          return;
        }
        if (page.lines.length === 0) return;
        const merged = appendLines(held, page.lines);
        setLines(merged.lines);
        if (merged.trimmed) {
          setTrimmed(true);
          setHasMore(true);
        }
      } finally {
        busy.current = false;
      }
    }
    const start = () => {
      if (timer === undefined) timer = setInterval(poll, LIVE_INTERVAL_MS);
    };
    const stop = () => {
      if (timer !== undefined) clearInterval(timer);
      timer = undefined;
    };
    const visibility = () => {
      if (document.hidden) stop();
      else {
        start();
        void poll();
      }
    };
    if (!document.hidden) start();
    document.addEventListener("visibilitychange", visibility);
    return () => {
      current = false;
      stop();
      document.removeEventListener("visibilitychange", visibility);
    };
  }, [live, status, id, loadFirst]);

  useLayoutEffect(() => {
    const element = box.current;
    if (!element) return;
    if (before.current !== null) {
      element.scrollTop += element.scrollHeight - before.current;
      before.current = null;
    } else if (stick.current) {
      element.scrollTop = element.scrollHeight;
    }
  }, [lines]);

  async function loadOlder() {
    const first = lines[0];
    if (!first || loadingOlder) return;
    setLoadingOlder(true);
    const result = await fetchWorkerLogs(id, {
      before: first.id,
      limit: OLDER_PAGE,
    });
    setLoadingOlder(false);
    if (!result.ok) {
      setFailure(result.code);
      return;
    }
    before.current = box.current?.scrollHeight ?? null;
    stick.current = false;
    setLines((held) => prependLines(held, result.value.lines));
    setHasMore(result.value.has_more);
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm text-mute">{t("workers.logs.lead")}</p>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <label htmlFor={liveId} className="flex items-center gap-2">
          <input
            id={liveId}
            type="checkbox"
            checked={live}
            onChange={(event) => setLive(event.target.checked)}
          />
          <span>{t("workers.logs.live")}</span>
        </label>
        <span className="text-sm text-mute">{t("workers.logs.liveHint")}</span>
        {session && (
          <span className="text-sm text-mute">
            {t("workers.logs.session")} · {session}
          </span>
        )}
      </div>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {status === "loading" && <p role="status">{t("workers.logs.loading")}</p>}
      {status === "ready" && lines.length === 0 && (
        <p className="text-mute">{t("workers.logs.empty")}</p>
      )}
      {hasMore && lines.length < MAX_KEPT_LINES && (
        <div>
          <button
            type="button"
            className={buttonClass("secondary")}
            disabled={loadingOlder}
            onClick={() => void loadOlder()}
          >
            {loadingOlder
              ? t("workers.logs.loadingOlder")
              : t("workers.logs.older")}
          </button>
        </div>
      )}
      {trimmed && (
        <p className="text-sm text-mute">
          {t("workers.logs.kept", { count: MAX_KEPT_LINES })}
        </p>
      )}
      {lines.length > 0 && (
        // A scrollable region the keyboard can reach. It is not a live region: a line is not announced.
        <div
          ref={box}
          role="region"
          tabIndex={0}
          aria-label={t("workers.logs.region")}
          aria-live="off"
          onScroll={(event) => {
            const e = event.currentTarget;
            stick.current =
              e.scrollHeight - e.scrollTop - e.clientHeight <= AT_BOTTOM_PX;
          }}
          className="max-h-96 overflow-auto rounded-lg border border-line-strong bg-side p-3 font-mono text-xs text-ink"
        >
          <ol className="m-0 flex list-none flex-col gap-0.5 p-0">
            {lines.map((line) => (
              <Line key={line.id} line={line} />
            ))}
          </ol>
        </div>
      )}
    </div>
  );
}
