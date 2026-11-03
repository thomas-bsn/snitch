import {
  Bot,
  CircleDashed,
  CircleHelp,
  PencilLine,
  UserCheck,
  type LucideIcon,
} from "lucide-react";
import type { MessageKey } from "../i18n/messages";
import type { TranscriptSegment } from "./libraryApi";

/** How far a segment's speaker is settled, as the reader sees it. */
export type SpeakerState =
  | "identified"
  | "suggestion"
  | "unknown"
  | "corrected"
  | "automatic";

export const SPEAKER_LOOKS: Record<
  SpeakerState,
  { icon: LucideIcon; label: MessageKey; className: string }
> = {
  identified: {
    icon: UserCheck,
    label: "clip.speaker.identified",
    className: "bg-accent-soft text-accent-ink",
  },
  suggestion: {
    icon: CircleHelp,
    label: "clip.speaker.suggestion",
    className: "bg-warn-bg text-warn-fg",
  },
  unknown: {
    icon: CircleDashed,
    label: "clip.speaker.unknown",
    className: "bg-side text-mute",
  },
  corrected: {
    icon: PencilLine,
    label: "clip.speaker.corrected",
    className: "bg-ok-bg text-ok-fg",
  },
  automatic: {
    icon: Bot,
    label: "clip.speaker.automatic",
    className: "bg-accent-soft text-accent-ink",
  },
};

/** The state of a segment's speaker, or null when the clip has no voice for it. */
export function speakerState(
  segment: Pick<
    TranscriptSegment,
    | "local_speaker_id"
    | "local_speaker_label"
    | "local_speaker_corrected"
    | "local_speaker_decision_level"
  >,
): SpeakerState | null {
  if (segment.local_speaker_id === undefined && !segment.local_speaker_label) {
    return null;
  }
  if (segment.local_speaker_corrected) return "corrected";
  switch (segment.local_speaker_decision_level) {
    case "human":
      return "identified";
    case "automatic":
      return "automatic";
    case "suggestion":
      return "suggestion";
    default:
      return "unknown";
  }
}

/** Whether a voice still waits for a decision: not set aside, not settled by a person. */
export function voiceToDecide(voice: {
  disposition?: string | null;
  decision_level: string;
}): boolean {
  return !voice.disposition && voice.decision_level !== "human";
}
