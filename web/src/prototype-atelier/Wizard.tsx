// PROTOTYPE — setup wizard (7 steps). Only the admin creation can call the real API; the optional steps are stubs.
import { useState } from "react";
import { WorkerCreate } from "./WorkerCreate";

export type Lang = "en" | "fr";
type Mark = "done" | "skipped";

const D = {
  en: {
    title: "Set up your library",
    lead: "Create the administrator account first. Everything else is optional and nothing starts without your say-so.",
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
    skipped: "Skipped",
    configured: "Done",
    step: (n: number, t: number) => `Step ${n} of ${t}`,
    codeTitle: "Enter the installation code",
    codeHelp:
      "Run docker compose logs app and copy the code from the log line. It is valid for 15 minutes and works once.",
    codeLabel: "Installation code",
    adminTitle: "Create the administrator",
    adminLead:
      "This account manages access, workers and folders. You can add others later.",
    name: "Your name",
    password: "Password",
    passwordHelp: "12 to 256 characters. A passphrase works well.",
    confirm: "Confirm password",
    mismatch: "The passwords do not match.",
    back: "Back",
    next: "Continue",
    skip: "Skip for now",
    create: "Create administrator",
    save: "Save and continue",
    discordTitle: "Sign in with Discord",
    discordLead:
      "Let members sign in with Discord. You use your own Discord application: nothing is shared with anyone else.",
    appId: "Application ID",
    secret: "Secret",
    ret: "Return URL to register in Discord",
    embedTitle: "Semantic search",
    embedLead:
      "Search by meaning, not only by exact words. Point Snitch at an OpenAI-compatible embeddings endpoint. Full-text search works without it.",
    endpoint: "Endpoint",
    model: "Model",
    workerTitle: "Create your first worker",
    workerLead:
      "A worker transcribes clips and analyses voices on your gaming PC. Name it, then copy its compose file to that machine.",
    workerName: "Worker name",
    workerCreate: "Create worker",
    workerShown:
      "Copy this now. The token is shown only once and Snitch cannot show it again.",
    workerCopy: "Copy compose file",
    workerCopied: "Copied",
    workerDone: "I copied it",
    importTitle: "Import your first videos",
    importLead:
      "Upload clips from your browser. Transfers resume if they are interrupted. Nothing is scanned or processed unless you ask.",
    drop: "Drop videos here, or choose files",
    importLater: "Open the Import page",
    summaryTitle: "All set",
    summaryLead:
      "You can finish the optional steps later, from Administration.",
    open: "Open the library",
    language: "Language",
    preview: "Preview: nothing is saved",
    todo: (where: string) => `Skipped. Finish it in ${where}.`,
    where: [
      "Administration › Users & access",
      "Administration › Semantic index",
      "Administration › Workers",
      "Import",
    ],
  },
  fr: {
    title: "Configurez votre bibliothèque",
    lead: "Créez d’abord le compte administrateur. Le reste est facultatif et rien ne démarre sans votre accord.",
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
    skipped: "Passée",
    configured: "Faite",
    step: (n: number, t: number) => `Étape ${n} sur ${t}`,
    codeTitle: "Saisissez le code d’installation",
    codeHelp:
      "Lancez docker compose logs app et copiez le code de la ligne du journal. Il est valable 15 minutes et ne sert qu’une fois.",
    codeLabel: "Code d’installation",
    adminTitle: "Créez l’administrateur",
    adminLead:
      "Ce compte gère les accès, les workers et les dossiers. Vous pourrez en ajouter d’autres.",
    name: "Votre nom",
    password: "Mot de passe",
    passwordHelp: "12 à 256 caractères. Une phrase de passe convient bien.",
    confirm: "Confirmer le mot de passe",
    mismatch: "Les mots de passe ne correspondent pas.",
    back: "Retour",
    next: "Continuer",
    skip: "Passer pour l’instant",
    create: "Créer l’administrateur",
    save: "Enregistrer et continuer",
    discordTitle: "Connexion avec Discord",
    discordLead:
      "Permettez la connexion avec Discord. Vous utilisez votre propre application Discord : rien n’est partagé avec personne.",
    appId: "Identifiant de l’application",
    secret: "Secret",
    ret: "URL de retour à enregistrer dans Discord",
    embedTitle: "Recherche sémantique",
    embedLead:
      "Cherchez par le sens, pas seulement par les mots exacts. Indiquez un point d’accès d’embeddings compatible OpenAI. La recherche plein texte fonctionne sans.",
    endpoint: "Point d’accès",
    model: "Modèle",
    workerTitle: "Créez votre premier worker",
    workerLead:
      "Un worker transcrit les clips et analyse les voix sur votre PC de jeu. Nommez-le, puis copiez son fichier compose sur cette machine.",
    workerName: "Nom du worker",
    workerCreate: "Créer le worker",
    workerShown:
      "Copiez ceci maintenant. Le jeton n’est affiché qu’une fois et Snitch ne pourra plus le montrer.",
    workerCopy: "Copier le fichier compose",
    workerCopied: "Copié",
    workerDone: "Je l’ai copié",
    importTitle: "Importez vos premières vidéos",
    importLead:
      "Envoyez des clips depuis le navigateur. Un transfert interrompu reprend. Rien n’est scanné ni traité sans votre demande.",
    drop: "Déposez des vidéos ici, ou choisissez des fichiers",
    importLater: "Ouvrir la page Importer",
    summaryTitle: "Tout est prêt",
    summaryLead:
      "Vous pourrez terminer les étapes facultatives plus tard, depuis l’administration.",
    open: "Ouvrir la bibliothèque",
    language: "Langue",
    preview: "Aperçu : rien n’est enregistré",
    todo: (where: string) => `Passée. À terminer dans ${where}.`,
    where: [
      "Administration › Utilisateurs et accès",
      "Administration › Index sémantique",
      "Administration › Workers",
      "Importer",
    ],
  },
};

