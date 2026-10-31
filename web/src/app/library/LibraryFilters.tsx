import { buttonClass } from "../components/buttons";
import { DateField, Select } from "../components/FilterFields";
import { useI18n } from "../i18n";
import {
  CLIP_STATES,
  type ClipFilterOptions,
  type LibraryFilters as Filters,
  type Person,
} from "./libraryApi";

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
        label={t("search.filters.visibility")}
        value={filters.visibility}
        onChange={(visibility) => onChange({ visibility })}
      >
        <option value="">{t("library.filters.any")}</option>
        <option value="shared">{t("search.filters.shared")}</option>
        <option value="private">{t("search.filters.private")}</option>
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
