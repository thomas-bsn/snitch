package httpapi

import "testing"

// Only a repository of the exact Hugging Face host can be inspected: any other host, scheme, port or
// credential in the pasted URL is refused before any request leaves Snitch.
func TestParseHFSource(t *testing.T) {
	const rev = "cd03eee90fbec18297ac31b8c21546e596b7f71c"
	ok := []struct{ in, repo, revision string }{
		{"nvidia/diar_streaming_sortformer_4spk-v2.1", "nvidia/diar_streaming_sortformer_4spk-v2.1", ""},
		{"  nvidia/diar_streaming_sortformer_4spk-v2.1  ", "nvidia/diar_streaming_sortformer_4spk-v2.1", ""},
		{"https://huggingface.co/nvidia/diar_streaming_sortformer_4spk-v2.1", "nvidia/diar_streaming_sortformer_4spk-v2.1", ""},
		{"https://huggingface.co/nvidia/diar_streaming_sortformer_4spk-v2.1/", "nvidia/diar_streaming_sortformer_4spk-v2.1", ""},
		{"huggingface.co/pyannote/segmentation-3.0", "pyannote/segmentation-3.0", ""},
		{"https://huggingface.co/nvidia/x/tree/" + rev, "nvidia/x", rev},
		{"https://huggingface.co/nvidia/x/resolve/" + rev + "/model.nemo", "nvidia/x", rev},
		{"https://huggingface.co/nvidia/x/tree/main", "nvidia/x", "main"},
	}
	for _, c := range ok {
		repo, revision, err := parseHFSource(c.in, "huggingface.co", "https")
		if err != nil || repo != c.repo || revision != c.revision {
			t.Errorf("%q: got %q %q %v", c.in, repo, revision, err)
		}
	}
	bad := []string{
		"", "gpt2", "a/b/c", "../etc/passwd", "owner/..", "owner/na me", "owner/name?x=1",
		"http://huggingface.co/owner/name",
		"https://evil.example/owner/name",
		"https://huggingface.co.evil.example/owner/name",
		"https://evilhuggingface.co/owner/name",
		"https://huggingface.co@evil.example/owner/name",
		"https://user:pw@huggingface.co/owner/name",
		"https://huggingface.co:8443/owner/name",
		"https://127.0.0.1/owner/name",
		"https://[::1]/owner/name",
		"file:///etc/passwd",
		"https://huggingface.co/owner/name/tree/..%2f..",
		"ftp://huggingface.co/owner/name",
	}
	for _, in := range bad {
		if repo, _, err := parseHFSource(in, "huggingface.co", "https"); err == nil {
			t.Errorf("%q must be refused, got %q", in, repo)
		}
	}
}
