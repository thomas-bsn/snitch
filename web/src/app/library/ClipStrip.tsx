import { ChevronDown, ChevronLeft, ChevronRight, Play } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useI18n } from "../i18n";
import { formatTimecode } from "./clipFormat";
import type { Neighbour, Sequence } from "./clipNeighbours";
import { ExplorerThumbnail } from "./ExplorerThumbnail";

const OPEN_KEY = "snitch.clip.strip";

/**
 * Captures of one game share a long prefix ("Counter-Strike 2 2026.11."): in a narrow tile,
 * what tells them apart is cut off. The prefix common to the whole list is left out, at a
 * word boundary, and so is the extension.
 */
function shortTitles(items: Neighbour[]): string[] {
  const names = items.map((item) =>
    item.title.replace(/\.[a-z0-9]{2,4}$/i, ""),
  );
  if (names.length < 2) return names;
  let prefix = names[0];
  for (const name of names) {
    while (!name.startsWith(prefix)) prefix = prefix.slice(0, -1);
  }
  const cut = Math.max(prefix.lastIndexOf(" "), prefix.lastIndexOf("_"));
  const length = cut + 1;
  return length >= 6 ? names.map((name) => name.slice(length) || name) : names;
}

function storedOpen() {
  try {
    return window.localStorage.getItem(OPEN_KEY) !== "closed";
  } catch {
    return true;
  }
}

function storeOpen(open: boolean) {
  try {
    window.localStorage.setItem(OPEN_KEY, open ? "open" : "closed");
  } catch {
    // Without storage, the strip opens again on the next visit.
  }
}

const arrowClass =
  "inline-flex size-11 shrink-0 items-center justify-center rounded-lg border border-line-strong bg-surface text-ink transition-colors hover:bg-side disabled:cursor-not-allowed disabled:opacity-40 md:size-8";

/** The arrows and the position: always there, strip open or folded. */
export function ClipPager({
  sequence,
  index,
  open,
  onToggle,
  onGo,
}: {
  sequence: Sequence;
  index: number;
  open: boolean;
  onToggle: () => void;
  onGo: (target: number) => void;
}) {
  const { t } = useI18n();
  const previous = sequence.items[index - 1];
  const next = sequence.items[index + 1];
  const where =
    sequence.kind === "search"
      ? t("clip.nav.searchLabel", { q: sequence.label })
      : sequence.label
        ? sequence.label.split("/").join(" › ")
        : t("clip.back");
  return (
    <nav
      aria-label={t("clip.nav.label")}
      className="ml-auto flex items-center gap-1.5"
    >
      <span className="mr-1 max-w-64 truncate text-xs">
        <span className="font-medium text-ink tabular-nums">
          {t("clip.nav.position", {
            index: index + 1,
            total: sequence.items.length,
          })}
        </span>
        <span className="hidden sm:inline">
          {" · "}
          {where}
        </span>
      </span>
      <button
        type="button"
        className={arrowClass}
        disabled={!previous}
        onClick={() => onGo(index - 1)}
        aria-label={
          previous
            ? t("clip.nav.previous", { name: previous.title })
            : t("clip.nav.first")
        }
        title={previous ? `${previous.title} (Shift+P)` : t("clip.nav.first")}
        aria-keyshortcuts="Shift+P"
      >
        <ChevronLeft size={18} aria-hidden="true" />
      </button>
      <button
        type="button"
        className={arrowClass}
        disabled={!next}
        onClick={() => onGo(index + 1)}
        aria-label={
          next ? t("clip.nav.next", { name: next.title }) : t("clip.nav.last")
        }
        title={next ? `${next.title} (Shift+N)` : t("clip.nav.last")}
        aria-keyshortcuts="Shift+N"
      >
        <ChevronRight size={18} aria-hidden="true" />
      </button>
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        aria-controls="clip-strip"
        className="inline-flex min-h-11 items-center gap-1 rounded-lg px-2 text-xs font-medium text-mute hover:bg-side hover:text-ink md:min-h-8"
      >
        <ChevronDown
          size={16}
          aria-hidden="true"
          className={`transition-transform motion-reduce:transition-none ${open ? "rotate-180" : ""}`}
        />
        {t(open ? "clip.nav.hide" : "clip.nav.show")}
      </button>
    </nav>
  );
}

