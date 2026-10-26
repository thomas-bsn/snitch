// PROTOTYPE — voice verification: thumbnail to preview, "Decide" drawer to pick or enter someone. Sample data, no writes.
import { useEffect, useState } from "react";

type Candidate = { name: string; pct: number };
type Case = {
  id: number;
  clip: string;
  when: string;
  speaker: string;
  speech: string;
  level: "unknown" | "ambiguous";
  candidates: Candidate[];
  group: string;
  from: string;
  to: string;
  bg: string;
};

const CASES: Case[] = [
  {
    id: 1,
    clip: "Clutch 1v3 sur Bind",
    when: "Oct 12",
    speaker: "Speaker 2",
    speech: "14 s of speech",
    level: "ambiguous",
    candidates: [
      { name: "Léa", pct: 61 },
      { name: "Marc", pct: 22 },
    ],
    group: "Heard in 9 clips",
    from: "0:09",
    to: "0:21",
    bg: "linear-gradient(135deg,#c9d6ff,#e2e2e2)",
  },
  {
    id: 2,
    clip: "Premier try du raid",
    when: "Sept 28",
    speaker: "Speaker 1",
    speech: "42 s of speech",
    level: "unknown",
    candidates: [],
    group: "Heard in 6 clips",
    from: "1:02",
    to: "1:31",
    bg: "linear-gradient(135deg,#ffd6c9,#f5e6e0)",
  },
  {
    id: 3,
    clip: "Fou rire après le plant",
    when: "Oct 11",
    speaker: "Speaker 3",
    speech: "8 s of speech",
    level: "ambiguous",
    candidates: [
      { name: "Marc", pct: 48 },
      { name: "Thomas", pct: 45 },
    ],
    group: "Heard in 4 clips",
    from: "0:33",
    to: "0:41",
    bg: "linear-gradient(135deg,#c9f0dd,#e2f0f5)",
  },
  {
    id: 4,
    clip: "Ace en finale",
    when: "Oct 5",
    speaker: "Speaker 2",
    speech: "21 s of speech",
    level: "unknown",
    candidates: [],
    group: "Heard in 2 clips",
    from: "0:12",
    to: "0:33",
    bg: "linear-gradient(135deg,#f3d9ff,#e6e6fa)",
  },
];
const PEOPLE = ["Léa", "Marc", "Thomas"];

function Level({ c }: { c: Case }) {
  return c.level === "ambiguous" ? (
    <span className="pill warn">⇄ Ambiguous</span>
  ) : (
    <span className="pill bad">? Unknown</span>
  );
}

function Thumb({
  c,
  onPlay,
  big,
}: {
  c: Case;
  onPlay?: () => void;
  big?: boolean;
}) {
  return (
    <button
      className={`vthumb${big ? " big" : ""}`}
      style={{ background: c.bg }}
      onClick={onPlay}
      aria-label={`Play excerpt of ${c.speaker} in ${c.clip}`}
    >
      <span className="vplay">▶</span>
      <span className="dur">
        {c.from}–{c.to}
      </span>
    </button>
  );
}

