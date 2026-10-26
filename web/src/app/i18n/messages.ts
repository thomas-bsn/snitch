import { en } from "./en";

/** Shape every language file must have: the English file is the reference. */
export type Messages = typeof en;

type Leaves<T, Prefix extends string = ""> = {
  [K in keyof T & string]: T[K] extends string
    ? `${Prefix}${K}`
    : Leaves<T[K], `${Prefix}${K}.`>;
}[keyof T & string];

/** Dotted path of any text, for example "menu.signOut". */
export type MessageKey = Exclude<Leaves<Messages>, `errors.${string}`>;

export type Params = Record<string, string | number>;

function lookup(messages: unknown, path: string): string | undefined {
  let node = messages;
  for (const part of path.split(".")) {
    if (typeof node !== "object" || node === null) return undefined;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : undefined;
}

function interpolate(template: string, params?: Params): string {
  if (!params) return template;
  return template.replace(/\{(\w+)\}/g, (whole, name: string) =>
    name in params ? String(params[name]) : whole,
  );
}

export function translate(
  messages: Messages,
  key: MessageKey,
  params?: Params,
): string {
  return interpolate(lookup(messages, key) ?? key, params);
}

/**
 * Text for a server error code, or a generic message that shows the code when this
 * language has no text for it.
 */
export function translateError(messages: Messages, code: string): string {
  const known = (messages.errors as Record<string, string>)[code];
  return known ?? interpolate(messages.errorFallback, { code });
}
