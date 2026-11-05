import { useEffect, useRef, useState, type RefObject } from "react";
import { VolumeBar } from "../components/VolumeBar";
import { getVolume, setVolume, subscribeVolume } from "../components/volume";
import { useI18n } from "../i18n";

/** How often the moment kept in the address follows the playback. */
const FOLLOW_EVERY_SECONDS = 5;

/**
 * The video of a clip. It opens at the moment the address names, and tells the page where
 * it is when playback pauses, seeks, ends, and every few seconds while it plays.
 */
export function ClipPlayer({
  clipId,
  videoRef,
  startAt,
  onMoment,
  onPlayhead,
}: {
  clipId: number;
  videoRef: RefObject<HTMLVideoElement | null>;
  startAt: number;
  /** The moment worth keeping in the address. */
  onMoment: (seconds: number) => void;
  /** The position, for highlighting the segment being played. */
  onPlayhead: (seconds: number) => void;
}) {
  const { t } = useI18n();
  const [failed, setFailed] = useState(false);
  // Only the moment of arrival counts: later changes of the address come from this player.
  const arrival = useRef(startAt);
  const lastKept = useRef(startAt);

  useEffect(() => {
    setFailed(false);
  }, [clipId]);

  // The video follows the shared volume, and the volume of its own controls feeds it back.
  useEffect(() => {
    const apply = () => {
      if (videoRef.current) videoRef.current.volume = getVolume();
    };
    apply();
    return subscribeVolume(apply);
  }, [videoRef, clipId]);

  function keep(video: HTMLVideoElement) {
    const seconds = Math.floor(video.currentTime);
    lastKept.current = seconds;
    onMoment(seconds);
  }

  return (
    <div className="flex flex-col gap-2">
      <video
        ref={videoRef}
        src={`/api/clips/${clipId}/media`}
        controls
        preload="metadata"
        aria-label={t("clip.videoLabel")}
        className="aspect-video w-full rounded-xl border border-line bg-black"
        onLoadedMetadata={(event) => {
          const video = event.currentTarget;
          if (arrival.current > 0 && arrival.current < video.duration) {
            video.currentTime = arrival.current;
            onPlayhead(arrival.current);
          }
          arrival.current = 0;
        }}
        onTimeUpdate={(event) => {
          const video = event.currentTarget;
          onPlayhead(video.currentTime);
          if (
            !video.paused &&
            Math.abs(video.currentTime - lastKept.current) >=
              FOLLOW_EVERY_SECONDS
          ) {
            keep(video);
          }
        }}
        onPause={(event) => keep(event.currentTarget)}
        onSeeked={(event) => keep(event.currentTarget)}
        onEnded={(event) => keep(event.currentTarget)}
        onVolumeChange={(event) => setVolume(event.currentTarget.volume)}
        onError={() => setFailed(true)}
      >
        <track kind="captions" />
      </video>
      <VolumeBar />
      {failed && (
        <p role="status" className="m-0 text-mute">
          {t("clip.videoFailed")}
        </p>
      )}
    </div>
  );
}
