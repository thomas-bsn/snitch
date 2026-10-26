// PROTOTYPE — workers list + detail drawer (machine, models, compose, token). Sample data, no writes.
import { useEffect, useState } from "react";
import { WorkerCreate } from "./WorkerCreate";
import { useWorkers, type W } from "./store";

const TOKEN = "snw_9f3c1e7a42b8d05e6a91c4f3";

function compose(name: string, token: string) {
  return `services:
  worker:
    image: ghcr.io/thomas-bsn/snitch-worker:latest
    restart: unless-stopped
    environment:
      SNITCH_API_URL: https://snitch.example
      SNITCH_WORKER_TOKEN: ${token}
      TRANSCRIPTION_API_URL: A_RENSEIGNER
      TRANSCRIPTION_MODEL: A_RENSEIGNER
      HF_TOKEN: A_RENSEIGNER
    # worker: ${name}`;
}

export function Workers() {
  const workers = useWorkers();
  const [open, setOpen] = useState<W | null>(null);
  const [adding, setAdding] = useState(false);
  const [confirm, setConfirm] = useState<"regen" | "revoke" | null>(null);
  const [fresh, setFresh] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (confirm) setConfirm(null);
        else {
          setOpen(null);
          setAdding(false);
          setFresh(null);
        }
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [confirm]);

  const close = () => {
    setOpen(null);
    setAdding(false);
    setFresh(null);
    setConfirm(null);
  };
  const copy = () => {
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  const token = adding ? TOKEN : fresh;

  return (
    <div className="verif">
      <div className="panel">
        <div className="panel-h">
          <b>Workers</b>
          <button className="btn pri" onClick={() => setAdding(true)}>
            Add worker
          </button>
        </div>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Last pulse</th>
              <th>Models</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {workers.map((w) => (
              <tr key={w.id} className="clickrow" onClick={() => setOpen(w)}>
                <td>
                  <b>{w.name}</b>
                </td>
                <td>
                  <span className={`pill ${w.online ? "ok" : "bad"}`}>
                    {w.online ? "● Online" : "○ Offline"}
                  </span>
                </td>
                <td>{w.seen}</td>
                <td className="mute">
                  {w.models.length
                    ? w.models.map((m) => m[1].split(" · ")[0]).join(" · ")
                    : "Not declared"}
                </td>
                <td>
                  <button
                    className="btn"
                    onClick={(e) => {
                      e.stopPropagation();
                      setOpen(w);
                    }}
                  >
                    Open
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {workers.length === 0 && (
          <div className="empty" style={{ margin: 16 }}>
            No worker yet. Add your first worker to start transcribing.
          </div>
        )}
      </div>

      {open && (
        <>
          <div className="scrim dim" onClick={close} />
          <aside
            className="drawer wide"
            role="dialog"
            aria-modal="true"
            aria-label={adding ? "Add worker" : `Worker ${open?.name}`}
          >
            <header>
              <div>
                <h2 style={{ margin: 0 }}>
                  {adding ? "New worker created" : open!.name}
                </h2>
                {open && !adding && (
                  <span className={`pill ${open.online ? "ok" : "bad"}`}>
                    {open.online ? "● Online" : "○ Offline"}
                  </span>
                )}
              </div>
              <button className="btn" onClick={close} aria-label="Close">
                ✕
              </button>
            </header>
            <div className="drawer-body">
              {adding && (
                <div className="banner warn">
                  Copy this now. For security, the token is shown only once and
                  Snitch cannot show it again.
                </div>
              )}

              {open && !adding && (
                <>
                  <section className="dsec">
                    <h3>Machine</h3>
                    {open.machine.length ? (
                      <>
                        <dl>
                          {open.machine.map(([k, v]) => (
                            <div key={k}>
                              <dt>{k}</dt>
                              <dd>{v}</dd>
                            </div>
                          ))}
                        </dl>
                        <span className="mute">
                          Reported by the worker at each pulse. Proposed: today
                          the worker only reports its models.
                        </span>
                      </>
                    ) : (
                      <span className="mute">
                        This worker has not reported anything yet. Start it to
                        see its machine.
                      </span>
                    )}
                  </section>

                  <section className="dsec">
                    <h3>Activity</h3>
                    <dl>
                      <div>
                        <dt>Last pulse</dt>
                        <dd>{open.seen}</dd>
                      </div>
                      <div>
                        <dt>Concurrency</dt>
                        <dd>{open.conc} task at a time</dd>
                      </div>
                      <div>
                        <dt>Created</dt>
                        <dd>{open.created}</dd>
                      </div>
                    </dl>
                    {open.jobs.length > 0 && (
                      <div className="panel">
                        <table>
                          <tbody>
                            {open.jobs.map(([t, c, s]) => (
                              <tr key={c}>
                                <td>{t}</td>
                                <td>{c}</td>
                                <td className="mute">{s}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </section>

                  <section className="dsec">
                    <h3>Models in use</h3>
                    {open.models.length ? (
                      <dl>
                        {open.models.map(([k, v]) => (
                          <div key={k}>
                            <dt>{k}</dt>
                            <dd>{v}</dd>
                          </div>
                        ))}
                      </dl>
                    ) : (
                      <span className="mute">Not declared.</span>
                    )}
                  </section>
                </>
              )}

              <section className="dsec">
                <h3>Compose file for this machine</h3>
                {!adding && !fresh && (
                  <span className="mute">
                    The token is not stored in readable form. To get a compose
                    file with a working token, generate a new one: the current
                    token stops working.
                  </span>
                )}
                <pre>
                  {compose(
                    adding ? "new-worker" : open!.name,
                    token ?? "<token shown once, at creation>",
                  )}
                </pre>
                <div className="row">
                  <button className="btn" onClick={copy}>
                    {copied ? "Copied" : "Copy"}
                  </button>
                  <button className="btn">Download compose.yaml</button>
                  {!adding && !fresh && (
                    <button className="btn" onClick={() => setConfirm("regen")}>
                      Generate a new token…
                    </button>
                  )}
                </div>
                {fresh && (
                  <div className="banner warn">
                    New token generated. Copy it now: it will not be shown
                    again.
                  </div>
                )}
              </section>

              {!adding && (
                <section className="dsec danger">
                  <h3>Revoke</h3>
                  <span className="mute">
                    A revoked worker can no longer claim tasks. Its history
                    stays.
                  </span>
                  <div>
                    <button
                      className="btn bad"
                      onClick={() => setConfirm("revoke")}
                    >
                      Revoke this worker…
                    </button>
                  </div>
                </section>
              )}
            </div>
            <footer>
              <button className="btn pri" onClick={close}>
                {adding ? "I copied it" : "Done"}
              </button>
            </footer>
          </aside>
        </>
      )}

      {adding && (
        <>
          <div className="scrim dim" onClick={close} />
          <aside
            className="drawer wide"
            role="dialog"
            aria-modal="true"
            aria-label="Add worker"
          >
            <header>
              <h2 style={{ margin: 0 }}>Add a worker</h2>
              <button className="btn" onClick={close} aria-label="Close">
                ✕
              </button>
            </header>
            <div className="drawer-body">
              <p className="mute" style={{ margin: 0 }}>
                Tell Snitch where the worker will find Whisper. Snitch writes
                the compose file for that machine.
              </p>
              <WorkerCreate onDone={close} />
            </div>
          </aside>
        </>
      )}

      {confirm && (
        <>
          <div
            className="scrim dim"
            style={{ zIndex: 62 }}
            onClick={() => setConfirm(null)}
          />
          <div
            className="modal"
            style={{ zIndex: 63, width: "min(440px,92vw)" }}
            role="alertdialog"
            aria-modal="true"
            aria-label="Confirm"
          >
            <h2 style={{ margin: 0 }}>
              {confirm === "regen"
                ? "Generate a new token?"
                : `Revoke ${open?.name}?`}
            </h2>
            <p className="mute" style={{ margin: 0 }}>
              {confirm === "regen"
                ? "The current token stops working immediately. Update the compose file on that machine."
                : "This worker will stop receiving tasks. You cannot undo this."}
            </p>
            <div className="row" style={{ justifyContent: "flex-end" }}>
              <button className="btn" onClick={() => setConfirm(null)}>
                Cancel
              </button>
              <button
                className={`btn ${confirm === "revoke" ? "bad" : "pri"}`}
                onClick={() => {
                  if (confirm === "regen") setFresh(TOKEN);
                  else close();
                  setConfirm(null);
                }}
              >
                {confirm === "regen" ? "Generate" : "Revoke"}
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
