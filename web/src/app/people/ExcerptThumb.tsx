import { Play } from "lucide-react";
import { Modal } from "../components/Modal";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { excerptUrl, type VerificationCase } from "./peopleApi";

// Decorative tiles: one of a few tints, picked from the voice so a case keeps its look.
const tints = [
  "bg-linear-to-br from-indigo-200 to-zinc-200 dark:from-indigo-900 dark:to-zinc-800",
  "bg-linear-to-br from-orange-200 to-stone-200 dark:from-orange-900 dark:to-stone-800",
  "bg-linear-to-br from-emerald-200 to-sky-100 dark:from-emerald-900 dark:to-sky-900",
  "bg-linear-to-br from-fuchsia-200 to-violet-100 dark:from-fuchsia-900 dark:to-violet-900",
];

/** The thumbnail of a case: a button that opens its excerpt. */
export function ExcerptThumb({
  item,
  onOpen,
}: {
  item: VerificationCase;
  onOpen: () => void;
}) {
  const { t } = useI18n();
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={t("people.verification.play", {
        label: item.label,
        clip: item.clip_name,
      })}
      className={`relative flex size-16 shrink-0 items-center justify-center rounded-lg ${tints[item.local_speaker_id % tints.length]} text-ink`}
    >
      <Play className="size-5" aria-hidden="true" />
    </button>
  );
}

/** The excerpt of a case, ready to play, with the way on to the decision. */
export function ExcerptDialog({
  item,
  onClose,
  onDecide,
}: {
  item: VerificationCase | undefined;
  onClose: () => void;
  onDecide: (item: VerificationCase) => void;
}) {
  const { t } = useI18n();
  return (
    <Modal
      open={item !== undefined}
      onOpenChange={(open) => !open && onClose()}
      title={t("people.excerpt.title", { label: item?.label ?? "" })}
      closeLabel={t("people.excerpt.close")}
    >
      {item && (
        <>
          <p className="text-mute">{item.clip_name}</p>
          <audio
            controls
            autoPlay
            aria-label={t("people.excerpt.audio", {
              label: item.label,
              clip: item.clip_name,
            })}
            src={excerptUrl(item.local_speaker_id)}
            className="w-full"
          />
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className={buttonClass("secondary")}
            >
              {t("people.excerpt.close")}
            </button>
            <button
              type="button"
              onClick={() => onDecide(item)}
              className={buttonClass("primary")}
            >
              {t("people.excerpt.decide")}
            </button>
          </div>
        </>
      )}
    </Modal>
  );
}
