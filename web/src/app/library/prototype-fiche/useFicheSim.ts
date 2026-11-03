// PROTOTYPE — état en mémoire et direct simulé par minuteries.
import { useCallback, useEffect, useRef, useState } from "react";
import {
  cascade,
  failureLogs,
  initialSegments,
  initialSteps,
  initialVoices,
  PREREQUISITES,
  STEP_ORDER,
  whisperErrorLogs,
  type Segment,
  type Step,
  type StepKey,
  type Voice,
} from "./data";

export type Sim = ReturnType<typeof useFicheSim>;

function now() {
  return new Date().toTimeString().slice(0, 8);
}

/** Étapes à lancer : la sélection, ce qui lui manque en amont, et ce qui en dépend en aval. */
export function launchPlan(steps: Step[], selected: StepKey[]) {
  const set = new Set<StepKey>(cascade(selected));
  let grew = true;
  while (grew) {
    grew = false;
    for (const k of [...set])
      for (const p of PREREQUISITES[k]) {
        const st = steps.find((s) => s.key === p)!;
        if (st.status !== "done" && !set.has(p)) {
          set.add(p);
          grew = true;
        }
      }
  }
  const plan = STEP_ORDER.filter((k) => set.has(k));
  const added = plan.filter((k) => !selected.includes(k));
  return { plan, added };
}

