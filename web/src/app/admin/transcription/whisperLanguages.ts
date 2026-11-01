import type { Language } from "../../preferences";

/**
 * The language codes Whisper accepts (its tokenizer's list, large-v3 included): the only
 * choice for the library language. Whisper's own codes are kept as sent to the API ("jw"
 * for Javanese). Keep in sync with app/internal/httpapi/whisper_languages.go, which a Go
 * test compares with this list.
 */
// prettier-ignore
export const WHISPER_LANGUAGES = [
  "en", "zh", "de", "es", "ru", "ko", "fr", "ja", "pt", "tr",
  "pl", "ca", "nl", "ar", "sv", "it", "id", "hi", "fi", "vi",
  "he", "uk", "el", "ms", "cs", "ro", "da", "hu", "ta", "no",
  "th", "ur", "hr", "bg", "lt", "la", "mi", "ml", "cy", "sk",
  "te", "fa", "lv", "bn", "sr", "az", "sl", "kn", "et", "mk",
  "br", "eu", "is", "hy", "ne", "mn", "bs", "kk", "sq", "sw",
  "gl", "mr", "pa", "si", "km", "sn", "yo", "so", "af", "oc",
  "ka", "be", "tg", "sd", "gu", "am", "yi", "lo", "uz", "fo",
  "ht", "ps", "tk", "nn", "mt", "sa", "lb", "my", "bo", "tl",
  "mg", "as", "tt", "haw", "ln", "ha", "ba", "jw", "su", "yue",
] as const;

/** Whisper's names, used when the browser cannot name a language itself. */
// prettier-ignore
const WHISPER_NAMES: Record<string, string> = {
  en: "English", zh: "Chinese", de: "German", es: "Spanish", ru: "Russian",
  ko: "Korean", fr: "French", ja: "Japanese", pt: "Portuguese", tr: "Turkish",
  pl: "Polish", ca: "Catalan", nl: "Dutch", ar: "Arabic", sv: "Swedish",
  it: "Italian", id: "Indonesian", hi: "Hindi", fi: "Finnish", vi: "Vietnamese",
  he: "Hebrew", uk: "Ukrainian", el: "Greek", ms: "Malay", cs: "Czech",
  ro: "Romanian", da: "Danish", hu: "Hungarian", ta: "Tamil", no: "Norwegian",
  th: "Thai", ur: "Urdu", hr: "Croatian", bg: "Bulgarian", lt: "Lithuanian",
  la: "Latin", mi: "Maori", ml: "Malayalam", cy: "Welsh", sk: "Slovak",
  te: "Telugu", fa: "Persian", lv: "Latvian", bn: "Bengali", sr: "Serbian",
  az: "Azerbaijani", sl: "Slovenian", kn: "Kannada", et: "Estonian",
  mk: "Macedonian", br: "Breton", eu: "Basque", is: "Icelandic", hy: "Armenian",
  ne: "Nepali", mn: "Mongolian", bs: "Bosnian", kk: "Kazakh", sq: "Albanian",
  sw: "Swahili", gl: "Galician", mr: "Marathi", pa: "Punjabi", si: "Sinhala",
  km: "Khmer", sn: "Shona", yo: "Yoruba", so: "Somali", af: "Afrikaans",
  oc: "Occitan", ka: "Georgian", be: "Belarusian", tg: "Tajik", sd: "Sindhi",
  gu: "Gujarati", am: "Amharic", yi: "Yiddish", lo: "Lao", uz: "Uzbek",
  fo: "Faroese", ht: "Haitian Creole", ps: "Pashto", tk: "Turkmen",
  nn: "Nynorsk", mt: "Maltese", sa: "Sanskrit", lb: "Luxembourgish",
  my: "Myanmar", bo: "Tibetan", tl: "Tagalog", mg: "Malagasy", as: "Assamese",
  tt: "Tatar", haw: "Hawaiian", ln: "Lingala", ha: "Hausa", ba: "Bashkir",
  jw: "Javanese", su: "Sundanese", yue: "Cantonese",
};

/** Whisper codes that differ from the standard code the browser knows. */
const STANDARD_CODES: Record<string, string> = { jw: "jv" };

export type WhisperLanguageOption = {
  /** The code sent to the API. */
  value: string;
  /** "Name (code)", in the interface language. */
  label: string;
};

function namer(language: Language): (code: string) => string {
  let names: Intl.DisplayNames | undefined;
  try {
    names = new Intl.DisplayNames([language], {
      type: "language",
      fallback: "none",
    });
  } catch {
    names = undefined;
  }
  return (code) => {
    let name: string | undefined;
    try {
      name = names?.of(STANDARD_CODES[code] ?? code);
    } catch {
      name = undefined; // Not a well-formed code.
    }
    if (!name) return WHISPER_NAMES[code] ?? code;
    return name.charAt(0).toLocaleUpperCase(language) + name.slice(1);
  };
}

const label = (name: (code: string) => string, code: string) =>
  `${name(code)} (${code})`;

/** "Name (code)" for a Whisper code, in the interface language. */
export function whisperLanguageLabel(code: string, language: Language) {
  return code ? label(namer(language), code) : "";
}

/**
 * The library language choices, named in the interface language and sorted by name.
 * A saved code missing from the list (stored before the list existed) is still shown.
 */
export function whisperLanguageOptions(
  language: Language,
  current?: string,
): WhisperLanguageOption[] {
  const name = namer(language);
  const codes: string[] = [...WHISPER_LANGUAGES];
  if (current && !codes.includes(current)) codes.push(current);
  return codes
    .map((code) => ({ value: code, label: label(name, code) }))
    .sort((a, b) => a.label.localeCompare(b.label, language));
}
