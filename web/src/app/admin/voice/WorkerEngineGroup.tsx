import { useState, type FormEvent } from "react";
import { useI18n } from "../../i18n";

/** A named administrative group; saving membership never orders an engine. */
export function WorkerEngineGroup({
  name,
  busy,
  onSave,
}: {
  name: string;
  busy: boolean;
  onSave: (name: string) => Promise<void>;
}) {
  const { t } = useI18n();
  const [value, setValue] = useState(name);
  function submit(event: FormEvent) {
    event.preventDefault();
    void onSave(value.trim());
  }
  return (
    <form className="engine-bar" onSubmit={submit}>
      <label>
        {t("workerEngines.groupName")}
        <input
          className="engine-search"
          value={value}
          maxLength={80}
          onChange={(event) => setValue(event.target.value)}
        />
      </label>
      <button
        className="voice-btn"
        type="submit"
        disabled={busy || value.trim() === name}
      >
        {t("workerEngines.saveGroup")}
      </button>
    </form>
  );
}
