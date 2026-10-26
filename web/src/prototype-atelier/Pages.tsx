// PROTOTYPE — every Snitch screen in the Atelier look, with sample data. Throwaway, read-only.
import { useState } from "react";
import { Verification } from "./Verification";
import { VoiceGroups } from "./Groups";
import { Workers } from "./Workers";
import { Members } from "./Members";
import { Transcription } from "./Transcription";
import { DeleteAccount } from "./DeleteAccount";

type Cell = string | { chip: string; tone?: "ok" | "warn" | "bad" | "info" };
type Block =
  | {
      k: "table";
      title?: string;
      cols: string[];
      rows: Cell[][];
      action?: string;
    }
  | {
      k: "form";
      title: string;
      lead?: string;
      fields: [string, string][];
      button: string;
    }
  | { k: "stats"; items: [string, string][] }
  | { k: "toggle"; label: string; help?: string; on?: boolean }
  | { k: "banner"; tone: "warn" | "info"; text: string }
  | { k: "code"; title: string; text: string }
  | {
      k: "results";
      items: {
        tc: string;
        clip: string;
        before: string;
        hit: string;
        after: string;
        who: string;
      }[];
    }
  | {
      k: "cards";
      items: { title: string; sub: string; meta: string; action: string }[];
    }
  | { k: "drop" }
  | { k: "verification" }
  | { k: "groups" }
  | { k: "workers" }
  | { k: "members" }
  | { k: "transcription" }
  | { k: "deleteaccount" }
  | { k: "progress"; items: [string, number, string][] };
type Tab = { name: string; blocks: Block[] };
type PageDef = { title: string; lead: string; action?: string; tabs: Tab[] };

const chip = (
  c: string,
  tone: "ok" | "warn" | "bad" | "info" = "info",
): Cell => ({ chip: c, tone });

export const ADMIN_TABS: { id: string; label: string }[] = [
  { id: "workers", label: "Workers" },
  { id: "jobs", label: "Job queue" },
  { id: "sources", label: "Folders" },
  { id: "transcription", label: "Transcription" },
  { id: "voice", label: "Voice processing" },
  { id: "index", label: "Semantic index" },
  { id: "users", label: "Users & access" },
  { id: "exports", label: "Exports" },
];

