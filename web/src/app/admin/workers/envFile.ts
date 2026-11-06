/** Placeholder Snitch writes for every value the administrator provides by hand. */
const PLACEHOLDER = "A_RENSEIGNER";

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
