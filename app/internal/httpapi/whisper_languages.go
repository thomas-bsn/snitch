package httpapi

import "slices"

// whisperLanguages lists the language codes Whisper accepts (its tokenizer's LANGUAGES,
// large-v3 included), which is the only choice offered for the library language.
// Whisper's own codes are kept as is: "jw" for Javanese, "haw" and "yue" have three letters.
// Keep in sync with web/src/app/admin/transcription/whisperLanguages.ts.
var whisperLanguages = []string{
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
}

func validTranscriptionLanguage(language string) bool {
	return slices.Contains(whisperLanguages, language)
}