const PAGES: Record<string, PageDef> = {
  search: {
    title: "Search",
    lead: "Find a moment by what was said. Results open the video at that timecode.",
    tabs: [
      {
        name: "Results",
        blocks: [
          {
            k: "results",
            items: [
              {
                tc: "0:14",
                clip: "Clutch 1v3 sur Bind",
                before: "Il est derrière la ",
                hit: "caisse",
                after: ", côté B.",
                who: "Léa · corrected",
              },
              {
                tc: "1:02",
                clip: "Fou rire après le plant",
                before: "Mets la ",
                hit: "caisse",
                after: " devant la porte.",
                who: "Marc · automatic",
              },
              {
                tc: "0:48",
                clip: "Premier try du raid",
                before: "On bloque derrière la ",
                hit: "caisse",
                after: " et on attend.",
                who: "Unknown voice",
              },
            ],
          },
        ],
      },
    ],
  },
  people: {
    title: "People",
    lead: "Name the voices Snitch hears. A human confirmation always comes first.",
    action: "New person",
    tabs: [
      {
        name: "People",
        blocks: [
          {
            k: "table",
            cols: ["Person", "Confirmed voices", "Clips", "Account"],
            rows: [
              ["Léa", "14", "92", "lea"],
              ["Marc", "9", "61", "marc"],
              ["Thomas", "22", "130", chip("Linked", "ok")],
            ],
          },
        ],
      },
      {
        name: "Voice groups",
        blocks: [{ k: "groups" }],
      },
      {
        name: "Verification",
        blocks: [{ k: "verification" }],
      },
    ],
  },
  import: {
    title: "Import",
    lead: "Upload clips from your browser. Transfers resume if they are interrupted.",
    tabs: [
      {
        name: "Upload",
        blocks: [
          { k: "drop" },
          {
            k: "progress",
            items: [
              ["raid_night_03.mp4", 64, "4.1 GB of 6.4 GB"],
              ["bind_clutch.mp4", 100, "Done"],
              ["lan_party.mkv", 12, "Paused"],
            ],
          },
        ],
      },
    ],
  },
  workers: {
    title: "Workers",
    lead: "A worker transcribes and analyses voices on your gaming PC.",
    tabs: [
      {
        name: "Workers",
        blocks: [{ k: "workers" }],
      },
    ],
  },
  jobs: {
    title: "Job queue",
    lead: "Nothing starts by itself. You schedule work, then pause or resume it here.",
    tabs: [
      {
        name: "Queue",
        blocks: [
          {
            k: "toggle",
            label: "Pause the queue",
            help: "Running tasks finish, new ones wait.",
            on: false,
          },
          {
            k: "stats",
            items: [
              ["Running", "1"],
              ["Waiting", "12"],
              ["Failed", "2"],
              ["Concurrency", "1"],
            ],
          },
          {
            k: "table",
            cols: ["Task", "Clip", "State", ""],
            rows: [
              [
                "Transcription",
                "Clutch 1v3 sur Bind",
                chip("Running", "info"),
                "Cancel",
              ],
              [
                "Voice processing",
                "Ace en finale",
                chip("Waiting", "warn"),
                "Cancel",
              ],
              [
                "Transcription",
                "Le bug du siècle",
                chip("Failed · model unreachable", "bad"),
                "Retry",
              ],
            ],
          },
        ],
      },
    ],
  },
  sources: {
    title: "Folders",
    lead: "Allowed folders Snitch can reconcile. Nothing is scanned until you ask.",
    action: "Add folder",
    tabs: [
      {
        name: "Folders",
        blocks: [
          {
            k: "banner",
            tone: "warn",
            text: "21% of the tracked files in “raid-captures” are missing. Check that the drive is mounted before scanning again.",
          },
          {
            k: "table",
            cols: ["Folder", "Files", "Last scan", "Microphone track", ""],
            rows: [
              ["/clips/ranked", "182", "Yesterday", "Track 2 · Léa", "Scan"],
              ["/clips/raid-captures", "66", "3 days ago", "Not set", "Scan"],
            ],
          },
        ],
      },
    ],
  },
  transcription: {
    title: "Transcription",
    lead: "How clips are transcribed, and the game terms the model should know.",
    tabs: [
      {
        name: "Model",
        blocks: [{ k: "transcription" }],
      },
      {
        name: "Vocabulary",
        blocks: [
          {
            k: "table",
            cols: ["Term", "Variants", "Preferred spelling", ""],
            rows: [
              ["Bind", "bine, bayned", "Bind", "Edit"],
              ["Ace", "esse", "Ace", "Edit"],
            ],
            action: "Add term",
          },
        ],
      },
    ],
  },
  voice: {
    title: "Voice processing",
    lead: "Diarization, matching and automatic attribution. Automation is off by default.",
    tabs: [
      {
        name: "Pipeline",
        blocks: [
          {
            k: "banner",
            tone: "info",
            text: "A real voice model needs your confirmation before first use.",
          },
          {
            k: "table",
            cols: ["Step", "Version", "State"],
            rows: [
              [
                "Diarization",
                "community-1",
                chip("Model not confirmed", "warn"),
              ],
              ["Fingerprints", "v1", chip("Ready", "ok")],
            ],
          },
        ],
      },
      {
        name: "Matching",
        blocks: [
          {
            k: "form",
            title: "Thresholds",
            fields: [
              ["Automatic above", "0.85"],
              ["Suggestion above", "0.60"],
            ],
            button: "Save thresholds",
          },
        ],
      },
      {
        name: "Automation",
        blocks: [
          {
            k: "stats",
            items: [
              ["Measured precision", "91%"],
              ["Held-out clips", "38"],
              ["Criteria met", "No"],
            ],
          },
          {
            k: "toggle",
            label: "Automatic attribution",
            help: "Off: matches stay suggestions. Displayed values are not a guarantee.",
            on: false,
          },
        ],
      },
    ],
  },
  index: {
    title: "Semantic index",
    lead: "Optional. Without it, full-text search keeps working.",
    tabs: [
      {
        name: "Embeddings",
        blocks: [
          {
            k: "form",
            title: "Embeddings endpoint",
            fields: [
              ["Endpoint", "http://embed:11434/v1/embeddings"],
              ["Model", "Not set"],
            ],
            button: "Save",
          },
          {
            k: "stats",
            items: [
              ["Indexed segments", "0"],
              ["Waiting to reindex", "0"],
              ["Active profile", "None"],
            ],
          },
        ],
      },
    ],
  },
  users: {
    title: "Users & access",
    lead: "Approve new accounts and set how people sign in.",
    tabs: [
      { name: "Members", blocks: [{ k: "members" }] },
      {
        name: "Pending",
        blocks: [
          {
            k: "table",
            cols: ["Name", "Requested", ""],
            rows: [
              ["alex", "Today", "Approve"],
              ["sam", "2 days ago", "Approve"],
            ],
          },
        ],
      },
      {
        name: "Sign-in",
        blocks: [
          { k: "toggle", label: "Require approval for new accounts", on: true },
          {
            k: "form",
            title: "Discord (optional)",
            lead: "Use your own Discord application, then register the return URL shown below.",
            fields: [
              ["Application ID", ""],
              ["Secret", ""],
              ["Return URL", "http://localhost:8080/api/auth/discord/callback"],
            ],
            button: "Save Discord",
          },
        ],
      },
    ],
  },
  exports: {
    title: "Exports",
    lead: "Take your data out. Exports keep technical column names.",
    tabs: [
      {
        name: "Corrections",
        blocks: [
          {
            k: "form",
            title: "Corrected segments",
            lead: "Original text, corrected text, author, date and version. No error rate is computed.",
            fields: [],
            button: "Download export",
          },
        ],
      },
    ],
  },
  account: {
    title: "Account settings",
    lead: "Your sign-in, your voice, your preferences.",
    tabs: [
      {
        name: "Profile",
        blocks: [
          {
            k: "toggle",
            label: "Link your Discord identity",
            help: "Lets you sign in with Discord.",
            on: false,
          },
          {
            k: "form",
            title: "Preferences",
            fields: [["Language", "English"]],
            button: "Save",
          },
          { k: "deleteaccount" },
        ],
      },
      {
        name: "My voice",
        blocks: [
          {
            k: "banner",
            tone: "info",
            text: "You are identified in 22 clips. Deleting your voice profile erases samples, fingerprints and attributions; your voices become unknown.",
          },
          {
            k: "form",
            title: "Delete my voice profile",
            fields: [],
            button: "Delete voice profile",
          },
        ],
      },
    ],
  },
};

