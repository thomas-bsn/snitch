import { useSyncExternalStore } from "react";

// One volume for every player of the application, kept in this browser.
const KEY = "snitch.volume";
const listeners = new Set<() => void>();

function read(): number {
  try {
    const stored = Number(localStorage.getItem(KEY));
    if (localStorage.getItem(KEY) !== null && stored >= 0 && stored <= 1)
      return stored;
  } catch {
    // Storage can be blocked: the volume then lasts for the page only.
  }
  return 1;
}

let level = read();

export const getVolume = () => level;

export function setVolume(value: number) {
  const next = Math.min(1, Math.max(0, value));
  if (next === level) return;
  level = next;
  try {
    localStorage.setItem(KEY, String(next));
  } catch {
    // Not persisted.
  }
  listeners.forEach((listener) => listener());
}

export function subscribeVolume(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export const useVolume = () =>
  useSyncExternalStore(subscribeVolume, getVolume, getVolume);
