import type { ReactNode } from "react";
import {
  AudioLines,
  Fingerprint,
  Waves,
  Check,
  CircleAlert,
} from "lucide-react";
import type { components } from "../../../api.gen";
import type { Kind } from "./ModelsByWorker";
import { useI18n } from "../../i18n";
export const modelKinds: Kind[] = ["diarization", "fingerprint", "separation"];
export const installKinds: Kind[] = ["diarization", "fingerprint"];
export type Model = components["schemas"]["VoiceModel"];
export function KindIcon({ kind }: { kind: Kind }) {
  const Icon =
    kind === "diarization"
      ? AudioLines
      : kind === "fingerprint"
        ? Fingerprint
        : Waves;
  return <Icon size={18} aria-hidden />;
}
export function VoiceBadge({
  children,
  tone = "",
}: {
  children: ReactNode;
  tone?: string;
}) {
  return (
    <span className={`voice-badge ${tone}`}>
      {tone === "ok" ? (
        <Check size={12} aria-hidden />
      ) : tone === "bad" || tone === "warn" ? (
        <CircleAlert size={12} aria-hidden />
      ) : null}
      {children}
    </span>
  );
}
export function ModelBadges({ model }: { model: Model }) {
  const { t } = useI18n();
  return (
    <div className="voice-inline">
      {model.active && (
        <VoiceBadge tone="ok">{t("voiceModels.active")}</VoiceBadge>
      )}
      {model.candidate && (
        <VoiceBadge tone="accent">{t("voiceModels.candidate")}</VoiceBadge>
      )}
      {!model.supported && (
        <VoiceBadge tone="warn">{t("voiceWorkspace.unsupported")}</VoiceBadge>
      )}
      {!model.enabled && (
        <VoiceBadge>{t("voiceWorkspace.disabled")}</VoiceBadge>
      )}
      {model.supported &&
        model.enabled &&
        !model.active &&
        !model.candidate && (
          <VoiceBadge>{t("voiceWorkspace.supported")}</VoiceBadge>
        )}
    </div>
  );
}
