import React, { useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import createClient from "openapi-fetch";
import type { paths, components } from "./api.gen";
import "./style.css";

const api = createClient<paths>();
type Setup = components["schemas"]["Setup"];
type Session = components["schemas"]["Session"];
type Settings = components["schemas"]["Settings"];
type User = components["schemas"]["User"];
type Source = components["schemas"]["Source"];
type SourceFile = components["schemas"]["SourceFile"];
type SourceWithFiles = Source & { files: SourceFile[] };
const messages: Record<string, string> = {
  unavailable:
    "Le service est temporairement indisponible. Réessayez dans un instant.",
  discord_denied:
    "Connexion Discord annulée. Vous pouvez vous connecter avec votre compte local.",
  discord_unavailable:
    "Discord est indisponible. Réessayez ou utilisez votre compte local.",
  invalid_oauth_state:
    "Le retour Discord est invalide ou expiré. Recommencez la connexion.",
  link_session_required:
    "La session à l’origine de la liaison a expiré ou changé. Recommencez depuis votre compte.",
  discord_already_linked:
    "Cette identité Discord est déjà liée. L’association existante est conservée.",
  invalid_credentials_or_pending:
    "Identifiants incorrects ou inscription en attente d’approbation.",
  setup_closed_or_invalid_code:
    "Installation déjà terminée, ou code invalide / expiré.",
  username_unavailable: "Ce nom est déjà utilisé.",
  invalid_credentials_format:
    "Nom de 3 à 64 caractères et mot de passe de 12 à 256 caractères requis.",
  rate_limited: "Trop de tentatives. Patientez quelques secondes.",
  busy: "Le serveur est occupé. Réessayez dans quelques secondes.",
  discord_secret_required:
    "Saisissez le secret de votre application Discord pour activer la connexion.",
  discord_credentials_invalid:
    "L’identifiant et le secret Discord ne peuvent pas contenir uniquement des espaces.",
  forbidden: "Accès réservé aux administrateurs.",
  csrf_rejected: "Session expirée. Rechargez la page.",
};
function checked<T>(result: { data?: T; error?: { error: string } }): T {
  if (result.error)
    throw new Error(
      messages[result.error.error] ?? "La demande a échoué. Réessayez.",
    );
  if (!result.data) throw new Error("Réponse du serveur absente.");
  return result.data;
}
function App() {
  const [setup, setSetup] = useState<Setup>();
  const [session, setSession] = useState<Session>();
  const [settings, setSettings] = useState<Settings>();
  const [pending, setPending] = useState<User[]>([]);
  const [sources, setSources] = useState<SourceWithFiles[]>([]);
  const [allowedRoots, setAllowedRoots] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState(
    (() => {
      const notice = new URLSearchParams(location.search).get("notice") ?? "";
      return notice === "pending"
        ? "Inscription Discord en attente d’approbation."
        : (messages[notice] ?? "");
    })(),
  );
  const [register, setRegister] = useState(false);
  const [busy, setBusy] = useState(false);
  const csrf = {
    "X-CSRF-Token": session?.csrf_token ?? "",
    Origin: location.origin,
  };
  async function refresh() {
    const state = checked(await api.GET("/api/setup"));
    setSetup(state);
    document.title = state.display_name;
    const me = await api.GET("/api/auth/me");
    setSession(me.data);
    if (me.data?.user.role === "admin") {
      setSettings(checked(await api.GET("/api/admin/settings")));
      setPending(checked(await api.GET("/api/admin/users")));
      await refreshSources();
    } else {
      setSources([]);
      setAllowedRoots([]);
    }
  }
  async function refreshSources() {
    const result = checked(await api.GET("/api/admin/sources"));
    setAllowedRoots(result.allowed_roots);
    const withFiles = await Promise.all(
      result.items.map(async (source) => {
        const files = checked(
          await api.GET("/api/admin/sources/{id}/files", {
            params: { path: { id: source.id } },
          }),
        );
        return { ...source, files: files.items };
      }),
    );
    setSources(withFiles);
  }
  useEffect(() => {
    refresh().catch((e: Error) => setError(e.message));
  }, []);
  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await action();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Erreur inattendue.");
    } finally {
      setBusy(false);
    }
  }
  async function credentials(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const body = {
      username: String(data.get("username")),
      password: String(data.get("password")),
    };
    await run(async () => {
      if (!setup?.initialized) {
        checked(
          await api.POST("/api/setup", {
            body: { ...body, code: String(data.get("code")) },
            params: { header: { Origin: location.origin } },
          }),
        );
      } else if (register) {
        const result = checked(
          await api.POST("/api/auth/register", {
            body,
            params: { header: { Origin: location.origin } },
          }),
        );
        setNotice(
          result.status === "pending"
            ? "Inscription en attente d’approbation par un administrateur."
            : "Compte créé. Vous pouvez vous connecter.",
        );
        setRegister(false);
        return;
      } else {
        checked(
          await api.POST("/api/auth/login", {
            body,
            params: { header: { Origin: location.origin } },
          }),
        );
      }
      await refresh();
    });
  }
  async function discord(link: boolean) {
    await run(async () => {
      const result = link
        ? checked(
            await api.POST("/api/auth/discord/link", {
              params: { header: csrf },
            }),
          )
        : checked(
            await api.POST("/api/auth/discord/start", {
              params: { header: { Origin: location.origin } },
            }),
          );
      location.assign(result.url);
    });
  }
  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    await run(async () => {
      checked(
        await api.PUT("/api/admin/settings", {
          params: { header: csrf },
          body: {
            client_id: String(data.get("client_id")),
            client_secret: String(data.get("client_secret")),
            registration_approval: data.get("approval") === "on",
          },
        }),
      );
      await refresh();
      setNotice("Réglages enregistrés.");
    });
    form.querySelector<HTMLInputElement>('input[name="client_secret"]')!.value =
      "";
  }
  async function disableDiscord() {
    await run(async () => {
      checked(
        await api.PUT("/api/admin/settings", {
          params: { header: csrf },
          body: {
            client_id: "",
            registration_approval: settings!.registration_approval,
          },
        }),
      );
      await refresh();
      setNotice("Discord désactivé. Les comptes locaux restent disponibles.");
    });
  }
  async function addSource(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    await run(async () => {
      checked(
        await api.POST("/api/admin/sources", {
          params: { header: csrf },
          body: {
            name: String(data.get("source_name")),
            path: String(data.get("source_path")),
          },
        }),
      );
      form.reset();
      await refreshSources();
      setNotice("Source ajoutée. Ses clips apparaîtront après vérification.");
    });
  }
  async function scanSource(source: Source) {
    await run(async () => {
      checked(
        await api.POST("/api/admin/sources/{id}/scan", {
          params: { path: { id: source.id }, header: csrf },
          body: {},
        }),
      );
      await refreshSources();
      setNotice(`Réconciliation planifiée pour « ${source.name} ».`);
    });
  }
  async function disableSource(source: Source) {
    await run(async () => {
      checked(
        await api.DELETE("/api/admin/sources/{id}", {
          params: { path: { id: source.id }, header: csrf },
        }),
      );
      await refreshSources();
      setNotice(
        `Source « ${source.name} » désactivée ; son historique est conservé.`,
      );
    });
  }
  return (
    <main>
      <header>
        <span className="brand-mark">R</span>
        <div>
          <p className="eyebrow">VOTRE MÉMOIRE DE JEU</p>
          <h1>{setup?.display_name ?? "Receipts"}</h1>
        </div>
      </header>
      {error && (
        <p role="alert" className="alert error">
          {error}
        </p>
      )}
      {notice && (
        <p role="status" className="alert">
          {notice}
        </p>
      )}
      {!setup ? (
        <p>Chargement…</p>
      ) : !session ? (
        <section className="card auth">
          <p className="eyebrow">
            {!setup.initialized ? "PREMIER DÉMARRAGE · PHASE 0" : "BIENVENUE"}
          </p>
          <h2>
            {!setup.initialized
              ? "Installer votre espace"
              : register
                ? "Demander un compte"
                : "Retrouver votre espace"}
          </h2>
          <p>
            {!setup.initialized
              ? "Créez le premier compte administrateur avec le code temporaire affiché dans les logs du serveur."
              : "Connectez-vous pour accéder à votre espace."}
          </p>
          <form onSubmit={credentials}>
            {!setup.initialized && (
              <label>
                Code d’installation
                <input name="code" required autoComplete="off" />
              </label>
            )}
            <label>
              Nom d’utilisateur
              <input
                name="username"
                required
                minLength={3}
                maxLength={64}
                autoComplete="username"
              />
            </label>
            <label>
              Mot de passe
              <input
                name="password"
                type="password"
                required
                minLength={12}
                maxLength={256}
                autoComplete={
                  !setup.initialized || register
                    ? "new-password"
                    : "current-password"
                }
              />
            </label>
            <p className="hint">12 caractères minimum.</p>
            <button disabled={busy} type="submit">
              {!setup.initialized
                ? "Créer l’administrateur"
                : register
                  ? "Demander l’accès"
                  : "Se connecter"}
            </button>
          </form>
          {setup.initialized && (
            <button
              className="secondary"
              disabled={busy}
              onClick={() => setRegister(!register)}
            >
              {register ? "Retour à la connexion" : "Créer un compte"}
            </button>
          )}
          {setup.initialized && setup.discord_enabled && (
            <>
              <button
                className="secondary"
                disabled={busy}
                onClick={() => discord(false)}
              >
                Continuer avec Discord
              </button>
              <p className="hint">
                Vous serez redirigé vers Discord. Seule votre identité publique
                est demandée.
              </p>
            </>
          )}
        </section>
      ) : (
        <>
          <div className="toolbar">
            <p>
              Connecté : <strong>{session.user.username}</strong> ·{" "}
              {session.user.role === "admin" ? "Administrateur" : "Utilisateur"}
            </p>
            <button
              className="secondary"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  checked(
                    await api.POST("/api/auth/logout", {
                      params: { header: csrf },
                    }),
                  );
                  setSession(undefined);
                  await refresh();
                })
              }
            >
              Se déconnecter
            </button>
          </div>
          <section className="card">
            <p className="eyebrow">LE SOCLE EST PRÊT</p>
            <h2>Votre bibliothèque arrive à la prochaine phase.</h2>
            <p>
              L’accès et l’installation sont disponibles. Les vidéos ne sont pas
              encore importées.
            </p>
            <ol>
              {setup.deferred_steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          </section>
          {setup.discord_enabled && (
            <section className="card">
              <h2>Lier votre identité Discord</h2>
              <p>
                Cette action associe votre compte actuel à votre identité
                Discord, sans fusion par pseudo. Elle ouvre Discord.
              </p>
              <button disabled={busy} onClick={() => discord(true)}>
                Lier Discord
              </button>
            </section>
          )}
          {session.user.role === "admin" && settings && (
            <section className="card">
              <h2>Accès et Discord</h2>
              <p>
                Discord est facultatif. Laissez l’identifiant vide pour le
                désactiver.
              </p>
              <p>
                {settings.client_id && settings.secret_configured
                  ? "Configuration Discord enregistrée. Vérifiez aussi l’URL de retour dans Discord."
                  : settings.client_id
                    ? "Configuration Discord incomplète : ajoutez le secret pour l’activer."
                    : "Discord désactivé : les comptes locaux restent disponibles."}
              </p>
              <p>
                Créez votre application dans le{" "}
                <a
                  href="https://discord.com/developers/applications"
                  target="_blank"
                  rel="noreferrer"
                >
                  Discord Developer Portal
                </a>
                , puis copiez son Client ID et son Client Secret depuis les
                réglages OAuth2. Ajoutez exactement cette URL aux redirections
                autorisées.
              </p>
              <p className="hint">
                URL de retour à renseigner dans Discord : {location.origin}
                /api/auth/discord/callback
              </p>
              <p className="hint">
                Enregistrer ces réglages ne contacte pas Discord. Une connexion
                ou une liaison ouvre Discord, puis le serveur échange avec ce
                service pour lire l’identité publique (portée identify
                uniquement).
              </p>
              <form
                onSubmit={save}
                key={`${settings.client_id}-${settings.registration_approval}`}
              >
                <label>
                  Identifiant de l’application Discord
                  <input
                    name="client_id"
                    defaultValue={settings.client_id}
                    maxLength={128}
                  />
                </label>
                <label>
                  Secret Discord
                  <input
                    name="client_secret"
                    type="password"
                    autoComplete="new-password"
                    maxLength={2048}
                    placeholder={
                      settings.secret_configured
                        ? "Configuré — laisser vide pour conserver"
                        : "Non configuré"
                    }
                  />
                </label>
                <p className="hint">
                  {settings.secret_configured
                    ? "Secret configuré. Laissez le champ vide pour le conserver."
                    : "Aucun secret configuré. Il est nécessaire pour activer Discord."}
                </p>
                <label className="checkbox">
                  <input
                    type="checkbox"
                    name="approval"
                    defaultChecked={settings.registration_approval}
                  />
                  Approbation administrateur requise pour les nouveaux comptes
                </label>
                <button disabled={busy}>Enregistrer les réglages</button>
                <button type="button" disabled={busy} onClick={disableDiscord}>
                  {settings.client_id
                    ? "Désactiver Discord"
                    : "Continuer sans Discord"}
                </button>
              </form>
            </section>
          )}
          {session.user.role === "admin" && (
            <section className="card">
              <h2>Inscriptions en attente ({pending.length})</h2>
              {pending.length === 0 ? (
                <p>Aucune demande en attente.</p>
              ) : (
                <ul>
                  {pending.map((user) => (
                    <li className="pending" key={user.id}>
                      <span>{user.username}</span>
                      <button
                        disabled={busy}
                        onClick={() =>
                          run(async () => {
                            checked(
                              await api.POST("/api/admin/users/{id}/approve", {
                                params: { path: { id: user.id }, header: csrf },
                              }),
                            );
                            await refresh();
                          })
                        }
                      >
                        Approuver {user.username}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          )}
          {session.user.role === "admin" && (
            <section className="card">
              <h2>Sources dossier</h2>
              {allowedRoots.length === 0 ? (
                <p className="hint">
                  Aucun dossier n’est monté pour Receipts. Configurez
                  CLIPS_ROOTS et un accès hôte en lecture seule dans votre
                  environnement.
                </p>
              ) : (
                <>
                  <p>
                    Dossiers autorisés : {allowedRoots.join(", ")}. Les
                    originaux restent en lecture seule.
                  </p>
                  <form onSubmit={addSource}>
                    <label>
                      Nom de la source
                      <input name="source_name" required maxLength={80} />
                    </label>
                    <label>
                      Chemin du dossier autorisé
                      <input name="source_path" required maxLength={4096} />
                    </label>
                    <button disabled={busy}>Ajouter le dossier</button>
                  </form>
                </>
              )}
              <button
                className="secondary"
                disabled={busy}
                onClick={() => run(refreshSources)}
              >
                Actualiser les sources
              </button>
              {sources.length === 0 ? (
                <p>Aucune source dossier configurée.</p>
              ) : (
                <ul>
                  {sources.map((source) => (
                    <li className="source" key={source.id}>
                      <div className="toolbar">
                        <strong>{source.name}</strong>
                        <span>
                          {source.enabled ? source.health : "désactivée"}
                        </span>
                      </div>
                      <p className="hint">{source.path}</p>
                      {source.health_message && (
                        <p role="status" className="alert">
                          {source.health_message}
                        </p>
                      )}
                      <p className="hint">
                        Dernière réconciliation :{" "}
                        {source.last_scan_at
                          ? new Date(source.last_scan_at).toLocaleString(
                              "fr-FR",
                            )
                          : "aucune"}
                      </p>
                      <button
                        className="secondary"
                        disabled={busy || !source.enabled}
                        onClick={() => scanSource(source)}
                      >
                        Réconcilier
                      </button>
                      {source.enabled && (
                        <button
                          className="secondary"
                          disabled={busy}
                          onClick={() => disableSource(source)}
                        >
                          Désactiver
                        </button>
                      )}
                      <ul>
                        {source.files.map((file) => (
                          <li className="source-file" key={file.relative_path}>
                            <strong>{file.relative_path}</strong> ·{" "}
                            {file.status}
                            {file.error_code && (
                              <span> · {file.error_code}</span>
                            )}
                            {file.status === "ready" && (
                              <p className="hint">
                                {file.duration_seconds?.toFixed(2) ??
                                  "Durée inconnue"}{" "}
                                s
                                {file.video_codec
                                  ? ` · ${file.video_codec}`
                                  : ""}
                                {file.recorded_at
                                  ? ` · ${file.recorded_at}${file.recorded_at_estimated ? " (estimée)" : ""}`
                                  : ""}
                              </p>
                            )}
                          </li>
                        ))}
                      </ul>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          )}
        </>
      )}
      {setup && !session && (
        <section className="card">
          <h2>Les prochaines étapes</h2>
          <p>La bibliothèque sera disponible à la phase 1.</p>
          <ol>
            {setup.deferred_steps.map((step) => (
              <li key={step}>{step}</li>
            ))}
          </ol>
        </section>
      )}
      <footer>Phase 0 · Installation et accès · Aucun média traité</footer>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<App />);
