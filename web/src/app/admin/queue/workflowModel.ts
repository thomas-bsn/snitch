import type { components } from "../../../api.gen";
export type Job = components["schemas"]["Job"];
export type ClipWork = {
  clipId: number;
  jobs: Job[];
  current: Job;
  finished: boolean;
  updated: number;
};
export function groupClipWork(jobs: Job[]): ClipWork[] {
  const groups = new Map<number, Map<string, Job>>();
  for (const job of jobs) {
    const steps = groups.get(job.clip_id) ?? new Map<string, Job>();
    const previous = steps.get(job.step);
    if (!previous || job.id > previous.id) steps.set(job.step, job);
    groups.set(job.clip_id, steps);
  }
  return [...groups]
    .map(([clipId, steps]) => {
      const items = [...steps.values()];
      const active = items.filter((j) => j.status !== "done");
      const current =
        active.find((j) => j.status === "leased") ?? active[0] ?? items[0];
      return {
        clipId,
        jobs: items,
        current,
        finished: items.every((j) => j.status === "done"),
        updated: Math.max(...items.map((j) => Date.parse(j.updated_at))),
      };
    })
    .sort((a, b) => b.updated - a.updated);
}
