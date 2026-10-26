import { useId, type ReactNode } from "react";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import {
  CLIP_STATES,
  type ClipFilterOptions,
  type LibraryFilters as Filters,
  type Person,
} from "./libraryApi";

const control =
  "h-11 w-full min-w-0 rounded-lg border border-line-strong bg-surface px-2.5 text-ink md:h-9";

function Select({
  label,
  value,
  onChange,
  children,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <label htmlFor={id} className="text-xs font-medium text-mute">
        {label}
      </label>
      <select
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className={control}
      >
        {children}
      </select>
    </div>
  );
}

function DateField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const id = useId();
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <label htmlFor={id} className="text-xs font-medium text-mute">
        {label}
      </label>
      <input
        id={id}
        type="date"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className={control}
      />
    </div>
  );
}

/** The five filters of the library (date, Source, owner, Person, state), kept in the address. */
export function LibraryFilters({
  filters,
  options,
  persons,
  active,
  onChange,
  onClear,
}: {
  filters: Filters;
  options: ClipFilterOptions | null;
  persons: Person[];
  active: boolean;
  onChange: (patch: Partial<Filters>) => void;
  onClear: () => void;
}) {
  const { t } = useI18n();
  return (
    <form
      role="search"
      aria-label={t("library.filters.label")}
      onSubmit={(event) => event.preventDefault()}
      className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-7"
    >
      <DateField
        label={t("library.filters.from")}
        value={filters.from}
        onChange={(from) => onChange({ from })}
      />
      <DateField
        label={t("library.filters.to")}
        value={filters.to}
        onChange={(to) => onChange({ to })}
      />
      <Select
        label={t("library.filters.source")}
        value={filters.source}
        onChange={(source) => onChange({ source })}
      >
        <option value="">{t("library.filters.any")}</option>
        {options?.sources.map((source) => (
          <option key={source.id} value={String(source.id)}>
            {source.name}
          </option>
        ))}
      </Select>
      <Select
        label={t("library.filters.owner")}
        value={filters.owner}
        onChange={(owner) => onChange({ owner })}
      >
        <option value="">{t("library.filters.any")}</option>
        {options?.owners.map((owner) => (
          <option key={owner.id} value={String(owner.id)}>
            {owner.username}
          </option>
        ))}
      </Select>
      <Select
        label={t("library.filters.person")}
        value={filters.person}
        onChange={(person) => onChange({ person })}
      >
        <option value="">{t("library.filters.any")}</option>
        <option value="unknown">{t("library.filters.unknownVoices")}</option>
        {persons.map((person) => (
          <option key={person.id} value={String(person.id)}>
            {person.display_name}
          </option>
        ))}
      </Select>
      <Select
        label={t("library.filters.state")}
        value={filters.state}
        onChange={(state) => onChange({ state })}
      >
        <option value="">{t("library.filters.any")}</option>
        {CLIP_STATES.map((state) => (
          <option key={state} value={state}>
            {t(`library.state.${state}`)}
          </option>
        ))}
      </Select>
      <div className="flex items-end">
        <button
          type="button"
          onClick={onClear}
          disabled={!active}
          className={`${buttonClass("secondary")} w-full`}
        >
          {t("library.filters.clear")}
        </button>
      </div>
    </form>
  );
}
