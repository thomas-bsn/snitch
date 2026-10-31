import type { MessageKey } from "../i18n/messages";
import type { SetupStep } from "./setupApi";

/** The seven screens of the wizard, in order. */
export const STEP_IDS = [
  "code",
  "admin",
  "location",
  "discord",
  "semantic",
  "worker",
  "import",
  "summary",
] as const;

export type StepId = (typeof STEP_IDS)[number];

export const STEP_LABELS: Record<StepId, MessageKey> = {
  code: "setup.steps.code",
  admin: "setup.steps.admin",
  location: "metadata.title",
  discord: "setup.steps.discord",
  semantic: "setup.steps.semantic",
  worker: "setup.steps.worker",
  import: "setup.steps.import",
  summary: "setup.steps.summary",
};

/** The four optional steps, with the name the server keeps their progress under and where to finish them. */
export const OPTIONAL_STEPS: {
  id: StepId;
  server: SetupStep;
  where: string;
  whereLabel: MessageKey;
}[] = [
  {
    id: "location",
    server: "location",
    where: "/admin/metadata",
    whereLabel: "metadata.title",
  },
  {
    id: "discord",
    server: "discord",
    where: "/admin/users",
    whereLabel: "setup.where.discord",
  },
  {
    id: "semantic",
    server: "semantic_search",
    where: "/admin/semantic",
    whereLabel: "setup.where.semantic",
  },
  {
    id: "worker",
    server: "worker",
    where: "/admin/workers",
    whereLabel: "setup.where.worker",
  },
  {
    id: "import",
    server: "import",
    where: "/import",
    whereLabel: "setup.where.import",
  },
];

export function serverStep(id: StepId): SetupStep | undefined {
  return OPTIONAL_STEPS.find((step) => step.id === id)?.server;
}
