package httpapi

import "testing"

func TestValidEngineVersion(t *testing.T) {
	commit := "cf724ac337d1ebc7d0dda1e23fb80916f52927a5"
	nemo, speechbrain := engineFamilies["nemo"], engineFamilies["speechbrain"]
	if nemo.GitURL != "https://github.com/NVIDIA/NeMo.git" || speechbrain.GitURL != "" {
		t.Fatalf("only the NeMo family names a Git repository, got %q and %q", nemo.GitURL, speechbrain.GitURL)
	}
	cases := []struct {
		name    string
		family  engineFamily
		version string
		want    bool
	}{
		{"published version", nemo, "3.0.0", true},
		{"pinned commit of a family that allows it", nemo, "git-" + commit, true},
		{"pinned commit of a family that does not", speechbrain, "git-" + commit, false},
		{"short commit", nemo, "git-cf724ac", false},
		{"branch name", nemo, "git-main", false},
		{"upper case commit", nemo, "git-CF724AC337D1EBC7D0DDA1E23FB80916F52927A5", false},
		{"commit with a suffix", nemo, "git-" + commit + "0", false},
		{"url", nemo, "git+https://example.com/x.git@" + commit, false},
		{"empty", nemo, "", false},
	}
	for _, c := range cases {
		if got := validEngineVersion(c.family, c.version); got != c.want {
			t.Errorf("%s: validEngineVersion(%q) = %v, want %v", c.name, c.version, got, c.want)
		}
	}
}
