import { useEffect } from "react";
import { api, errorCode } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { useI18n } from "../../i18n";
import { VoiceAdvanced } from "./VoiceAdvanced";
import { VoiceHeader } from "./VoiceHeader";
import { VoiceInventory } from "./VoiceInventory";
import { VoiceModelCards } from "./VoiceModelCards";

async function load() {
  try {
    const [overview, models, pipeline, matching, accuracy] = await Promise.all([
      api.GET("/api/admin/voice-overview"),
      api.GET("/api/admin/voice-models"),
      api.GET("/api/admin/voice-pipeline"),
      api.GET("/api/voice-matching"),
      api.GET("/api/admin/voice-accuracy"),
    ]);
    if (
      overview.data &&
      models.data &&
      pipeline.data &&
      matching.data &&
      accuracy.data
    )
      return {
        ok: true as const,
        value: {
          overview: overview.data,
          models: models.data,
          pipeline: pipeline.data,
          matching: matching.data,
          accuracy: accuracy.data,
        },
      };
    return {
      ok: false as const,
      code:
        errorCode(
          overview.error ??
            models.error ??
            pipeline.error ??
            matching.error ??
            accuracy.error,
        ) ?? "unknown",
    };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

/** Administration, voice processing: state and models first, thresholds folded under "Advanced". */
export function VoiceTab() {
  const { t, error } = useI18n();
  const { state, reload } = useLoaded(load);
  const preparing =
    state.status === "ready" &&
    state.value.models.models.some((m) =>
      m.workers.some(
        (w) => w.state === "downloading" || w.state === "verifying",
      ),
    );
  useEffect(() => {
    if (!preparing) return;
    const timer = window.setInterval(reload, 3000);
    return () => window.clearInterval(timer);
  }, [preparing, reload]);
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  const { overview, models, pipeline, matching, accuracy } = state.value;
  return (
    <div className="flex max-w-4xl flex-col gap-6">
      <h2 className="text-lg font-semibold">{t("voiceAdmin.title")}</h2>
      <VoiceHeader
        overview={overview}
        accuracy={accuracy}
        catalogue={models}
        onChanged={reload}
      />
      <VoiceModelCards catalogue={models} onChanged={reload} />
      <VoiceInventory overview={overview} />
      <VoiceAdvanced
        pipeline={pipeline.config}
        matching={matching}
        accuracy={accuracy}
        onChanged={reload}
      />
    </div>
  );
}
