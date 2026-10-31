import { Search } from "lucide-react";
import { useEffect, useId, useState, type FormEvent } from "react";
import { buttonClass } from "../components/buttons";
import { DateField, Select } from "../components/FilterFields";
import { useI18n } from "../i18n";
import { CLIP_STATES, type ClipFilterOptions } from "../library/libraryApi";
import type { Person } from "../library/libraryApi";
import { SEARCH_MODES, type SearchFilters } from "./searchApi";

/** The query, the way of searching and the filters of the search screen. */
export function SearchForm({
  filters,
  options,
  persons,
  active,
  onSearch,
  onChange,
  onClear,
}: {
  filters: SearchFilters;
  options: ClipFilterOptions | null;
  persons: Person[];
  active: boolean;
  onSearch: (query: string) => void;
  onChange: (patch: Partial<SearchFilters>) => void;
  onClear: () => void;
}) {
  const { t } = useI18n();
  const queryId = useId();
  const hintId = useId();
  const [query, setQuery] = useState(filters.q);

  // The header field also searches: the box follows what the address says.
  useEffect(() => setQuery(filters.q), [filters.q]);

  function submit(event: FormEvent) {
    event.preventDefault();
    onSearch(query.trim());
  }

  return (
    <form
      role="search"
      aria-label={t("search.form.label")}
      onSubmit={submit}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1">
        <label htmlFor={queryId} className="text-xs font-medium text-mute">
          {t("search.form.query")}
        </label>
        <div className="flex gap-2">
          <input
            id={queryId}
            type="search"
            value={query}
            maxLength={200}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("search.form.placeholder")}
            aria-describedby={hintId}
            className="h-11 min-w-0 flex-1 rounded-lg border border-line-strong bg-surface px-3 text-ink placeholder:text-mute md:h-9"
          />
          <button type="submit" className={buttonClass("primary")}>
            <Search className="size-4" aria-hidden="true" />
            {t("search.form.submit")}
          </button>
        </div>
        <p id={hintId} className="text-xs text-mute">
          {t(`search.modes.${filters.mode}.hint`)}
        </p>
      </div>

      <fieldset className="m-0 flex min-w-0 flex-col gap-1 border-0 p-0">
        <legend className="mb-1 p-0 text-xs font-medium text-mute">
          {t("search.modes.label")}
        </legend>
        <div className="flex flex-wrap gap-2">
          {SEARCH_MODES.map((mode) => (
            <label
              key={mode}
              className="inline-flex min-h-11 cursor-pointer items-center rounded-lg border border-line-strong px-3 text-ink has-checked:border-accent has-checked:bg-accent-soft has-checked:text-accent-ink has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-accent md:min-h-9"
            >
              <input
                type="radio"
                name="search-mode"
                value={mode}
                checked={filters.mode === mode}
                onChange={() => onChange({ mode })}
                className="sr-only"
              />
              {t(`search.modes.${mode}.name`)}
            </label>
          ))}
        </div>
      </fieldset>

      <div
        role="group"
        aria-label={t("search.filters.label")}
        className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-7"
      >
        <p className="col-span-full text-sm text-mute">
          {t("metadata.dateFilterZone")}
        </p>
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
          label={t("search.filters.visibility")}
          value={filters.visibility}
          onChange={(visibility) => onChange({ visibility })}
        >
          <option value="">{t("library.filters.any")}</option>
          <option value="shared">{t("search.filters.shared")}</option>
          <option value="private">{t("search.filters.private")}</option>
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
      </div>
    </form>
  );
}
