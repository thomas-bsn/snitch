import { useState } from "react";
import {
  Download,
  History,
  LockKeyhole,
  Server,
  Settings2,
  ShieldCheck,
  Waves,
} from "lucide-react";
import { useSearchParams } from "react-router";
import type { components } from "../../../api.gen";
import { useI18n } from "../../i18n";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";
import { Modal } from "../../components/Modal";
import { ModelsByWorker, type Row } from "./ModelsByWorker";
import { ApplyToWorkers } from "./ApplyToWorkers";
import { OrderHistory, OrderHistoryContent } from "./OrderHistory";
import { VoiceCatalogue, VoiceReferenceDetails } from "./VoiceCatalogue";
import { VoiceModelAdd } from "./VoiceModelAdd";
import { VoiceHuggingFace } from "./VoiceHuggingFace";
import { VoiceHeader } from "./VoiceHeader";
import { VoiceAdvanced } from "./VoiceAdvanced";
import { VoiceInventory } from "./VoiceInventory";
import { FingerprintTransition } from "./FingerprintTransition";
import type { Catalogue } from "./VoiceModelCards";
import { modelKinds, installKinds } from "./VoiceModelUI";
import "./voice-workspace.css";

type Props = {
  catalogue: Catalogue;
  workers: Row[];
  overview: components["schemas"]["VoiceOverview"];
  pipeline: components["schemas"]["VoicePipeline"];
  matching: components["schemas"]["VoiceMatchingSettings"];
  accuracy: components["schemas"]["VoiceAccuracy"];
  onChanged: () => void;
};
/** The selected B layout: Worker matrix, reference catalogue and order history share one workspace. */
export function VoiceWorkspace({
  catalogue,
  workers,
  overview,
  pipeline,
  matching,
  accuracy,
  onChanged,
}: Props) {
  const { t } = useI18n();
  const { user } = useSession();
  const canEdit = user !== null && isSuperAdministrator(user);
  const [params, setParams] = useSearchParams();
  const view = params.get("view");
  const tab = view === "catalogue" || view === "history" ? view : "workers";
  const [selected, setSelected] = useState<string[]>([]);
  const [applying, setApplying] = useState<{ modelId?: string }>();
  const [details, setDetails] = useState<string>();
  const [settings, setSettings] = useState<"identification" | "access">();
  const [transition, setTransition] = useState<string>();
  const [historyWorker, setHistoryWorker] = useState("");
  const [history, setHistory] = useState<{ workerId?: string }>();
  const [orderCount, setOrderCount] = useState<number>();
  function changeTab(value: string) {
    setParams((p) => {
      if (value === "workers") p.delete("view");
      else p.set("view", value);
      return p;
    });
  }
  const openHistory = (id = "") => setHistory({ workerId: id });
  const chosen = catalogue.models.find((m) => m.id === details);
  return (
    <div className="voice-workspace">
      <div className="voice-heading">
        <div>
          <h2>{t("voiceWorkspace.title")}</h2>
          <p>{t("voiceWorkspace.lead")}</p>
        </div>
        <div className="voice-heading-actions">
          <button
            type="button"
            className="voice-btn"
            onClick={() => openHistory()}
          >
            <History size={16} aria-hidden />
            {t("voiceWorkspace.history")}
          </button>
          <button
            type="button"
            className="voice-btn"
            onClick={() => setApplying({})}
          >
            <Server size={16} aria-hidden />
            {t("voiceModels.byWorker.applyMany")}
          </button>
          {canEdit && (
            <VoiceModelAdd
              onChanged={() => {
                changeTab("catalogue");
                onChanged();
              }}
            />
          )}
        </div>
      </div>
      <div
        className="voice-tabs"
        role="tablist"
        aria-label={t("voiceAdmin.title")}
        onKeyDown={(e) => {
          const keys = ["workers", "catalogue", "history"];
          const index = keys.indexOf(tab);
          const next =
            e.key === "ArrowRight"
              ? (index + 1) % 3
              : e.key === "ArrowLeft"
                ? (index + 2) % 3
                : e.key === "Home"
                  ? 0
                  : e.key === "End"
                    ? 2
                    : -1;
          if (next < 0) return;
          e.preventDefault();
          changeTab(keys[next]);
          document.getElementById(`voice-tab-${keys[next]}`)?.focus();
        }}
      >
        {(["workers", "catalogue", "history"] as const).map((key) => (
          <button
            type="button"
            key={key}
            id={`voice-tab-${key}`}
            role="tab"
            aria-selected={tab === key}
            aria-controls={`voice-panel-${key}`}
            tabIndex={tab === key ? 0 : -1}
            onClick={() => changeTab(key)}
          >
            {t(`voiceWorkspace.${key}`)}
            <span>
              {key === "workers"
                ? workers.length
                : key === "catalogue"
                  ? catalogue.models.length
                  : (orderCount ?? "—")}
            </span>
          </button>
        ))}
      </div>
      <div className="voice-production">
        <span>
          <ShieldCheck size={16} aria-hidden />
          {t("voiceWorkspace.production")}
        </span>
        {modelKinds.map((kind) => (
          <div key={kind}>
            <span>{t(`voiceModels.byWorker.type.${kind}`)}</span>
            <strong>
              {catalogue.models.find((m) => m.kind === kind && m.active)
                ?.label ||
                catalogue.models.find((m) => m.kind === kind && m.active)
                  ?.repo ||
                (kind === "separation"
                  ? t("voiceWorkspace.unavailable")
                  : overview.active_models[kind])}
            </strong>
          </div>
        ))}
      </div>
      <section
        role="tabpanel"
        id={`voice-panel-${tab}`}
        aria-labelledby={`voice-tab-${tab}`}
        tabIndex={0}
      >
        {tab === "workers" ? (
          <>
            <div className="voice-toolbar">
              <p>
                {selected.length
                  ? t("voiceWorkspace.selected", { count: selected.length })
                  : t("voiceWorkspace.desiredHint")}
              </p>
              <button
                type="button"
                className="voice-btn primary"
                onClick={() => setApplying({})}
              >
                <Download size={15} aria-hidden />
                {t("voiceModels.byWorker.applyMany")}
              </button>
            </div>
            <ModelsByWorker
              catalogue={catalogue}
              workers={workers}
              selected={selected}
              onSelected={setSelected}
              onChanged={onChanged}
              onHistory={openHistory}
              onInspect={setDetails}
              onTransition={setTransition}
            />
            <div className="voice-notice">
              <Waves size={18} aria-hidden />
              {t("voiceWorkspace.separationHint")}
            </div>
          </>
        ) : tab === "catalogue" ? (
          <VoiceCatalogue
            catalogue={catalogue}
            onInspect={setDetails}
            onDeploy={(modelId) => setApplying({ modelId })}
          />
        ) : null}
        <div hidden={tab !== "history"}>
          <OrderHistoryContent
            workers={workers}
            workerId={historyWorker}
            onWorkerChange={setHistoryWorker}
            onCount={setOrderCount}
          />
        </div>
      </section>
      <FingerprintTransition
        catalogue={catalogue}
        target={transition}
        onClose={() => setTransition(undefined)}
        onChanged={onChanged}
        onPrepare={(modelId) => setApplying({ modelId })}
      />
      <div className="voice-footer">
        <span>
          <ShieldCheck size={16} aria-hidden />
          {t("voiceWorkspace.productionNote")}
        </span>
        <button type="button" onClick={() => setSettings("identification")}>
          {t("voiceWorkspace.settings")}
          <Settings2 size={16} aria-hidden />
        </button>
        <button type="button" onClick={() => setSettings("access")}>
          {t("voiceWorkspace.access")}
          <LockKeyhole size={16} aria-hidden />
        </button>
      </div>
      {history && (
        <OrderHistory
          key={history.workerId ?? "all"}
          open
          workers={workers}
          initialWorkerId={history.workerId}
          onClose={() => setHistory(undefined)}
        />
      )}
      {applying && (
        <ApplyToWorkers
          open
          onClose={() => {
            setApplying(undefined);
            onChanged();
          }}
          workers={workers}
          catalogue={catalogue}
          kinds={installKinds}
          initialModelId={applying.modelId}
          initialWorkerIds={selected}
          onChanged={onChanged}
        />
      )}
      {chosen && (
        <VoiceReferenceDetails
          key={chosen.id}
          model={chosen}
          canEdit={canEdit}
          onClose={() => setDetails(undefined)}
          onChanged={onChanged}
          onDeploy={() => {
            setDetails(undefined);
            setApplying({ modelId: chosen.id });
          }}
          onTransition={() => {
            setDetails(undefined);
            setTransition(chosen.id);
          }}
        />
      )}
      <Modal
        open={settings !== undefined}
        onOpenChange={(open) => !open && setSettings(undefined)}
        title={t(
          settings === "access"
            ? "voiceWorkspace.access"
            : "voiceWorkspace.settings",
        )}
        closeLabel={t("voiceModels.byWorker.close")}
        variant="side"
      >
        {settings === "access" ? (
          <VoiceHuggingFace />
        ) : settings === "identification" ? (
          <>
            <VoiceHeader
              overview={overview}
              accuracy={accuracy}
              catalogue={catalogue}
              onChanged={onChanged}
            />
            <VoiceAdvanced
              pipeline={pipeline}
              matching={matching}
              accuracy={accuracy}
              onChanged={onChanged}
            />
            <VoiceInventory overview={overview} />
          </>
        ) : null}
      </Modal>
    </div>
  );
}
