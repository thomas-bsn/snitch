/** One letter without its case or accent, so that "joue" finds "Joué". */
const plain = (letter: string) =>
  letter.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase();

export type Excerpt = { before: string; hit: string; after: string };

/**
 * Cuts the quote around the passage that was found. The server gives the matched text, which
 * can differ from the quote by case or accents; when it is not found as such (a typo found by
 * an approximate match), the whole quote is returned without a highlight.
 */
export function excerpt(context: string, matched: string): Excerpt {
  const letters = Array.from(context);
  const haystack = letters.map(plain);
  const needle = Array.from(matched, plain);
  if (needle.length === 0) return { before: context, hit: "", after: "" };
  for (let start = 0; start + needle.length <= haystack.length; start += 1) {
    if (needle.every((letter, i) => haystack[start + i] === letter)) {
      const end = start + needle.length;
      return {
        before: letters.slice(0, start).join(""),
        hit: letters.slice(start, end).join(""),
        after: letters.slice(end).join(""),
      };
    }
  }
  return { before: context, hit: "", after: "" };
}
