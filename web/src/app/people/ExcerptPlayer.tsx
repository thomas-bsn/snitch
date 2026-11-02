import { Pause, Play } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useI18n } from "../i18n";
import { excerptUrl } from "./peopleApi";

// One excerpt at a time: the player that starts stops the one that was playing.
let current: (() => void) | undefined;

/**
 * A reusable player for the excerpt of a local speaker. The button names the voice and its
 * clip and is pressed while the excerpt plays.
 */
export function ExcerptPlayer({
  localSpeakerId,
  label,
  clip,
}: {
  localSpeakerId: number;
  label: string;
  clip: string;
}) {
  const { t } = useI18n();
  const audio = useRef<HTMLAudioElement>(null);
  const [playing, setPlaying] = useState(false);

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
    if (current) current();
    current = () => element.pause();
    void element.play().catch(() => setPlaying(false));
  }

  const Icon = playing ? Pause : Play;
  return (
    <>
      <button
        type="button"
        aria-pressed={playing}
        onClick={toggle}
        className="flex size-11 shrink-0 items-center justify-center rounded-full border border-line bg-surface text-ink hover:bg-side"
        aria-label={t("people.excerpt.player", { label, clip })}
      >
        <Icon className="size-4" aria-hidden="true" />
      </button>
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
