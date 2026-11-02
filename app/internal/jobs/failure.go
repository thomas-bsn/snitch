package jobs

import "regexp"

// failureActions is the closed catalogue of failure causes (contract FailureCode) with the recovery
// action advised to an Administrator. Only the server decides the action; the Worker sends a code.
var failureActions = map[string]string{
	"worker_failed":                "check_worker_logs",
	"provider_unavailable":         "check_transcription_service",
	"invalid_input":                "check_clip_media",
	"audio_preparation_failed":     "check_clip_media",
	"lease_expired":                "check_worker_connection",
	"max_attempts_exhausted":       "relaunch_job",
	"cancelled_by_admin":           "relaunch_job",
	"input_version_superseded":     "none",
	"worker_revoked":               "check_worker_connection",
	"whisper_network":              "check_transcription_service",
	"whisper_timeout":              "check_transcription_service",
	"whisper_http_error":           "check_transcription_service",
	"whisper_model_missing":        "install_transcription_model",
	"whisper_invalid_response":     "check_transcription_service",
	"pyannote_model_missing":       "install_voice_model",
	"pyannote_access_denied":       "check_huggingface_token",
	"gpu_unavailable":              "check_gpu",
	"gpu_out_of_memory":            "check_gpu",
	"diarization_inference_failed": "check_worker_logs",
	"hf_auth_required":             "check_huggingface_token",
	"hf_not_found":                 "check_voice_model_repository",
	"hf_network":                   "check_worker_network",
	"models_volume_unwritable":     "check_models_volume",
	"hf_cache_error":               "check_models_volume",
}

func knownFailureCode(code string) bool {
	_, ok := failureActions[code]
	return ok
}

// RecommendedAction returns the advised recovery action code, or "" for an empty or unknown cause.
func RecommendedAction(code string) string { return failureActions[code] }

// FailureCodes lists the catalogue, for the consistency test with the contract.
func FailureCodes() []string {
	codes := make([]string, 0, len(failureActions))
	for code := range failureActions {
		codes = append(codes, code)
	}
	return codes
}

var (
	providerPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	modelPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
)

// Diagnostic is the optional, typed and bounded detail a Worker may attach to a failure.
// An older Worker sends none of it.
type Diagnostic struct {
	Provider   *string
	Model      *string
	Revision   *string
	HTTPStatus *int
	DurationMS *int
}

func (d Diagnostic) valid() bool {
	if d.Provider != nil && !providerPattern.MatchString(*d.Provider) {
		return false
	}
	for _, v := range []*string{d.Model, d.Revision} {
		if v != nil && !modelPattern.MatchString(*v) {
			return false
		}
	}
	if d.HTTPStatus != nil && (*d.HTTPStatus < 100 || *d.HTTPStatus > 599) {
		return false
	}
	return d.DurationMS == nil || (*d.DurationMS >= 0 && *d.DurationMS <= 86400000)
}
