// PROTOTYPE — "Delete my account" with a typed confirmation. Nothing is deleted; the call is stubbed.
import { useEffect, useState } from "react";

const WORDS = ["delete", "supprimer"];

export function DeleteAccount() {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const [done, setDone] = useState(false);
  const ok = WORDS.includes(text.trim().toLowerCase());

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  const close = () => {
    setOpen(false);
    setText("");
  };

  return (
    <div className="panel pad danger">
      <b>Delete my account</b>
      <p className="mute">
        Permanently deletes your account and signs you out everywhere. This
        cannot be undone.
      </p>
      {done && (
        <div className="banner info" role="status">
          Preview: nothing was deleted.
        </div>
      )}
      <div>
        <button className="btn bad" onClick={() => setOpen(true)}>
          Delete my account…
        </button>
      </div>
      {open && (
        <>
          <div className="scrim dim" onClick={close} />
          <div
            className="modal"
            style={{ width: "min(460px,92vw)" }}
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="del-t"
          >
            <h2 id="del-t" style={{ margin: 0 }}>
              Delete your account?
            </h2>
            <p className="mute" style={{ margin: 0 }}>
              This is permanent. To confirm, type <b>delete</b> (or{" "}
              <b>supprimer</b>) below.
            </p>
            <label>
              Confirmation
              <input
                value={text}
                onChange={(e) => setText(e.target.value)}
                autoFocus
                autoComplete="off"
                spellCheck={false}
                placeholder="delete"
              />
            </label>
            <div className="row" style={{ justifyContent: "flex-end" }}>
              <button className="btn" onClick={close}>
                Cancel
              </button>
              <button
                className="btn bad solid"
                disabled={!ok}
                onClick={() => {
                  close();
                  setDone(true);
                }}
              >
                Delete my account
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
