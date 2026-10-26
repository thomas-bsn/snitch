// PROTOTYPE — unknown voice groups: list of groups, "Listen and name" drawer with excerpts, exclude, name. Sample data, no writes.
import { useEffect, useState } from "react";

type Voice = {
  id: number;
  clip: string;
  when: string;
  from: string;
  to: string;
  bg: string;
};
type Group = { id: number; voices: Voice[] };
const BG = [
  "linear-gradient(135deg,#c9d6ff,#e2e2e2)",
  "linear-gradient(135deg,#ffd6c9,#f5e6e0)",
  "linear-gradient(135deg,#c9f0dd,#e2f0f5)",
  "linear-gradient(135deg,#f3d9ff,#e6e6fa)",
];
const mk = (
  id: number,
  clip: string,
  when: string,
  from: string,
  to: string,
): Voice => ({ id, clip, when, from, to, bg: BG[id % 4] });
const GROUPS: Group[] = [
  {
    id: 1,
    voices: [
      mk(1, "Clutch 1v3 sur Bind", "Oct 12", "0:09", "0:21"),
      mk(2, "Premier try du raid", "Sept 28", "1:02", "1:31"),
      mk(3, "Ace en finale", "Oct 5", "0:12", "0:33"),
      mk(4, "Le bug du siècle", "Oct 3", "0:04", "0:11"),
      mk(5, "Debrief de fin de partie", "Oct 1", "2:10", "2:44"),
      mk(6, "Défaite à égalité", "Oct 9", "0:50", "1:08"),
    ],
  },
  {
    id: 2,
    voices: [
      mk(7, "Fou rire après le plant", "Oct 11", "0:33", "0:41"),
      mk(8, "Cinématique du boss", "Oct 7", "0:03", "0:09"),
      mk(9, "Soirée LAN", "Aug 30", "3:20", "3:41"),
    ],
  },
];
const PEOPLE = ["Léa", "Marc", "Thomas"];

