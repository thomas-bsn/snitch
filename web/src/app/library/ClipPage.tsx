import { Tabs } from "@base-ui/react/tabs";
import { ArrowLeft } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import { Link, useLocation, useParams, useSearchParams } from "react-router";
import { useFormatDate } from "../admin/format";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { ClipMetadata } from "./ClipMetadata";
import { ClipPlayer } from "./ClipPlayer";
import { ClipStatePill } from "./ClipStatePill";
import { HistoryTab } from "./HistoryTab";
import { Retranscribe } from "./Retranscribe";
import { ProgressBanner } from "./ProgressBanner";
import { TranscriptTab } from "./TranscriptTab";
import { VoicesTab } from "./VoicesTab";
import { formatTimecode, parseMoment } from "./clipFormat";
import { fetchClip, fetchClipVoice, type Clip } from "./libraryApi";
import { TranscribeButton } from "./TranscribeButton";

const tabClass =
  "-mb-px inline-flex h-11 shrink-0 cursor-pointer items-center border-b-2 border-transparent px-3 font-medium whitespace-nowrap text-mute outline-none hover:text-ink data-[active]:border-accent data-[active]:text-ink md:h-10";

type TabName = "transcript" | "voices" | "history";

function ClipView({ clip, onChanged }: { clip: Clip; onChanged: () => void }) {
  const { t } = useI18n();
  const formatDate = useFormatDate();
  const location = useLocation();
  const [params, setParams] = useSearchParams();
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const [startAt] = useState(() => parseMoment(params.get("t")));
  const [playhead, setPlayhead] = useState(startAt);
  const [tab, setTab] = useState<TabName>("transcript");
  const voice = useLoaded(
    useCallback(() => fetchClipVoice(clip.id), [clip.id]),
  );
  useDocumentTitle(clip.original_name);

  const back = (location.state as { from?: string } | null)?.from ?? "/library";
  const segments = useMemo(
    () => clip.transcription?.segments ?? [],
    [clip.transcription],
  );

  // The address keeps the moment being watched, so that sharing it shares that instant.
  const keepMoment = useCallback(
    (seconds: number) => {
      setParams(
        (current) => {
          const next = new URLSearchParams(current);
          if (seconds > 0) next.set("t", String(seconds));
          else next.delete("t");
          return next;
        },
        { replace: true, state: location.state },
      );
    },
    [setParams, location.state],
  );

  function seek(seconds: number) {
    const video = videoRef.current;
    if (video) {
      video.currentTime = seconds;
      void video.play().catch(() => undefined);
    }
    setPlayhead(seconds);
    keepMoment(Math.floor(seconds));
  }

  const date = clip.recorded_at
    ? formatDate(clip.recorded_at)
    : t("clip.speaker.unknown");
  const voiceData = voice.state.status === "ready" ? voice.state.value : null;

  return (
    <section className="flex flex-col gap-4 px-4 py-4 md:px-7 md:py-5">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <Link
          to={back}
          className="inline-flex min-h-11 items-center gap-1.5 text-mute hover:text-ink md:min-h-8"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          {t(back.startsWith("/search") ? "clip.backToSearch" : "clip.back")}
        </Link>
        <ClipStatePill status={clip.status} />
      </div>

      <div className="grid gap-5 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
        <div className="flex min-w-0 flex-col gap-4">
          <ClipPlayer
            clipId={clip.id}
            videoRef={videoRef}
            startAt={startAt}
            onMoment={keepMoment}
            onPlayhead={setPlayhead}
          />
          <div>
            <h1 className="m-0 text-[22px] font-semibold break-words">
              {clip.original_name}
            </h1>
            <p className="mt-1 mb-0 text-mute">
              {[
                t("clip.meta.owner", {
                  owner: clip.owner_deleted
                    ? t("library.deletedOwner")
                    : clip.owner,
                }),
                date,
                typeof clip.duration_seconds === "number"
                  ? formatTimecode(clip.duration_seconds)
                  : null,
                t(
                  clip.visibility === "shared"
                    ? "clip.meta.shared"
                    : "clip.meta.private",
                ),
              ]
                .filter(Boolean)
                .join(" · ")}
            </p>
            {clip.sources && clip.sources.length > 0 && (
              <p className="mt-0.5 mb-0 text-mute">
                {t("clip.meta.sources", { sources: clip.sources.join(", ") })}
              </p>
            )}
          </div>
          <ClipMetadata clip={clip} onChanged={onChanged} />
          <ProgressBanner clip={clip} voice={voiceData} />
          <TranscribeButton
            clipId={clip.id}
            progress={clip.progress}
            audioTrackSelection={clip.audio_track_selection}
            hasCorrections={clip.transcription !== undefined}
            onChanged={onChanged}
            audioTracks={clip.tracks?.filter((t) => t.type === "audio") ?? []}
          />
          {clip.transcription && (
            <Retranscribe
              id={clip.id}
              stream={clip.transcription.audio_stream_index}
            />
          )}
        </div>

        <Tabs.Root
          value={tab}
          onValueChange={(value) => setTab(value as TabName)}
          className="flex min-w-0 flex-col rounded-xl border border-line"
        >
          <Tabs.List
            aria-label={t("clip.tabs.label")}
            tabIndex={0}
            className="flex gap-1 overflow-x-auto border-b border-line px-3"
          >
            <Tabs.Tab value="transcript" className={tabClass}>
              {t("clip.tabs.transcript")}
            </Tabs.Tab>
            <Tabs.Tab value="voices" className={tabClass}>
              {t("clip.tabs.voices")}
            </Tabs.Tab>
            <Tabs.Tab value="history" className={tabClass}>
              {t("clip.tabs.history")}
            </Tabs.Tab>
          </Tabs.List>
          <Tabs.Panel
            value="transcript"
            className="max-h-[70vh] overflow-y-auto outline-none"
          >
            <TranscriptTab
              clipId={clip.id}
              jobId={clip.transcription?.transcription_job_id}
              speakers={clip.transcription?.local_speakers ?? []}
              onChanged={onChanged}
              segments={segments}
              playhead={playhead}
              onSeek={seek}
            />
          </Tabs.Panel>
          <Tabs.Panel
            value="voices"
            className="max-h-[70vh] overflow-y-auto outline-none"
          >
            <VoicesTab voice={voiceData} />
          </Tabs.Panel>
          <Tabs.Panel
            value="history"
            className="max-h-[70vh] overflow-y-auto outline-none"
          >
            <HistoryTab
              clipId={clip.id}
              segments={segments}
              jobId={clip.transcription?.transcription_job_id}
              onChanged={onChanged}
            />
          </Tabs.Panel>
        </Tabs.Root>
      </div>
    </section>
  );
}

/** The page of one clip: the player and the transcript side by side, with the clip's progress. */
export function ClipPage() {
  const { t, error } = useI18n();
  const { id } = useParams();
  const clipId = /^[1-9]\d*$/.test(id ?? "") ? Number(id) : 0;
  const clip = useLoaded(
    useCallback(
      () =>
        clipId > 0
          ? fetchClip(clipId)
          : Promise.resolve({ ok: false as const, code: "invalid_clip_id" }),
      [clipId],
    ),
  );
  useDocumentTitle(t("clip.back"));

  if (clip.state.status === "loading") {
    return (
      <p className="p-6 text-mute" role="status">
        {t("clip.loading")}
      </p>
    );
  }
  if (clip.state.status === "failed") {
    return (
      <section className="flex flex-col gap-4 px-4 py-6 md:px-7">
        <h1 className="text-[22px] font-semibold">{t("clip.back")}</h1>
        <ErrorAlert>{error(clip.state.code)}</ErrorAlert>
        <div>
          <Link to="/library" className="text-accent-ink underline">
            {t("clip.back")}
          </Link>
        </div>
      </section>
    );
  }
  return (
    <ClipView
      key={clip.state.value.id}
      clip={clip.state.value}
      onChanged={clip.reload}
    />
  );
}
