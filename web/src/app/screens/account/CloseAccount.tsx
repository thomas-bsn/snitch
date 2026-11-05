import { useState } from "react";
import { useNavigate } from "react-router";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";
import { Section } from "./Section";

export function CloseAccount() {
  const { t, error } = useI18n();
  const { user, session, reload } = useSession();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [confirmation, setConfirmation] = useState("");
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  if (!user) return null;
  async function close() {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.DELETE("/api/account", {
        params: { header: mutation(session?.csrf_token) },
      });
      if (r.error) {
        setFailure(errorCode(r.error) ?? "unknown");
        return;
      }
      await reload();
      navigate("/login", { replace: true });
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  const allowed = ["delete", "supprimer"].includes(
    confirmation.trim().toLowerCase(),
  );
  return (
    <Section title={t("closeAccount.zone")} danger>
      <h3 className="font-semibold">{t("closeAccount.title")}</h3>
      <p className="text-sm leading-relaxed text-mute">
        {t("closeAccount.explanation")}
      </p>
      {isSuperAdministrator(user) ? (
        <p>{t("closeAccount.protected")}</p>
      ) : (
        <div>
          <button
            type="button"
            className={buttonClass("danger")}
            onClick={() => {
              setConfirmation("");
              setFailure(undefined);
              setOpen(true);
            }}
          >
            {t("closeAccount.open")}
          </button>
        </div>
      )}
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        title={t("closeAccount.title")}
        cancelLabel={t("closeAccount.cancel")}
        confirmLabel={t("closeAccount.confirm")}
        danger
        busy={busy}
        confirmDisabled={!allowed}
        onConfirm={() => void close()}
      >
        <p>{t("closeAccount.confirmHint")}</p>
        <Field
          label={t("closeAccount.type")}
          value={confirmation}
          onChange={(e) => setConfirmation(e.target.value)}
          autoComplete="off"
        />
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      </ConfirmDialog>
    </Section>
  );
}
