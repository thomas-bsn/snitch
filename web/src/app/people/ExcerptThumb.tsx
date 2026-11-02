// Decorative tiles: one of a few tints, picked from the voice so a case keeps its look.
const tints = [
  "bg-linear-to-br from-indigo-200 to-zinc-200 dark:from-indigo-900 dark:to-zinc-800",
  "bg-linear-to-br from-orange-200 to-stone-200 dark:from-orange-900 dark:to-stone-800",
  "bg-linear-to-br from-emerald-200 to-sky-100 dark:from-emerald-900 dark:to-sky-900",
  "bg-linear-to-br from-fuchsia-200 to-violet-100 dark:from-fuchsia-900 dark:to-violet-900",
];

/** The tint of a voice's tile. */
export const tintOf = (localSpeakerId: number) =>
  tints[localSpeakerId % tints.length];
