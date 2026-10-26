// PROTOTYPE — "Atelier" look on real data (setup, session, clips). Throwaway.
// Question: does this wizard + library feel right with the real API? Read-only
// except the real setup/login/logout calls; wizard steps 3-7 are stubs.
import { useEffect, useState } from "react";
import createClient from "openapi-fetch";
import type { paths, components } from "../api.gen";
import "./atelier.css";
import { ClipView } from "./ClipView";
import { Wizard } from "./Wizard";
import { AdminPage, PrototypePage } from "./Pages";

const api = createClient<paths>();
type Setup = components["schemas"]["Setup"];
type Session = components["schemas"]["Session"];
type Clip = components["schemas"]["Clip"];

const T = {
  en: {
    setupTitle: "Set up your library",
    setupLead:
      "Create the administrator account first. Everything else is optional and nothing starts without your say-so.",
    steps: [
      "Installation code",
      "Administrator",
      "Discord",
      "Semantic search",
      "First worker",
      "Import videos",
      "Summary",
    ],
    optional: "Optional",
    step: (n: number, t: number) => `Step ${n} of ${t}`,
    code: "Installation code",
    codeHelp:
      "Find it with: docker compose logs app. Valid 15 minutes, single use.",
    codeTitle: "Enter the installation code",
    adminTitle: "Create the administrator",
    adminLead:
      "This account manages access, workers and sources. You can add others later.",
    name: "Your name",
    password: "Password",
    passwordHelp: "12 to 256 characters. Use a passphrase.",
    confirm: "Confirm password",
    mismatch: "Passwords do not match.",
    back: "Back",
    next: "Continue",
    skip: "Skip for now",
    create: "Create administrator",
    stub: "Prototype stub: nothing is saved here.",
    discord: [
      "Connect Discord",
      "Let members sign in with Discord. You bring your own Discord application.",
    ],
    embed: [
      "Semantic search",
      "Point Snitch at an OpenAI-compatible embeddings endpoint. Full-text search works without it.",
    ],
    worker: [
      "Create your first worker",
      "A worker transcribes clips on your gaming PC. You will get a ready-made compose file.",
    ],
    imp: [
      "Import your first videos",
      "Upload from the browser. Nothing is scanned or processed unless you ask.",
    ],
    summary: "All set",
    summaryLead:
      "You can finish the optional steps later from the administration pages.",
    open: "Open the library",
    language: "Language",
    library: "Library",
    search: "Search",
    people: "People",
    importNav: "Import",
    searchPh: "Search the lyrics…",
    clips: (n: number) => `${n} clips`,
    filters: ["Date", "Source", "Owner", "Person", "State"],
    empty: "No clips yet. Import your first videos to get started.",
    ready: "✓ Ready",
    other: "… Not ready",
    signIn: "Sign in",
    signInLead: "Use your Snitch account.",
    signOut: "Sign out",
  },
  fr: {
    setupTitle: "Configurez votre bibliothèque",
    setupLead:
      "Créez d’abord le compte administrateur. Le reste est facultatif et rien ne démarre sans votre accord.",
    steps: [
      "Code d’installation",
      "Administrateur",
      "Discord",
      "Recherche sémantique",
      "Premier worker",
      "Import de vidéos",
      "Récapitulatif",
    ],
    optional: "Facultatif",
    step: (n: number, t: number) => `Étape ${n} sur ${t}`,
    code: "Code d’installation",
    codeHelp:
      "Lisible avec : docker compose logs app. Valable 15 minutes, à usage unique.",
    codeTitle: "Saisissez le code d’installation",
    adminTitle: "Créez l’administrateur",
    adminLead:
      "Ce compte gère les accès, les workers et les sources. Vous pourrez en ajouter d’autres.",
    name: "Votre nom",
    password: "Mot de passe",
    passwordHelp: "12 à 256 caractères. Une phrase de passe convient.",
    confirm: "Confirmer le mot de passe",
    mismatch: "Les mots de passe ne correspondent pas.",
    back: "Retour",
    next: "Continuer",
    skip: "Passer pour l’instant",
    create: "Créer l’administrateur",
    stub: "Maquette : rien n’est enregistré ici.",
    discord: [
      "Connecter Discord",
      "Permettez la connexion par Discord. Vous fournissez votre propre application Discord.",
    ],
    embed: [
      "Recherche sémantique",
      "Indiquez un point d’accès d’embeddings compatible OpenAI. La recherche plein texte fonctionne sans.",
    ],
    worker: [
      "Créer votre premier worker",
      "Un worker transcrit les clips sur votre PC de jeu. Vous recevrez un fichier compose prêt à l’emploi.",
    ],
    imp: [
      "Importer vos premières vidéos",
      "Envoyez depuis le navigateur. Rien n’est scanné ni traité sans votre demande.",
    ],
    summary: "Tout est prêt",
    summaryLead:
      "Vous pourrez terminer les étapes facultatives plus tard depuis l’administration.",
    open: "Ouvrir la bibliothèque",
    language: "Langue",
    library: "Bibliothèque",
    search: "Recherche",
    people: "Personnes",
    importNav: "Importer",
    searchPh: "Rechercher dans les paroles…",
    clips: (n: number) => `${n} clips`,
    filters: ["Date", "Source", "Propriétaire", "Personne", "État"],
    empty:
      "Aucun clip pour l’instant. Importez vos premières vidéos pour commencer.",
    ready: "✓ Prêt",
    other: "… Pas prêt",
    signIn: "Connexion",
    signInLead: "Utilisez votre compte Snitch.",
    signOut: "Déconnexion",
  },
};
type Lang = keyof typeof T;

