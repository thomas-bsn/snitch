import { Combobox } from "@base-ui/react/combobox";
import { Check, ChevronDown } from "lucide-react";
import { useId, useMemo } from "react";
import { useI18n } from "../../i18n";
import {
  whisperLanguageOptions,
  type WhisperLanguageOption,
} from "./whisperLanguages";

type WhisperLanguageFieldProps = {
  label: string;
  /** The Whisper code, or "" when none is chosen yet. */
  value: string;
  onChange: (code: string) => void;
  required?: boolean;
};

/**
 * The library language: a searchable list of the languages Whisper supports, named in the
 * interface language with their code. Searching matches the name and the code; the value
 * stays the Whisper code.
 */
export function WhisperLanguageField({
  label,
  value,
  onChange,
  required,
}: WhisperLanguageFieldProps) {
  const { t, language } = useI18n();
  const id = useId();
  const options = useMemo(
    () => whisperLanguageOptions(language, value),
    [language, value],
  );
  const selected = options.find((option) => option.value === value) ?? null;

  return (
    <Combobox.Root<WhisperLanguageOption>
      items={options}
      value={selected}
      onValueChange={(option) => {
        if (option) onChange(option.value);
      }}
      isItemEqualToValue={(item, current) => item.value === current.value}
      required={required}
      locale={language}
      autoHighlight
    >
      <div className="flex flex-col gap-1.5">
        <label htmlFor={id} className="font-medium">
          {label}
        </label>
        <Combobox.InputGroup className="relative">
          <Combobox.Input
            id={id}
            placeholder={t("transAdmin.languageSearch")}
            className="h-11 w-full rounded-lg border border-line-strong bg-surface pl-3 pr-11 text-ink placeholder:text-mute md:h-10"
          />
          <Combobox.Trigger
            aria-label={t("transAdmin.languageShowAll")}
            className="absolute inset-y-0 right-0 flex w-11 items-center justify-center rounded-r-lg text-mute"
          >
            <ChevronDown aria-hidden className="size-4" />
          </Combobox.Trigger>
        </Combobox.InputGroup>
      </div>
      <Combobox.Portal>
        <Combobox.Positioner sideOffset={4} className="z-50 outline-none">
          <Combobox.Popup className="w-[var(--anchor-width)] max-w-[var(--available-width)] rounded-xl border border-line bg-surface p-1.5 text-ink shadow-pop">
            <Combobox.Empty>
              <p className="px-2.5 py-2 text-sm text-mute">
                {t("transAdmin.languageEmpty")}
              </p>
            </Combobox.Empty>
            <Combobox.List className="max-h-[min(20rem,var(--available-height))] overflow-y-auto overscroll-contain outline-none data-[empty]:hidden">
              {(option: WhisperLanguageOption) => (
                <Combobox.Item
                  key={option.value}
                  value={option}
                  className="flex min-h-11 cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-2 text-sm outline-none data-[highlighted]:bg-side md:min-h-9"
                >
                  <span className="flex size-4 shrink-0 items-center">
                    <Combobox.ItemIndicator>
                      <Check aria-hidden className="size-4 text-accent-ink" />
                    </Combobox.ItemIndicator>
                  </span>
                  {option.label}
                </Combobox.Item>
              )}
            </Combobox.List>
          </Combobox.Popup>
        </Combobox.Positioner>
      </Combobox.Portal>
    </Combobox.Root>
  );
}