const TOKEN = "snw_9f3c1e7a42b8d05e6a91c4f3";

export function Wizard({
  lang,
  setLang,
  preview,
  createAdmin,
  onFinish,
}: {
  lang: Lang;
  setLang: (l: Lang) => void;
  preview: boolean;
  createAdmin: (
    code: string,
    username: string,
    password: string,
  ) => Promise<string | undefined>;
  onFinish: () => void;
}) {
  const t = D[lang];
  const total = t.steps.length;
  const [step, setStep] = useState(0);
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [marks, setMarks] = useState<Record<number, Mark>>({});
  const [created, setCreated] = useState(false);
  const [copied, setCopied] = useState(false);

  const mark = (m: Mark) => {
    setMarks({ ...marks, [step]: m });
    setStep(step + 1);
  };
  const skipBtn = (
    <button type="button" className="btn" onClick={() => mark("skipped")}>
      {t.skip}
    </button>
  );

  async function admin(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError("");
    const d = new FormData(e.currentTarget);
    if (d.get("password") !== d.get("confirm")) return setError(t.mismatch);
    setBusy(true);
    const err = await createAdmin(
      code,
      String(d.get("username")),
      String(d.get("password")),
    );
    setBusy(false);
    if (err) {
      setError(err);
      return;
    }
    setStep(2);
  }

  const head = (title: string, lead?: string) => (
    <div>
      <div className="eyebrow">{t.step(step + 1, total)}</div>
      <h2>{title}</h2>
      {lead && (
        <p className="mute" style={{ margin: 0 }}>
          {lead}
        </p>
      )}
    </div>
  );

  return (
    <div className="at">
      <aside className="wiz-side">
        <div className="logo">
          <i />
          Snitch
        </div>
        <div>
          <h1>{t.title}</h1>
          <p className="mute">{t.lead}</p>
        </div>
        <ol>
          {t.steps.map((label, i) => {
            const m = marks[i];
            const cls = i < step ? "done" : i === step ? "on" : "";
            return (
              <li
                key={label}
                className={cls}
                aria-current={i === step ? "step" : undefined}
              >
                <span className="dot">
                  {i < step ? (m === "skipped" ? "–" : "✓") : i + 1}
                </span>
                {label}
                {m === "skipped" ? (
                  <span className="tag">{t.skipped}</span>
                ) : i >= 2 && i <= 5 && i >= step ? (
                  <span className="tag">{t.optional}</span>
                ) : null}
              </li>
            );
          })}
        </ol>
        <div
          className="lang"
          style={{ flexDirection: "column", alignItems: "flex-start", gap: 10 }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
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
          {preview && <span className="pill warn">{t.preview}</span>}
        </div>
      </aside>

      <main className="wiz-main">
        {step === 0 && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              setStep(1);
            }}
          >
            {head(t.codeTitle, t.codeHelp)}
            <label>
              {t.codeLabel}
              <input
                value={code}
                onChange={(e) => setCode(e.target.value)}
                autoFocus
                required
                spellCheck={false}
              />
            </label>
            <div className="row">
              <button className="btn pri">{t.next}</button>
            </div>
          </form>
        )}

        {step === 1 && (
          <form onSubmit={admin}>
            {head(t.adminTitle, t.adminLead)}
            <label>
              {t.name}
              <input
                name="username"
                minLength={3}
                maxLength={64}
                required
                autoFocus
              />
            </label>
            <label>
              {t.password}
              <input
                name="password"
                type="password"
                minLength={12}
                maxLength={256}
                required
              />
              <span className="mute" style={{ fontWeight: 400, fontSize: 12 }}>
                {t.passwordHelp}
              </span>
            </label>
            <label>
              {t.confirm}
              <input name="confirm" type="password" required />
            </label>
            {error && (
              <div className="err" role="alert">
                {error}
              </div>
            )}
            <div className="row">
              <button type="button" className="btn" onClick={() => setStep(0)}>
                {t.back}
              </button>
              <button className="btn pri" disabled={busy}>
                {t.create}
              </button>
            </div>
          </form>
        )}

        {step === 2 && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              mark("done");
            }}
          >
            {head(t.discordTitle, t.discordLead)}
            <label>
              {t.appId}
              <input inputMode="numeric" />
            </label>
            <label>
              {t.secret}
              <input type="password" autoComplete="off" />
            </label>
            <label>
              {t.ret}
              <input
                readOnly
                value={`${location.origin}/api/auth/discord/callback`}
              />
            </label>
            <div className="row">
              {skipBtn}
              <button className="btn pri">{t.save}</button>
            </div>
          </form>
        )}

        {step === 3 && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              mark("done");
            }}
          >
            {head(t.embedTitle, t.embedLead)}
            <label>
              {t.endpoint}
              <input defaultValue="http://embed:11434/v1/embeddings" />
            </label>
            <label>
              {t.model}
              <input placeholder="nomic-embed-text" />
            </label>
            <div className="row">
              {skipBtn}
              <button className="btn pri">{t.save}</button>
            </div>
          </form>
        )}

        {step === 4 && (
          <div className="stepbox">
            {head(t.workerTitle, t.workerLead)}
            <WorkerCreate
              lang={lang}
              onDone={() => mark("done")}
              skip={skipBtn}
            />
          </div>
        )}

        {step === 5 && (
          <div className="stepbox">
            {head(t.importTitle, t.importLead)}
            <div className="drop">{t.drop}</div>
            <div className="row">
              {skipBtn}
              <button className="btn pri" onClick={() => mark("done")}>
                {t.next}
              </button>
            </div>
          </div>
        )}

        {step === 6 && (
          <div className="stepbox">
            {head(t.summaryTitle, t.summaryLead)}
            <div className="panel">
              <table>
                <tbody>
                  <tr>
                    <td>{t.steps[1]}</td>
                    <td>
                      <span className="pill ok">✓ {t.configured}</span>
                    </td>
                  </tr>
                  {[2, 3, 4, 5].map((i) => (
                    <tr key={i}>
                      <td>{t.steps[i]}</td>
                      <td>
                        {marks[i] === "done" ? (
                          <span className="pill ok">✓ {t.configured}</span>
                        ) : (
                          <span className="mute">{t.todo(t.where[i - 2])}</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="row">
              <button className="btn pri" onClick={onFinish}>
                {t.open}
              </button>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
