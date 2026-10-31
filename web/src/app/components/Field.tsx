import { useId, type InputHTMLAttributes } from "react";

type FieldProps = {
  label: string;
  hint?: string;
} & Omit<InputHTMLAttributes<HTMLInputElement>, "id">;

/** A labelled text field: the label and the hint are tied to the input. */
export function Field({ label, hint, ...input }: FieldProps) {
  const id = useId();
  const hintId = `${id}-hint`;
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="font-medium">
        {label}
      </label>
      <input
        {...input}
        id={id}
        aria-describedby={hint ? hintId : undefined}
        className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink read-only:bg-side read-only:text-mute md:h-10"
      />
      {hint && (
        <p id={hintId} className="text-xs text-mute">
          {hint}
        </p>
      )}
    </div>
  );
}
