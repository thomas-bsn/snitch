import { useCallback, useState, type ReactNode } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { Field } from "../../components/Field";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { useFormatDate } from "../format";

type Detail = components["schemas"]["MemberDetail"];
type Action = "reset" | "unlink" | "delete";

const selectClass =
  "h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10";

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <fieldset className="flex flex-col gap-3 border-0 p-0">
      <legend className="mb-1 text-xs font-semibold tracking-wide text-mute uppercase">
        {title}
      </legend>
      {children}
    </fieldset>
  );
}

type Props = {
  /** The member being edited; the panel is closed while it is undefined. */
  id: number | undefined;
  /** Name shown in the panel's title while the member is loading. */
  name: string;
  onClose: () => void;
  /** The list must be read again: something about a member changed. */
  onChanged: () => void;
};

export function MemberEditor({ id, name, onClose, onChanged }: Props) {
  const { t } = useI18n();
  return (
    <Modal
      open={id !== undefined}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      title={t("members.edit.title", { name })}
      closeLabel={t("members.edit.close")}
      variant="side"
    >
      {id !== undefined && (
        <Body key={id} id={id} onClose={onClose} onChanged={onChanged} />
      )}
    </Modal>
  );
}

function Body({
  id,
  onClose,
  onChanged,
}: {
  id: number;
  onClose: () => void;
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const format = useFormatDate();
  const load = useCallback(async () => {
    try {
      const r = await api.GET("/api/admin/members/{id}", {
        params: { path: { id } },
      });
      return r.data
        ? { ok: true as const, value: r.data }
        : { ok: false as const, code: errorCode(r.error) ?? "unknown" };
    } catch {
      return { ok: false as const, code: "unavailable" };
    }
  }, [id]);
  const { state, reload } = useLoaded(load);
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  return (
    <Form
      member={state.value}
      csrf={session?.csrf_token}
      format={format}
      onClose={onClose}
      onSaved={() => {
        reload();
        onChanged();
      }}
      onGone={() => {
        onChanged();
        onClose();
      }}
    />
  );
}

function Form({
  member,
  csrf,
  format,
  onClose,
  onSaved,
  onGone,
}: {
  member: Detail;
  csrf: string | undefined;
  format: (iso: string) => string;
  onClose: () => void;
  onSaved: () => void;
  onGone: () => void;
}) {
  const { t, error } = useI18n();
  const name = member.display_name || member.username;
  const [username, setUsername] = useState(member.username);
  const [displayName, setDisplayName] = useState(member.display_name ?? "");
  const [language, setLanguage] = useState(member.language ?? "");
  const [role, setRole] = useState(member.role);
  const [approved, setApproved] = useState(member.approved);
  const [action, setAction] = useState<Action>();
  const [hide, setHide] = useState(false);
  const [password, setPassword] = useState<string>();
  const [copied, setCopied] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [busy, setBusy] = useState(false);
  const { edit, manage, set_role: setRoleAllowed } = member.permissions;
  const patch: components["schemas"]["MemberPatch"] = {};
  if (username !== member.username) patch.username = username;
  if (displayName.trim() !== (member.display_name ?? ""))
    patch.display_name = displayName.trim() === "" ? null : displayName;
  if (language !== (member.language ?? ""))
    patch.language = language === "" ? null : (language as "en" | "fr");
  if (setRoleAllowed && role !== member.role && role !== "super_admin")
    patch.role = role;
  if (manage && approved !== member.approved) patch.approved = approved;
  const dirty = Object.keys(patch).length > 0;

  const path = { id: member.id };
  const header = mutation(csrf);
  async function run(
    call: () => Promise<{ error?: unknown }>,
    done: () => void,
  ) {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    try {
      const r = await call();
      if (r.error) {
        setFailure(errorCode(r.error) ?? "unknown");
        return;
      }
      done();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setAction(undefined);
    }
  }
  function save() {
    void run(
      () =>
        api.PATCH("/api/admin/members/{id}", {
          params: { path, header },
          body: patch,
        }),
      () => {
        setNotice(t("members.edit.saved"));
        onSaved();
      },
    );
  }
  function confirm() {
    if (action === "reset") {
      setBusy(true);
      setFailure(undefined);
      void api
        .POST("/api/admin/members/{id}/password", { params: { path, header } })
        .then((r) => {
          if (r.data) {
            setPassword(r.data.temporary_password);
            setCopied(false);
            onSaved();
          } else setFailure(errorCode(r.error) ?? "unknown");
        })
        .catch(() => setFailure("unavailable"))
        .finally(() => {
          setBusy(false);
          setAction(undefined);
        });
    } else if (action === "unlink") {
      void run(
        () =>
          api.DELETE("/api/admin/members/{id}/discord", {
            params: { path, header },
          }),
        onSaved,
      );
    } else if (action === "delete") {
      void run(
        () =>
          api.DELETE("/api/admin/members/{id}", {
            params: { path, header },
            body: { hide_clips: hide },
          }),
        onGone,
      );
    }
  }
  const dialogTitle = {
    reset: t("members.edit.resetTitle"),
    unlink: t("members.edit.unlinkTitle"),
    delete: t("members.deleteTitle"),
  };
  const dialogBody = {
    reset: t("members.edit.resetBody", { name }),
    unlink: t("members.edit.unlinkBody", { name }),
    delete: t("members.deleteBody", { name }),
  };
  const dialogConfirm = {
    reset: t("members.edit.resetConfirm"),
    unlink: t("members.edit.unlinkDiscord"),
    delete: t("members.deleteConfirm"),
  };

  return (
    <form
      className="flex flex-1 flex-col gap-5"
      onSubmit={(e) => {
        e.preventDefault();
        if (dirty && !busy) save();
      }}
    >
      <p className="-mt-2 text-xs text-mute">
        {t("members.edit.joined", { date: format(member.created_at) })} ·{" "}
        {t("members.edit.subtitle", { count: member.clip_count })}
      </p>
      {!edit && <ErrorAlert>{t("members.edit.readOnly")}</ErrorAlert>}
      {edit && !manage && (
        <StatusNotice>{t("members.edit.selfNote")}</StatusNotice>
      )}
      {notice && <StatusNotice>{notice}</StatusNotice>}
      {failure && !action && <ErrorAlert>{error(failure)}</ErrorAlert>}

      <Section title={t("members.edit.identity")}>
        <Field
          label={t("members.edit.username")}
          hint={t("members.edit.usernameHint")}
          value={username}
          readOnly={!edit}
          autoComplete="off"
          onChange={(e) => setUsername(e.target.value)}
        />
        <Field
          label={t("members.edit.displayName")}
          hint={t("members.edit.displayNameHint")}
          value={displayName}
          readOnly={!edit}
          maxLength={100}
          autoComplete="off"
          onChange={(e) => setDisplayName(e.target.value)}
        />
        <div className="flex flex-col gap-1.5">
          <label htmlFor="member-language" className="font-medium">
            {t("members.edit.language")}
          </label>
          <select
            id="member-language"
            className={selectClass}
            value={language}
            disabled={!edit}
            onChange={(e) => setLanguage(e.target.value)}
          >
            <option value="">{t("members.edit.languageDefault")}</option>
            <option value="fr">Français</option>
            <option value="en">English</option>
          </select>
        </div>
      </Section>

      {manage && (
        <Section title={t("members.edit.access")}>
          <div className="flex flex-col gap-1.5">
            <label htmlFor="member-role" className="font-medium">
              {t("members.edit.role")}
            </label>
            <select
              id="member-role"
              className={selectClass}
              value={role}
              disabled={!setRoleAllowed}
              onChange={(e) => setRole(e.target.value as Detail["role"])}
            >
              <option value="user">{t("roles.user")}</option>
              <option value="admin">{t("roles.admin")}</option>
              {member.role === "super_admin" && (
                <option value="super_admin">{t("roles.superAdmin")}</option>
              )}
            </select>
          </div>
          <div className="flex flex-col gap-1">
            <label className="flex min-h-11 items-center gap-2">
              <input
                type="checkbox"
                checked={approved}
                onChange={(e) => setApproved(e.target.checked)}
              />
              {t("members.edit.approved")}
            </label>
            <p className="text-xs text-mute">
              {t("members.edit.approvedHint")}
            </p>
          </div>
        </Section>
      )}

      {manage && (
        <Section title={t("members.edit.security")}>
          <p className="text-mute">
            {member.discord_linked
              ? t("members.edit.discordLinked", {
                  name: member.discord_username
                    ? ` (${member.discord_username})`
                    : "",
                })
              : t("members.edit.discordNone")}
          </p>
          {member.discord_linked && !member.has_password && (
            <p className="text-xs text-mute">{t("members.edit.noPassword")}</p>
          )}
          <div className="flex flex-wrap gap-2">
            {member.discord_linked && member.has_password && (
              <button
                type="button"
                className={buttonClass()}
                onClick={() => setAction("unlink")}
              >
                {t("members.edit.unlinkDiscord")}
              </button>
            )}
            <button
              type="button"
              className={buttonClass()}
              onClick={() => setAction("reset")}
            >
              {t("members.edit.resetPassword")}
            </button>
            <button
              type="button"
              className={buttonClass()}
              disabled={busy || member.session_count === 0}
              onClick={() =>
                void run(
                  () =>
                    api.DELETE("/api/admin/members/{id}/sessions", {
                      params: { path, header },
                    }),
                  () => {
                    setNotice(t("members.edit.sessionsClosed"));
                    onSaved();
                  },
                )
              }
            >
              {t("members.edit.revokeSessions", {
                count: member.session_count,
              })}
            </button>
          </div>
          {password && (
            <div className="flex flex-col gap-2 rounded-lg border border-line-strong bg-side p-3">
              <p className="font-medium">
                {t("members.edit.temporaryPassword", { name })}
              </p>
              <div className="flex items-center gap-2">
                <output
                  aria-label={t("members.edit.temporaryPassword", { name })}
                  className="min-w-0 flex-1 rounded-md bg-surface px-2 py-1.5 font-mono break-all select-all"
                >
                  {password}
                </output>
                <button
                  type="button"
                  className={buttonClass()}
                  onClick={() => {
                    void navigator.clipboard
                      ?.writeText(password)
                      .then(() => setCopied(true))
                      .catch(() => undefined);
                  }}
                >
                  {copied ? t("members.edit.copied") : t("members.edit.copy")}
                </button>
              </div>
              <p className="text-xs text-mute">
                {t("members.edit.temporaryHint")}
              </p>
            </div>
          )}
        </Section>
      )}

      {manage && (
        <div className="flex flex-col gap-2 rounded-lg border border-bad-fg/40 p-3">
          <p className="font-medium text-bad-fg">
            {t("members.edit.dangerTitle")}
          </p>
          <p className="text-xs text-mute">{t("members.edit.dangerBody")}</p>
          <div>
            <button
              type="button"
              className={buttonClass("danger")}
              onClick={() => {
                setHide(false);
                setAction("delete");
              }}
            >
              {t("members.delete")}
            </button>
          </div>
        </div>
      )}

      <div className="sticky -bottom-5 z-10 -mx-5 mt-auto flex items-center justify-end gap-2 border-t border-line bg-surface px-5 py-3">
        {!dirty && edit && (
          <span className="mr-auto text-xs text-mute">
            {t("members.edit.noChanges")}
          </span>
        )}
        <button type="button" className={buttonClass()} onClick={onClose}>
          {t("members.cancel")}
        </button>
        <button
          type="submit"
          className={buttonClass("primary")}
          disabled={!edit || !dirty || busy}
        >
          {t("members.edit.save")}
        </button>
      </div>

      <ConfirmDialog
        open={!!action}
        onOpenChange={(open) => {
          if (!open) setAction(undefined);
        }}
        title={action ? dialogTitle[action] : ""}
        cancelLabel={t("members.cancel")}
        confirmLabel={action ? dialogConfirm[action] : ""}
        danger={action === "delete"}
        busy={busy}
        onConfirm={confirm}
      >
        {action && (
          <>
            <p>{dialogBody[action]}</p>
            {action === "delete" && (
              <label className="flex min-h-11 items-center gap-2">
                <input
                  type="checkbox"
                  checked={hide}
                  onChange={(e) => setHide(e.target.checked)}
                />
                {t("members.hide")}
              </label>
            )}
            {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          </>
        )}
      </ConfirmDialog>
    </form>
  );
}
