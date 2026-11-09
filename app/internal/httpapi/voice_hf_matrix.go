package httpapi

import (
	"slices"
	"strings"
)

// The adapter matrix (ADR 0038). The engine of a reference is never chosen by a person: Snitch derives it from
// the public metadata of the repository, among the adapters that have an implementation and a test.
//
// Approved adapters, in the order that decides when several fit the same repository (first match wins, so the
// answer is deterministic and does not depend on tag order or on the user):
//
//  1. nemo-titanet / speechbrain-ecapa - fingerprint, with explicit family tags.
//  2. nemo - diarization, for repositories declaring NeMo or a *.nemo artifact.
//  3. pyannote - diarization and fingerprint, with a pyannote library/tag.
//
// Metadata proposes an adapter; only its successful local inference proves readiness.
//
// Transformers is in the target matrix but has no Worker adapter yet: a repository that only fits it stays
// unsupported until an adapter is implemented and tested. Separation has no adapter before #268.
// A repository that carries remote code (custom_code tag, auto_map) is never supported: it would execute
// code of the repository.
type hfAdapter struct {
	engine string
	kinds  []string
	fits   func(m hfMetadata) bool
}

var hfAdapters = []hfAdapter{
	{"nemo-titanet", []string{"fingerprint"}, func(m hfMetadata) bool {
		return (m.LibraryName == "nemo" || slices.Contains(m.Tags, "nemo")) && slices.Contains(m.Tags, "titanet")
	}},
	{"speechbrain-ecapa", []string{"fingerprint"}, func(m hfMetadata) bool {
		return (m.LibraryName == "speechbrain" || slices.Contains(m.Tags, "speechbrain")) && slices.Contains(m.Tags, "ecapa-tdnn")
	}},
	{"nemo", []string{"diarization"}, func(m hfMetadata) bool {
		if m.LibraryName == "nemo" || slices.Contains(m.Tags, "nemo") {
			return true
		}
		for _, f := range m.Siblings {
			if strings.HasSuffix(strings.ToLower(f.Name), ".nemo") {
				return true
			}
		}
		return false
	}},
	{"pyannote", []string{"diarization", "fingerprint"}, func(m hfMetadata) bool {
		return m.LibraryName == "pyannote-audio" || slices.Contains(m.Tags, "pyannote-audio") || slices.Contains(m.Tags, "pyannote")
	}},
	// After pyannote: a pyannote wrapper of a WeSpeaker network stays on the embedded engine. A WeSpeaker repository
	// of its own runs on the on-demand WeSpeaker family (#375, ADR 0056), which has no signed release.
	{"wespeaker", []string{"fingerprint"}, func(m hfMetadata) bool {
		return m.LibraryName == "wespeaker" || slices.Contains(m.Tags, "wespeaker")
	}},
}

// Tags that declare a type. A type is proposed only when exactly one family is declared: the repository name
// and the pipeline tag are not evidence (a name is free text, an ASR pipeline tag says nothing here).
var hfKindTags = []struct {
	kind string
	tags []string
}{
	{"diarization", []string{"speaker-diarization", "diarization"}},
	{"fingerprint", []string{"speaker-embedding", "speaker-verification"}},
	{"separation", []string{"audio-source-separation", "speech-separation", "source-separation"}},
}

type hfAssessment struct {
	Engine            string   // approved adapter, empty when none
	CompatibleKinds   []string // types that adapter can run
	ProposedKind      string   // empty when the metadata do not settle the type
	UnsupportedReason string   // empty when supported
}

func assessHF(m hfMetadata) hfAssessment {
	var a hfAssessment
	var declared []string
	for _, k := range hfKindTags {
		if slices.ContainsFunc(k.tags, func(t string) bool { return slices.Contains(m.Tags, t) }) {
			declared = append(declared, k.kind)
		}
	}
	if len(declared) == 1 {
		a.ProposedKind = declared[0]
	}
	switch auto := strings.TrimSpace(string(m.Config.AutoMap)); {
	case slices.Contains(m.Tags, "custom_code") || (auto != "" && auto != "null" && auto != "{}"):
		a.UnsupportedReason = "the repository requires remote code, which Snitch never runs"
		return a
	}
	for _, ad := range hfAdapters {
		if ad.fits(m) {
			a.Engine, a.CompatibleKinds = ad.engine, ad.kinds
			break
		}
	}
	switch {
	case a.Engine == "":
		a.UnsupportedReason = "no approved adapter can load this repository"
	case a.ProposedKind == "separation":
		a.CompatibleKinds = nil
		a.UnsupportedReason = "no approved adapter runs separation yet"
	}
	return a
}

// supports tells whether the reference can be installed as the given type, with the reason when not.
func (a hfAssessment) supports(kind string) (bool, string) {
	switch {
	case a.UnsupportedReason != "":
		return false, a.UnsupportedReason
	case kind == "separation":
		return false, "no approved adapter runs separation yet"
	case !slices.Contains(a.CompatibleKinds, kind):
		return false, "the " + a.Engine + " adapter cannot run a " + kind + " model"
	}
	return true, ""
}
