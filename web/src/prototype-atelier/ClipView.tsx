// PROTOTYPE — clip page, copied from the Atelier "page d'un clip" artboard. Sample data, read-only.
import { useState } from "react";

const P = { bg: "#ececfb", fg: "#3a3aa8" };
const Q = { bg: "#fdeedd", fg: "#8a4a07" };
const U = { bg: "#eeeef1", fg: "#4b4b55" };
const SEGMENTS = [
  {
    tc: "0:02",
    who: "Léa",
    c: P,
    note: "Identified · suggestion",
    text: "Attends, attends, il en reste trois.",
  },
  {
    tc: "0:09",
    who: "Speaker 2",
    c: U,
    note: "Unknown",
    text: "Je les tiens, ne bouge pas.",
  },
  {
    tc: "0:14",
    who: "Léa",
    c: P,
    note: "Corrected · Thomas",
    text: "Il est derrière la caisse, côté B.",
    hl: true,
  },
  {
    tc: "0:21",
    who: "Speaker 2",
    c: U,
    note: "Unknown",
    text: "Un de moins, il en reste deux.",
  },
  {
    tc: "0:30",
    who: "Marc",
    c: Q,
    note: "Identified · automatic",
    text: "Non mais c’est pas possible, regarde ça.",
  },
  {
    tc: "0:41",
    who: "Léa",
    c: P,
    note: "Identified · suggestion",
    text: "Gagné ! On le refait quand tu veux.",
  },
];
const TABS = ["Transcript", "Voices", "History"];

export function ClipView({
  onBack,
  back,
}: {
  onBack: () => void;
  back: string;
}) {
  const [tab, setTab] = useState(0);
  return (
    <>
      <div className="crumbs">
        <button className="back" onClick={onBack}>
          ← {back}
        </button>
        <span className="mute">/</span>
        <b>Clutch 1v3 sur Bind</b>
        <span className="state ready">✓ Transcribed</span>
      </div>
      <div className="clip">
        <div className="main">
          <div className="player">
            <div className="bar">
              <button className="play" aria-label="Play">
                ▶
              </button>
              <span>0:14 / 0:48</span>
              <div className="track">
                <i />
              </div>
            </div>
          </div>
          <div>
            <h1>Clutch 1v3 sur Bind</h1>
            <div className="mute" style={{ marginTop: 4 }}>
              Ranked · Oct 12 · 0:48 · imported by Thomas · sample data
            </div>
          </div>
          <div style={{ display: "flex", gap: 8 }}>
            <button className="btn">Visibility: Private ▾</button>
            <button className="btn">Retranscribe…</button>
          </div>
        </div>
        <aside className="read">
          <div className="tabs pad" role="tablist">
            {TABS.map((t, i) => (
              <button
                key={t}
                role="tab"
                aria-selected={i === tab}
                className={i === tab ? "on" : ""}
                onClick={() => setTab(i)}
              >
                {t}
              </button>
            ))}
          </div>
          <div className="segs">
            {tab === 0 &&
              SEGMENTS.map((s) => (
                <div key={s.tc} className={`seg${s.hl ? " hl" : ""}`}>
                  <span className="tc">{s.tc}</span>
                  <div>
                    <div className="who">
                      <span
                        className="spk"
                        style={{ background: s.c.bg, color: s.c.fg }}
                      >
                        {s.who}
                      </span>
                      <span className="note">{s.note}</span>
                    </div>
                    <p>{s.text}</p>
                  </div>
                </div>
              ))}
            {tab === 1 && (
              <p className="mute" style={{ padding: 16 }}>
                Three local speakers detected. Léa: 3 segments, Marc: 1, Speaker
                2: 2 (unknown).
              </p>
            )}
            {tab === 2 && (
              <p className="mute" style={{ padding: 16 }}>
                Segment 0:14 corrected by Thomas. Restore the original or an
                earlier version from here.
              </p>
            )}
          </div>
        </aside>
      </div>
    </>
  );
}
