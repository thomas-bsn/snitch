import { useSyncExternalStore } from "react";

// Tracks whether the server answers. Any request that gets no usable answer marks it
// unreachable; the interface then shows a message with a "try again" button, and the
// retry clears the mark before loading everything again.

let unreachable = false;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((listener) => listener());
}

export function markUnreachable() {
  if (unreachable) return;
  unreachable = true;
  emit();
}

export function markReachable() {
  if (!unreachable) return;
  unreachable = false;
  emit();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useServerUnreachable(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => unreachable,
    () => false,
  );
}
