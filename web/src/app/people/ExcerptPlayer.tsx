import { Pause, Play } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { getVolume, subscribeVolume } from "../components/volume";
import { useI18n } from "../i18n";
import { excerptUrl } from "./peopleApi";

// One audio at a time: the player that starts stops the one that was playing.
let current: (() => void) | undefined;

/** Stops whatever plays and registers `stop` as what to stop when the next one starts. */
export function claimPlayback(stop: () => void) {
  if (current) current();
  current = stop;
}

/**
 * A reusable player for the excerpt of a local speaker. The button names the voice and its
 * clip and is pressed while the excerpt plays.
 */
export function ExcerptPlayer({
  localSpeakerId,
  label,
  clip,
  shortcut,
  link = false,
  compact = false,
}: {
  /** A small square icon button, for a line of a list. */
  compact?: boolean;
  /** A text link ("Listen to an excerpt") instead of the round icon button. */
  link?: boolean;
  /** Key (data-shortcut) a screen with keyboard shortcuts presses this button with. */
  shortcut?: string;
  localSpeakerId: number;
  label: string;
  clip: string;
}) {
  const { t } = useI18n();
  const audio = useRef<HTMLAudioElement>(null);
  const [playing, setPlaying] = useState(false);

  // The excerpt follows the shared volume.
  useEffect(() => {
    const apply = () => {
      if (audio.current) audio.current.volume = getVolume();
    };
    apply();
    return subscribeVolume(apply);
  }, []);

  // A player leaving the screen does not keep playing.
  useEffect(() => {
    const element = audio.current;
    return () => {
      element?.pause();
    };
  }, []);

  function toggle() {
    const element = audio.current;
    if (!element) return;
    if (playing) {
      element.pause();
      return;
    }
    claimPlayback(() => element.pause());
    void element.play().catch(() => setPlaying(false));
  }

  const Icon = playing ? Pause : Play;
  return (
    <>
      {link ? (
        <button
          type="button"
          aria-pressed={playing}
          onClick={toggle}
          aria-label={`${t("clip.voices.excerpt")} (${label}, ${clip})`}
          className="inline-flex min-h-11 items-center gap-1 text-xs text-accent-ink hover:underline md:min-h-0"
        >
          <Icon className="size-3.5" aria-hidden="true" />
          <span aria-hidden="true">{t("clip.voices.excerpt")}</span>
        </button>
      ) : (
        <button
          type="button"
          aria-pressed={playing}
          data-shortcut={shortcut}
          aria-keyshortcuts={shortcut === " " ? "Space" : shortcut}
          onClick={toggle}
          className={
            compact
              ? "flex size-11 shrink-0 items-center justify-center rounded-md text-mute hover:bg-side md:size-8"
              : "flex size-11 shrink-0 items-center justify-center rounded-full border border-line bg-surface text-ink hover:bg-side"
          }
          aria-label={t("people.excerpt.player", { label, clip })}
        >
          <Icon className="size-4" aria-hidden="true" />
        </button>
      )}
      <audio
        ref={audio}
        preload="none"
        src={excerptUrl(localSpeakerId)}
        onPlay={() => setPlaying(true)}
        onPause={() => setPlaying(false)}
        onEnded={() => setPlaying(false)}
      />
    </>
  );
}
