/** Placeholder Snitch writes for every value the administrator provides by hand. */
const PLACEHOLDER = "A_RENSEIGNER";

const SAFE_VALUE = /^[A-Za-z0-9_./:@%+=,-]+$/;

/** A Docker Compose .env value: bare when safe, otherwise double-quoted and escaped. */
function envValue(value: string): string {
  if (SAFE_VALUE.test(value)) return value;
  const escaped = value
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\$/g, "\\$")
    .replace(/\n/g, "\\n")
    .replace(/\r/g, "\\r");
  return `"${escaped}"`;
}

/**
 * Writes the Hugging Face token into the generated .env file. This happens in the browser
 * only: the token is never sent to Snitch, which could not give it back.
 */
export function withHuggingFaceToken(env: string, token: string): string {
  const value = token.trim();
  if (!value) return env;
  return env.replace(
    new RegExp(`^HF_TOKEN=${PLACEHOLDER}$`, "m"),
    () => `HF_TOKEN=${envValue(value)}`,
  );
}

/** Values still to fill in by hand in a generated .env file. */
export function placeholdersLeft(env: string): string[] {
  return env
    .split("\n")
    .filter((line) => line.endsWith(`=${PLACEHOLDER}`))
    .map((line) => line.slice(0, line.indexOf("=")));
}

/** Offers a text as a downloaded file, without any request. */
export function downloadText(filename: string, text: string) {
  const url = URL.createObjectURL(new Blob([text], { type: "text/plain" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}
