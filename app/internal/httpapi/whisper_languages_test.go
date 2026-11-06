package httpapi

import (
	"os"
	"regexp"
	"slices"
	"testing"
)

// The web interface offers exactly the languages the server accepts.
func TestWhisperLanguagesMatchTheWebInterface(t *testing.T) {
	source, err := os.ReadFile("../../../web/src/app/admin/transcription/whisperLanguages.ts")
	if err != nil {
		t.Fatal(err)
	}
	list := regexp.MustCompile(`(?s)WHISPER_LANGUAGES = \[(.*?)\]`).FindSubmatch(source)
	if list == nil {
		t.Fatal("WHISPER_LANGUAGES not found")
	}
	var web []string
	for _, code := range regexp.MustCompile(`"([a-z]+)"`).FindAllSubmatch(list[1], -1) {
		web = append(web, string(code[1]))
	}
	if !slices.Equal(web, whisperLanguages) {
		t.Fatalf("web list differs from the server list:\nweb    %v\nserver %v", web, whisperLanguages)
	}
	if len(whisperLanguages) != 100 {
		t.Fatalf("Whisper supports 100 languages, got %d", len(whisperLanguages))
	}
}
