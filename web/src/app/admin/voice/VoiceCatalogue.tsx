import { useEffect, useState, type FormEvent } from "react";
import {
  ChevronRight,
  Download,
  LockKeyhole,
  MoreHorizontal,
  Pencil,
  Search,
} from "lucide-react";
import { api, errorCode, mutation } from "../../api/client";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { ErrorAlert } from "../../components/Alert";
import { Modal } from "../../components/Modal";
import { Field } from "../../components/Field";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { buttonClass } from "../../components/buttons";
import type { Catalogue } from "./VoiceModelCards";
import { KindIcon, ModelBadges, modelKinds, type Model } from "./VoiceModelUI";
const mb = (n: number) => Math.round(n / 1_000_000);

export function VoiceCatalogue({
  catalogue,
  onInspect,
  onDeploy,
}: {
  catalogue: Catalogue;
  onInspect: (id: string) => void;
  onDeploy: (id: string) => void;
}) {
  const { t } = useI18n();
  const [kind, setKind] = useState("all");
  const [search, setSearch] = useState("");
  const models = catalogue.models.filter(
    (m) =>
      (kind === "all" || m.kind === kind) &&
      `${m.label} ${m.repo}`
        .toLocaleLowerCase()
        .includes(search.toLocaleLowerCase()),
  );
  return (
    <>
      <div className="voice-catalogue-tools">
        <div
          className="voice-filter"
          role="group"
          aria-label={t("voiceWorkspace.catalogue")}
        >
          {["all", ...modelKinds].map((key) => (
            <button
              type="button"
              key={key}
              aria-pressed={kind === key}
              onClick={() => setKind(key)}
            >
              {key === "all"
                ? t("voiceWorkspace.all")
                : t(`voiceModels.byWorker.type.${key as Model["kind"]}`)}
              <span>
                {key === "all"
                  ? catalogue.models.length
                  : catalogue.models.filter((m) => m.kind === key).length}
              </span>
            </button>
          ))}
        </div>
        <label className="voice-search">
          <Search size={16} aria-hidden />
          <span className="sr-only">{t("voiceWorkspace.search")}</span>
          <input
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("voiceWorkspace.searchPlaceholder")}
          />
        </label>
      </div>
      <div className="voice-catalogue-grid">
        {models.map((m) => {
          const count = m.workers.filter((w) => w.state === "ready").length;
          return (
            <article
              className={`voice-reference ${!m.supported ? "unsupported" : ""}`}
              key={m.id}
              aria-label={m.repo}
            >
              <div className="voice-reference-head">
                <span className="voice-ref-icon">
                  <KindIcon kind={m.kind} />
                </span>
                <div>
                  <small>{t(`voiceModels.byWorker.type.${m.kind}`)}</small>
                  <h4>
                    <button type="button" onClick={() => onInspect(m.id)}>
                      {m.label || m.repo}
                    </button>
                  </h4>
                </div>
                <button
                  type="button"
                  className="voice-icon-btn"
                  aria-label={t("voiceWorkspace.detailsOf", {
                    model: m.label || m.repo,
                  })}
                  onClick={() => onInspect(m.id)}
                >
                  <MoreHorizontal size={19} aria-hidden />
                </button>
              </div>
              <ModelBadges model={m} />
              <p className="voice-repo">{m.repo}</p>
              <div className="voice-ref-facts">
                <span>
                  {m.revision.slice(0, 7)}{" "}
                  <LockKeyhole className="inline" size={11} aria-hidden />
                </span>
                <span>
                  {mb(m.size_bytes)} {t("voiceWorkspace.volumeUnit")}
                </span>
                <span>
                  {t(
                    m.gated
                      ? "voiceWorkspace.restricted"
                      : "voiceWorkspace.openAccess",
                  )}
                </span>
                {m.separated_voices && m.sample_rate_hz && (
                  <span>
                    {t("voiceWorkspace.separationLimits", {
                      voices: m.separated_voices,
                      rate: m.sample_rate_hz / 1000,
                    })}
                  </span>
                )}
              </div>
              <p>
                {!m.supported
                  ? m.unsupported_reason
                  : count
                    ? t("voiceWorkspace.workerCount", { count })
                    : t("voiceWorkspace.notPresent")}
              </p>
              <div className="voice-card-footer">
                <button
                  type="button"
                  className="voice-btn quiet"
                  onClick={() => onInspect(m.id)}
                >
                  {t("voiceWorkspace.details")}
                  <ChevronRight size={14} aria-hidden />
                </button>
                <button
                  type="button"
                  className="voice-btn"
                  disabled={!m.supported || !m.enabled}
                  onClick={() => onDeploy(m.id)}
                >
                  {t(
                    count ? "voiceWorkspace.deploy" : "voiceWorkspace.install",
                  )}
                  <Download size={14} aria-hidden />
                </button>
              </div>
            </article>
          );
        })}
      </div>
      {models.length === 0 && (
        <div className="voice-empty">{t("voiceWorkspace.noMatch")}</div>
      )}
    </>
  );
}

