package httpapi

// voiceCatalogEntry is one model Snitch knows how to run. The catalogue is data fixed in the code:
// no search, no free repository, because an unknown repository is not executable and loading it
// would run arbitrary code (pickle). Each entry pins a revision.
type voiceCatalogEntry struct {
	ID                  string
	Kind                string // diarization or fingerprint
	Engine              string
	Repo                string
	Revision            string
	License             string
	Gated               bool
	MaxSpeakers         int // 0: not limited by the model
	SizeBytes           int64
	SupportsNumSpeakers bool
}

// voiceCatalog is a variable only so a test can add a synthetic second model; it is never written at runtime.
var voiceCatalog = []voiceCatalogEntry{
	{
		ID: "pyannote-community-1", Kind: "diarization", Engine: "pyannote",
		Repo: "pyannote/speaker-diarization-community-1", Revision: "3533c8cf8e369892e6b79ff1bf80f7b0286a54ee",
		License: "cc-by-4.0", Gated: true, SizeBytes: 33695573, SupportsNumSpeakers: true,
	},
	{
		// Sortformer streaming: not gated (NVIDIA Open Model License), 4 speakers at most, no speaker count hint.
		ID: "nvidia-sortformer-4spk-v2.1", Kind: "diarization", Engine: "nemo",
		Repo: "nvidia/diar_streaming_sortformer_4spk-v2.1", Revision: "cd03eee90fbec18297ac31b8c21546e596b7f71c",
		License: "nvidia-open-model-license", MaxSpeakers: 4, SizeBytes: 471403401,
	},
	{
		ID: "pyannote-wespeaker-resnet34", Kind: "fingerprint", Engine: "pyannote",
		Repo: "pyannote/wespeaker-voxceleb-resnet34-LM", Revision: "837717ddb9ff5507820346191109dc79c958d614",
		License: "cc-by-4.0", SizeBytes: 26650477,
	},
}

// voiceRealProvider tells whether a pipeline provider runs a catalogue model (the fake one runs none).
// The provider of a diarization model is its engine.
func voiceRealProvider(provider string) bool {
	return provider == "pyannote" || provider == "nemo"
}

func voiceCatalogByID(id string) (voiceCatalogEntry, bool) {
	for _, entry := range voiceCatalog {
		if entry.ID == id {
			return entry, true
		}
	}
	return voiceCatalogEntry{}, false
}

func voiceCatalogByRepo(kind, repo string) (voiceCatalogEntry, bool) {
	for _, entry := range voiceCatalog {
		if entry.Kind == kind && entry.Repo == repo {
			return entry, true
		}
	}
	return voiceCatalogEntry{}, false
}

// voiceFailureCauses is the closed vocabulary a Worker may report for a failed preparation.
var voiceFailureCauses = map[string]bool{
	"hf_token_missing": true, "hf_access_denied": true, "disk_full": true, "runtime_incompatible": true, "inference_failed": true,
}