function Chip({ c }: { c: Cell }) {
  if (typeof c === "string") return <>{c}</>;
  return <span className={`pill ${c.tone ?? "info"}`}>{c.chip}</span>;
}

function Blk({ b }: { b: Block }) {
  const [on, setOn] = useState(b.k === "toggle" ? !!b.on : false);
  switch (b.k) {
    case "table":
      return (
        <div className="panel">
          {(b.title || b.action) && (
            <div className="panel-h">
              <b>{b.title ?? ""}</b>
              {b.action && <button className="btn">{b.action}</button>}
            </div>
          )}
          <table>
            <thead>
              <tr>
                {b.cols.map((c) => (
                  <th key={c}>{c}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {b.rows.map((r, i) => (
                <tr key={i}>
                  {r.map((c, j) => (
                    <td key={j}>
                      <Chip c={c} />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
    case "form":
      return (
        <div className="panel pad">
          <b>{b.title}</b>
          {b.lead && <p className="mute">{b.lead}</p>}
          {b.fields.map(([l, v]) => (
            <label key={l}>
              {l}
              <input defaultValue={v} />
            </label>
          ))}
          <div>
            <button className="btn pri">{b.button}</button>
          </div>
        </div>
      );
    case "stats":
      return (
        <div className="stats">
          {b.items.map(([l, v]) => (
            <div key={l} className="stat">
              <span className="mute">{l}</span>
              <b>{v}</b>
            </div>
          ))}
        </div>
      );
    case "toggle":
      return (
        <div className="panel pad row2">
          <div>
            <b>{b.label}</b>
            {b.help && <p className="mute">{b.help}</p>}
          </div>
          <button
            role="switch"
            aria-checked={on}
            className={`sw${on ? " on" : ""}`}
            aria-label={b.label}
            onClick={() => setOn(!on)}
          >
            <i />
          </button>
        </div>
      );
    case "banner":
      return <div className={`banner ${b.tone}`}>{b.text}</div>;
    case "code":
      return (
        <div className="panel pad">
          <b>{b.title}</b>
          <pre>{b.text}</pre>
        </div>
      );
    case "results":
      return (
        <div className="panel">
          {b.items.map((r) => (
            <div key={r.tc + r.clip} className="res">
              <span className="tc">{r.tc}</span>
              <div>
                <p className="quote">
                  {r.before}
                  <mark>{r.hit}</mark>
                  {r.after}
                </p>
                <span className="mute">
                  {r.clip} · {r.who}
                </span>
              </div>
            </div>
          ))}
        </div>
      );
    case "cards":
      return (
        <div className="grid2">
          {b.items.map((c) => (
            <div key={c.title} className="panel pad">
              <b>{c.title}</b>
              <span className="mute">{c.sub}</span>
              <span className="mute">{c.meta}</span>
              <div>
                <button className="btn">{c.action}</button>
              </div>
            </div>
          ))}
        </div>
      );
    case "deleteaccount":
      return <DeleteAccount />;
    case "transcription":
      return <Transcription />;
    case "members":
      return <Members />;
    case "workers":
      return <Workers />;
    case "groups":
      return <VoiceGroups />;
    case "verification":
      return <Verification />;
    case "drop":
      return <div className="drop">Drop videos here, or choose files</div>;
    case "progress":
      return (
        <div className="panel">
          {b.items.map(([n, p, s]) => (
            <div key={n} className="up">
              <span>{n}</span>
              <div className="pbar" role="progressbar" aria-valuenow={p}>
                <i style={{ width: `${p}%` }} />
              </div>
              <span className="mute">{s}</span>
            </div>
          ))}
        </div>
      );
  }
}

export function PrototypePage({
  id,
  embedded,
}: {
  id: string;
  embedded?: boolean;
}) {
  const page = PAGES[id];
  const [tab, setTab] = useState(0);
  if (!page) return null;
  const cur = page.tabs[Math.min(tab, page.tabs.length - 1)];
  const tabs =
    page.tabs.length > 1 ? (
      embedded ? (
        <div className="subtabs" role="tablist">
          {page.tabs.map((t, i) => (
            <button
              key={t.name}
              role="tab"
              aria-selected={i === tab}
              className={i === tab ? "on" : ""}
              onClick={() => setTab(i)}
            >
              {t.name}
            </button>
          ))}
        </div>
      ) : (
        <div className="tabs" role="tablist">
          {page.tabs.map((t, i) => (
            <button
              key={t.name}
              role="tab"
              aria-selected={i === tab}
              className={i === tab ? "on" : ""}
              onClick={() => setTab(i)}
            >
              {t.name}
            </button>
          ))}
        </div>
      )
    ) : null;
  return (
    <section
      className={embedded ? "" : "page"}
      style={
        embedded
          ? { display: "flex", flexDirection: "column", gap: 18 }
          : undefined
      }
    >
      <div className="phead">
        <div>
          {embedded ? (
            <h2 style={{ margin: 0 }}>{page.title}</h2>
          ) : (
            <h1>{page.title}</h1>
          )}
          <p className="mute">{page.lead}</p>
        </div>
        {page.action && <button className="btn pri">{page.action}</button>}
      </div>
      {tabs}
      <div className="blocks">
        {cur.blocks.map((b, i) => (
          <Blk key={i} b={b} />
        ))}
      </div>
    </section>
  );
}

export function AdminPage() {
  const [id, setId] = useState(ADMIN_TABS[0].id);
  return (
    <section className="page" style={{ paddingTop: 20 }}>
      <h1>Administration</h1>
      <div className="tabs" role="tablist" aria-label="Administration">
        {ADMIN_TABS.map((t) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={t.id === id}
            className={t.id === id ? "on" : ""}
            onClick={() => setId(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>
      <PrototypePage id={id} embedded key={id} />
    </section>
  );
}
