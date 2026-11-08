import { useState, type FormEvent } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";
import { useSession } from "../../session";

type Inspection = components["schemas"]["VoiceModelInspection"];
type Kind = Exclude<components["schemas"]["VoiceModelAdd"]["kind"], "vad">;

const kinds: Kind[] = ["diarization", "fingerprint", "separation"];
const kindKeys: Record<Kind, MessageKey> = {
  diarization: "voiceModels.add.kind.diarization",
  fingerprint: "voiceModels.add.kind.fingerprint",
  separation: "voiceModels.add.kind.separation",
};

/**
 * Assisted addition of a Hugging Face repository (Super-administrator). One field; Snitch inspects the public
 * metadata, proposes a type only when it is reliable, and derives the engine itself: the engine is shown, never
 * asked. A repository without an approved adapter is explained and cannot be added, whatever type is chosen.
 */
export function VoiceModelAdd({ onChanged }: { onChanged: () => void }) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const headers = mutation(session?.csrf_token);
  const [open, setOpen] = useState(false);
  const [source, setSource] = useState("");
  const [inspection, setInspection] = useState<Inspection>();
  const [kind, setKind] = useState<Kind>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [added, setAdded] = useState<string>();

  function reset() {
    setSource("");
    setInspection(undefined);
    setKind(undefined);
    setFailure(undefined);
  }

  async function inspect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFailure(undefined);
    setInspection(undefined);
    setKind(undefined);
    try {
      const result = await api.POST("/api/admin/voice-models/inspection", {
        params: { header: headers },
        body: { source },
      });
      if (result.data) {
        setInspection(result.data);
        setKind((result.data.proposed_kind as Kind | null) ?? undefined);
      } else setFailure(errorCode(result.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  const kindSupported =
    inspection !== undefined &&
    inspection.supported &&
    kind !== undefined &&
    inspection.compatible_kinds.includes(kind);

  async function add() {
    if (!inspection || !kind || !kindSupported) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.POST("/api/admin/voice-models", {
        params: { header: headers },
        body: { source, revision: inspection.revision, kind },
      });
      if (result.data) {
        setAdded(result.data.label || result.data.repo);
        setOpen(false);
        reset();
        onChanged();
      } else setFailure(errorCode(result.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div>
        <button
          type="button"
          className={buttonClass("primary")}
          onClick={() => {
            setAdded(undefined);
            setOpen(true);
          }}
        >
          {t("voiceModels.add.open")}
        </button>
      </div>
      {added && (
        <StatusNotice>
          {t("voiceModels.add.added", { model: added })}
        </StatusNotice>
      )}
      <Modal
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (!next) reset();
        }}
        title={t("voiceModels.add.title")}
        closeLabel={t("voiceModels.add.close")}
      >
        <form onSubmit={inspect} className="flex items-end gap-2">
          <div className="min-w-0 flex-1">
            <Field
              label={t("voiceModels.add.source")}
              hint={t("voiceModels.add.sourceHint")}
              value={source}
              onChange={(e) => setSource(e.target.value)}
              spellCheck={false}
              autoComplete="off"
              required
            />
          </div>
          <button
            type="submit"
            className={buttonClass("primary")}
            disabled={busy || source.trim() === ""}
          >
            {busy && !inspection
              ? t("voiceModels.add.inspecting")
              : t("voiceModels.add.inspect")}
          </button>
        </form>
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {inspection && (
          <div className="flex flex-col gap-3">
            {!inspection.supported ? (
              <ErrorAlert>
                {t("voiceModels.add.unsupported", {
                  reason: inspection.unsupported_reason,
                })}
              </ErrorAlert>
            ) : inspection.proposed_kind ? (
              <StatusNotice>
                {t("voiceModels.add.proposed", {
                  kind: t(kindKeys[inspection.proposed_kind as Kind]),
                })}
              </StatusNotice>
            ) : (
              <StatusNotice>{t("voiceModels.add.uncertain")}</StatusNotice>
            )}
            {inspection.existing_ids.length > 0 && (
              <StatusNotice>
                {t("voiceModels.add.existing", {
                  ids: inspection.existing_ids.join(", "),
                })}
              </StatusNotice>
            )}
            <div
              role="group"
              aria-label={t("voiceModels.add.typeLabel")}
              className="flex flex-wrap gap-2"
            >
              {kinds.map((k) => (
                <button
                  key={k}
                  type="button"
                  aria-pressed={kind === k}
                  onClick={() => setKind(k)}
                  className={`min-h-11 min-w-28 flex-1 rounded-lg border px-3 text-sm font-medium md:min-h-10 ${
                    kind === k
                      ? "border-accent-solid bg-accent-soft text-accent-ink"
                      : "border-line-strong bg-surface text-ink hover:bg-side"
                  }`}
                >
                  {t(kindKeys[k])}
                </button>
              ))}
            </div>
            {inspection.supported && kind && !kindSupported && (
              <ErrorAlert>
                {t("voiceModels.add.kindUnsupported", {
                  kind: t(kindKeys[kind]),
                })}
              </ErrorAlert>
            )}
            <dl
              aria-label={t("voiceModels.add.recap")}
              className="rounded-lg border border-line px-3 text-sm"
            >
              {[
                [t("voiceModels.add.model"), inspection.repo],
                [
                  t("voiceModels.add.type"),
                  kind ? t(kindKeys[kind]) : t("voiceModels.add.typeToChoose"),
                ],
                [
                  t("voiceModels.add.engine"),
                  inspection.engine
                    ? inspection.engine === "nemo"
                      ? t("voiceModels.add.engines.nemo")
                      : inspection.engine === "pyannote"
                        ? t("voiceModels.add.engines.pyannote")
                        : inspection.engine
                    : t("voiceModels.add.engineNone"),
                ],
                [
                  t("voiceModels.add.revision"),
                  `${inspection.revision.slice(0, 7)}…`,
                ],
                [
                  t("voiceModels.add.license"),
                  inspection.license || t("voiceModels.add.licenseNone"),
                ],
                [
                  t("voiceModels.add.gated"),
                  inspection.gated
                    ? t("voiceModels.add.gatedYes")
                    : t("voiceModels.add.gatedNo"),
                ],
              ].map(([label, value]) => (
                <div
                  key={label}
                  className="flex justify-between gap-3 border-b border-line py-2 last:border-b-0"
                >
                  <dt className="text-mute">{label}</dt>
                  <dd className="break-all text-right font-medium">{value}</dd>
                </div>
              ))}
            </dl>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => {
                  setOpen(false);
                  reset();
                }}
              >
                {t("voiceModels.add.cancel")}
              </button>
              <button
                type="button"
                className={buttonClass("primary")}
                disabled={busy || !kindSupported}
                onClick={() => void add()}
              >
                {t("voiceModels.add.submit")}
              </button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