const NAV = [
  { id: "library", icon: "M2 3h12v10H2zM6 3v10" },
  { id: "search", icon: "M7 2a5 5 0 1 0 0 10A5 5 0 0 0 7 2zM11 11l3 3" },
  {
    id: "people",
    icon: "M8 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM2 14c0-3 3-4 6-4s6 1 6 4",
  },
  { id: "import", icon: "M8 11V2M4 6l4-4 4 4M2 14h12" },
];
const grads = [
  "linear-gradient(135deg,#c9d6ff,#e2e2e2)",
  "linear-gradient(135deg,#ffd6c9,#f5e6e0)",
  "linear-gradient(135deg,#c9f0dd,#e2f0f5)",
  "linear-gradient(135deg,#f3d9ff,#e6e6fa)",
];
function dur(s?: number | null) {
  if (s == null) return "—";
  return `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, "0")}`;
}

export function AtelierPrototype() {
  const [lang, setLang] = useState<Lang>("en");
  const t = T[lang];
  const [setup, setSetup] = useState<Setup>();
  const [session, setSession] = useState<Session>();
  const [clips, setClips] = useState<Clip[]>([]);
  const [total, setTotal] = useState(0);
  const [step, setStep] = useState(0);
  const [wizard, setWizard] = useState(false);
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const preview =
    new URLSearchParams(location.search).get("preview") === "wizard";
  const [openClip, setOpenClip] = useState(false);
  const [theme, setTheme] = useState<"light" | "dark" | "system">(() => {
    try {
      const v = localStorage.getItem("snitch-at-theme");
      return v === "dark" || v === "system" ? v : "light";
    } catch {
      return "light";
    }
  });
  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      const dark = theme === "dark" || (theme === "system" && mq.matches);
      document.documentElement.dataset.atTheme = dark ? "dark" : "light";
    };
    apply();
    try {
      localStorage.setItem("snitch-at-theme", theme);
    } catch {
      /* storage unavailable */
    }
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, [theme]);
  const [view, setView] = useState("library");
  const [menu, setMenu] = useState(false);
  function go(v: string) {
    setView(v);
    setOpenClip(false);
    setMenu(false);
  }
  const [loadError, setLoadError] = useState(false);

  async function refresh() {
    if (preview) {
      setSetup({
        initialized: false,
        display_name: "Snitch",
        discord_enabled: false,
        phase: 6,
        deferred_steps: [],
      });
      setWizard(true);
      return;
    }
    const s = await api.GET("/api/setup");
    setSetup(s.data);
    if (s.data && !s.data.initialized) setWizard(true);
    const me = await api.GET("/api/auth/me");
    setSession(me.data);
    if (me.data) {
      const r = await fetch("/api/clips?page=1&page_size=24", {
        credentials: "same-origin",
      });
      if (r.ok) {
        const j = await r.json();
        setClips(j.items);
        setTotal(j.total);
      }
    }
  }
  function load() {
    setLoadError(false);
    refresh().catch(() => setLoadError(true));
  }
  useEffect(load, []);

  const langBox = (
    <div className="lang">
      <span>{t.language}</span>
      <select
        className="btn"
        value={lang}
        onChange={(e) => setLang(e.target.value as Lang)}
      >
        <option value="en">English</option>
        <option value="fr">Français</option>
      </select>
    </div>
  );

  async function login(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError("");
    const d = new FormData(e.currentTarget);
    const r = await api.POST("/api/auth/login", {
      body: {
        username: String(d.get("username")),
        password: String(d.get("password")),
      },
      params: { header: { Origin: location.origin } },
    });
    if (r.error) return setError(`${(r.error as { error: string }).error}`);
    await refresh();
  }

  async function logout() {
    await api.POST("/api/auth/logout", {
      params: {
        header: {
          "X-CSRF-Token": session?.csrf_token ?? "",
          Origin: location.origin,
        },
      },
    });
    setSession(undefined);
  }

  if (loadError)
    return (
      <div className="at">
        <main className="wiz-main">
          <div className="stepbox">
            <h2>Can’t reach Snitch</h2>
            <p className="mute">
              The server did not answer. Check that it is running, then try
              again.
            </p>
            <div className="row">
              <button className="btn pri" onClick={load}>
                Try again
              </button>
            </div>
          </div>
        </main>
      </div>
    );
  if (!setup) return <div className="at" />;

  if (wizard) {
    return (
      <Wizard
        lang={lang}
        setLang={setLang}
        preview={preview}
        onFinish={() => {
          if (preview) location.search = "?variant=atelier";
          else {
            setWizard(false);
            void refresh();
          }
        }}
        createAdmin={async (c, username, password) => {
          if (preview) return undefined;
          const r = await api.POST("/api/setup", {
            body: { username, password, code: c },
            params: { header: { Origin: location.origin } },
          });
          if (r.error) return `${(r.error as { error: string }).error}`;
          await refresh();
          return undefined;
        }}
      />
    );
  }

  if (!session) {
    return (
      <div className="at">
        <main className="wiz-main">
          <form onSubmit={login}>
            <div className="logo">
              <i />
              Snitch
            </div>
            <div>
              <h2>{t.signIn}</h2>
              <p className="mute">{t.signInLead}</p>
            </div>
            <label>
              {t.name}
              <input name="username" required />
            </label>
            <label>
              {t.password}
              <input name="password" type="password" required />
            </label>
            {error && <div className="err">{error}</div>}
            <div className="row">
              <button className="btn pri">{t.signIn}</button>
            </div>
            {langBox}
          </form>
        </main>
      </div>
    );
  }

  return (
    <div className="at">
      <nav className="side">
        <div className="logo">
          <i />
          Snitch
        </div>
        {NAV.map((n) => (
          <button
            key={n.id}
            className={`nav${view === n.id ? " on" : ""}`}
            onClick={() => {
              setView(n.id);
              setOpenClip(false);
            }}
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              aria-hidden="true"
            >
              <path d={n.icon} />
            </svg>
            {n.id === "library"
              ? t.library
              : n.id === "search"
                ? t.search
                : n.id === "people"
                  ? t.people
                  : t.importNav}
          </button>
        ))}
      </nav>
      <div className="grow">
        <header className="top">
          <label className="searchbox">
            <svg
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              aria-hidden="true"
            >
              <circle cx="7" cy="7" r="4.5" />
              <path d="M10.5 10.5 14 14" />
            </svg>
            <input
              placeholder={t.searchPh}
              aria-label={t.search}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  setView("search");
                  setOpenClip(false);
                }
              }}
            />
          </label>
          <button
            className="btn pri"
            onClick={() => {
              setView("import");
              setOpenClip(false);
            }}
          >
            {t.importNav}
          </button>
          <div className="menuwrap">
            <button
              className="avatar"
              aria-haspopup="menu"
              aria-expanded={menu}
              aria-label="Account menu"
              onClick={() => setMenu(!menu)}
            >
              {session.user.username.slice(0, 2).toUpperCase()}
            </button>
            {menu && (
              <>
                <div className="scrim" onClick={() => setMenu(false)} />
                <div className="menu" role="menu">
                  <div className="who">
                    <b>{session.user.username}</b>
                    <span className="mute">
                      {session.user.role === "admin"
                        ? "Administrator"
                        : "Member"}
                    </span>
                  </div>
                  <button
                    className="item"
                    role="menuitem"
                    onClick={() => go("account")}
                  >
                    Account settings
                  </button>
                  {session.user.role === "admin" && (
                    <button
                      className="item"
                      role="menuitem"
                      onClick={() => go("admin")}
                    >
                      Administration
                    </button>
                  )}
                  <hr />
                  <div className="themerow">
                    <span>Theme</span>
                    <div className="seg3" role="group" aria-label="Theme">
                      {(
                        [
                          [
                            "light",
                            "Light",
                            "M8 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM8 1v2M8 13v2M1 8h2M13 8h2M3 3l1.4 1.4M11.6 11.6 13 13M3 13l1.4-1.4M11.6 4.4 13 3",
                          ],
                          [
                            "dark",
                            "Dark",
                            "M13 9.5A5.5 5.5 0 0 1 6.5 3 5.5 5.5 0 1 0 13 9.5z",
                          ],
                          ["system", "System", "M2 3h12v8H2zM6 14h4M8 11v3"],
                        ] as const
                      ).map(([k, label, d]) => (
                        <button
                          key={k}
                          aria-label={label}
                          title={label}
                          aria-pressed={theme === k}
                          className={theme === k ? "on" : ""}
                          onClick={() => setTheme(k)}
                        >
                          <svg
                            width="16"
                            height="16"
                            viewBox="0 0 16 16"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="1.5"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            aria-hidden="true"
                          >
                            <path d={d} />
                          </svg>
                        </button>
                      ))}
                    </div>
                  </div>
                  {langBox}
                  <hr />
                  <button className="item" role="menuitem" onClick={logout}>
                    {t.signOut}
                  </button>
                </div>
              </>
            )}
          </div>
        </header>
        {view === "admin" ? (
          <AdminPage />
        ) : view !== "library" ? (
          <PrototypePage id={view} key={view} />
        ) : openClip ? (
          <ClipView onBack={() => setOpenClip(false)} back={t.library} />
        ) : (
          <section className="page">
            <div style={{ display: "flex", alignItems: "baseline", gap: 12 }}>
              <h1>{t.library}</h1>
              <span className="mute">{t.clips(total)}</span>
            </div>
            <div className="chips">
              {t.filters.map((f) => (
                <button key={f} className="chip">
                  {f} ▾
                </button>
              ))}
            </div>
            {clips.length === 0 ? (
              <div className="empty">
                {t.empty}
                <div style={{ marginTop: 12 }}>
                  <button className="btn" onClick={() => setOpenClip(true)}>
                    View a sample clip
                  </button>
                </div>
              </div>
            ) : (
              <div className="grid">
                {clips.map((c) => (
                  <article
                    key={c.id}
                    className="card"
                    onClick={() => setOpenClip(true)}
                  >
                    <div
                      className="thumb"
                      style={{ background: grads[c.id % grads.length] }}
                    >
                      <span className="dur">{dur(c.duration_seconds)}</span>
                    </div>
                    <div style={{ fontWeight: 500 }}>{c.original_name}</div>
                    <div className="meta">
                      <span>
                        {c.owner} ·{" "}
                        {new Date(c.recorded_at).toLocaleDateString(lang)}
                      </span>
                      <span
                        className={`state ${c.status === "ready" ? "ready" : "other"}`}
                      >
                        {c.status === "ready" ? t.ready : t.other}
                      </span>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </section>
        )}
      </div>
    </div>
  );
}