export function Verification() {
  const [cases, setCases] = useState(CASES);
  const [level, setLevel] = useState<"all" | "unknown" | "ambiguous">("all");
  const [preview, setPreview] = useState<Case | null>(null);
  const [deciding, setDeciding] = useState<Case | null>(null);
  const [pick, setPick] = useState("");
  const [newName, setNewName] = useState("");
  const [q, setQ] = useState("");
  const [toast, setToast] = useState("");

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setPreview(null);
        setDeciding(null);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const shown = cases.filter((c) => level === "all" || c.level === level);
  const n = (l: string) => cases.filter((c) => c.level === l).length;
  const open = (c: Case) => {
    setDeciding(c);
    setPick("");
    setNewName("");
    setQ("");
  };
  const confirm = () => {
    if (!deciding) return;
    const who = pick === "__new" ? newName.trim() : pick;
    setCases(cases.filter((c) => c.id !== deciding.id));
    setToast(
      `Saved: ${deciding.speaker} is ${who}. Matching will be recomputed.`,
    );
    setDeciding(null);
    setTimeout(() => setToast(""), 4000);
  };
  const ready = pick === "__new" ? newName.trim().length >= 1 : pick !== "";
  const others = PEOPLE.filter(
    (p) =>
      p.toLowerCase().includes(q.toLowerCase()) &&
      !deciding?.candidates.some((c) => c.name === p),
  );

  return (
    <div className="verif">
      <div className="stats">
        <button
          className={`stat btnstat${level === "all" ? " on" : ""}`}
          onClick={() => setLevel("all")}
        >
          <span className="mute">To decide</span>
          <b>{cases.length}</b>
        </button>
        <button
          className={`stat btnstat${level === "unknown" ? " on" : ""}`}
          onClick={() => setLevel("unknown")}
        >
          <span className="mute">? Unknown</span>
          <b>{n("unknown")}</b>
        </button>
        <button
          className={`stat btnstat${level === "ambiguous" ? " on" : ""}`}
          onClick={() => setLevel("ambiguous")}
        >
          <span className="mute">⇄ Ambiguous</span>
          <b>{n("ambiguous")}</b>
        </button>
      </div>
      <p className="mute" style={{ margin: 0 }}>
        Most frequent voices first: naming one removes the most uncertainty.
        Deciding is always a human confirmation.
      </p>
      {shown.length === 0 && (
        <div className="empty">Nothing left to decide here.</div>
      )}
      <div className="cases">
        {shown.map((c) => (
          <article key={c.id} className="case">
            <Thumb c={c} onPlay={() => setPreview(c)} />
            <div className="case-body">
              <div className="case-top">
                <b>{c.speaker}</b>
                <Level c={c} />
              </div>
              <span className="mute">
                {c.clip} · {c.when} · {c.speech}
              </span>
              <span className="mute">{c.group}</span>
              {c.candidates.length > 0 && (
                <div className="cands">
                  {c.candidates.map((x) => (
                    <span key={x.name} className="pill info">
                      {x.name} {x.pct}%
                    </span>
                  ))}
                </div>
              )}
            </div>
            <button className="btn pri" onClick={() => open(c)}>
              Decide
            </button>
          </article>
        ))}
      </div>

      {preview && (
        <>
          <div className="scrim dim" onClick={() => setPreview(null)} />
          <div
            className="modal"
            role="dialog"
            aria-modal="true"
            aria-label="Excerpt preview"
          >
            <div className="player" style={{ background: preview.bg }}>
              <div className="bar">
                <button className="play" aria-label="Play">
                  ▶
                </button>
                <span>
                  {preview.from} / {preview.to}
                </span>
                <div className="track">
                  <i />
                </div>
              </div>
            </div>
            <div className="modal-f">
              <div>
                <b>{preview.speaker}</b>{" "}
                <span className="mute">· {preview.clip}</span>
              </div>
              <div className="row">
                <button className="btn" onClick={() => setPreview(null)}>
                  Close
                </button>
                <button
                  className="btn pri"
                  onClick={() => {
                    const c = preview;
                    setPreview(null);
                    open(c);
                  }}
                >
                  Decide
                </button>
              </div>
            </div>
          </div>
        </>
      )}

      {deciding && (
        <>
          <div className="scrim dim" onClick={() => setDeciding(null)} />
          <aside
            className="drawer"
            role="dialog"
            aria-modal="true"
            aria-label="Decide who is speaking"
          >
            <header>
              <h2 style={{ margin: 0 }}>Who is {deciding.speaker}?</h2>
              <button
                className="btn"
                onClick={() => setDeciding(null)}
                aria-label="Close"
              >
                ✕
              </button>
            </header>
            <div className="drawer-body">
              <div className="mini">
                <Thumb c={deciding} onPlay={() => setPreview(deciding)} />
                <div>
                  <b>{deciding.clip}</b>
                  <div className="mute">
                    Excerpt {deciding.from}–{deciding.to} · {deciding.group}
                  </div>
                  <Level c={deciding} />
                </div>
              </div>

              {deciding.candidates.length > 0 && (
                <fieldset>
                  <legend>Closest matches</legend>
                  {deciding.candidates.map((c) => (
                    <label
                      key={c.name}
                      className={`opt${pick === c.name ? " on" : ""}`}
                    >
                      <input
                        type="radio"
                        name="who"
                        checked={pick === c.name}
                        onChange={() => setPick(c.name)}
                      />
                      <span>{c.name}</span>
                      <span className="mute">{c.pct}% similar</span>
                    </label>
                  ))}
                </fieldset>
              )}

              <fieldset>
                <legend>
                  {deciding.candidates.length
                    ? "Someone else"
                    : "Existing people"}
                </legend>
                <input
                  className="find"
                  placeholder="Search people…"
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                  aria-label="Search people"
                />
                {others.map((p) => (
                  <label key={p} className={`opt${pick === p ? " on" : ""}`}>
                    <input
                      type="radio"
                      name="who"
                      checked={pick === p}
                      onChange={() => setPick(p)}
                    />
                    <span>{p}</span>
                  </label>
                ))}
                {others.length === 0 && <span className="mute">No match.</span>}
              </fieldset>

              <fieldset>
                <legend>Someone new</legend>
                <label className={`opt${pick === "__new" ? " on" : ""}`}>
                  <input
                    type="radio"
                    name="who"
                    checked={pick === "__new"}
                    onChange={() => setPick("__new")}
                  />
                  <span>Create a person</span>
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
              <button className="btn" onClick={() => setDeciding(null)}>
                Skip for now
              </button>
              <button className="btn pri" disabled={!ready} onClick={confirm}>
                Confirm
              </button>
            </footer>
          </aside>
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
