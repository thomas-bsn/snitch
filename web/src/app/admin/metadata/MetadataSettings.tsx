import { useEffect, useState } from "react";
import { api, mutation } from "../../api/client";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import zones from "./zones.json";

const countries = zones as Record<string, string[]>;
const allZones = Array.from(new Set(Object.values(countries).flat())).sort();

export function MetadataSettings({
  onSaved,
  reprocess = false,
}: {
  onSaved?: () => void;
  reprocess?: boolean;
}) {
  const { t, language } = useI18n();
  const { session } = useSession();
  const [country, setCountry] = useState("");
  const [zone, setZone] = useState("");
  const [savedLocation, setSavedLocation] = useState({
    country: "",
    timezone: "",
  });
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    void api
      .GET("/api/admin/metadata-location")
      .then(({ data }) => {
        if (data) {
          setCountry(data.country);
          setZone(data.timezone);
          setSavedLocation(data);
        }
      })
      .catch(() => setMessage(t("metadata.error")));
  }, [t]);
  const names = new Intl.DisplayNames([language], { type: "region" });
  const options = Object.keys(countries).sort((a, b) =>
    (names.of(a) ?? a).localeCompare(names.of(b) ?? b, language),
  );
  async function save() {
    setBusy(true);
    setMessage("");
    try {
      const { response } = await api.PUT("/api/admin/metadata-location", {
        params: { header: mutation(session?.csrf_token) },
        body: { country, timezone: zone },
      });
      if (!response.ok) throw new Error();
      setSavedLocation({ country, timezone: zone });
      setMessage(t("metadata.saved"));
      onSaved?.();
    } catch {
      setMessage(t("metadata.error"));
    } finally {
      setBusy(false);
    }
  }
  async function run() {
    if (!window.confirm(t("metadata.confirm"))) return;
    setBusy(true);
    setMessage("");
    try {
      let after = 0;
      let updated = 0;
      let unavailable = 0;
      while (true) {
        const { response, data } = await api.POST(
          "/api/admin/clip-metadata/reprocess",
          {
            params: { header: mutation(session?.csrf_token), query: { after } },
          },
        );
        if (!response.ok || !data) throw new Error();
        updated += data.updated;
        unavailable += data.unavailable;
        setMessage(t("metadata.result", { updated, unavailable }));
        if (data.done) break;
        if (data.next <= after) throw new Error();
        after = data.next;
      }
    } catch {
      setMessage(t("metadata.error"));
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="flex max-w-xl flex-col gap-4">
      {reprocess && (
        <h2 className="text-lg font-semibold">{t("metadata.title")}</h2>
      )}
      <p>{t("metadata.explain")}</p>
      <label className="flex flex-col gap-1">
        {t("metadata.country")}
        <select
          value={country}
          onChange={(e) => {
            const value = e.target.value;
            setCountry(value);
            setZone(countries[value]?.[0] ?? "");
          }}
          className="rounded border border-line bg-canvas p-2"
        >
          <option value="">{t("metadata.choose")}</option>
          {options.map((c) => (
            <option key={c} value={c}>
              {names.of(c) ?? c}
            </option>
          ))}
        </select>
      </label>
      <label className="flex flex-col gap-1">
        {t("metadata.timezone")}
        <select
          value={zone}
          onChange={(e) => setZone(e.target.value)}
          className="rounded border border-line bg-canvas p-2"
          disabled={!country}
        >
          {Array.from(
            new Set([...(countries[country] ?? []), ...allZones]),
          ).map((z) => (
            <option key={z} value={z}>
              {z}
            </option>
          ))}
        </select>
      </label>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          className={buttonClass("primary")}
          onClick={() => void save()}
          disabled={busy || !country || !zone}
        >
          {t("metadata.save")}
        </button>
        {reprocess && (
          <button
            type="button"
            className={buttonClass()}
            onClick={() => void run()}
            disabled={
              busy ||
              !zone ||
              country !== savedLocation.country ||
              zone !== savedLocation.timezone
            }
          >
            {t("metadata.reprocess")}
          </button>
        )}
      </div>
      {message && <p role="status">{message}</p>}
    </section>
  );
}
