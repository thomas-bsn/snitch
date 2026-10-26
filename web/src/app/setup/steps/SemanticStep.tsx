import { useCallback } from "react";
import { api, errorCode } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { StatusPill } from "../../components/StatusPill";
import { useI18n } from "../../i18n";
import { StepHeader } from "../StepHeader";

async function fetchEmbeddingStatus() {
  const { data, error } = await api.GET("/api/admin/embeddings");
  return data
    ? ({ ok: true, value: data } as const)
    : ({ ok: false, code: errorCode(error) ?? "unknown" } as const);
}

/**
 * Step 4 (optional): semantic search. The embeddings service is part of the deployment but
 * the model is chosen in its variables, so this step shows where the instance stands and
 * what to do; full-text search works without it. Nothing is downloaded or indexed here.
 */
export function SemanticStep({
  onDone,
  skip,
}: {
  onDone: () => Promise<void>;
  skip: React.ReactNode;
}) {
  const { t, error } = useI18n();
  const { state } = useLoaded(useCallback(fetchEmbeddingStatus, []));
  const active = state.status === "ready" && state.value.model !== "";
  return (
    <div className="flex max-w-xl flex-col gap-5">
      <StepHeader
        position={4}
        total={7}
        title={t("setup.semantic.title")}
        lead={t("setup.semantic.lead")}
      />
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {state.status === "ready" && (
        <p className="flex flex-wrap items-center gap-2">
          <span className="font-medium">{t("setup.semantic.status")}</span>
          {active ? (
            <StatusPill tone="ok">
              {t("setup.semantic.active", { model: state.value.model })}
            </StatusPill>
          ) : (
            <StatusPill tone="idle">{t("setup.semantic.inactive")}</StatusPill>
          )}
        </p>
      )}
      {!active && (
        <div className="flex flex-col gap-2 rounded-lg border border-line bg-side p-4">
          <p>{t("setup.semantic.howTo")}</p>
          <code className="rounded-md bg-surface px-3 py-2 font-mono text-sm select-all">
            EMBEDDINGS_MODEL=nomic-embed-text
          </code>
          <code className="rounded-md bg-surface px-3 py-2 font-mono text-sm select-all">
            docker compose exec embed ollama pull nomic-embed-text
          </code>
          <p className="text-sm text-mute">{t("setup.semantic.howToHint")}</p>
        </div>
      )}
      <div className="flex flex-wrap gap-2">
        {skip}
        <button
          type="button"
          onClick={() => void onDone()}
          className={buttonClass("primary")}
        >
          {t("setup.continue")}
        </button>
      </div>
    </div>
  );
}
