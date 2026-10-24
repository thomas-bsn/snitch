package httpapi

import "testing"

func TestResolveAudioTrackRequiresExplicitSelectionForAmbiguousMultitrack(t *testing.T) {
	tracks := []audioTrack{{Index: 1, Type: "audio", Title: "Game", Language: "fra", Channels: 2, ChannelLayout: "stereo"}, {Index: 2, Type: "audio", Title: "Voice", Language: "fra", Channels: 2, ChannelLayout: "stereo"}}
	if _, ok := resolveAudioTrack(tracks, audioTrackRule{}); ok {
		t.Fatal("ambiguous audio streams must not select the first stream")
	}
	title := "Voice"
	if got, ok := resolveAudioTrack(tracks, audioTrackRule{Title: &title}); !ok || got != 2 {
		t.Fatalf("title rule got %d, %v", got, ok)
	}
	title = "Missing"
	fallback := 1
	if got, ok := resolveAudioTrack(tracks, audioTrackRule{Title: &title, FallbackAudioStreamIndex: &fallback}); !ok || got != 1 {
		t.Fatalf("fallback got %d, %v", got, ok)
	}
	language := "fra"
	if _, ok := resolveAudioTrack(tracks, audioTrackRule{Language: &language, FallbackAudioStreamIndex: &fallback}); ok {
		t.Fatal("fallback must not hide an ambiguous rule match")
	}
}
