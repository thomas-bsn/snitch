import { ChevronDown } from "lucide-react";
import { useId, useState } from "react";
import { buttonClass } from "../components/buttons";
import { DateField, Select } from "../components/FilterFields";
import { useI18n } from "../i18n";
import {
  CLIP_STATES,
  type ClipFilterOptions,
  type Person,
} from "../library/libraryApi";
import type { SearchFilters } from "./searchApi";

/** The filters of the search screen: a rail on desktop, a collapsed panel on a phone. */
export function SearchForm({
  filters,
  options,
  persons,
  active,
  onChange,
  onClear,
}: {
  filters: SearchFilters;
  options: ClipFilterOptions | null;
  persons: Person[];
  active: boolean;
  onChange: (patch: Partial<SearchFilters>) => void;
  onClear: () => void;
}) {
  const { t } = useI18n();
  const panelId = useId();
  const [open, setOpen] = useState(false);

  return (
    <aside className="min-w-0 rounded-xl border border-line bg-surface lg:sticky lg:top-4">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((value) => !value)}
        className="flex min-h-11 w-full items-center justify-between gap-2 px-3.5 text-sm font-semibold text-ink lg:hidden"
      >
        {t("search.filters.heading")}
        <ChevronDown
          aria-hidden="true"
          className={`size-4 text-mute transition-transform ${
            open ? "rotate-180" : ""
          }`}
        />
      </button>
      <form
        id={panelId}
        aria-label={t("search.filters.label")}
        onSubmit={(event) => event.preventDefault()}
        className={`${open ? "flex" : "hidden"} flex-col gap-2.5 p-3.5 lg:flex`}
      >
        <h2 className="m-0 hidden text-[10.5px] font-semibold uppercase tracking-[0.08em] text-mute lg:block">
          {t("search.filters.heading")}
        </h2>
        <p className="text-xs text-mute">{t("metadata.dateFilterZone")}</p>
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
        <button
          type="button"
          onClick={onClear}
          disabled={!active}
          className={`${buttonClass("secondary")} mt-1 w-full`}
        >
          {t("library.filters.clear")}
        </button>
      </form>
    </aside>
  );
}
