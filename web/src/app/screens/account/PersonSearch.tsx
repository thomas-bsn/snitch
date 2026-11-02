import { useEffect, useState } from "react";
import { ErrorAlert } from "../../components/Alert";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { fetchPersonPage, type Person } from "../../people/peopleApi";

const SEARCH_DELAY_MS = 250;
const SEARCH_SIZE = 8;

const optionClass = (active: boolean) =>
  `flex min-h-11 w-full items-center rounded-lg border px-3 py-2 text-left text-sm ${
    active ? "border-accent bg-accent-soft text-accent-ink" : "border-line"
  }`;

type Found =
  | { status: "loading" }
  | { status: "failed"; code: string }
  | { status: "ready"; items: Person[]; total: number };

/**
 * Searching a Person by name: the server filters and counts, after a short pause in the
 * typing, and only the first page is shown. When more people match than are shown, the
 * message tells to type more of the name.
 */
export function PersonSearch({
  selected,
  onSelect,
}: {
  selected: number | undefined;
  onSelect: (person: Person) => void;
}) {
  const { t, error } = useI18n();
  const [query, setQuery] = useState("");
  const [found, setFound] = useState<Found>({ status: "loading" });

  useEffect(() => {
    let current = true;
    setFound({ status: "loading" });
    const timer = setTimeout(() => {
      void fetchPersonPage(query.trim(), 1, SEARCH_SIZE).then((result) => {
        if (!current) return;
        setFound(
          result.ok
            ? {
                status: "ready",
                items: result.value.items,
                total: result.value.total,
              }
            : { status: "failed", code: result.code },
        );
      });
    }, SEARCH_DELAY_MS);
    return () => {
      current = false;
      clearTimeout(timer);
    };
  }, [query]);

  return (
    <div className="flex flex-col gap-3">
      <Field
        type="search"
        label={t("account.voice.linkSearch")}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        maxLength={100}
        autoComplete="off"
      />
      {found.status === "failed" && (
        <ErrorAlert>{error(found.code)}</ErrorAlert>
      )}
      {found.status === "loading" && (
        <p className="text-mute">{t("account.voice.linkSearching")}</p>
      )}
      {found.status === "ready" && (
        <>
          <ul
            aria-label={t("account.voice.linkSearch")}
            className="flex list-none flex-col gap-1 p-0"
          >
            {found.items.map((person) => (
              <li key={person.id}>
                <button
                  type="button"
                  aria-pressed={selected === person.id}
                  className={optionClass(selected === person.id)}
                  onClick={() => onSelect(person)}
                >
                  {person.display_name}
                </button>
              </li>
            ))}
          </ul>
          {found.items.length === 0 && (
            <p className="text-mute">{t("account.voice.linkNoMatches")}</p>
          )}
          {found.total > found.items.length && (
            <p className="text-sm text-mute">
              {t("account.voice.linkMore", {
                shown: found.items.length,
                total: found.total,
              })}
            </p>
          )}
        </>
      )}
    </div>
  );
}
