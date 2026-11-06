import { Search } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { useLocation, useNavigate } from "react-router";
import { useI18n } from "../i18n";

/** The search field of the header: submitting opens the search screen. */
export function HeaderSearch() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const location = useLocation();
  const [query, setQuery] = useState("");

  // On the search screen the field shows what is being searched.
  useEffect(() => {
    if (location.pathname === "/search") {
      setQuery(new URLSearchParams(location.search).get("q") ?? "");
    }
  }, [location.pathname, location.search]);

  function submit(event: FormEvent) {
    event.preventDefault();
    const trimmed = query.trim();
    const current = new URLSearchParams(location.search);
    const params = new URLSearchParams();
    for (const key of [
      "from",
      "to",
      "source",
      "visibility",
      "owner",
      "person",
      "state",
      // The Library filter rides along so it is still there on return.
      "filter",
    ]) {
      const value = current.get(key);
      if (value) params.set(key, value);
    }
    if (trimmed) params.set("q", trimmed);
    else params.delete("q");
    params.delete("page");
    navigate(
      `/search${params.size ? `?${params.toString().replace(/\+/g, "%20")}` : ""}`,
    );
  }

  return (
    <form
      role="search"
      onSubmit={submit}
      className="flex h-11 min-w-0 max-w-[460px] flex-1 items-center gap-2 rounded-lg border border-line-strong px-3 text-mute focus-within:border-accent md:h-9"
    >
      <Search className="size-4 shrink-0" aria-hidden="true" />
      <input
        type="search"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        aria-label={t("header.searchLabel")}
        placeholder={t("header.searchPlaceholder")}
        className="h-full min-h-11 min-w-0 flex-1 bg-transparent text-ink outline-none placeholder:text-mute"
      />
    </form>
  );
}
