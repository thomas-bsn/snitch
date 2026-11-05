import { Volume1, Volume2, VolumeX } from "lucide-react";
import { useI18n } from "../i18n";
import { setVolume, useVolume } from "./volume";

/** The volume slider shared by every player; it follows the level kept in this browser. */
export function VolumeBar() {
  const { t } = useI18n();
  const volume = useVolume();
  const Icon = volume === 0 ? VolumeX : volume < 0.5 ? Volume1 : Volume2;
  return (
    <div className="flex items-center gap-2 text-mute">
      <button
        type="button"
        aria-label={t("volume.mute")}
        aria-pressed={volume === 0}
        onClick={() => setVolume(volume === 0 ? 1 : 0)}
        className="flex size-11 items-center justify-center rounded-md hover:bg-side md:size-8"
      >
        <Icon className="size-4" aria-hidden="true" />
      </button>
      <input
        type="range"
        min={0}
        max={100}
        step={1}
        value={Math.round(volume * 100)}
        onChange={(event) => setVolume(Number(event.target.value) / 100)}
        aria-label={t("volume.label")}
        className="h-11 w-32 accent-[var(--color-accent,currentColor)] md:h-8"
      />
      <span aria-hidden="true" className="w-9 text-xs tabular-nums">
        {Math.round(volume * 100)} %
      </span>
    </div>
  );
}