export function useFicheSim() {
  const [steps, setSteps] = useState<Step[]>(initialSteps);
  const [voices, setVoices] = useState<Voice[]>(initialVoices);
  const [segments, setSegments] = useState<Segment[]>(initialSegments);
  const [logs, setLogs] = useState<Record<string, string[]>>({
    fingerprints: failureLogs(),
  });
  const [failNext, setFailNext] = useState<"none" | "fingerprints" | "transcription">("fingerprints");
  const [voiceWorkerMissing, setVoiceWorkerMissing] = useState(false);
  const failRef = useRef(failNext);
  failRef.current = failNext;
  const missingRef = useRef(voiceWorkerMissing);
  missingRef.current = voiceWorkerMissing;

  const running = steps.some((s) => s.status === "running" || s.status === "pending" || s.status === "waiting");

  // Le moteur : fait avancer l'étape en cours, puis démarre la suivante en attente.
  useEffect(() => {
    if (!running) return;
    const timer = setInterval(() => {
      setSteps((prev) => {
        const next = prev.map((s) => ({ ...s }));
        const cur = next.find((s) => s.status === "running");
        if (cur) {
          cur.progress = Math.min(100, (cur.progress ?? 0) + (cur.key === "transcription" ? 4 : 9));
          const line = `${now()}  DEBUG progress step=${cur.key} pct=${cur.progress}`;
          setLogs((l) => ({ ...l, [cur.key]: [...(l[cur.key] ?? []), line].slice(-50) }));
          const failAt = failRef.current === cur.key && cur.progress >= 63;
          if (failAt) {
            cur.status = "failed";
            cur.progress = undefined;
            cur.error =
              cur.key === "transcription"
                ? { code: "provider_unavailable", message: "Le service de transcription ne répond pas (HTTP 503 « model is loading ») après 5 tentatives.", at: "à l'instant" }
                : { code: "inference_failed", message: "Le modèle d'empreintes n'a pas pu traiter un segment (étape : extraction du segment). Le segment dépasse la fin de l'audio.", at: "à l'instant" };
            setLogs((l) => ({
              ...l,
              [cur.key]: [...(l[cur.key] ?? []), ...(cur.key === "transcription" ? whisperErrorLogs : failureLogs().slice(-4))].slice(-50),
            }));
            // Ce qui dépendait d'elle ne peut plus partir.
            for (const s of next)
              if (s.status === "pending" && PREREQUISITES[s.key].includes(cur.key)) s.status = "cancelled";
            for (const s of next)
              if (s.status === "pending" && PREREQUISITES[s.key].some((p) => next.find((x) => x.key === p)!.status === "cancelled"))
                s.status = "cancelled";
          } else if (cur.progress >= 100) {
            cur.status = "done";
            cur.progress = undefined;
            cur.durationS = 10 + Math.round(Math.random() * 200);
            setLogs((l) => ({ ...l, [cur.key]: [...(l[cur.key] ?? []), `${now()}  INFO  step_done step=${cur.key}`].slice(-50) }));
          }
          return next;
        }
        const ready = next.find(
          (s) =>
            (s.status === "pending" || s.status === "waiting") &&
            PREREQUISITES[s.key].every((p) => next.find((x) => x.key === p)!.status === "done"),
        );
        if (ready) {
          const isVoice = ["diarization", "fingerprints"].includes(ready.key);
          if (isVoice && missingRef.current) {
            ready.status = "waiting";
            ready.waitCause = "Aucun worker voix connecté — reprendra seul dès qu'un worker avec les modèles vocaux sera en ligne.";
            return next;
          }
          ready.status = "running";
          ready.waitCause = undefined;
          ready.progress = 0;
          ready.worker = ready.key === "indexing" || ready.key === "matching" ? undefined : "gpu-salon";
          setLogs((l) => ({ ...l, [ready.key]: [`${now()}  INFO  job_claimed step=${ready.key} worker=gpu-salon lease=120s`] }));
        }
        return next;
      });
    }, 350);
    return () => clearInterval(timer);
  }, [running]);

  const launch = useCallback((selected: StepKey[]) => {
    setSteps((prev) => {
      const { plan } = launchPlan(prev, selected);
      return prev.map((s) =>
        plan.includes(s.key) ? { ...s, status: "pending", error: undefined, progress: undefined, durationS: undefined } : s,
      );
    });
  }, []);

  const cancel = useCallback(() => {
    setSteps((prev) =>
      prev.map((s) => (s.status === "pending" || s.status === "waiting" ? { ...s, status: "cancelled" } : s)),
    );
  }, []);

  const remove = useCallback((selected: StepKey[]) => {
    const all = cascade(selected);
    setSteps((prev) =>
      prev.map((s) => (all.includes(s.key) ? { ...s, status: "absent", error: undefined, durationS: undefined, worker: undefined } : s)),
    );
    if (all.includes("transcription")) setSegments([]);
    if (all.includes("diarization")) setVoices([]);
    else if (all.includes("matching"))
      setVoices((v) => v.map((x) => (x.state === "ignored" || x.state === "random" ? x : { ...x, state: "unknown", person: undefined, candidates: undefined })));
    setLogs((l) => Object.fromEntries(Object.entries(l).filter(([k]) => !all.includes(k as StepKey))));
  }, []);

  const reset = useCallback(() => {
    setSteps(initialSteps());
    setVoices(initialVoices);
    setSegments(initialSegments);
    setLogs({ fingerprints: failureLogs() });
  }, []);

  const updateVoice = useCallback((id: number, patch: Partial<Voice>) => {
    setVoices((v) => v.map((x) => (x.id === id ? { ...x, ...patch } : x)));
  }, []);

  const voiceActions = {
    validate: (v: Voice, person: string) => updateVoice(v.id, { state: "validated", person, candidates: undefined }),
    /** « Ce n'est pas cette personne » : la voix redevient speaker_n et le refus est retenu. */
    reject: (v: Voice) =>
      updateVoice(v.id, {
        state: "unknown",
        refused: v.person ? [...v.refused, v.person] : v.refused,
        person: undefined,
        candidates: v.candidates?.filter((c) => c !== v.person),
      }),
    ignore: (v: Voice) => updateVoice(v.id, { state: "ignored" }),
    random: (v: Voice) => updateVoice(v.id, { state: "random" }),
    restore: (v: Voice) => updateVoice(v.id, { state: v.person ? "suggested" : "unknown" }),
  };

  const editSegment = useCallback((id: number, text: string) => {
    setSegments((s) => s.map((x) => (x.id === id ? { ...x, text, edited: true } : x)));
  }, []);

  return {
    steps,
    voices,
    segments,
    logs,
    running,
    failNext,
    setFailNext,
    voiceWorkerMissing,
    setVoiceWorkerMissing,
    launch,
    cancel,
    remove,
    reset,
    voiceActions,
    editSegment,
  };
}
