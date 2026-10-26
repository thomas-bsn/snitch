import { Plus } from "lucide-react";
import { Link, useNavigate, useParams } from "react-router";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { StatusPill } from "../../components/StatusPill";
import { useDocumentTitle } from "../../components/useDocumentTitle";
import { useI18n } from "../../i18n";
import { useFormatDate } from "../format";
import { WorkerCreateForm } from "./WorkerCreateForm";
import { WorkerDetailPage } from "./WorkerDetailPage";
import { fetchWorkers, useLoaded, type Worker } from "./workersApi";
import { workerModelSummary, workerStatus } from "./workerView";

/** The Workers tab of the administration: the list, the creation form and a Worker's page. */
export function WorkersTab() {
  const { id } = useParams();
  if (id === "new") return <CreateWorker />;
  if (id) return <WorkerDetailPage id={id} />;
  return <WorkersList />;
}

function CreateWorker() {
  const { t } = useI18n();
  const navigate = useNavigate();
  useDocumentTitle(t("workers.create.title"));
  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("workers.create.title")}</h2>
      <p className="max-w-xl text-mute">{t("workers.create.lead")}</p>
      <WorkerCreateForm
        onDone={(worker) => navigate(`/admin/workers/${worker.id}`)}
        secondaryAction={
          <Link to="/admin/workers" className={buttonClass("secondary")}>
            {t("workers.create.cancel")}
          </Link>
        }
      />
    </div>
  );
}

function WorkersList() {
  const { t, error } = useI18n();
  const formatDate = useFormatDate();
  const { state } = useLoaded(fetchWorkers);
  useDocumentTitle(t("workers.title"));

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="max-w-xl text-mute">{t("workers.lead")}</p>
        <Link to="/admin/workers/new" className={buttonClass("primary")}>
          <Plus className="size-4" />
          {t("workers.add")}
        </Link>
      </div>
      {state.status === "loading" && (
        <p role="status">{t("workers.loading")}</p>
      )}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {state.status === "ready" && state.value.items.length === 0 && (
        <p className="rounded-lg border border-dashed border-line-strong p-6 text-mute">
          {t("workers.empty")}
        </p>
      )}
      {state.status === "ready" && state.value.items.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-line">
          <table className="w-full min-w-[34rem] border-collapse text-left">
            <thead className="bg-side text-xs text-mute">
              <tr>
                <th scope="col" className="px-3 py-2 font-medium">
                  {t("workers.columns.name")}
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  {t("workers.columns.status")}
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  {t("workers.columns.lastPulse")}
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  {t("workers.columns.models")}
                </th>
              </tr>
            </thead>
            <tbody>
              {state.value.items.map((worker) => (
                <WorkerRow
                  key={worker.id}
                  worker={worker}
                  lastPulse={
                    worker.last_seen_at
                      ? formatDate(worker.last_seen_at)
                      : t("workers.never")
                  }
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function WorkerRow({
  worker,
  lastPulse,
}: {
  worker: Worker;
  lastPulse: string;
}) {
  const { t } = useI18n();
  const { tone, label } = workerStatus(worker);
  const summary = workerModelSummary(worker.models);
  return (
    <tr className="border-t border-line">
      <th scope="row" className="px-3 py-2.5 font-semibold">
        <Link
          to={`/admin/workers/${worker.id}`}
          aria-label={t("workers.open", { name: worker.name })}
          className="inline-flex min-h-11 items-center text-accent-ink underline-offset-2 hover:underline md:min-h-0"
        >
          {worker.name}
        </Link>
      </th>
      <td className="px-3 py-2.5">
        <StatusPill tone={tone}>{t(label)}</StatusPill>
      </td>
      <td className="px-3 py-2.5 text-mute">{lastPulse}</td>
      <td className="px-3 py-2.5 text-mute">
        {summary.length > 0 ? summary.join(" · ") : t("workers.notDeclared")}
      </td>
    </tr>
  );
}