/** Catalogue metadata is editable only for super-administrators; identity and adapter stay pinned. */
export function VoiceReferenceDetails({
  model,
  canEdit,
  onClose,
  onChanged,
  onDeploy,
  onTransition,
}: {
  model: Model;
  canEdit: boolean;
  onClose: () => void;
  onChanged: () => void;
  onDeploy: () => void;
  onTransition: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState({
    label: model.label,
    license: model.license,
    size: String(model.size_bytes / 1_000_000),
    vram: String(model.vram_bytes / 1_000_000),
    enabled: model.enabled,
    family: model.family ?? "",
  });
  // The families come from the server table; the model may only take one that covers its task.
  const [families, setFamilies] = useState<string[]>([]);
  useEffect(() => {
    if (!editing || !canEdit) return;
    let current = true;
    api
      .GET("/api/admin/voice-engines/families")
      .then((r) => {
        if (current && r.data)
          setFamilies(
            r.data.families
              .filter((f) => f.kinds.includes(model.kind))
              .map((f) => f.family),
          );
      })
      .catch(() => {});
    return () => {
      current = false;
    };
  }, [editing, canEdit, model.kind]);
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [activating, setActivating] = useState(false);
  async function run(action: () => Promise<{ error?: unknown }>) {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await action();
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        setEditing(false);
        setActivating(false);
        onChanged();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  function save(e: FormEvent) {
    e.preventDefault();
    void run(() =>
      api.PATCH("/api/admin/voice-models/{id}", {
        params: {
          path: { id: model.id },
          header: mutation(session?.csrf_token),
        },
        body: {
          label: draft.label,
          license: draft.license,
          size_bytes: Math.round(Number(draft.size) * 1_000_000),
          vram_bytes: Math.round(Number(draft.vram) * 1_000_000),
          enabled: draft.enabled,
          // Only sent when changed: an unchanged form never touches the engine contract of the model.
          ...(draft.family !== (model.family ?? "")
            ? { family: draft.family }
            : {}),
        },
      }),
    );
  }
  return (
    <Modal
      open
      onOpenChange={(open) => !open && !busy && onClose()}
      title={editing ? t("voiceWorkspace.edit") : model.label || model.repo}
      closeLabel={t("voiceModels.byWorker.close")}
      variant="side"
    >
      <ModelBadges model={model} />
      <p className="voice-repo">{model.repo}</p>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {editing ? (
        <form className="voice-form" onSubmit={save}>
          <Field
            label={t("voiceWorkspace.label")}
            value={draft.label}
            maxLength={100}
            required
            onChange={(e) => setDraft({ ...draft, label: e.target.value })}
          />
          <Field
            label={t("voiceWorkspace.licence")}
            value={draft.license}
            maxLength={100}
            onChange={(e) => setDraft({ ...draft, license: e.target.value })}
          />
          <div className="voice-form-grid">
            <Field
              label={t("voiceWorkspace.size")}
              type="number"
              min={0}
              step="any"
              required
              value={draft.size}
              onChange={(e) => setDraft({ ...draft, size: e.target.value })}
            />
            <Field
              label={t("voiceWorkspace.vram")}
              type="number"
              min={0}
              step="any"
              required
              value={draft.vram}
              onChange={(e) => setDraft({ ...draft, vram: e.target.value })}
            />
          </div>
          <label className="grid gap-1">
            {t("voiceWorkspace.family")}
            <select
              className="snitch-select"
              value={draft.family}
              onChange={(e) => setDraft({ ...draft, family: e.target.value })}
            >
              <option value="">{t("voiceWorkspace.familyNone")}</option>
              {families.map((f) => (
                <option key={f} value={f}>
                  {f}
                </option>
              ))}
            </select>
            <span className="text-sm text-mute">
              {t("voiceWorkspace.familyHint")}
            </span>
          </label>
          <label className="flex min-h-11 items-center gap-2">
            <input
              type="checkbox"
              checked={draft.enabled}
              onChange={(e) =>
                setDraft({ ...draft, enabled: e.target.checked })
              }
            />
            {t("voiceWorkspace.enabled")}
          </label>
          <p className="text-sm text-mute">{t("voiceWorkspace.editHint")}</p>
          <div className="flex flex-wrap gap-2">
            <button
              type="submit"
              className={buttonClass("primary")}
              disabled={busy}
            >
              {t("voiceWorkspace.save")}
            </button>
            <button
              type="button"
              className={buttonClass("secondary")}
              disabled={busy}
              onClick={() => setEditing(false)}
            >
              {t("voiceModels.cancelDialog")}
            </button>
          </div>
        </form>
      ) : (
        <>
          <dl className="voice-modal-facts">
            {[
              [
                t("voiceModels.byWorker.applyType"),
                t(`voiceModels.byWorker.type.${model.kind}`),
              ],
              [
                t("voiceWorkspace.engine"),
                model.engine || t("voiceWorkspace.unavailable"),
              ],
              [t("voiceWorkspace.revision"), model.revision],
              [
                t("voiceWorkspace.licence"),
                model.license || t("voiceWorkspace.unknown"),
              ],
              [t("voiceWorkspace.size"), mb(model.size_bytes)],
              [
                t("voiceWorkspace.vram"),
                model.vram_bytes
                  ? mb(model.vram_bytes)
                  : t("voiceWorkspace.unknown"),
              ],
              ...(model.kind === "separation"
                ? [
                    [
                      t("voiceWorkspace.separatedVoices"),
                      model.separated_voices ?? t("voiceWorkspace.unknown"),
                    ],
                    [
                      t("voiceWorkspace.sampleRate"),
                      model.sample_rate_hz
                        ? t("voiceWorkspace.sampleRateValue", {
                            rate: model.sample_rate_hz / 1000,
                          })
                        : t("voiceWorkspace.unknown"),
                    ],
                  ]
                : []),
            ].map(([key, value]) => (
              <div key={key}>
                <dt>{key}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
          {!model.supported && (
            <p className="text-sm text-mute">{model.unsupported_reason}</p>
          )}
          <p className="text-sm text-mute">
            {t("voiceWorkspace.catalogueHint")}
          </p>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className={buttonClass("primary")}
              disabled={!model.enabled || !model.supported}
              onClick={onDeploy}
            >
              {t("voiceWorkspace.deploy")}
            </button>
            {canEdit && (
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => setEditing(true)}
              >
                <Pencil size={15} aria-hidden />
                {t("voiceWorkspace.edit")}
              </button>
            )}
          </div>
          {model.kind === "fingerprint" &&
            !model.active &&
            model.supported &&
            model.enabled && (
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={onTransition}
              >
                {t("voiceWorkspace.transition")}
              </button>
            )}
          {model.kind === "diarization" &&
            model.supported &&
            model.enabled &&
            !model.active && (
              <div className="flex flex-wrap gap-2">
                {!model.candidate && (
                  <button
                    type="button"
                    className={buttonClass("secondary")}
                    disabled={busy}
                    onClick={() =>
                      void run(() =>
                        api.PUT("/api/admin/voice-models/candidate", {
                          params: { header: mutation(session?.csrf_token) },
                          body: { model_id: model.id },
                        }),
                      )
                    }
                  >
                    {t("voiceModels.choose")}
                  </button>
                )}
                <button
                  type="button"
                  className={buttonClass("secondary")}
                  disabled={
                    busy ||
                    !model.workers.some(
                      (w) => w.connected && w.state === "ready",
                    )
                  }
                  onClick={() => setActivating(true)}
                >
                  {t("voiceModels.activate")}
                </button>
                {model.candidate && (
                  <button
                    type="button"
                    className={buttonClass("secondary")}
                    disabled={busy}
                    onClick={() =>
                      void run(() =>
                        api.DELETE("/api/admin/voice-models/candidate", {
                          params: { header: mutation(session?.csrf_token) },
                        }),
                      )
                    }
                  >
                    {t("voiceModels.cancel")}
                  </button>
                )}
              </div>
            )}
          <ConfirmDialog
            open={activating}
            onOpenChange={setActivating}
            title={t("voiceModels.confirmTitle", { model: model.repo })}
            cancelLabel={t("voiceModels.cancelDialog")}
            confirmLabel={t("voiceModels.confirmActivate")}
            busy={busy}
            onConfirm={() =>
              void run(() =>
                api.POST("/api/admin/voice-models/{id}/activate", {
                  params: {
                    path: { id: model.id },
                    header: mutation(session?.csrf_token),
                  },
                }),
              )
            }
          >
            <p>{t("voiceModels.effectVersion")}</p>
            <p>{t("voiceModels.effectRunning")}</p>
            <p>{t("voiceModels.effectClips")}</p>
            <p>
              {t("voiceModels.effectReady", {
                names:
                  model.workers
                    .filter((w) => w.state === "ready")
                    .map((w) => w.name)
                    .join(", ") || "—",
              })}
            </p>
            {model.workers.some((w) => w.state !== "ready") && (
              <p>
                {t("voiceModels.effectNotReady", {
                  names: model.workers
                    .filter((w) => w.state !== "ready")
                    .map((w) => w.name)
                    .join(", "),
                })}
              </p>
            )}
            {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          </ConfirmDialog>
        </>
      )}
    </Modal>
  );
}
