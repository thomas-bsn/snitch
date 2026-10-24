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
type Upload = components["schemas"]["Upload"];
type Clip = components["schemas"]["Clip"];
type Job = components["schemas"]["Job"];
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
  const [uploads, setUploads] = useState<Upload[]>([]);
  const [clips, setClips] = useState<Clip[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [jobPage, setJobPage] = useState(1);
  const [jobTotal, setJobTotal] = useState(0);
  const [jobFilters, setJobFilters] = useState({
    status: "" as "" | "pending" | "leased" | "done" | "failed",
    clip_id: "",
    step: "",
  });
  const [clipPage, setClipPage] = useState(1);
  const [clipTotal, setClipTotal] = useState(0);
  const [uploadVisibility, setUploadVisibility] = useState<
    "shared" | "private"
  >("shared");
  const [clipFilters, setClipFilters] = useState({
    visibility: "",
    status: "",
    owner_id: "",
    source_id: "",
    from: "",
    to: "",
  });
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
    if (me.data) {
      await refreshUploads();
      await refreshClips(1);
    }
    if (me.data?.user.role === "admin") {
      setSettings(checked(await api.GET("/api/admin/settings")));
      setPending(checked(await api.GET("/api/admin/users")));
      await refreshJobs(1);
      await refreshSources();
    } else {
      setSources([]);
      setAllowedRoots([]);
      setJobs([]);
    }
  }
  async function refreshClips(page: number) {
    const query = new URLSearchParams({ page: String(page), page_size: "12" });
    for (const [key, value] of Object.entries(clipFilters))
      if (value) query.set(key, value);
    const response = await fetch(`/api/clips?${query}`, {
      credentials: "same-origin",
    });
    if (!response.ok) throw new Error("Impossible de charger la bibliothèque.");
    const result = await response.json();
    setClips(result.items);
    setClipPage(result.page);
    setClipTotal(result.total);
  }
  async function refreshUploads() {
    const result = await api.GET("/api/uploads", { params: { header: csrf } });
    if (result.data) setUploads(result.data.items);
  }
  async function refreshJobs(page: number) {
    const result = checked(
      await api.GET("/api/admin/jobs", {
        params: {
          query: {
            page,
            page_size: 10,
            status: jobFilters.status || undefined,
            clip_id: jobFilters.clip_id
              ? Number(jobFilters.clip_id)
              : undefined,
            step: jobFilters.step || undefined,
          },
        },
      }),
    );
    setJobs(result.items);
    setJobPage(result.page);
    setJobTotal(result.total);
  }
  async function scheduleJob(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    await run(async () => {
      const result = await api.POST("/api/admin/jobs", {
        params: { header: csrf },
        body: {
          clip_id: Number(data.get("clip_id")),
          step: String(data.get("step")),
          input_version: String(data.get("input_version")),
          priority: Number(data.get("priority")),
        },
      });
      checked(result);
      form.reset();
      await refreshJobs(1);
      setNotice(
        result.response.status === 201
          ? "Tâche planifiée."
          : "Cette étape et cette version sont déjà dans la file.",
      );
    });
  }
  async function filterJobs(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await run(() => refreshJobs(1));
  }
  async function addUpload(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const file =
      form.querySelector<HTMLInputElement>('input[type="file"]')?.files?.[0];
    if (!file) return;
    await run(async () => {
      const upload = checked(
        await api.POST("/api/uploads", {
          params: { header: csrf },
          body: {
            filename: file.name,
            size_bytes: file.size,
            visibility: uploadVisibility,
          },
        }),
      );
      for (let offset = upload.offset_bytes; offset < file.size; ) {
        const end = Math.min(offset + upload.chunk_size_bytes, file.size);
        const response = await fetch(`/api/uploads/${upload.id}/chunks`, {
          method: "PUT",
          credentials: "same-origin",
          headers: {
            ...csrf,
            "Content-Type": "application/octet-stream",
            "Content-Range": `bytes ${offset}-${end - 1}/${file.size}`,
          },
          body: file.slice(offset, end),
        });
        const body = await response.json();
        if (!response.ok)
          throw new Error(
            messages[body.error] ?? "L’envoi du fichier a échoué.",
          );
        offset = body.offset_bytes;
        setNotice(
          `Envoi de ${file.name} : ${Math.round((offset / file.size) * 100)} %`,
        );
      }
      checked(
        await api.POST("/api/uploads/{id}/complete", {
          params: { path: { id: upload.id }, header: csrf },
        }),
      );
      form.reset();
      await refreshUploads();
      await refreshClips(1);
      setNotice(
        `« ${file.name} » a été vérifiée et ajoutée à la bibliothèque.`,
      );
    });
  }
  async function resumeUpload(upload: Upload) {
    const input = document.createElement("input");
    input.type = "file";
    input.onchange = async () => {
      const file = input.files?.[0];
      if (!file) return;
      if (file.size !== upload.size_bytes) {
        setError(
          "Sélectionnez le même fichier (nom et taille) pour reprendre cet envoi.",
        );
        return;
      }
      await run(async () => {
        for (let offset = upload.offset_bytes; offset < file.size; ) {
          const end = Math.min(offset + upload.chunk_size_bytes, file.size);
          const response = await fetch(`/api/uploads/${upload.id}/chunks`, {
            method: "PUT",
            credentials: "same-origin",
            headers: {
              ...csrf,
              "Content-Type": "application/octet-stream",
              "Content-Range": `bytes ${offset}-${end - 1}/${file.size}`,
            },
            body: file.slice(offset, end),
          });
          const body = await response.json();
          if (!response.ok)
            throw new Error(
              messages[body.error] ?? "La reprise de l’envoi a échoué.",
            );
          offset = body.offset_bytes;
        }
        checked(
          await api.POST("/api/uploads/{id}/complete", {
            params: { path: { id: upload.id }, header: csrf },
          }),
        );
        await refreshUploads();
        await refreshClips(1);
      });
    };
    input.click();
  }
  async function cancelUpload(upload: Upload) {
    await run(async () => {
      checked(
        await api.DELETE("/api/uploads/{id}", {
          params: { path: { id: upload.id }, header: csrf },
        }),
      );
      await refreshUploads();
      await refreshClips(1);
    });
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
            <p className="eyebrow">IMPORTER UNE VIDÉO</p>
            <h2>Ajouter à la bibliothèque</h2>
            <p>
              Les envois sont découpés pour reprendre après une coupure. Ils
              apparaissent dans la bibliothèque après validation.
            </p>
            <form onSubmit={addUpload}>
              <label>
                Visibilité
                <select
                  value={uploadVisibility}
                  onChange={(event) =>
                    setUploadVisibility(
                      event.target.value as "private" | "shared",
                    )
                  }
                >
                  <option value="shared">Partagée avec les membres</option>
                  <option value="private">
                    Privée (vous et les administrateurs)
                  </option>
                </select>
              </label>
              <label>
                Fichier vidéo
                <input
                  type="file"
                  accept="video/*,.mkv,.mp4,.mov,.webm"
                  required
                />
              </label>
              <button disabled={busy}>Envoyer la vidéo</button>
            </form>
            <h3>Envois à reprendre ({uploads.length})</h3>
            {uploads.length === 0 ? (
              <p>Aucun envoi en attente.</p>
            ) : (
              <ul>
                {uploads.map((upload) => (
                  <li className="source" key={upload.id}>
                    <strong>{upload.original_name}</strong> ·{" "}
                    {upload.offset_bytes} / {upload.size_bytes} octets ·{" "}
                    {upload.status}
                    <div className="toolbar">
                      <button
                        className="secondary"
                        disabled={busy || upload.status !== "uploading"}
                        onClick={() => resumeUpload(upload)}
                      >
                        Reprendre
                      </button>
                      <button
                        className="secondary"
                        disabled={busy || upload.status !== "uploading"}
                        onClick={() => cancelUpload(upload)}
                      >
                        Annuler
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>
          <section className="card" aria-label="Bibliothèque">
            <h2>Bibliothèque ({clipTotal})</h2>
            <form
              onSubmit={(event) => {
                event.preventDefault();
                run(() => refreshClips(1));
              }}
            >
              <label>
                Visibilité
                <select
                  value={clipFilters.visibility}
                  onChange={(e) =>
                    setClipFilters({
                      ...clipFilters,
                      visibility: e.target.value,
                    })
                  }
                >
                  <option value="">Toutes</option>
                  <option value="shared">Partagées</option>
                  <option value="private">Privées</option>
                </select>
              </label>
              <label>
                État
                <select
                  value={clipFilters.status}
                  onChange={(e) =>
                    setClipFilters({ ...clipFilters, status: e.target.value })
                  }
                >
                  <option value="">Tous</option>
                  <option value="ready">Prêtes</option>
                  <option value="waiting_stability">En attente</option>
                  <option value="unreadable">Illisibles</option>
                  <option value="missing">Original introuvable</option>
                </select>
              </label>
              <label>
                Propriétaire (ID)
                <input
                  type="number"
                  min="1"
                  value={clipFilters.owner_id}
                  onChange={(e) =>
                    setClipFilters({ ...clipFilters, owner_id: e.target.value })
                  }
                />
              </label>
              <label>
                Source (ID)
                <input
                  type="number"
                  min="1"
                  value={clipFilters.source_id}
                  onChange={(e) =>
                    setClipFilters({
                      ...clipFilters,
                      source_id: e.target.value,
                    })
                  }
                />
              </label>
              <label>
                Du
                <input
                  type="date"
                  value={clipFilters.from}
                  onChange={(e) =>
                    setClipFilters({ ...clipFilters, from: e.target.value })
                  }
                />
              </label>
              <label>
                Au
                <input
                  type="date"
                  value={clipFilters.to}
                  onChange={(e) =>
                    setClipFilters({ ...clipFilters, to: e.target.value })
                  }
                />
              </label>
              <button disabled={busy}>Filtrer</button>
            </form>
            {clips.length === 0 ? (
              <p>Aucun clip visible pour ces filtres.</p>
            ) : (
              <ul>
                {clips.map((clip) => (
                  <li className="source" key={clip.id}>
                    <strong>{clip.original_name}</strong> · {clip.status} ·{" "}
                    {clip.visibility} · {clip.owner}
                    <p className="hint">
                      {new Date(clip.recorded_at).toLocaleDateString("fr-FR")} ·{" "}
                      {clip.duration_seconds?.toFixed(2) ?? "Durée inconnue"} s
                      · {clip.video_codec || "codec inconnu"}
                    </p>
                    <details>
                      <summary>Détails et lecture</summary>
                      <p>
                        {clip.format_name} · audio{" "}
                        {clip.audio_codec || "absent"} · {clip.size_bytes}{" "}
                        octets · sources {clip.sources?.join(", ") || "upload"}
                      </p>
                      {clip.uploaders.length > 0 && (
                        <p>Importé par {clip.uploaders.join(", ")}</p>
                      )}
                      {clip.status === "ready" && (
                        <video
                          controls
                          preload="metadata"
                          src={`/api/clips/${clip.id}/media`}
                          aria-label={`Lecture ${clip.original_name}`}
                        />
                      )}
                    </details>
                  </li>
                ))}
              </ul>
            )}
            <div className="toolbar">
              <button
                className="secondary"
                disabled={busy || clipPage <= 1}
                onClick={() => run(() => refreshClips(clipPage - 1))}
              >
                Précédente
              </button>
              <span>
                Page {clipPage} · {clipTotal} clips
              </span>
              <button
                className="secondary"
                disabled={busy || clipPage * 12 >= clipTotal}
                onClick={() => run(() => refreshClips(clipPage + 1))}
              >
                Suivante
              </button>
            </div>
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
              <h2>File de tâches ({jobTotal})</h2>
              <p className="hint">
                Les tâches restent en attente jusqu’à ce qu’un worker prenne en
                charge leur étape.
              </p>
              <form onSubmit={scheduleJob}>
                <label>
                  ID du clip
                  <input name="clip_id" type="number" min="1" required />
                </label>
                <label>
                  Étape
                  <input
                    name="step"
                    required
                    maxLength={64}
                    pattern="[a-z][a-z0-9_.-]{0,63}"
                  />
                </label>
                <label>
                  Version d’entrée
                  <input name="input_version" required maxLength={128} />
                </label>
                <label>
                  Priorité
                  <input
                    name="priority"
                    type="number"
                    min="-100"
                    max="100"
                    defaultValue="0"
                  />
                </label>
                <button disabled={busy}>Planifier l’étape</button>
              </form>
              <form onSubmit={filterJobs}>
                <label>
                  État de la tâche
                  <select
                    value={jobFilters.status}
                    onChange={(event) =>
                      setJobFilters({
                        ...jobFilters,
                        status: event.target.value as typeof jobFilters.status,
                      })
                    }
                  >
                    <option value="">Tous les états</option>
                    <option value="pending">En attente</option>
                    <option value="leased">Louée</option>
                    <option value="done">Terminée</option>
                    <option value="failed">Échouée</option>
                  </select>
                </label>
                <label>
                  ID du clip à filtrer
                  <input
                    type="number"
                    min="1"
                    value={jobFilters.clip_id}
                    onChange={(event) =>
                      setJobFilters({
                        ...jobFilters,
                        clip_id: event.target.value,
                      })
                    }
                  />
                </label>
                <label>
                  Étape à filtrer
                  <input
                    value={jobFilters.step}
                    onChange={(event) =>
                      setJobFilters({ ...jobFilters, step: event.target.value })
                    }
                  />
                </label>
                <button className="secondary" disabled={busy}>
                  Filtrer les tâches
                </button>
              </form>
              {jobs.length === 0 ? (
                <p>Aucune tâche ne correspond aux filtres.</p>
              ) : (
                <ul aria-label="Tâches">
                  {jobs.map((job) => (
                    <li key={job.id}>
                      <strong>{job.step}</strong>
                      <p>
                        Tâche {job.id} · clip {job.clip_id} · {job.status} ·
                        priorité {job.priority} · tentatives {job.attempts}/
                        {job.max_attempts}
                      </p>
                      <p className="hint">
                        Version {job.input_version} · créée{" "}
                        {new Date(job.created_at).toLocaleString("fr-FR")}
                      </p>
                    </li>
                  ))}
                </ul>
              )}
              <div className="toolbar">
                <button
                  className="secondary"
                  disabled={busy || jobPage <= 1}
                  onClick={() => run(() => refreshJobs(jobPage - 1))}
                >
                  Tâches précédentes
                </button>
                <span>Page {jobPage}</span>
                <button
                  className="secondary"
                  disabled={busy || jobPage * 10 >= jobTotal}
                  onClick={() => run(() => refreshJobs(jobPage + 1))}
                >
                  Tâches suivantes
                </button>
              </div>
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
      <footer>
        Phase 1 · Ingestion et bibliothèque
        {session && (
          <>
            {" "}
            · <span>Worker : phase 2</span> · <span>Embeddings : phase 4</span>
          </>
        )}
      </footer>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<App />);
