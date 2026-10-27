import { AlertDialog } from "@base-ui/react/alert-dialog";
import type { ReactNode } from "react";
import { buttonClass } from "./buttons";

type ConfirmDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  children: ReactNode;
  cancelLabel: string;
  confirmLabel: string;
  onConfirm: () => void;
  /** A destructive action gets the warning look. */
  danger?: boolean;
  busy?: boolean;
  confirmDisabled?: boolean;
};

/**
 * A modal confirmation: focus is trapped, Escape cancels, and the title and the text are
 * the accessible name and description of the dialog.
 */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  children,
  cancelLabel,
  confirmLabel,
  onConfirm,
  danger,
  busy,
  confirmDisabled,
}: ConfirmDialogProps) {
  return (
    <AlertDialog.Root open={open} onOpenChange={onOpenChange}>
      <AlertDialog.Portal>
        <AlertDialog.Backdrop className="fixed inset-0 z-40 bg-black/40" />
        <AlertDialog.Viewport className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <AlertDialog.Popup className="flex w-full max-w-md flex-col gap-3 rounded-xl border border-line bg-surface p-5 text-ink shadow-pop outline-none">
            <AlertDialog.Title className="text-lg font-semibold">
              {title}
            </AlertDialog.Title>
            <AlertDialog.Description
              render={<div />}
              className="flex flex-col gap-2 text-mute"
            >
              {children}
            </AlertDialog.Description>
            <div className="mt-2 flex justify-end gap-2">
              <AlertDialog.Close className={buttonClass("secondary")}>
                {cancelLabel}
              </AlertDialog.Close>
              <button
                type="button"
                disabled={busy || confirmDisabled}
                onClick={onConfirm}
                className={buttonClass(danger ? "danger" : "primary")}
              >
                {confirmLabel}
              </button>
            </div>
          </AlertDialog.Popup>
        </AlertDialog.Viewport>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}
