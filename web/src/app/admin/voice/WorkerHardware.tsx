import { useI18n } from "../../i18n";
import type { EngineWorker } from "./engineOrders";

const gibibytes = (bytes: number) => (bytes / 1024 ** 3).toFixed(1);

/** What a Worker reported of its machine: the constraints an engine release is checked against. */
export function WorkerHardware({ worker }: { worker: EngineWorker }) {
  const { t } = useI18n();
  const hw = worker.hardware;
  if (!hw)
    return <p className="engine-note">{t("workerEngines.hardware.unknown")}</p>;
  return (
    <p
      className="engine-hardware"
      aria-label={t("workerEngines.hardware.title")}
    >
      <span>{t("workerEngines.hardware.python", { value: hw.python })}</span>
      <span>
        {t("workerEngines.hardware.platform", { value: hw.platform })}
      </span>
      <span>
        {hw.cuda
          ? t("workerEngines.hardware.cuda", { value: hw.cuda })
          : t("workerEngines.hardware.noCuda")}
      </span>
      <span>
        {t("workerEngines.hardware.ram", { value: gibibytes(hw.ram_bytes) })}
      </span>
      <span>
        {t("workerEngines.hardware.vram", { value: gibibytes(hw.vram_bytes) })}
      </span>
      <span>
        {t("workerEngines.hardware.disk", {
          value: gibibytes(hw.disk_free_bytes),
        })}
        {worker.disk_reserved_bytes > 0 &&
          ` (${t("workerEngines.hardware.reserved", { value: gibibytes(worker.disk_reserved_bytes) })})`}
      </span>
    </p>
  );
}
