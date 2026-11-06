import { useId } from "react";
import { buttonClass } from "../components/buttons";
import { DateField, Select } from "../components/FilterFields";
import { useI18n } from "../i18n";
import { EXPLORER_STAGES, EXPLORER_STATES } from "./explorerApi";
import type { ClipFilterOptions, Person } from "./libraryApi";

/** The explorer's filters, kept in the address. */
export type ExplorerFilters = {
  q: string;
  state: string;
  /** Processing stage: keeps the videos that reached it, and the folders that lead to them. */
  stage: string;
  game: string;
  from: string;
  to: string;
  visibility: string;
  owner: string;
  person: string;
  /** "pending" keeps the videos with voices to process. */
  voices: string;
};

/** The library filters: game, date, visibility, owner and state. Person and voices to process. */
export function LibraryFilters({
  filters,
  options,
  persons,
  active,
  onChange,
  onClear,
}: {
  filters: ExplorerFilters;
  options: ClipFilterOptions | null;
  persons: Person[];
  active: boolean;
  onChange: (patch: Partial<ExplorerFilters>) => void;
  onClear: () => void;
}) {
  const { t } = useI18n();
  const gameId = useId();
  return (
    <form
      role="search"
      aria-label={t("library.filters.label")}
      onSubmit={(event) => event.preventDefault()}
      className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6"
    >
      <p className="col-span-full text-sm text-mute">
        {t("metadata.dateFilterZone")}
      </p>
      <div className="flex min-w-0 flex-col gap-1">
        <label htmlFor={gameId} className="text-xs font-medium text-mute">
          {t("metadata.game")}
        </label>
        <input
          id={gameId}
          value={filters.game}
          onChange={(event) => onChange({ game: event.target.value })}
          className="h-11 w-full min-w-0 rounded-lg border border-line-strong bg-surface px-2.5 text-ink md:h-9"
        />
      </div>
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
        {persons.map((person) => (
          <option key={person.id} value={String(person.id)}>
            {person.display_name}
          </option>
        ))}
      </Select>
      <Select
        label={t("library.filters.voices")}
        value={filters.voices}
        onChange={(voices) => onChange({ voices })}
      >
        <option value="">{t("library.filters.any")}</option>
        <option value="pending">{t("library.filters.voicesPending")}</option>
      </Select>
      <Select
        label={t("library.filters.state")}
        value={filters.state}
        onChange={(state) => onChange({ state })}
      >
        <option value="">{t("library.filters.any")}</option>
        {EXPLORER_STATES.map((state) => (
          <option key={state} value={state}>
            {t(`library.explorer.state.${state}`)}
          </option>
        ))}
      </Select>
      <Select
        label={t("library.filters.stage")}
        value={filters.stage}
        onChange={(stage) => onChange({ stage })}
      >
        <option value="">{t("library.filters.stages.all")}</option>
        {EXPLORER_STAGES.map((stage) => (
          <option key={stage} value={stage}>
            {t(`library.filters.stages.${stage}`)}
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