export function VoiceGroups() {
  const [groups, setGroups] = useState(GROUPS);
  const [open, setOpen] = useState<Group | null>(null);
  const [excluded, setExcluded] = useState<number[]>([]);
  const [pick, setPick] = useState("");
  const [newName, setNewName] = useState("");
  const [play, setPlay] = useState<Voice | null>(null);
  const [toast, setToast] = useState("");

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (play) setPlay(null);
        else setOpen(null);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [play]);

  const start = (g: Group) => {
    setOpen(g);
    setExcluded([]);
    setPick("");
    setNewName("");
  };
  const kept = open ? open.voices.filter((v) => !excluded.includes(v.id)) : [];
  const ready =
    kept.length > 0 && (pick === "__new" ? newName.trim() !== "" : pick !== "");
  const who = pick === "__new" ? newName.trim() : pick;
  const name = () => {
    if (!open) return;
    setGroups(groups.filter((g) => g.id !== open.id));
    setToast(
      `Saved: ${kept.length} voices are now ${who}.${excluded.length ? ` ${excluded.length} left out.` : ""}`,
    );
    setOpen(null);
    setTimeout(() => setToast(""), 4000);
  };
  const toggle = (id: number) =>
    setExcluded(
      excluded.includes(id)
        ? excluded.filter((x) => x !== id)
        : [...excluded, id],
    );

  return (
    <div className="verif">
      <p className="mute" style={{ margin: 0 }}>
        Voices that sound alike, found across clips. Snitch only proposes:
        nothing is attributed until you name the group.
      </p>
      {groups.length === 0 && (
        <div className="empty">No groups left to name.</div>
      )}
      <div className="grid2">
        {groups.map((g) => (
          <div key={g.id} className="panel pad">
            <div className="vstack">
              {g.voices.slice(0, 4).map((v, i) => (
                <span
                  key={v.id}
                  className="vchip"
                  style={{ background: v.bg, zIndex: 9 - i }}
                />
              ))}
              {g.voices.length > 4 && (
                <span className="vmore">+{g.voices.length - 4}</span>
              )}
            </div>
            <b>Group of {g.voices.length} close voices</b>
            <span className="mute">
              Heard in {g.voices.length} clips · {g.voices[0].clip},{" "}
              {g.voices[1].clip}…
            </span>
            <span className="mute">Never confirmed automatically</span>
            <div>
              <button className="btn pri" onClick={() => start(g)}>
                Listen and name
              </button>
            </div>
          </div>
        ))}
      </div>

      {open && (
        <>
          <div className="scrim dim" onClick={() => setOpen(null)} />
          <aside
            className="drawer wide"
            role="dialog"
            aria-modal="true"
            aria-label="Name a voice group"
          >
            <header>
              <div>
                <h2 style={{ margin: 0 }}>Is this one person?</h2>
                <span className="mute">
                  {kept.length} of {open.voices.length} voices selected
                </span>
              </div>
              <button
                className="btn"
                onClick={() => setOpen(null)}
                aria-label="Close"
              >
                ✕
              </button>
            </header>
            <div className="drawer-body">
              <p className="mute" style={{ margin: 0 }}>
                Listen to a few excerpts. Leave out any voice that is someone
                else: the others stay together.
              </p>
              <div className="vgrid">
                {open.voices.map((v) => {
                  const out = excluded.includes(v.id);
                  return (
                    <div key={v.id} className={`vcell${out ? " out" : ""}`}>
                      <button
                        className="vthumb"
                        style={{ background: v.bg }}
                        onClick={() => setPlay(v)}
                        aria-label={`Play excerpt from ${v.clip}`}
                      >
                        <span className="vplay">▶</span>
                        <span className="dur">
                          {v.from}–{v.to}
                        </span>
                      </button>
                      <b className="vt">{v.clip}</b>
                      <span className="mute">{v.when}</span>
                      <button className="btn" onClick={() => toggle(v.id)}>
                        {out ? "Put back" : "Not this person"}
                      </button>
                    </div>
                  );
                })}
              </div>

              <fieldset>
                <legend>Who is it?</legend>
                {PEOPLE.map((p) => (
                  <label key={p} className={`opt${pick === p ? " on" : ""}`}>
                    <input
                      type="radio"
                      name="gwho"
                      checked={pick === p}
                      onChange={() => setPick(p)}
                    />
                    <span>{p}</span>
                  </label>
                ))}
                <label className={`opt${pick === "__new" ? " on" : ""}`}>
                  <input
                    type="radio"
                    name="gwho"
                    checked={pick === "__new"}
                    onChange={() => setPick("__new")}
                  />
                  <span>Someone new</span>
                </label>
                {pick === "__new" && (
                  <input
                    className="find"
                    autoFocus
                    placeholder="Name"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    aria-label="New person name"
                  />
                )}
              </fieldset>
            </div>
            <footer>
              <button className="btn" onClick={() => setOpen(null)}>
                Not now
              </button>
              <button className="btn pri" disabled={!ready} onClick={name}>
                {ready
                  ? `Name ${kept.length} voices as ${who}`
                  : "Name these voices"}
              </button>
            </footer>
          </aside>
        </>
      )}

      {play && (
        <>
          <div
            className="scrim dim"
            style={{ zIndex: 62 }}
            onClick={() => setPlay(null)}
          />
          <div
            className="modal"
            style={{ zIndex: 63 }}
            role="dialog"
            aria-modal="true"
            aria-label="Excerpt preview"
          >
            <div className="player" style={{ background: play.bg }}>
              <div className="bar">
                <button className="play" aria-label="Play">
                  ▶
                </button>
                <span>
                  {play.from} / {play.to}
                </span>
                <div className="track">
                  <i />
                </div>
              </div>
            </div>
            <div className="modal-f">
              <div>
                <b>{play.clip}</b> <span className="mute">· {play.when}</span>
              </div>
              <button className="btn" onClick={() => setPlay(null)}>
                Close
              </button>
            </div>
          </div>
        </>
      )}
      {toast && (
        <div className="toast" role="status">
          {toast}
        </div>
      )}
    </div>
  );
}
