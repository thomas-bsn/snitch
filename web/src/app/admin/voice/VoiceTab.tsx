import { useEffect } from "react";
import { api, errorCode } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { useI18n } from "../../i18n";
import { VoiceWorkspace } from "./VoiceWorkspace";

async function load() {
  try {
    const [overview, models, eligibility, matching, accuracy, workers] =
      await Promise.all([
        api.GET("/api/admin/voice-overview"),
        api.GET("/api/admin/voice-models"),
        api.GET("/api/admin/voice-eligibility"),
        api.GET("/api/voice-matching"),
        api.GET("/api/admin/voice-accuracy"),
        api.GET("/api/admin/voice-models/workers"),
      ]);
    if (
      overview.data &&
      models.data &&
      eligibility.data &&
      matching.data &&
      accuracy.data &&
      workers.data
    )
      return {
        ok: true as const,
        value: {
          overview: overview.data,
          models: models.data,
          eligibility: eligibility.data,
          matching: matching.data,
          accuracy: accuracy.data,
          workers: workers.data,
        },
      };
    return {
      ok: false as const,
      code:
        errorCode(
          overview.error ??
            models.error ??
            eligibility.error ??
            matching.error ??
            accuracy.error ??
            workers.error,
        ) ?? "unknown",
    };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

/** Shared live data for the Worker matrix, catalogue and history. */
export function VoiceTab() {
  const { t, error } = useI18n();
  const { state, reload } = useLoaded(load);
  const preparing =
    state.status === "ready" &&
    state.value.workers.workers.some((w) =>
      w.models.some((m) =>
        [
          "requested",
          "downloading",
          "verifying",
          "switching",
          "removing",
        ].includes(m.state),
      ),
    );
  useEffect(() => {
    const timer = window.setInterval(reload, preparing ? 3000 : 10000);
    return () => window.clearInterval(timer);
  }, [preparing, reload]);
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  const { overview, models, eligibility, matching, accuracy, workers } =
    state.value;
  return (
    <VoiceWorkspace
      catalogue={models}
      workers={workers.workers}
      overview={overview}
      eligibility={eligibility}
      matching={matching}
      accuracy={accuracy}
      onChanged={reload}
    />
  );
}
