/** A field where typing text is expected: the shortcuts leave it alone. */
export function isTextField(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  if (target instanceof HTMLTextAreaElement) return true;
  if (target instanceof HTMLSelectElement) return true;
  return (
    target instanceof HTMLInputElement &&
    !["radio", "checkbox", "button", "submit"].includes(target.type)
  );
}

/** Space already activates a focused control: the shortcut leaves those alone. */
export function isInteractive(target: EventTarget | null) {
  return (
    target instanceof HTMLElement &&
    target.closest("button, a, summary, [role=button], input") !== null
  );
}
