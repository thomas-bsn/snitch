// PROTOTYPE — arborescence factice de /videos pour l'explorateur et l'import.

export type VideoState = "unindexed" | "indexed" | "running" | "transcribed" | "analysed" | "failed";

export type Node =
  | { kind: "folder"; name: string; children: Node[]; private?: boolean; pending?: boolean }
  | { kind: "video"; name: string; state: VideoState; duration?: string; date: string; size: string; title?: string; todo?: number };

export const STATE_LABEL: Record<VideoState, string> = {
  unindexed: "Non indexée",
  indexed: "Non analysée",
  running: "En cours",
  transcribed: "Transcrite",
  analysed: "Analysée",
  failed: "Échec",
};

export const STATE_TONE: Record<VideoState, string> = {
  unindexed: "border border-dashed border-line-strong text-mute",
  indexed: "bg-side text-mute",
  running: "bg-accent-soft text-accent-ink",
  transcribed: "bg-ok-bg text-ok-fg",
  analysed: "bg-ok-bg text-ok-fg",
  failed: "bg-bad-bg text-bad-fg",
};

const v = (name: string, state: VideoState, date: string, extra: Partial<Extract<Node, { kind: "video" }>> = {}): Node => ({
  kind: "video",
  name,
  state,
  date,
  size: extra.size ?? "1,2 Go",
  duration: state === "unindexed" ? undefined : (extra.duration ?? "14:20"),
  ...extra,
});

export const root: Node = {
  kind: "folder",
  name: "videos",
  children: [
    {
      kind: "folder",
      name: "2026",
      children: [
        {
          kind: "folder",
          name: "novembre",
          children: [
            v("reunion-prepa-s12.mp4", "failed", "28 oct.", { title: "Réunion de préparation — tournage S12", duration: "38:12", todo: 4 }),
            v("essai-micro-hf.mov", "running", "2 nov."),
            v("rush-nuit-01.mp4", "unindexed", "3 nov.", { size: "4,1 Go" }),
            v("rush-nuit-02.mp4", "unindexed", "3 nov.", { size: "3,8 Go" }),
          ],
        },
        {
          kind: "folder",
          name: "octobre",
          children: [
            {
              kind: "folder",
              name: "parking",
              children: [
                v("prise-1.mp4", "analysed", "24 oct."),
                v("prise-2.mp4", "analysed", "24 oct."),
                v("prise-3.mp4", "analysed", "24 oct.", { title: "Rushes parking — prise 3", todo: 3 }),
                v("prise-4.mp4", "unindexed", "24 oct."),
              ],
            },
            v("interview-claire.mp4", "analysed", "19 oct.", { title: "Interview Claire — making-of", todo: 1 }),
            v("brief-lumiere.mp4", "transcribed", "12 oct.", { title: "Brief équipe lumière" }),
            v("broll-ville.mp4", "indexed", "10 oct."),
          ],
        },
        { kind: "folder", name: "septembre", children: [v("visio-budget.mp4", "analysed", "30 sept."), v("reperages-studio-b.mp4", "analysed", "21 sept.")] },
      ],
    },
    { kind: "folder", name: "archives-2025", children: [v("bout-a-bout-v3.mov", "unindexed", "12 déc. 2025", { size: "18 Go" })] },
    { kind: "folder", name: "perso-thomas", private: true, children: [v("vlog-01.mp4", "indexed", "2 sept.")] },
    v("import-sans-dossier.mp4", "transcribed", "1 nov."),
  ],
};

export function at(path: string[]): Extract<Node, { kind: "folder" }> {
  let cur = root as Extract<Node, { kind: "folder" }>;
  for (const p of path) cur = cur.children.find((c) => c.kind === "folder" && c.name === p) as typeof cur;
  return cur;
}

export function countVideos(n: Node): { total: number; unindexed: number; done: number } {
  if (n.kind === "video") return { total: 1, unindexed: n.state === "unindexed" ? 1 : 0, done: n.state === "analysed" || n.state === "transcribed" ? 1 : 0 };
  return n.children.map(countVideos).reduce((a, b) => ({ total: a.total + b.total, unindexed: a.unindexed + b.unindexed, done: a.done + b.done }), { total: 0, unindexed: 0, done: 0 });
}

export const ficheFor = (n: Extract<Node, { kind: "video" }>) =>
  `/prototype/fiche-video?variant=C${n.state === "unindexed" || n.state === "indexed" ? "&etat=jamais" : ""}`;
