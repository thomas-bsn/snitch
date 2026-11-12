import type { MessageKey, Params } from "../i18n/messages";
import type { AppNotification } from "./notificationsApi";

type Translate = (key: MessageKey, params?: Params) => string;

/** The sentence that tells what a notification is about, shared by the bell and the page. */
export function notificationText(item: AppNotification, t: Translate): string {
  const clip = item.clip_name;
  if (item.kind === "account_request") {
    const name = item.account_name ?? "";
    if (item.state === "active")
      return t("notifications.accountRequest", { name });
    return item.account_outcome === "approved"
      ? t("notifications.accountApproved", { name })
      : t("notifications.accountRefused");
  }
  if (item.kind === "folder_request_decided")
    return item.folder_outcome === "approved"
      ? t("notifications.folderApproved", { path: `/${item.folder_path}` })
      : t("notifications.folderRefused", { path: `/${item.folder_path}` });
  if (item.generic)
    return item.state === "resolved"
      ? t("notifications.incidentResolved")
      : t("notifications.incidentGeneric");
  if (item.kind !== "needs_speaker_review") {
    if (item.state === "resolved") return t("notifications.incidentResolved");
    const count = item.pending_count;
    const step = item.step;
    if (item.kind === "job_blocked")
      return item.error_code === "transcription_not_validated"
        ? t("notifications.incidentBlockedNotValidated", { count })
        : item.error_code === "worker_not_claiming"
          ? t("notifications.incidentBlockedNotClaiming", { count })
          : t("notifications.incidentBlockedMismatch", { count });
    return item.kind === "worker_unavailable"
      ? t("notifications.incidentWorker", { count })
      : item.kind === "provider_unavailable"
        ? t("notifications.incidentProvider", { count })
        : item.kind === "model_missing"
          ? t("notifications.incidentModel", { count })
          : t("notifications.incidentJob", { count, step });
  }
  if (item.state === "resolved")
    return t("notifications.speakerReviewResolved", { clip });
  return item.pending_count === 1
    ? t("notifications.speakerReviewOne", { clip })
    : t("notifications.speakerReview", { clip, count: item.pending_count });
}

/** The label of the direct link of an active action, or undefined when there is none. */
export function notificationActionLabel(
  item: AppNotification,
): MessageKey | undefined {
  if (item.state !== "active" || item.generic) return undefined;
  if (item.kind === "needs_speaker_review") return "notifications.actionVerify";
  if (item.kind === "account_request")
    return "notifications.accountRequestAction";
  return item.kind === "worker_unavailable" || item.kind === "job_blocked"
    ? "notifications.incidentWorkerAction"
    : "notifications.incidentJobAction";
}

/** An incident (technical error) rather than a voice to verify. */
export const isError = (item: AppNotification) =>
  item.kind !== "needs_speaker_review" &&
  item.kind !== "folder_request_decided" &&
  item.kind !== "account_request";