function Tile({
  item,
  current,
  onGo,
  kind,
  label,
}: {
  label: string;
  item: Neighbour;
  current: boolean;
  onGo: () => void;
  kind: Sequence["kind"];
}) {
  const { t } = useI18n();
  const sub =
    kind === "search" && item.at !== undefined
      ? t("clip.nav.at", { time: formatTimecode(item.at) })
      : typeof item.durationSeconds === "number"
        ? formatTimecode(item.durationSeconds)
        : "";
  return (
    <button
      type="button"
      onClick={onGo}
      aria-current={current ? "true" : undefined}
      data-strip-clip={item.clipId}
      title={item.excerpt ? `“${item.excerpt}”` : item.title}
      className={`flex w-36 shrink-0 flex-col gap-1.5 rounded-xl border p-1.5 text-left transition-colors ${
        current
          ? "border-accent bg-accent-soft"
          : "border-transparent hover:bg-side"
      }`}
    >
      <span className="relative flex aspect-video w-full items-center justify-center overflow-hidden rounded-md bg-[#1b1b22] text-white/60">
        <Play size={18} aria-hidden="true" />
        <ExplorerThumbnail clipId={item.clipId} />
        {sub && (
          <span className="absolute right-1 bottom-1 rounded bg-black/70 px-1 font-mono text-[11px] text-white">
            {sub}
          </span>
        )}
        {current && (
          <span className="absolute inset-0 rounded-md ring-2 ring-accent ring-inset" />
        )}
      </span>
      <span
        className={`line-clamp-2 text-xs leading-snug [overflow-wrap:anywhere] ${
          current ? "font-semibold text-accent-ink" : "text-ink"
        }`}
      >
        {label}
      </span>
    </button>
  );
}

/** The clips of the list, the current one in the middle, scrolled with the arrows. */
export function ClipStrip({
  sequence,
  index,
  onGo,
}: {
  sequence: Sequence;
  index: number;
  onGo: (target: number) => void;
}) {
  const { t } = useI18n();
  const list = useRef<HTMLOListElement | null>(null);
  const current = sequence.items[index]?.clipId;
  const labels = useMemo(() => shortTitles(sequence.items), [sequence.items]);

  // The current clip comes back to the middle of the strip, without moving the page.
  useEffect(() => {
    const box = list.current;
    const tile = box?.querySelector<HTMLElement>(
      `[data-strip-clip="${current}"]`,
    );
    if (!box || !tile) return;
    const reduce = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    box.scrollTo({
      left: tile.offsetLeft - box.clientWidth / 2 + tile.clientWidth / 2,
      behavior: reduce ? "auto" : "smooth",
    });
  }, [current]);

  return (
    <div id="clip-strip" className="border-b border-line bg-side/60">
      <ol
        ref={list}
        aria-label={t("clip.nav.label")}
        className="m-0 flex list-none gap-1 overflow-x-auto scroll-smooth px-4 py-2 [scrollbar-width:thin] md:px-7"
      >
        {sequence.items.map((item, i) => (
          <li key={`${item.clipId}-${item.at ?? ""}`}>
            <Tile
              item={item}
              current={i === index}
              kind={sequence.kind}
              label={labels[i]}
              onGo={() => onGo(i)}
            />
          </li>
        ))}
      </ol>
    </div>
  );
}

export function useStripOpen() {
  const [open, setOpen] = useState(storedOpen);
  return [
    open,
    () =>
      setOpen((value) => {
        storeOpen(!value);
        return !value;
      }),
  ] as const;
}

/** The end of a clip, when the next one follows: it starts after a few seconds unless cancelled. */
export function NextCountdown({
  next,
  onGo,
  onCancel,
}: {
  next: Neighbour;
  onGo: () => void;
  onCancel: () => void;
}) {
  const { t } = useI18n();
  const [left, setLeft] = useState(5);
  const go = useRef(onGo);
  go.current = onGo;
  useEffect(() => {
    if (left <= 0) {
      go.current();
      return;
    }
    const timer = window.setTimeout(() => setLeft((n) => n - 1), 1000);
    return () => window.clearTimeout(timer);
  }, [left]);
  return (
    <div className="absolute inset-0 z-10 flex items-center justify-center rounded-xl bg-black/75 p-4 text-white">
      <div className="flex max-w-xs flex-col items-center gap-3 text-center">
        <p className="m-0 text-sm text-white/80" role="status">
          {t("clip.nav.countdown", { seconds: left })}
        </p>
        <span className="relative flex aspect-video w-40 items-center justify-center overflow-hidden rounded-md bg-[#1b1b22] text-white/60">
          <Play size={18} aria-hidden="true" />
          <ExplorerThumbnail clipId={next.clipId} />
        </span>
        <p className="m-0 font-semibold [overflow-wrap:anywhere]">
          {next.title}
        </p>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={onCancel}
            className="inline-flex h-11 items-center rounded-lg border border-white/40 px-4 text-sm hover:bg-white/10 md:h-9"
          >
            {t("clip.nav.cancel")}
          </button>
          <button
            type="button"
            onClick={onGo}
            className="inline-flex h-11 items-center rounded-lg bg-white px-4 text-sm font-medium text-black hover:bg-white/90 md:h-9"
          >
            {t("clip.nav.playNow")}
          </button>
        </div>
      </div>
    </div>
  );
}
