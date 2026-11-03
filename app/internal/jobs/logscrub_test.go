package jobs

import "testing"

// The table mirrors worker/tests/test_logging.py: the Worker and the server must mask the same
// values. Every value is synthetic.
func TestScrubLogLine(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"bearer header", "Authorization: Bearer abc.def", "Authorization: Bearer [masked]"},
		{"hugging face token", "token hf_SyntheticToken123 ok", "token [masked] ok"},
		{"lease token", "lease rw_synthetic-token", "lease [masked]"},
		{"credentialed url", "https://ann:pw-synth@hub.example/path?sig=abc", "https://hub.example/[masked]"},
		{"bare url keeps its host", "https://hub.example", "https://hub.example"},
		{"clip path", "clip /private/var/synthetic/audio.wav done", "clip [masked_path] done"},
		{"media file name", "synthetic-recording.mp4 uploaded", "[masked_path] uploaded"},
		{"stable codes stay", "hf_not_found job 42 claimed for step voice.fingerprint", "hf_not_found job 42 claimed for step voice.fingerprint"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ScrubLogLine(c.in); got != c.want {
				t.Fatalf("ScrubLogLine(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
