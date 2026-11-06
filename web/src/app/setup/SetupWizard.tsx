import { useEffect, useMemo, useRef, useState } from "react";
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
import { HuggingFaceStep } from "./steps/HuggingFaceStep";
import { ImportStep } from "./steps/ImportStep";
import { SemanticStep } from "./steps/SemanticStep";
import { SummaryStep } from "./steps/SummaryStep";
import { WorkerStep } from "./steps/WorkerStep";
import {
  OPTIONAL_STEPS,
  STEP_IDS,
  STEP_LABELS,
  serverStep,
  type StepId,
} from "./steps";
import { WizardSidebar, standingOf, type Standing } from "./WizardSidebar";
import { MetadataSettings } from "../admin/metadata/MetadataSettings";
import { StepHeader } from "./StepHeader";

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

  const [step, setStep] = useState<StepId>(installing ? "code" : "location");
  const [visited, setVisited] = useState<ReadonlySet<StepId>>(
    () => new Set([installing ? "code" : "location"]),
  );
  const [furthest, setFurthest] = useState(installing ? 0 : 2);
  const main = useRef<HTMLElement>(null);
  function showStep(next: StepId) {
    setVisited((previous) => new Set([...previous, next]));
    setFurthest((previous) => Math.max(previous, STEP_IDS.indexOf(next)));
    setStep(next);
  }
  useEffect(() => {
    main.current?.querySelector<HTMLElement>("[data-active-step] h1")?.focus();
  }, [step]);
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
        showStep(firstStepToDo(result.value));
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
      const preservedState =
        progress && stepState(progress, known) === "done" ? "done" : state;
      const result = await record(known, preservedState);
      if (!result.ok) {
        setRecordFailure(result.code);
        return;
      }
      setProgress(result.value);
    }
    showStep(STEP_IDS[STEP_IDS.indexOf(id) + 1]);
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
      <WizardSidebar
        standing={standing}
        optional={OPTIONAL_IDS}
        available={new Set(STEP_IDS.slice(0, furthest + 1))}
        onSelect={showStep}
      />
      <main
        ref={main}
        className="flex flex-1 flex-col gap-5 px-5 py-8 md:px-12 md:py-12"
      >
        {resuming && !loadFailure && <p role="status">{t("app.loading")}</p>}
        {loadFailure && <ErrorAlert>{error(loadFailure)}</ErrorAlert>}
        {recordFailure && <ErrorAlert>{error(recordFailure)}</ErrorAlert>}
        {step === "code" && installing && (
          <CodeStep
            code={code}
            onCodeChange={setCode}
            onValid={() => showStep("admin")}
          />
        )}
        {step === "admin" && installing && (
          <AdminStep
            draft={draft}
            onChange={setDraft}
            onBack={() => showStep("code")}
            onSubmit={createAccount}
          />
        )}
        {progress && !installing && (step === "code" || step === "admin") && (
          <div data-active-step className="flex max-w-xl flex-col gap-5">
            <StepHeader
              position={STEP_IDS.indexOf(step) + 1}
              total={9}
              title={t(STEP_LABELS[step])}
            />
            <p>{t("setup.completed")}</p>
            <button
              type="button"
              className={buttonClass("primary")}
              onClick={() => showStep(STEP_IDS[STEP_IDS.indexOf(step) + 1])}
            >
              {t("setup.continue")}
            </button>
          </div>
        )}
        {progress && step === "location" && (
          <div data-active-step className="flex max-w-xl flex-col gap-5">
            <StepHeader position={3} total={9} title={t("metadata.title")} />
            <MetadataSettings onSaved={() => void leave("location", "done")} />
            {skip("location")}
          </div>
        )}
        {progress && visited.has("discord") && (
          <div
            hidden={step !== "discord"}
            data-active-step={step === "discord" ? "" : undefined}
          >
            <DiscordStep
              onDone={() => leave("discord", "done")}
              skip={skip("discord")}
            />
          </div>
        )}
        {progress && visited.has("semantic") && (
          <div
            hidden={step !== "semantic"}
            data-active-step={step === "semantic" ? "" : undefined}
          >
            <SemanticStep
              onDone={() => leave("semantic", "done")}
              skip={skip("semantic")}
            />
          </div>
        )}
        {progress && visited.has("worker") && (
          <div
            hidden={step !== "worker"}
            data-active-step={step === "worker" ? "" : undefined}
          >
            {stepState(progress, "worker") === "done" ? (
              <div className="flex max-w-xl flex-col gap-5">
                <StepHeader
                  position={6}
                  total={9}
                  title={t("setup.worker.title")}
                />
                <p>{t("setup.completed")}</p>
                <button
                  type="button"
                  className={buttonClass("primary")}
                  onClick={() => showStep("import")}
                >
                  {t("setup.continue")}
                </button>
              </div>
            ) : (
              <WorkerStep
                onDone={() => leave("worker", "done")}
                skip={skip("worker")}
              />
            )}
          </div>
        )}
        {progress && visited.has("huggingface") && (
          <div
            hidden={step !== "huggingface"}
            data-active-step={step === "huggingface" ? "" : undefined}
          >
            <HuggingFaceStep
              onDone={() => leave("huggingface", "done")}
              skip={skip("huggingface")}
            />
          </div>
        )}
        {progress && step === "import" && (
          <div data-active-step>
            <ImportStep
              onDone={() => leave("import", "done")}
              skip={skip("import")}
            />
          </div>
        )}
        {progress && step === "summary" && (
          <div data-active-step>
            <SummaryStep progress={progress} />
          </div>
        )}
        {progress && step !== "code" && (
          <div>
            <button
              type="button"
              onClick={() => showStep(STEP_IDS[STEP_IDS.indexOf(step) - 1])}
              className={buttonClass()}
            >
              {t("setup.back")}
            </button>
          </div>
        )}
      </main>
    </div>
  );
}
