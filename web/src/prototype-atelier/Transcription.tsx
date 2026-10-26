// PROTOTYPE — Administration › Transcription: the library's main model, then the workers with the name each one sends to its server.
import { useEffect, useState } from "react";
import { store, useProfile, useWorkers, type W } from "./store";

export function Transcription() {
  const profile = useProfile();
  const workers = useWorkers();
  const [model, setModel] = useState<string | null>(null);
  const [language, setLanguage] = useState<string | null>(null);
  const [adv, setAdv] = useState(false);
  const [sel, setSel] = useState<W | null>(null);
  const [alias, setAlias] = useState("");
  const [saved, setSaved] = useState("");
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setSel(null);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  if (!profile)
    return (
      <div className="empty">
        <b style={{ color: "var(--ink)" }}>No transcription model yet</b>
        <p style={{ margin: "6px 0 0" }}>
          It is set when you create your first worker, in Administration ›
          Workers. Every later worker shares it.
        </p>
      </div>
    );

  const m = model ?? profile.model;
  const l = language ?? profile.language;
  const dirty = m.trim() !== profile.model || l !== profile.language;
  const open = (w: W) => {
    setSel(w);
    setAlias(w.alias);
  };

  return (
    <div className="verif">
      <div className="panel pad">
        <div className="phead">
          <div>
            <b>Main model</b>
            <p className="mute" style={{ marginTop: 2 }}>
              Used by every worker and for every clip. Version {profile.version}
              .
            </p>
          </div>
        </div>
        <label>
          Whisper model name
          <input value={m} onChange={(e) => setModel(e.target.value)} />
        </label>
        <label>
          Language
          <select value={l} onChange={(e) => setLanguage(e.target.value)}>
            <option value="fr">Français (fr)</option>
            <option value="en">English (en)</option>
            <option value="es">Español (es)</option>
            <option value="de">Deutsch (de)</option>
          </select>
          <span className="help">One language for the whole library.</span>
        </label>
        <button
          className="linkbtn"
          onClick={() => setAdv(!adv)}
          aria-expanded={adv}
        >
          {adv ? "▾" : "▸"} Speech detection (advanced)
        </button>
        {adv && (
          <div className="advgrid">
            <label>
              Detection
              <select defaultValue="off">
                <option value="off">Off</option>
                <option value="energy">Energy</option>
              </select>
            </label>
            <label>
              Energy threshold
              <input defaultValue="0.02" />
            </label>
            <label>
              Minimum speech (ms)
              <input defaultValue="100" />
            </label>
            <label>
              Minimum silence (ms)
              <input defaultValue="300" />
            </label>
            <label>
              Padding (ms)
              <input defaultValue="100" />
            </label>
          </div>
        )}
        {dirty && (
          <div className="banner info">
            Saving creates a new version. Results already computed do not
            change, and you choose which clips to redo.
          </div>
        )}
        {saved && (
          <div className="banner info" role="status">
            {saved}
          </div>
        )}
        <div>
          <button
            className="btn pri"
            disabled={!dirty || m.trim() === ""}
            onClick={() => {
              store.saveProfile(m.trim(), l);
              setModel(null);
              setLanguage(null);
              setSaved("Saved as a new version.");
              setTimeout(() => setSaved(""), 3000);
            }}
          >
            Save and version
          </button>
        </div>
      </div>

      <div className="panel">
        <div className="panel-h">
          <b>Workers</b>
          <span className="mute">
            Select one to see the name it sends to its server
          </span>
        </div>
        <table>
          <thead>
            <tr>
              <th>Worker</th>
              <th>Whisper server</th>
              <th>Name sent to the server</th>
              <th>Language</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {workers.map((w) => (
              <tr
                key={w.id}
                className={`clickrow${sel?.id === w.id ? " sel" : ""}`}
                onClick={() => open(w)}
              >
                <td>
                  <b>{w.name}</b>
                </td>
                <td>{w.server}</td>
                <td>
                  {w.alias ? (
                    w.alias
                  ) : (
                    <span className="mute">{profile.model} (same)</span>
                  )}
                </td>
                <td>{profile.language}</td>
                <td>
                  <span className={`pill ${w.online ? "ok" : "bad"}`}>
                    {w.online ? "● Online" : "○ Offline"}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {sel && (
        <>
          <div className="scrim dim" onClick={() => setSel(null)} />
          <div
            className="modal"
            style={{
              width: "min(520px,92vw)",
              maxHeight: "88vh",
              overflow: "auto",
            }}
            role="dialog"
            aria-modal="true"
            aria-label={`Worker ${sel.name}`}
          >
            <h2 style={{ margin: 0 }}>{sel.name}</h2>
            <label>
              Whisper server
              <input readOnly value={sel.server} />
              <span className="help">Declared by the worker.</span>
            </label>
            <label>
              Language
              <input readOnly value={`${profile.language} (library)`} />
              <span className="help">
                Same for every worker. Change it in the main model above.
              </span>
            </label>
            <label>
              Name sent to this server
              <input
                value={alias}
                onChange={(e) => setAlias(e.target.value)}
                placeholder={profile.model}
              />
              <span className="help">
                Leave empty to send the main model name. Fill it only if this
                server calls the same model by another name. Snitch cannot check
                that it is really the same model.
              </span>
            </label>
            <div className="row">
              <button className="btn" onClick={() => setSel(null)}>
                Close
              </button>
              <button
                className="btn pri"
                onClick={() => {
                  store.setAlias(sel.id, alias.trim());
                  setSel(null);
                }}
              >
                Save
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
