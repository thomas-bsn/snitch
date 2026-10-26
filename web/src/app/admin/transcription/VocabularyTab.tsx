import { useState, type FormEvent } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

type Term = components["schemas"]["VocabularyTerm"];
async function load() {
  try {
    const { data, error } = await api.GET("/api/admin/vocabulary");
    return data
      ? ({ ok: true, value: data } as const)
      : ({ ok: false, code: errorCode(error) ?? "unknown" } as const);
  } catch {
    return { ok: false, code: "unavailable" } as const;
  }
}
export function VocabularyTab() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  const [editing, setEditing] = useState<Term>();
  const [term, setTerm] = useState("");
  const [variants, setVariants] = useState("");
  const [preferred, setPreferred] = useState("");
  const [active, setActive] = useState(true);
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  function reset() {
    setEditing(undefined);
    setTerm("");
    setVariants("");
    setPreferred("");
    setActive(true);
  }
  async function save(event: FormEvent) {
    event.preventDefault();
    setFailure(undefined);
    setBusy(true);
    try {
      const body = {
        term: term.trim(),
        variants: variants
          .split(",")
          .map((v) => v.trim())
          .filter(Boolean),
        preferred_spelling: preferred.trim(),
        active,
      };
      const result = editing
        ? await api.PUT("/api/admin/vocabulary/{id}", {
            params: {
              path: { id: editing.id },
              header: mutation(session?.csrf_token),
            },
            body,
          })
        : await api.POST("/api/admin/vocabulary", {
            params: { header: mutation(session?.csrf_token) },
            body,
          });
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unknown");
        return;
      }
      reset();
      reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function remove(id: number) {
    setFailure(undefined);
    setBusy(true);
    try {
      const result = await api.DELETE("/api/admin/vocabulary/{id}", {
        params: { path: { id }, header: mutation(session?.csrf_token) },
      });
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unknown");
        return;
      }
      reset();
      reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("vocabAdmin.title")}</h2>
      <p className="text-mute">
        {t("vocabAdmin.version", { version: state.value.version })}
      </p>
      <form
        onSubmit={(event) => void save(event)}
        className="flex max-w-xl flex-col gap-3"
      >
        <Field
          label={t("vocabAdmin.term")}
          required
          maxLength={160}
          value={term}
          onChange={(event) => setTerm(event.target.value)}
        />
        <Field
          label={t("vocabAdmin.variants")}
          hint={t("vocabAdmin.variantsHint")}
          maxLength={4000}
          value={variants}
          onChange={(event) => setVariants(event.target.value)}
        />
        <Field
          label={t("vocabAdmin.preferred")}
          required
          maxLength={160}
          value={preferred}
          onChange={(event) => setPreferred(event.target.value)}
        />
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={active}
            onChange={(event) => setActive(event.target.checked)}
          />
          {t("vocabAdmin.active")}
        </label>
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        <div className="flex gap-2">
          <button disabled={busy} className={buttonClass("primary")}>
            {editing ? t("vocabAdmin.save") : t("vocabAdmin.add")}
          </button>
          {editing && (
            <button type="button" className={buttonClass()} onClick={reset}>
              {t("vocabAdmin.cancel")}
            </button>
          )}
        </div>
      </form>
      <ul className="flex flex-col gap-2">
        {state.value.items.map((item) => (
          <li key={item.id} className="rounded-lg border border-line p-3">
            <p className="font-medium">{item.preferred_spelling}</p>
            <p>{item.term}</p>
            <p>{item.variants.join(" · ")}</p>
            <p>
              {item.active ? t("vocabAdmin.active") : t("vocabAdmin.inactive")}
            </p>
            <div className="mt-2 flex gap-2">
              <button
                type="button"
                disabled={busy}
                className={buttonClass()}
                onClick={() => {
                  setEditing(item);
                  setTerm(item.term);
                  setVariants(item.variants.join(", "));
                  setPreferred(item.preferred_spelling);
                  setActive(item.active);
                }}
              >
                {t("vocabAdmin.edit")}
              </button>
              <button
                type="button"
                className={buttonClass("danger")}
                onClick={() => void remove(item.id)}
              >
                {t("vocabAdmin.remove")}
              </button>
            </div>
          </li>
        ))}
      </ul>
      {state.value.items.length === 0 && <p>{t("vocabAdmin.empty")}</p>}
    </section>
  );
}
