import { Dialog } from "@base-ui/react/dialog";
import { X } from "lucide-react";
import type { ReactNode } from "react";

type ModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** Accessible name of the close icon. */
  closeLabel: string;
  children: ReactNode;
  /** A centered window (default), or a panel sliding in from the right (wide for a grid). */
  variant?: "center" | "side" | "wide";
};

/**
 * A modal window with a form or content inside: focus is trapped, Escape closes it and the
 * title is its accessible name. For a plain yes/no question use ConfirmDialog.
 */
export function Modal({
  open,
  onOpenChange,
  title,
  closeLabel,
  children,
  variant = "center",
}: ModalProps) {
  const side = variant !== "center";
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Viewport
          className={
            side
              ? "fixed inset-0 z-50 flex justify-end"
              : "fixed inset-0 z-50 flex items-center justify-center p-4"
          }
        >
          <Dialog.Popup
            className={
              side
                ? `flex h-full w-full ${variant === "wide" ? "max-w-3xl" : "max-w-md"} flex-col gap-4 overflow-y-auto border-l border-line bg-surface p-5 text-ink shadow-pop outline-none`
                : "flex max-h-full w-full max-w-lg flex-col gap-4 overflow-y-auto rounded-xl border border-line bg-surface p-5 text-ink shadow-pop outline-none"
            }
          >
            <div className="flex items-start justify-between gap-3">
              <Dialog.Title className="text-lg font-semibold">
                {title}
              </Dialog.Title>
              <Dialog.Close
                aria-label={closeLabel}
                className="flex size-11 shrink-0 items-center justify-center rounded-md hover:bg-side md:size-8"
              >
                <X className="size-4" aria-hidden="true" />
              </Dialog.Close>
            </div>
            {children}
          </Dialog.Popup>
        </Dialog.Viewport>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
