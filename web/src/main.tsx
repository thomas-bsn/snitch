import React, { useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import createClient from "openapi-fetch";
import type { paths, components } from "./api.gen";
import "./style.css";
import { SpeakerBadge, VoicePanel } from "./voice/VoicePanel";
import {
  ClipPersons,
  PersonFilterSelect,
  ResultSpeaker,
  applyPersonFilter,
} from "./voice/PersonFilter";
import { SegmentText } from "./segments/SegmentText";
import {
  CorrectionsWarningPanel,
  describeCorrections,
  replacedCorrections,
  type CorrectionsToReplace,
} from "./segments/Retranscription";
import { SegmentHistory } from "./segments/SegmentHistory";
import { ObsoleteCorrections } from "./segments/ObsoleteCorrections";
import { SegmentSpeaker } from "./segments/SegmentSpeaker";
import { VoicePipelineCard } from "./voice/VoicePipelineCard";
import { VoiceMatchingCard } from "./voice/VoiceMatchingCard";
import { PersonsCard } from "./voice/PersonsCard";
import { MyVoiceCard } from "./voice/MyVoiceCard";
import { PersonAccountCard } from "./voice/PersonAccountCard";
import { VoiceAutomationCard } from "./voice/VoiceAutomationCard";
import { VoiceGroupsCard } from "./voice/VoiceGroupsCard";
import { VoiceVerificationCard } from "./voice/VoiceVerificationCard";
import { MicrophoneClipCard } from "./voice/MicrophoneClipCard";
import { MicrophoneSourceCard } from "./voice/MicrophoneSourceCard";
import { isAdministrator, isSuperAdministrator, roleLabel } from "./roles";

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
type JobEvent = components["schemas"]["JobEvent"];
type JobControls = components["schemas"]["JobControls"];
type Worker = components["schemas"]["Worker"];

type WorkerBundle = components["schemas"]["WorkerBundle"];
type WorkerModel = components["schemas"]["WorkerModel"];

function downloadTextFile(name: string, text: string) {
  const url = URL.createObjectURL(new Blob([text], { type: "text/plain" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
}

function describeWorkerModel(model?: WorkerModel): string {
  if (!model) return "non déclaré";
  return [model.provider, model.model, model.version, model.endpoint]
    .filter(Boolean)
    .join(" · ");
}

function describeWorkerModels(worker: Worker): string {
  const models = worker.models;
  if (
    !models ||
    (!models.transcription && !models.diarization && !models.fingerprint)
  )
    return "Modèles : non déclaré";
  return [
    `Transcription : ${describeWorkerModel(models.transcription)}`,
    `Diarisation : ${describeWorkerModel(models.diarization)}`,
    `Empreinte vocale : ${describeWorkerModel(models.fingerprint)}`,
  ].join(", ");
}
type VocabularyTerm = components["schemas"]["VocabularyTerm"];
type TranscriptionPipeline = components["schemas"]["TranscriptionPipeline"];
type SearchResult = {
  clip_id: number;
  clip_name: string;
  visibility: string;
  recorded_at: string;
  status: string;
  sources: string[];
  match_type: string;
  context: string;
  matched_text: string;
  timestamp: number;
  corrected?: boolean;
  person?: { id: number; display_name: string } | null;
  decision_level?: components["schemas"]["DecisionLevel"];
};
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
  job_not_dead: "Seules les tâches mortes peuvent être relancées.",
  job_not_cancellable: "Cette tâche ne peut plus être annulée.",
};
function checked<T>(result: { data?: T; error?: { error: string } }): T {
  if (result.error)
    throw new Error(
      messages[result.error.error] ?? "La demande a échoué. Réessayez.",
    );
  if (!result.data) throw new Error("Réponse du serveur absente.");
  return result.data;
}
function formatTimecode(seconds: number): string {
  const whole = Math.floor(Math.max(0, seconds));
  const hours = Math.floor(whole / 3600);
  const minutes = Math.floor((whole % 3600) / 60);
  const remainder = whole % 60;
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`
    : `${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`;
}
function checkedNoContent(result: { error?: { error: string } }): void {
  if (result.error)
    throw new Error(
      messages[result.error.error] ?? "La demande a échoué. Réessayez.",
    );
}
function highlighted(context: string, match: string): React.ReactNode {
  const start = context
    .toLocaleLowerCase("fr")
    .indexOf(match.toLocaleLowerCase("fr"));
  if (start < 0 || !match) return context;
  return (
    <>
      {context.slice(0, start)}
      <mark>{context.slice(start, start + match.length)}</mark>
      {context.slice(start + match.length)}
    </>
  );
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
  const [clipDetails, setClipDetails] = useState<Record<number, Clip>>({});
  const [searchQuery, setSearchQuery] = useState("");
  const [searchMode, setSearchMode] = useState("full_text");
  const [searchPage, setSearchPage] = useState(1);
  const [searchTotal, setSearchTotal] = useState(0);
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searchDegraded, setSearchDegraded] = useState(false);
  const [searchFilters, setSearchFilters] = useState({
    from: "",
    to: "",
    source_id: "",
    visibility: "",
    status: "",
  });
  const [searchPerson, setSearchPerson] = useState("");
  const [clipPerson, setClipPerson] = useState("");
  const [searchPlayback, setSearchPlayback] = useState<SearchResult>();
  const [selectedAudioTracks, setSelectedAudioTracks] = useState<
    Record<number, number>
  >({});
  const [jobs, setJobs] = useState<Job[]>([]);
  const [jobHistory, setJobHistory] = useState<Record<number, JobEvent[]>>({});
  const [jobControls, setJobControls] = useState<JobControls>({
    paused: false,
    max_concurrency: 1,
    updated_at: "",
  });
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [vocabulary, setVocabulary] = useState<VocabularyTerm[]>([]);
  const [vocabularyVersion, setVocabularyVersion] = useState("");
  const [pipeline, setPipeline] = useState<TranscriptionPipeline>();
  const [pipelineVersion, setPipelineVersion] = useState("");
  const [embeddingState, setEmbeddingState] = useState<{
    configured: boolean;
    provider: string;
    model: string;
    dimensions: number;
    state: string;
    error?: string;
    pending: number;
    done: number;
    failed: number;
    candidate_model?: string;
    candidate_state?: string;
    candidate_error?: string;
    candidate_total?: number;
    candidate_done?: number;
    candidate_pending?: number;
    candidate_failed?: number;
    reindex_pending?: number;
    reindex_failed?: number;
  }>();
  const [embeddingModel, setEmbeddingModel] = useState("");
  const [editingTerm, setEditingTerm] = useState<VocabularyTerm>();
  const [newWorkerToken, setNewWorkerToken] = useState("");
  const [newWorkerBundle, setNewWorkerBundle] = useState<WorkerBundle | null>(
    null,
  );
  const [jobPage, setJobPage] = useState(1);
  const [jobTotal, setJobTotal] = useState(0);
  const [jobFilters, setJobFilters] = useState({
    status: "" as
      | ""
      | "pending"
      | "leased"
      | "done"
      | "failed"
      | "retry_wait"
      | "cancelled"
      | "dead",
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
  const [batchPreview, setBatchPreview] = useState<{
    eligible: number;
    already_processed: number;
    already_queued: number;
    unavailable: number;
    selection_required: number;
    corrected: number;
    corrected_segments: number;
  }>();
  const [includeCorrected, setIncludeCorrected] = useState(false);
  const [retranscriptionWarning, setRetranscriptionWarning] = useState<{
    clipId: number;
    index: number;
    corrections: CorrectionsToReplace;
  }>();
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
  const [theme, setTheme] = useState<"dark" | "light">(() => {
    const saved =
      localStorage.getItem("snitch-theme") ??
      localStorage.getItem("receipts-theme");
    return saved === "light" ? "light" : "dark";
  });
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("snitch-theme", theme);
  }, [theme]);
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
    if (me.data && isAdministrator(me.data.user.role)) {
      setSettings(checked(await api.GET("/api/admin/settings")));
      await refreshVocabulary();
      await refreshPipeline();
      await refreshEmbeddings();
      setPending(checked(await api.GET("/api/admin/users")));
      await refreshJobs(1);
      await refreshJobControls();
      setWorkers(checked(await api.GET("/api/admin/workers")).items);
      await refreshSources();
    } else {
      setSources([]);
      setAllowedRoots([]);
      setJobs([]);
      setJobControls({ paused: false, max_concurrency: 1, updated_at: "" });
      setWorkers([]);
      setVocabulary([]);
      setNewWorkerToken("");
      setNewWorkerBundle(null);
    }
  }
  async function refreshClips(page: number) {
    const query = new URLSearchParams({ page: String(page), page_size: "12" });
    for (const [key, value] of Object.entries(clipFilters))
      if (value) query.set(key, value);
    applyPersonFilter(query, clipPerson);
    const response = await fetch(`/api/clips?${query}`, {
      credentials: "same-origin",
    });
    if (!response.ok) throw new Error("Impossible de charger la bibliothèque.");
    const result = await response.json();
    setClips(result.items);
    setClipPage(result.page);
    setClipTotal(result.total);
  }
  async function previewBatch() {
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(clipFilters))
      if (value) query.set(key, value);
    if (includeCorrected) query.set("include_corrected", "true");
    const response = await fetch(
      `/api/admin/transcriptions/batch-preview?${query}`,
      { credentials: "same-origin" },
    );
    if (!response.ok) throw new Error("Impossible de prévisualiser ce lot.");
    setBatchPreview(await response.json());
  }
  // Retranscribes a clip. A clip that carries corrections is refused by the server until
  // the administrator confirms what is replaced (warning panel, then confirm=true).
  async function retranscribeClip(
    clipId: number,
    index: number,
    confirm: boolean,
  ) {
    const query = { confirm_replace_corrections: confirm || undefined };
    const track = await api.PUT("/api/admin/clips/{id}/audio-track", {
      params: { path: { id: clipId }, header: csrf, query },
      body: { audio_stream_index: index },
    });
    const trackWarning = replacedCorrections(track.error);
    if (trackWarning) {
      setRetranscriptionWarning({ clipId, index, corrections: trackWarning });
      return;
    }
    checked(track);
    const result = await api.POST("/api/admin/clips/{id}/transcription", {
      params: { path: { id: clipId }, header: csrf, query },
      body: {},
    });
    const warning = replacedCorrections(result.error);
    if (warning) {
      setRetranscriptionWarning({ clipId, index, corrections: warning });
      return;
    }
    const response = checked(result);
    setRetranscriptionWarning(undefined);
    setNotice(`Transcription planifiée (tâche ${response.job.id}).`);
    await refreshJobs(1);
  }
  async function scheduleBatch() {
    if (
      !batchPreview?.eligible ||
      !window.confirm(
        `Planifier ${batchPreview.eligible} transcription(s) ?` +
          (includeCorrected && batchPreview.corrected > 0
            ? ` Les corrections de ${batchPreview.corrected} clip(s) corrigé(s) seront remplacées (conservées dans l’historique, obsolètes).`
            : ""),
      )
    )
      return;
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(clipFilters))
      if (value) query.set(key, value);
    const response = await fetch(
      `/api/admin/transcriptions/batch-schedule?${query}`,
      {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", ...csrf },
        body: JSON.stringify({
          confirm: true,
          include_corrected: includeCorrected,
        }),
      },
    );
    if (!response.ok) throw new Error("Impossible de planifier ce lot.");
    const result = await response.json();
    setNotice(`${result.scheduled} tâche(s) ajoutée(s) à la file.`);
    setBatchPreview(result.preview);
    await refreshJobs(1);
  }
  async function runSearch(page = 1) {
    const query = new URLSearchParams({
      q: searchQuery,
      mode: searchMode,
      page: String(page),
      page_size: "10",
    });
    for (const [key, value] of Object.entries(searchFilters))
      if (value) query.set(key, value);
    applyPersonFilter(query, searchPerson);
    const response = await fetch(`/api/search?${query}`, {
      credentials: "same-origin",
    });
    if (!response.ok)
      throw new Error("Impossible de rechercher dans les transcriptions.");
    const result = await response.json();
    setSearchResults(result.items);
    setSearchTotal(result.total);
    setSearchPage(result.page);
    setSearchDegraded(Boolean(result.degraded));
  }
  async function refreshVocabulary() {
    const result = checked(await api.GET("/api/admin/vocabulary"));
    setVocabulary(result.items);
    setVocabularyVersion(result.version);
  }
  async function refreshPipeline() {
    const result = checked(await api.GET("/api/admin/transcription-pipeline"));
    setPipeline(result.config);
    setPipelineVersion(result.version);
  }
  async function refreshEmbeddings() {
    const response = await fetch("/api/admin/embeddings", {
      credentials: "same-origin",
    });
    if (response.ok) setEmbeddingState(await response.json());
  }
  useEffect(() => {
    if (!session || !isAdministrator(session.user.role)) return;
    const timer = window.setInterval(() => {
      void refreshEmbeddings();
    }, 5000);
    return () => window.clearInterval(timer);
  }, [session?.user.role]);
  async function backfillEmbeddings() {
    const response = await fetch("/api/admin/embeddings/backfill", {
      method: "POST",
      credentials: "same-origin",
      headers: csrf,
    });
    if (!response.ok)
      throw new Error(
        "Impossible de lancer l’indexation des transcriptions existantes.",
      );
    const result = await response.json();
    setNotice(`${result.queued} transcription(s) ajoutée(s) à l’indexation.`);
    await refreshEmbeddings();
  }
  async function rebuildEmbeddings() {
    const response = await fetch("/api/admin/embeddings/rebuild", {
      method: "POST",
      credentials: "same-origin",
      headers: { ...csrf, "Content-Type": "application/json" },
      body: JSON.stringify({ model: embeddingModel }),
    });
    if (!response.ok)
      throw new Error("Impossible de lancer la reconstruction de l’index.");
    setNotice(
      "Nouvel index en préparation. L’index actif reste utilisé pendant le traitement.",
    );
    await refreshEmbeddings();
  }
  async function savePipeline(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const mode = String(form.get("vad_mode") || "off");
    const body: TranscriptionPipeline = {
      provider: "openai-compatible",
      model: String(form.get("model") || "default").trim(),
      language: String(form.get("language") || "fr").trim(),
      preparation: "wav-16khz-mono-v1",
      vad:
        mode === "energy"
          ? {
              mode: "energy",
              version: "energy-v1",
              threshold: Number(form.get("threshold") || 0.02),
              min_speech_ms: Number(form.get("min_speech_ms") || 100),
              min_silence_ms: Number(form.get("min_silence_ms") || 300),
              padding_ms: Number(form.get("padding_ms") || 100),
            }
          : { mode: "off", version: "vad-off-v1" },
    };
    await run(async () => {
      const saved = checked(
        await api.PUT("/api/admin/transcription-pipeline", {
          params: { header: csrf },
          body,
        }),
      );
      setPipeline(saved.config);
      setPipelineVersion(saved.version);
      setNotice(
        "Configuration enregistrée. Les prochaines transcriptions utiliseront cette version.",
      );
    });
  }
  async function saveVocabularyTerm(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const body = {
      term: String(form.get("term") ?? ""),
      variants: String(form.get("variants") ?? "")
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean),
      preferred_spelling: String(form.get("preferred_spelling") ?? ""),
      active: form.get("active") === "on",
    };
    setBusy(true);
    try {
      const result = editingTerm
        ? await api.PUT("/api/admin/vocabulary/{id}", {
            params: { path: { id: editingTerm.id }, header: csrf },
            body,
          })
        : await api.POST("/api/admin/vocabulary", {
            params: { header: csrf },
            body,
          });
      checked(result);
      setEditingTerm(undefined);
      formElement.reset();
      await refreshVocabulary();
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Impossible d’enregistrer le terme.",
      );
    } finally {
      setBusy(false);
    }
  }
  async function removeVocabularyTerm(id: number) {
    setBusy(true);
    try {
      checkedNoContent(
        await api.DELETE("/api/admin/vocabulary/{id}", {
          params: { path: { id }, header: csrf },
        }),
      );
      if (editingTerm?.id === id) setEditingTerm(undefined);
      await refreshVocabulary();
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Impossible de supprimer le terme.",
      );
    } finally {
      setBusy(false);
    }
  }
  async function loadClipDetail(id: number) {
    try {
      const detail = checked(
        await api.GET("/api/clips/{id}", { params: { path: { id } } }),
      );
      setClipDetails((current) => ({ ...current, [id]: detail }));
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Impossible de charger le clip.",
      );
    }
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
  async function refreshJobControls() {
    setJobControls(checked(await api.GET("/api/admin/job-controls")));
  }
  async function saveJobControls(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await run(async () => {
      setJobControls(
        checked(
          await api.PUT("/api/admin/job-controls", {
            params: { header: csrf },
            body: {
              paused: jobControls.paused,
              max_concurrency: jobControls.max_concurrency,
            },
          }),
        ),
      );
      setNotice("Contrôles de file enregistrés.");
    });
  }
  async function scheduleJob(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    await run(async () => {
      const request = (confirm: boolean) =>
        api.POST("/api/admin/jobs", {
          params: {
            header: csrf,
            query: { confirm_replace_corrections: confirm || undefined },
          },
          body: {
            clip_id: Number(data.get("clip_id")),
            step: String(data.get("step")),
            input_version: String(data.get("input_version")),
            priority: Number(data.get("priority")),
            max_attempts: Number(data.get("max_attempts")),
          },
        });
      let result = await request(false);
      const warning = replacedCorrections(result.error);
      if (warning) {
        if (
          !window.confirm(
            `Cette retranscription remplacerait des corrections humaines : ${describeCorrections(warning)}. Elles resteront dans l’historique, obsolètes. Continuer ?`,
          )
        )
          return;
        result = await request(true);
      }
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
  async function loadJobHistory(job: Job) {
    await run(async () => {
      const result = checked(
        await api.GET("/api/admin/jobs/{id}/history", {
          params: { path: { id: job.id } },
        }),
      );
      setJobHistory({ ...jobHistory, [job.id]: result.items });
    });
  }
  async function relaunchJob(job: Job) {
    await run(async () => {
      const request = (confirm: boolean) =>
        api.POST("/api/admin/jobs/{id}/retry", {
          params: {
            path: { id: job.id },
            header: csrf,
            query: { confirm_replace_corrections: confirm || undefined },
          },
        });
      let result = await request(false);
      const warning = replacedCorrections(result.error);
      if (warning) {
        if (
          !window.confirm(
            `Cette retranscription remplacerait des corrections humaines : ${describeCorrections(warning)}. Elles resteront dans l’historique, obsolètes. Continuer ?`,
          )
        )
          return;
        result = await request(true);
      }
      checked(result);
      await refreshJobs(jobPage);
      setNotice(`Tâche ${job.id} relancée.`);
    });
  }
  async function cancelJob(job: Job) {
    await run(async () => {
      checkedNoContent(
        await api.DELETE("/api/admin/jobs/{id}", {
          params: { path: { id: job.id }, header: csrf },
        }),
      );
      await refreshJobs(jobPage);
      setNotice(`Tâche ${job.id} annulée.`);
    });
  }
  async function registerWorker(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    await run(async () => {
      const result = checked(
        await api.POST("/api/admin/workers", {
          params: { header: csrf },
          body: {
            name: String(data.get("worker_name")),
            max_concurrency: Number(data.get("worker_capacity")),
          },
        }),
      );
      setNewWorkerToken(result.token);
      setNewWorkerBundle(result.bundle);
      form.reset();
      setWorkers(checked(await api.GET("/api/admin/workers")).items);
      setNotice(
        "Worker créé. Copiez son jeton maintenant : il ne sera plus affiché.",
      );
    });
  }
  async function revokeWorker(worker: Worker) {
    await run(async () => {
      checkedNoContent(
        await api.DELETE("/api/admin/workers/{id}", {
          params: { path: { id: worker.id }, header: csrf },
        }),
      );
      setNewWorkerToken("");
      setNewWorkerBundle(null);
      setWorkers(checked(await api.GET("/api/admin/workers")).items);
      setNotice(`Jeton du worker ${worker.name} révoqué.`);
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
    <main className={session ? "authenticated" : ""}>
      <header>
        <span className="brand-mark">R</span>
        <div>
          <p className="eyebrow">VOTRE MÉMOIRE DE JEU</p>
          <h1>{setup?.display_name ?? "Snitch"}</h1>
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
          <div className="app-shell">
            <aside className="sidebar" aria-label="Navigation principale">
              <a
                className="sidebar-brand"
                href="#library"
                aria-label="Snitch, bibliothèque"
              >
                <span className="brand-mark">R</span>
                <span>{setup.display_name}</span>
              </a>
              <p className="nav-label">Espace</p>
              <a className="nav-link active" href="#library">
                <span aria-hidden="true">▦</span> Bibliothèque
              </a>
              <a className="nav-link" href="#voice-verification">
                <span aria-hidden="true">✓</span> Vérifier les voix
              </a>
              <a className="nav-link" href="#voice-groups">
                <span aria-hidden="true">◍</span> Voix inconnues
              </a>
              <a className="nav-link" href="#uploads">
                <span aria-hidden="true">⇧</span> Imports
                {uploads.length > 0 && (
                  <span className="nav-count">{uploads.length}</span>
                )}
              </a>
              {isAdministrator(session.user.role) && (
                <>
                  <p className="nav-label">Administration</p>
                  <a className="nav-link" href="#job-queue">
                    <span aria-hidden="true">◷</span> File de tâches
                  </a>
                  <a className="nav-link" href="#workers">
                    <span aria-hidden="true">⌁</span> Workers
                  </a>
                  <a className="nav-link" href="#sources">
                    <span aria-hidden="true">▤</span> Sources
                  </a>
                  <a className="nav-link" href="#settings">
                    <span aria-hidden="true">⚙</span> Réglages
                  </a>
                </>
              )}
              <div className="sidebar-user">
                <span className="user-avatar">
                  {session.user.username.slice(0, 1).toUpperCase()}
                </span>
                <span className="user-copy">
                  <strong>{session.user.username}</strong>
                  <small>{roleLabel(session.user.role)}</small>
                </span>
              </div>
            </aside>
            <div className="workspace">
              <div className="toolbar">
                <p>
                  <span className="breadcrumb">ESPACE / MÉDIATHÈQUE</span>
                </p>
                <div className="toolbar-actions">
                  <div className="theme-toggle" role="group" aria-label="Thème">
                    <button
                      className={theme === "dark" ? "selected" : ""}
                      aria-pressed={theme === "dark"}
                      onClick={() => setTheme("dark")}
                    >
                      Sombre
                    </button>
                    <button
                      className={theme === "light" ? "selected" : ""}
                      aria-pressed={theme === "light"}
                      onClick={() => setTheme("light")}
                    >
                      Clair
                    </button>
                  </div>
                  <button
                    className="secondary logout-button"
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
              </div>
              <section className="card" id="uploads">
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
              <section
                className="card library-card"
                aria-label="Bibliothèque"
                id="library"
              >
                <div className="library-heading">
                  <div>
                    <p className="eyebrow">VOTRE COLLECTION</p>
                    <h2>Bibliothèque</h2>
                    <p>{clipTotal} clips · derniers ajouts en premier</p>
                  </div>
                  <a className="primary-link" href="#uploads">
                    ＋ Importer des clips
                  </a>
                </div>
                <section
                  className="transcript-search"
                  aria-label="Recherche dans les paroles"
                >
                  <h3>Rechercher dans les paroles</h3>
                  <form
                    onSubmit={(event) => {
                      event.preventDefault();
                      run(() => runSearch(1));
                    }}
                  >
                    <label>
                      Paroles
                      <input
                        aria-label="Rechercher dans les paroles"
                        value={searchQuery}
                        onChange={(event) => setSearchQuery(event.target.value)}
                        placeholder="Ex. bonjour tout le monde"
                        required
                      />
                    </label>
                    <label>
                      Type de recherche
                      <select
                        aria-label="Type de recherche"
                        value={searchMode}
                        onChange={(event) => setSearchMode(event.target.value)}
                      >
                        <option value="full_text">Plein texte</option>
                        <option value="exact">Mot exact</option>
                        <option value="hybrid">Sémantique + plein texte</option>
                      </select>
                    </label>
                    <label>
                      Du
                      <input
                        aria-label="Date de début de recherche"
                        type="date"
                        value={searchFilters.from}
                        onChange={(e) =>
                          setSearchFilters({
                            ...searchFilters,
                            from: e.target.value,
                          })
                        }
                      />
                    </label>
                    <label>
                      Au
                      <input
                        aria-label="Date de fin de recherche"
                        type="date"
                        value={searchFilters.to}
                        onChange={(e) =>
                          setSearchFilters({
                            ...searchFilters,
                            to: e.target.value,
                          })
                        }
                      />
                    </label>
                    <label>
                      Source (ID)
                      <input
                        aria-label="Source de recherche"
                        type="number"
                        min="1"
                        value={searchFilters.source_id}
                        onChange={(e) =>
                          setSearchFilters({
                            ...searchFilters,
                            source_id: e.target.value,
                          })
                        }
                      />
                    </label>
                    <label>
                      Visibilité
                      <select
                        aria-label="Visibilité de recherche"
                        value={searchFilters.visibility}
                        onChange={(e) =>
                          setSearchFilters({
                            ...searchFilters,
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
                      État du clip
                      <select
                        aria-label="État du clip recherché"
                        value={searchFilters.status}
                        onChange={(e) =>
                          setSearchFilters({
                            ...searchFilters,
                            status: e.target.value,
                          })
                        }
                      >
                        <option value="">Tous</option>
                        <option value="ready">Prêtes</option>
                        <option value="waiting_stability">En attente</option>
                        <option value="unreadable">Illisibles</option>
                        <option value="missing">Original introuvable</option>
                      </select>
                    </label>
                    <PersonFilterSelect
                      label="Personne de la recherche"
                      value={searchPerson}
                      onChange={setSearchPerson}
                    />
                    <button disabled={busy}>Rechercher</button>
                  </form>
                  {searchDegraded && (
                    <p role="status">
                      Recherche sémantique indisponible : résultats en plein
                      texte.
                    </p>
                  )}
                  {searchResults.length > 0 && (
                    <>
                      <p role="status">
                        {searchTotal} résultat{searchTotal > 1 ? "s" : ""} ·
                        page {searchPage}
                      </p>
                      <ol className="search-results">
                        {searchResults.map((item, index) => (
                          <li
                            key={`${item.clip_id}-${item.timestamp}-${index}`}
                          >
                            <strong>{item.clip_name}</strong> ·{" "}
                            {new Date(item.recorded_at).toLocaleDateString(
                              "fr-FR",
                            )}{" "}
                            ·{" "}
                            {item.visibility === "shared" ? "Partagé" : "Privé"}{" "}
                            ·{" "}
                            {item.match_type === "exact"
                              ? "Correspondance exacte"
                              : item.match_type === "approximate"
                                ? "Correspondance approximative"
                                : item.match_type === "semantic"
                                  ? "Correspondance sémantique"
                                  : item.match_type === "hybrid"
                                    ? "Sémantique + plein texte"
                                    : "Plein texte"}
                            <p>
                              {highlighted(item.context, item.matched_text)}
                              {item.corrected && (
                                <>
                                  {" "}
                                  <span className="segment-correction-badge">
                                    Corrigé
                                  </span>
                                </>
                              )}
                            </p>
                            <ResultSpeaker
                              person={item.person}
                              level={item.decision_level}
                            />
                            <button
                              className="secondary"
                              disabled={item.status !== "ready"}
                              onClick={() => setSearchPlayback(item)}
                            >
                              Ouvrir à {formatTimecode(item.timestamp)}
                            </button>
                          </li>
                        ))}
                      </ol>
                      <div className="toolbar">
                        <button
                          className="secondary"
                          disabled={busy || searchPage <= 1}
                          onClick={() => run(() => runSearch(searchPage - 1))}
                        >
                          Précédente
                        </button>
                        <span>
                          Page {searchPage} · {searchTotal} résultats
                        </span>
                        <button
                          className="secondary"
                          disabled={busy || searchPage * 10 >= searchTotal}
                          onClick={() => run(() => runSearch(searchPage + 1))}
                        >
                          Suivante
                        </button>
                      </div>
                    </>
                  )}
                  {searchQuery && searchResults.length === 0 && (
                    <p role="status">
                      Aucun résultat · {searchTotal} résultat trouvé
                    </p>
                  )}
                  {searchPlayback && (
                    <section
                      aria-label="Lecture du résultat"
                      className="search-playback"
                    >
                      <h4>
                        {searchPlayback.clip_name} ·{" "}
                        {formatTimecode(searchPlayback.timestamp)}
                      </h4>
                      <video
                        controls
                        autoPlay
                        preload="metadata"
                        src={`/api/clips/${searchPlayback.clip_id}/media`}
                        aria-label={`Lecture ${searchPlayback.clip_name}`}
                        onLoadedMetadata={(event) => {
                          event.currentTarget.currentTime =
                            searchPlayback.timestamp;
                          void event.currentTarget
                            .play()
                            .catch(() => undefined);
                        }}
                      />
                      <button
                        className="secondary"
                        onClick={() => setSearchPlayback(undefined)}
                      >
                        Fermer la lecture
                      </button>
                    </section>
                  )}
                </section>
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
                        setClipFilters({
                          ...clipFilters,
                          status: e.target.value,
                        })
                      }
                    >
                      <option value="">Tous</option>
                      <option value="ready">Prêtes</option>
                      <option value="waiting_stability">En attente</option>
                      <option value="unreadable">Illisibles</option>
                      <option value="missing">Original introuvable</option>
                    </select>
                  </label>
                  <label className="advanced-filter">
                    Propriétaire (ID)
                    <input
                      type="number"
                      min="1"
                      value={clipFilters.owner_id}
                      onChange={(e) =>
                        setClipFilters({
                          ...clipFilters,
                          owner_id: e.target.value,
                        })
                      }
                    />
                  </label>
                  <label className="advanced-filter">
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
                  <label className="advanced-filter">
                    Du
                    <input
                      type="date"
                      value={clipFilters.from}
                      onChange={(e) =>
                        setClipFilters({ ...clipFilters, from: e.target.value })
                      }
                    />
                  </label>
                  <label className="advanced-filter">
                    Au
                    <input
                      type="date"
                      value={clipFilters.to}
                      onChange={(e) =>
                        setClipFilters({ ...clipFilters, to: e.target.value })
                      }
                    />
                  </label>
                  <PersonFilterSelect
                    label="Personne de la bibliothèque"
                    value={clipPerson}
                    onChange={setClipPerson}
                  />
                  <button className="filter-submit" disabled={busy}>
                    Appliquer les filtres
                  </button>
                </form>
                {isAdministrator(session.user.role) && (
                  <section className="card" aria-label="Planification par lot">
                    <h3>Planifier les transcriptions visibles</h3>
                    <p className="hint">
                      Prévisualisez les clips selon les filtres ci-dessus. Rien
                      n’est ajouté à la file avant votre confirmation.
                    </p>
                    <button
                      className="secondary"
                      disabled={busy}
                      onClick={() => void run(previewBatch)}
                    >
                      Prévisualiser le lot
                    </button>
                    <label className="inline-check">
                      <input
                        type="checkbox"
                        checked={includeCorrected}
                        onChange={(event) => {
                          setIncludeCorrected(event.target.checked);
                          setBatchPreview(undefined);
                        }}
                      />{" "}
                      Inclure les clips corrigés (leurs corrections seront
                      remplacées)
                    </label>
                    {batchPreview && (
                      <div role="status" className="batch-counts">
                        <p>
                          {batchPreview.eligible} éligible(s) ·{" "}
                          {batchPreview.already_processed} déjà traité(s) ·{" "}
                          {batchPreview.already_queued} déjà en file
                        </p>
                        <p>
                          {batchPreview.unavailable} original(aux)
                          indisponible(s) · {batchPreview.selection_required}{" "}
                          piste(s) à sélectionner
                        </p>
                        {batchPreview.corrected > 0 && (
                          <p>
                            {batchPreview.corrected} clip(s) corrigé(s) (
                            {batchPreview.corrected_segments} correction(s)){" "}
                            {includeCorrected
                              ? "inclus à votre demande : leurs corrections seront remplacées"
                              : "exclu(s) du lot pour protéger les corrections"}
                          </p>
                        )}
                        <button
                          disabled={busy || batchPreview.eligible === 0}
                          onClick={() => void run(scheduleBatch)}
                        >
                          Confirmer et planifier {batchPreview.eligible}
                        </button>
                      </div>
                    )}
                  </section>
                )}
                {clips.length === 0 ? (
                  <p>Aucun clip visible pour ces filtres.</p>
                ) : (
                  <ul className="clip-grid">
                    {clips.map((clip) => (
                      <li className="clip-card" key={clip.id}>
                        <div
                          className={`clip-art clip-art-${clip.id % 5}`}
                          aria-hidden="true"
                        >
                          <span className="clip-art-label">
                            {clip.status === "ready"
                              ? "APERÇU VIDÉO"
                              : clip.status === "missing"
                                ? "ORIGINAL ABSENT"
                                : clip.status === "unreadable"
                                  ? "ILLISIBLE"
                                  : "EN ATTENTE"}
                          </span>
                          {clip.status === "ready" && (
                            <span className="clip-play">▶</span>
                          )}
                          <span className="clip-duration">
                            {clip.duration_seconds
                              ? `${Math.floor(clip.duration_seconds / 60)}:${String(Math.floor(clip.duration_seconds % 60)).padStart(2, "0")}`
                              : "—:—"}
                          </span>
                        </div>
                        <div className="clip-card-body">
                          <strong
                            className="clip-name"
                            title={clip.original_name}
                          >
                            {clip.original_name}
                          </strong>
                          <p className="clip-meta">
                            {new Date(clip.recorded_at).toLocaleDateString(
                              "fr-FR",
                            )}{" "}
                            · {clip.owner}
                          </p>
                          <ClipPersons persons={clip.persons} />
                          <div className="clip-card-footer">
                            <span className={`clip-state state-${clip.status}`}>
                              {clip.status === "ready"
                                ? "Disponible"
                                : clip.status === "missing"
                                  ? "Introuvable"
                                  : clip.status === "unreadable"
                                    ? "Illisible"
                                    : "En attente"}
                            </span>
                            <span className="visibility-chip">
                              {clip.visibility === "shared"
                                ? "Partagé"
                                : "Privé"}
                            </span>
                          </div>
                          <details
                            onToggle={(event) => {
                              if (
                                event.currentTarget.open &&
                                !clipDetails[clip.id]
                              )
                                void loadClipDetail(clip.id);
                            }}
                          >
                            <summary>Détails et lecture</summary>
                            <p>
                              {clip.format_name} · audio{" "}
                              {clip.audio_codec || "absent"} · {clip.size_bytes}{" "}
                              octets · sources{" "}
                              {clip.sources?.join(", ") || "upload"}
                            </p>
                            {clip.uploaders.length > 0 && (
                              <p>Importé par {clip.uploaders.join(", ")}</p>
                            )}
                            {clip.status === "ready" && (
                              <video
                                id={`clip-player-${clip.id}`}
                                controls
                                preload="metadata"
                                src={`/api/clips/${clip.id}/media`}
                                aria-label={`Lecture ${clip.original_name}`}
                              />
                            )}
                            {clipDetails[clip.id]?.audio_track_selection
                              ?.status === "selection_required" &&
                              selectedAudioTracks[clip.id] === undefined && (
                                <p role="status" className="hint">
                                  Plusieurs pistes audio sont possibles.
                                  Sélectionnez celle à transcrire.
                                </p>
                              )}
                            {clip.status === "ready" &&
                              isAdministrator(session.user.role) &&
                              clip.tracks.some(
                                (track) => track.type === "audio",
                              ) && (
                                <div className="transcription-controls">
                                  <label>
                                    Piste audio
                                    <select
                                      value={
                                        selectedAudioTracks[clip.id] ??
                                        clipDetails[clip.id]
                                          ?.audio_track_selection
                                          ?.audio_stream_index ??
                                        ""
                                      }
                                      onChange={(event) =>
                                        setSelectedAudioTracks({
                                          ...selectedAudioTracks,
                                          [clip.id]: Number(event.target.value),
                                        })
                                      }
                                    >
                                      <option value="" disabled>
                                        Sélectionner une piste
                                      </option>
                                      {clip.tracks
                                        .filter(
                                          (track) => track.type === "audio",
                                        )
                                        .map((track) => (
                                          <option
                                            key={track.index}
                                            value={track.index}
                                          >
                                            Piste {track.index} ·{" "}
                                            {track.title || "sans titre"} ·{" "}
                                            {track.language ||
                                              "langue inconnue"}{" "}
                                            ·{" "}
                                            {track.channel_layout ||
                                              `${track.channels ?? "?"} canaux`}
                                          </option>
                                        ))}
                                    </select>
                                  </label>
                                  <button
                                    className="secondary"
                                    onClick={() =>
                                      run(async () => {
                                        const index =
                                          selectedAudioTracks[clip.id] ??
                                          clipDetails[clip.id]
                                            ?.audio_track_selection
                                            ?.audio_stream_index;
                                        if (index === undefined)
                                          throw new Error(
                                            "Sélectionnez explicitement une piste audio.",
                                          );
                                        await retranscribeClip(
                                          clip.id,
                                          index,
                                          false,
                                        );
                                      })
                                    }
                                  >
                                    Transcrire
                                  </button>
                                  {retranscriptionWarning?.clipId ===
                                    clip.id && (
                                    <CorrectionsWarningPanel
                                      corrections={
                                        retranscriptionWarning.corrections
                                      }
                                      busy={busy}
                                      onCancel={() =>
                                        setRetranscriptionWarning(undefined)
                                      }
                                      onConfirm={() =>
                                        void run(() =>
                                          retranscribeClip(
                                            clip.id,
                                            retranscriptionWarning.index,
                                            true,
                                          ),
                                        )
                                      }
                                    />
                                  )}
                                </div>
                              )}
                            {clipDetails[clip.id]?.transcription &&
                              (() => {
                                const transcript =
                                  clipDetails[clip.id]!.transcription!;
                                return (
                                  <section
                                    className="clip-transcript"
                                    aria-label="Transcription horodatée"
                                  >
                                    <h3>
                                      Transcription · {transcript.provider} /{" "}
                                      {transcript.model}
                                    </h3>
                                    {transcript.segments.length === 0 ? (
                                      <p>Aucune parole détectée.</p>
                                    ) : (
                                      transcript.segments.map(
                                        (segment, segmentIndex) => (
                                          <p
                                            key={`${segment.start}-${segmentIndex}`}
                                          >
                                            <SpeakerBadge
                                              label={
                                                segment.local_speaker_label
                                              }
                                              personName={
                                                segment.local_speaker_person_name
                                              }
                                              level={
                                                segment.local_speaker_decision_level
                                              }
                                            />
                                            <button
                                              className="transcript-time"
                                              onClick={() => {
                                                const player =
                                                  document.getElementById(
                                                    `clip-player-${clip.id}`,
                                                  ) as HTMLVideoElement | null;
                                                if (player) {
                                                  player.currentTime =
                                                    segment.start;
                                                  void player.play();
                                                }
                                              }}
                                            >
                                              {formatTimecode(segment.start)}
                                            </button>{" "}
                                            <SegmentText
                                              clipId={clip.id}
                                              jobId={
                                                transcript.transcription_job_id
                                              }
                                              segment={segment}
                                              csrf={csrf}
                                              onChanged={() =>
                                                loadClipDetail(clip.id)
                                              }
                                            />
                                            <SegmentHistory
                                              clipId={clip.id}
                                              jobId={
                                                transcript.transcription_job_id
                                              }
                                              index={segment.index}
                                              version={
                                                segment.correction_version ?? 0
                                              }
                                              csrf={csrf}
                                              onChanged={() =>
                                                loadClipDetail(clip.id)
                                              }
                                            />
                                            <SegmentSpeaker
                                              clipId={clip.id}
                                              jobId={
                                                transcript.transcription_job_id
                                              }
                                              segment={segment}
                                              speakers={
                                                transcript.local_speakers ?? []
                                              }
                                              csrf={csrf}
                                              onChanged={() =>
                                                loadClipDetail(clip.id)
                                              }
                                            />
                                          </p>
                                        ),
                                      )
                                    )}
                                  </section>
                                );
                              })()}
                            {clipDetails[clip.id] && (
                              <ObsoleteCorrections
                                clipId={clip.id}
                                jobId={
                                  clipDetails[clip.id].transcription
                                    ?.transcription_job_id
                                }
                              />
                            )}
                            {clipDetails[clip.id] && (
                              <ClipPersons
                                persons={clipDetails[clip.id].persons}
                              />
                            )}
                            {clipDetails[clip.id] &&
                              clip.status === "ready" && (
                                <VoicePanel
                                  clipId={clip.id}
                                  isAdmin={isAdministrator(session.user.role)}
                                  csrf={csrf}
                                  reloadToken={clipDetails[clip.id]}
                                />
                              )}
                            {clipDetails[clip.id] &&
                              clip.status === "ready" && (
                                <MicrophoneClipCard
                                  clipId={clip.id}
                                  isAdmin={isAdministrator(session.user.role)}
                                  csrf={csrf}
                                  onChanged={() => void loadClipDetail(clip.id)}
                                />
                              )}
                            {clipDetails[clip.id] && (
                              <button
                                className="secondary"
                                onClick={() => void loadClipDetail(clip.id)}
                              >
                                Actualiser la transcription
                              </button>
                            )}
                          </details>
                        </div>
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
              <VoiceVerificationCard csrf={csrf} />
              <VoiceGroupsCard csrf={csrf} />
              {setup.discord_enabled && (
                <section className="card" id="settings">
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
              {isAdministrator(session.user.role) && (
                <section className="card" aria-label="Export des corrections">
                  <h2>Export des corrections</h2>
                  <p>
                    Segments dont le texte a été corrigé, avec le texte
                    d’origine, le texte corrigé, l’auteur, la date et la
                    version. Snitch ne calcule aucun taux d’erreur.
                  </p>
                  <a
                    href="/api/admin/segment-corrections/export?format=ndjson"
                    download="segment-corrections.ndjson"
                  >
                    Télécharger l’export des segments corrigés
                  </a>
                </section>
              )}
              {isAdministrator(session.user.role) && embeddingState && (
                <section className="card" aria-label="Index sémantique">
                  <h2>Index sémantique</h2>
                  <p>
                    {embeddingState.configured
                      ? `${embeddingState.provider} · Modèle ${embeddingState.model} · ${embeddingState.dimensions || "dimension à détecter"} dimensions`
                      : "Service d’embeddings non configuré. La recherche lexicale reste disponible."}
                  </p>
                  <p>
                    État : {embeddingState.state} · {embeddingState.pending} en
                    attente · {embeddingState.done} indexées ·{" "}
                    {embeddingState.failed} en échec
                  </p>
                  <p>
                    Segments corrigés en attente de réindexation :{" "}
                    {embeddingState.reindex_pending ?? 0}
                    {embeddingState.reindex_failed
                      ? ` (dont ${embeddingState.reindex_failed} en échec)`
                      : ""}
                  </p>
                  {embeddingState.error && (
                    <p role="status">Erreur : {embeddingState.error}</p>
                  )}
                  <button
                    className="secondary"
                    disabled={busy || !embeddingState.configured}
                    onClick={() => run(backfillEmbeddings)}
                  >
                    Indexer les transcriptions existantes
                  </button>
                  {embeddingState.candidate_state &&
                  embeddingState.candidate_state !== "failed" ? (
                    <>
                      <p role="status">
                        Index candidat {embeddingState.candidate_model} ·{" "}
                        {embeddingState.candidate_state} ·{" "}
                        {embeddingState.candidate_done}/
                        {embeddingState.candidate_total} indexées ·{" "}
                        {embeddingState.candidate_pending} en attente ·{" "}
                        {embeddingState.candidate_failed} en échec
                      </p>
                      {embeddingState.candidate_error && (
                        <p role="status">
                          Erreur : {embeddingState.candidate_error}
                        </p>
                      )}
                    </>
                  ) : (
                    <form
                      onSubmit={(event) => {
                        event.preventDefault();
                        run(rebuildEmbeddings);
                      }}
                    >
                      {embeddingState.candidate_state === "failed" && (
                        <p role="status">
                          La reconstruction a échoué. L’index actif reste
                          utilisé ; vous pouvez relancer la préparation.
                        </p>
                      )}
                      <label>
                        Modèle à reconstruire
                        <input
                          value={embeddingModel || embeddingState.model}
                          onChange={(event) =>
                            setEmbeddingModel(event.target.value)
                          }
                          maxLength={128}
                          required
                        />
                      </label>
                      <button
                        className="secondary"
                        disabled={busy || !embeddingState.configured}
                      >
                        {embeddingState.candidate_state === "failed"
                          ? "Recommencer la reconstruction"
                          : "Reconstruire et basculer l’index"}
                      </button>
                    </form>
                  )}
                </section>
              )}
              {isSuperAdministrator(session.user.role) && settings && (
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
                    réglages OAuth2. Ajoutez exactement cette URL aux
                    redirections autorisées.
                  </p>
                  <p className="hint">
                    URL de retour à renseigner dans Discord :{" "}
                    {settings.callback_url}
                  </p>
                  <p className="hint">
                    Enregistrer ces réglages ne contacte pas Discord. Une
                    connexion ou une liaison ouvre Discord, puis le serveur
                    échange avec ce service pour lire l’identité publique
                    (portée identify uniquement).
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
                      Approbation administrateur requise pour les nouveaux
                      comptes
                    </label>
                    <button disabled={busy}>Enregistrer les réglages</button>
                    <button
                      type="button"
                      disabled={busy}
                      onClick={disableDiscord}
                    >
                      {settings.client_id
                        ? "Désactiver Discord"
                        : "Continuer sans Discord"}
                    </button>
                  </form>
                </section>
              )}
              {isAdministrator(session.user.role) && (
                <section className="card" id="transcription-pipeline">
                  <h2>Pipeline de transcription</h2>
                  {pipeline && (
                    <>
                      <p className="hint">
                        Version active : {pipelineVersion}. Une modification ne
                        change pas les résultats déjà calculés.
                      </p>
                      <form onSubmit={savePipeline}>
                        <label>
                          Profil de modèle (doit correspondre à
                          TRANSCRIPTION_MODEL sur le worker)
                          <input
                            name="model"
                            required
                            maxLength={100}
                            defaultValue={pipeline.model}
                          />
                        </label>
                        <label>
                          Langue
                          <input
                            name="language"
                            required
                            maxLength={32}
                            defaultValue={pipeline.language}
                          />
                        </label>
                        <label>
                          Détection d’activité vocale
                          <select
                            name="vad_mode"
                            defaultValue={pipeline.vad.mode}
                          >
                            <option value="off">Désactivée</option>
                            <option value="energy">VAD énergie</option>
                          </select>
                        </label>
                        <label>
                          Seuil d’énergie (0 à 1)
                          <input
                            name="threshold"
                            type="number"
                            min="0.001"
                            max="0.999"
                            step="0.001"
                            defaultValue={pipeline.vad.threshold ?? 0.02}
                          />
                        </label>
                        <label>
                          Durée minimale de parole (ms)
                          <input
                            name="min_speech_ms"
                            type="number"
                            min="1"
                            max="5000"
                            defaultValue={pipeline.vad.min_speech_ms ?? 100}
                          />
                        </label>
                        <label>
                          Silence minimum entre zones (ms)
                          <input
                            name="min_silence_ms"
                            type="number"
                            min="1"
                            max="10000"
                            defaultValue={pipeline.vad.min_silence_ms ?? 300}
                          />
                        </label>
                        <label>
                          Marge autour des zones (ms)
                          <input
                            name="padding_ms"
                            type="number"
                            min="0"
                            max="2000"
                            defaultValue={pipeline.vad.padding_ms ?? 100}
                          />
                        </label>
                        <button disabled={busy}>
                          Enregistrer et versionner
                        </button>
                      </form>
                    </>
                  )}
                </section>
              )}
              <PersonsCard csrf={csrf} />
              <MyVoiceCard csrf={csrf} />
              {isAdministrator(session.user.role) && (
                <PersonAccountCard csrf={csrf} />
              )}
              {isAdministrator(session.user.role) && (
                <VoicePipelineCard csrf={csrf} />
              )}
              {isAdministrator(session.user.role) && (
                <VoiceMatchingCard csrf={csrf} />
              )}
              <VoiceAutomationCard
                isAdmin={isAdministrator(session.user.role)}
                csrf={csrf}
              />
              {isAdministrator(session.user.role) && (
                <section className="card" id="vocabulary">
                  <h2>Vocabulaire de transcription</h2>
                  <p className="hint">
                    Version active : {vocabularyVersion}. Ces termes sont
                    envoyés au fournisseur à la prochaine transcription
                    planifiée.
                  </p>
                  <form
                    onSubmit={saveVocabularyTerm}
                    key={editingTerm?.id ?? "new-vocabulary-term"}
                  >
                    <label>
                      Terme
                      <input
                        name="term"
                        required
                        maxLength={160}
                        defaultValue={editingTerm?.term ?? ""}
                      />
                    </label>
                    <label>
                      Variantes, séparées par des virgules
                      <input
                        name="variants"
                        maxLength={4000}
                        defaultValue={editingTerm?.variants.join(", ") ?? ""}
                      />
                    </label>
                    <label>
                      Graphie préférée
                      <input
                        name="preferred_spelling"
                        required
                        maxLength={160}
                        defaultValue={editingTerm?.preferred_spelling ?? ""}
                      />
                    </label>
                    <label className="checkbox">
                      <input
                        name="active"
                        type="checkbox"
                        defaultChecked={editingTerm?.active ?? true}
                      />
                      Actif pour les prochaines transcriptions
                    </label>
                    <button disabled={busy}>
                      {editingTerm
                        ? "Enregistrer le terme"
                        : "Ajouter le terme"}
                    </button>
                    {editingTerm && (
                      <button
                        type="button"
                        className="secondary"
                        onClick={() => setEditingTerm(undefined)}
                      >
                        Annuler
                      </button>
                    )}
                  </form>
                  <ul className="vocabulary-list">
                    {vocabulary.map((term) => (
                      <li key={term.id}>
                        <div>
                          <strong>{term.preferred_spelling}</strong>
                          <span>
                            {term.term}
                            {term.variants.length
                              ? ` · ${term.variants.join(" · ")}`
                              : ""}
                          </span>
                          <small>{term.active ? "Actif" : "Inactif"}</small>
                        </div>
                        <div className="actions">
                          <button
                            type="button"
                            className="secondary"
                            onClick={() => setEditingTerm(term)}
                          >
                            Modifier
                          </button>
                          <button
                            type="button"
                            className="secondary"
                            disabled={busy}
                            onClick={() => void removeVocabularyTerm(term.id)}
                          >
                            Supprimer
                          </button>
                        </div>
                      </li>
                    ))}
                  </ul>
                  {vocabulary.length === 0 && <p>Aucun terme enregistré.</p>}
                </section>
              )}
              {isAdministrator(session.user.role) && (
                <section
                  className="card"
                  aria-label="Contrôle de la file"
                  id="queue-controls"
                >
                  <h2>Contrôle de la file</h2>
                  <p aria-live="polite">
                    {jobControls.paused
                      ? "Acquisitions en pause"
                      : "Acquisitions actives"}
                  </p>
                  <form onSubmit={saveJobControls}>
                    <label className="checkbox">
                      <input
                        type="checkbox"
                        checked={jobControls.paused}
                        onChange={(event) =>
                          setJobControls({
                            ...jobControls,
                            paused: event.target.checked,
                          })
                        }
                      />
                      Suspendre les nouvelles acquisitions
                    </label>
                    <label>
                      Parallélisme global
                      <input
                        type="number"
                        min="1"
                        max="4"
                        required
                        value={jobControls.max_concurrency}
                        onChange={(event) =>
                          setJobControls({
                            ...jobControls,
                            max_concurrency: Number(event.target.value),
                          })
                        }
                      />
                    </label>
                    <p className="hint">
                      La limite globale s’applique en plus de la capacité de
                      chaque worker. Les tâches déjà actives peuvent se terminer
                      pendant une pause.
                    </p>
                    <button disabled={busy}>Enregistrer les contrôles</button>
                  </form>
                </section>
              )}
              {isAdministrator(session.user.role) && (
                <section className="card" id="workers">
                  <h2>Workers</h2>
                  <p className="hint">
                    Le jeton apparaît une seule fois après création.
                    Configurez-le sur le PC worker avec une URL HTTPS.
                  </p>
                  <form onSubmit={registerWorker}>
                    <label>
                      Nom du worker
                      <input
                        name="worker_name"
                        minLength={1}
                        maxLength={64}
                        required
                      />
                    </label>
                    <label>
                      Concurrence maximale
                      <input
                        name="worker_capacity"
                        type="number"
                        min="1"
                        max="4"
                        defaultValue="1"
                        required
                      />
                    </label>
                    <button disabled={busy}>Créer un worker</button>
                  </form>
                  {newWorkerToken && (
                    <p role="status">
                      Jeton à copier maintenant: <code>{newWorkerToken}</code>
                    </p>
                  )}
                  {newWorkerBundle && (
                    <section aria-label="Fichiers du worker">
                      <p className="hint">
                        Placez ces deux fichiers dans un même dossier du PC
                        worker, renseignez les valeurs « A_RENSEIGNER » du
                        fichier .env, puis lancez « docker compose up -d ». Les
                        navigateurs ne gardent pas toujours le nom .env au
                        téléchargement (snitch-worker.env) : renommez-le en .env
                        ou ajoutez --env-file snitch-worker.env à la commande.
                        Le .env contient le jeton : il ne sera plus affiché
                        après votre départ de cette page.
                      </p>
                      {newWorkerBundle.insecure_api_url && (
                        <p className="alert error" role="alert">
                          L’URL publique de Snitch n’est pas en HTTPS : le
                          worker refuse le HTTP. Passez l’instance en HTTPS
                          avant de lancer le worker.
                        </p>
                      )}
                      {(
                        [
                          ["compose.yaml", newWorkerBundle.compose],
                          [".env", newWorkerBundle.env],
                        ] as const
                      ).map(([name, text]) => (
                        <div key={name}>
                          <label>
                            Contenu de {name}
                            <textarea
                              readOnly
                              rows={name === ".env" ? 16 : 12}
                              value={text}
                            />
                          </label>
                          <button
                            type="button"
                            onClick={() =>
                              run(async () => {
                                await navigator.clipboard.writeText(text);
                                setNotice(`${name} copié.`);
                              })
                            }
                          >
                            Copier {name}
                          </button>{" "}
                          <button
                            type="button"
                            onClick={() =>
                              downloadTextFile(
                                name === ".env" ? "snitch-worker.env" : name,
                                text,
                              )
                            }
                          >
                            Télécharger {name}
                          </button>
                        </div>
                      ))}
                    </section>
                  )}
                  {workers.length === 0 ? (
                    <p>Aucun worker enregistré.</p>
                  ) : (
                    <ul aria-label="Workers">
                      {workers.map((worker) => (
                        <li key={worker.id}>
                          <strong>{worker.name}</strong>
                          <p>
                            {worker.revoked_at
                              ? "Révoqué"
                              : worker.connected
                                ? "En ligne"
                                : "Absent"}{" "}
                            · capacité annoncée {worker.reported_concurrency}/
                            {worker.max_concurrency} · dernière activité{" "}
                            {worker.last_seen_at
                              ? new Date(worker.last_seen_at).toLocaleString(
                                  "fr-FR",
                                )
                              : "jamais"}
                          </p>
                          <p className="hint">{describeWorkerModels(worker)}</p>
                          {!worker.revoked_at && (
                            <button
                              className="secondary"
                              disabled={busy}
                              onClick={() => revokeWorker(worker)}
                            >
                              Révoquer le jeton
                            </button>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </section>
              )}
              {isAdministrator(session.user.role) && (
                <section className="card" id="members">
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
                                  await api.POST(
                                    "/api/admin/users/{id}/approve",
                                    {
                                      params: {
                                        path: { id: user.id },
                                        header: csrf,
                                      },
                                    },
                                  ),
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
              {isAdministrator(session.user.role) && (
                <section className="card" id="job-queue">
                  <h2>File de tâches ({jobTotal})</h2>
                  <p className="hint">
                    Les tâches restent en attente jusqu’à ce qu’un worker prenne
                    en charge leur étape.
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
                    <label>
                      Nombre maximal de tentatives
                      <input
                        name="max_attempts"
                        type="number"
                        min="1"
                        max="20"
                        defaultValue="3"
                        required
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
                            status: event.target
                              .value as typeof jobFilters.status,
                          })
                        }
                      >
                        <option value="">Tous les états</option>
                        <option value="pending">En attente</option>
                        <option value="leased">Louée</option>
                        <option value="done">Terminée</option>
                        <option value="failed">Échouée</option>
                        <option value="retry_wait">Nouvel essai différé</option>
                        <option value="dead">Épuisée</option>
                        <option value="cancelled">Annulée</option>
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
                          setJobFilters({
                            ...jobFilters,
                            step: event.target.value,
                          })
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
                            priorité {job.priority} · cycle {job.cycle} ·
                            tentatives {job.attempts}/{job.max_attempts}
                          </p>
                          <p className="hint">
                            Version {job.input_version} · créée{" "}
                            {new Date(job.created_at).toLocaleString("fr-FR")}
                            {job.status === "retry_wait" &&
                              ` · prochain essai ${new Date(job.next_attempt_at).toLocaleString("fr-FR")}`}
                            {job.last_error && ` · erreur ${job.last_error}`}
                          </p>
                          <div className="toolbar">
                            <button
                              className="secondary"
                              disabled={busy}
                              onClick={() => loadJobHistory(job)}
                            >
                              Historique
                            </button>
                            {job.status === "dead" && (
                              <button
                                disabled={busy}
                                onClick={() => relaunchJob(job)}
                              >
                                Relancer
                              </button>
                            )}
                            {["pending", "retry_wait", "leased"].includes(
                              job.status,
                            ) && (
                              <button
                                className="secondary"
                                disabled={busy}
                                onClick={() => cancelJob(job)}
                              >
                                Annuler
                              </button>
                            )}
                          </div>
                          {jobHistory[job.id] && (
                            <ul aria-label={`Historique de la tâche ${job.id}`}>
                              {jobHistory[job.id].map((event) => (
                                <li key={event.id}>
                                  {event.event} · cycle {event.cycle}
                                  {event.attempt
                                    ? ` · tentative ${event.attempt}`
                                    : ""}
                                  {event.error_code
                                    ? ` · ${event.error_code}`
                                    : ""}{" "}
                                  ·{" "}
                                  {new Date(event.created_at).toLocaleString(
                                    "fr-FR",
                                  )}
                                </li>
                              ))}
                            </ul>
                          )}
                          {job.result && (
                            <pre aria-label={`Résultat de la tâche ${job.id}`}>
                              {JSON.stringify(job.result, null, 2)}
                            </pre>
                          )}
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
              {isAdministrator(session.user.role) && (
                <section className="card" id="sources">
                  <h2>Sources dossier</h2>
                  {allowedRoots.length === 0 ? (
                    <p className="hint">
                      Aucun dossier n’est monté pour Snitch. Configurez
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
                          <details className="source-track-rule">
                            <summary>
                              Règle de piste audio pour cette source
                            </summary>
                            <form
                              onSubmit={(event) => {
                                event.preventDefault();
                                const form = new FormData(event.currentTarget);
                                void run(async () => {
                                  const fallback = String(
                                    form.get("fallback_audio_stream_index") ||
                                      "",
                                  ).trim();
                                  const channels = String(
                                    form.get("channels") || "",
                                  ).trim();
                                  const body = {
                                    title:
                                      String(form.get("title") || "").trim() ||
                                      undefined,
                                    language:
                                      String(
                                        form.get("language") || "",
                                      ).trim() || undefined,
                                    channel_layout:
                                      String(
                                        form.get("channel_layout") || "",
                                      ).trim() || undefined,
                                    channels: channels
                                      ? Number(channels)
                                      : undefined,
                                    fallback_audio_stream_index: fallback
                                      ? Number(fallback)
                                      : undefined,
                                  };
                                  checked(
                                    await api.PUT(
                                      "/api/admin/sources/{id}/audio-track-rule",
                                      {
                                        params: {
                                          path: { id: source.id },
                                          header: csrf,
                                        },
                                        body,
                                      },
                                    ),
                                  );
                                  setNotice(
                                    "Règle de piste enregistrée; les clips résolus sont replanifiés.",
                                  );
                                });
                              }}
                            >
                              <label>
                                Titre de piste exact
                                <input name="title" placeholder="ex. Voix" />
                              </label>
                              <label>
                                Langue exacte
                                <input name="language" placeholder="ex. fra" />
                              </label>
                              <label>
                                Disposition
                                <input
                                  name="channel_layout"
                                  placeholder="ex. stereo"
                                />
                              </label>
                              <label>
                                Nombre de canaux exact
                                <input name="channels" type="number" min="1" />
                              </label>
                              <label>
                                Piste audio de repli (index global)
                                <input
                                  name="fallback_audio_stream_index"
                                  type="number"
                                  min="0"
                                />
                              </label>
                              <button className="secondary" disabled={busy}>
                                Enregistrer la règle
                              </button>
                            </form>
                          </details>
                          <MicrophoneSourceCard
                            sourceId={source.id}
                            csrf={csrf}
                          />
                          <ul>
                            {source.files.map((file) => (
                              <li
                                className="source-file"
                                key={file.relative_path}
                              >
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
            </div>
          </div>
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
