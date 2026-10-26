import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { useSession } from "../session";
import {
  fetchSetupProgress,
  stepState,
  useSetupActions,
  type SetupProgress,
} from "./setupApi";
import { AdminStep, type AccountDraft } from "./steps/AdminStep";
import { CodeStep } from "./steps/CodeStep";
import { DiscordStep } from "./steps/DiscordStep";
import { ImportStep } from "./steps/ImportStep";
import { SemanticStep } from "./steps/SemanticStep";
import { SummaryStep } from "./steps/SummaryStep";
import { WorkerStep } from "./steps/WorkerStep";
import { OPTIONAL_STEPS, STEP_IDS, serverStep, type StepId } from "./steps";
import { WizardSidebar, standingOf, type Standing } from "./WizardSidebar";

const OPTIONAL_IDS = new Set<StepId>(OPTIONAL_STEPS.map((step) => step.id));

/** The first optional step still to do, or the summary when each one was dealt with. */
function firstStepToDo(progress: SetupProgress): StepId {
  const next = OPTIONAL_STEPS.find(
    (step) => stepState(progress, step.server) === "todo",
  );
  return next?.id ?? "summary";
}

/**
 * The guided installation. Before any account exists it asks for the installation code and
 * creates the Super-administrator; afterwards it offers four optional steps, each of which
 * can be skipped, then a summary. Progress is kept by the server, so the wizard also serves
 * to resume where the person left off.
 */
export function SetupWizard() {
  const { t, error } = useI18n();
  const { setup, completeSetup, reload } = useSession();
  const navigate = useNavigate();
  const { record } = useSetupActions();
  const installing = !setup?.initialized;
  useDocumentTitle(t("setup.title"));

  const [step, setStep] = useState<StepId>(installing ? "code" : "discord");
  const [code, setCode] = useState("");
  const [draft, setDraft] = useState<AccountDraft>({
    username: "",
    password: "",
    confirmation: "",
  });
  const [progress, setProgress] = useState<SetupProgress>();
  const [loadFailure, setLoadFailure] = useState<string>();
  const [recordFailure, setRecordFailure] = useState<string>();

  // Resuming: read what the server knows and start at the first step left to do.
  useEffect(() => {
    if (installing) return;
    let current = true;
    void fetchSetupProgress().then((result) => {
      if (!current) return;
      if (result.ok) {
        setProgress(result.value);
        setStep(firstStepToDo(result.value));
      } else setLoadFailure(result.code);
    });
    return () => {
      current = false;
    };
  }, [installing]);

  const standing = useMemo(() => {
    const result = {} as Record<StepId, Standing>;
    for (const id of STEP_IDS) {
      let value: Standing = "todo";
      const known = serverStep(id);
      if (known && progress) value = standingOf(stepState(progress, known));
      else if (id === "code" || id === "admin")
        value =
          !installing || (id === "code" && step === "admin") ? "done" : "todo";
      if (id === step) value = "current";
      result[id] = value;
    }
    return result;
  }, [progress, step, installing]);

  async function createAccount(): Promise<string | undefined> {
    const outcome = await completeSetup(
      code.trim(),
      draft.username.trim(),
      draft.password,
    );
    if (!outcome.ok) return outcome.code;
    // The session is open: move to the address of the wizard, then let the gate reload.
    navigate("/setup", { replace: true });
    await reload();
    return undefined;
  }

  /** Notes how the person left a step, then shows the next one. */
  async function leave(id: StepId, state: "done" | "skipped") {
    const known = serverStep(id);
    if (known) {
      setRecordFailure(undefined);
      const result = await record(known, state);
      if (!result.ok) {
        setRecordFailure(result.code);
        return;
      }
      setProgress(result.value);
    }
    setStep(STEP_IDS[STEP_IDS.indexOf(id) + 1]);
  }

  const skip = (id: StepId) => (
    <button
      type="button"
      onClick={() => void leave(id, "skipped")}
      className={buttonClass()}
    >
      {t("setup.skip")}
    </button>
  );

  const resuming = !installing && !progress;
  return (
    <div className="flex min-h-dvh flex-col bg-canvas text-ink md:flex-row">
      <WizardSidebar standing={standing} optional={OPTIONAL_IDS} />
      <main className="flex flex-1 flex-col gap-5 px-5 py-8 md:px-12 md:py-12">
        {resuming && !loadFailure && <p role="status">{t("app.loading")}</p>}
        {loadFailure && <ErrorAlert>{error(loadFailure)}</ErrorAlert>}
        {recordFailure && <ErrorAlert>{error(recordFailure)}</ErrorAlert>}
        {step === "code" && installing && (
          <CodeStep
            code={code}
            onCodeChange={setCode}
            onValid={() => setStep("admin")}
          />
        )}
        {step === "admin" && installing && (
          <AdminStep
            draft={draft}
            onChange={setDraft}
            onBack={() => setStep("code")}
            onSubmit={createAccount}
          />
        )}
        {progress && step === "discord" && (
          <DiscordStep
            onDone={() => leave("discord", "done")}
            skip={skip("discord")}
          />
        )}
        {progress && step === "semantic" && (
          <SemanticStep
            onDone={() => leave("semantic", "done")}
            skip={skip("semantic")}
          />
        )}
        {progress && step === "worker" && (
          <WorkerStep
            onDone={() => leave("worker", "done")}
            skip={skip("worker")}
          />
        )}
        {progress && step === "import" && (
          <ImportStep
            onDone={() => leave("import", "done")}
            skip={skip("import")}
          />
        )}
        {progress && step === "summary" && <SummaryStep progress={progress} />}
      </main>
    </div>
  );
}
