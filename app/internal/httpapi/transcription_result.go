package httpapi

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

type transcriptionResult struct {
	Provider         string                  `json:"provider"`
	Model            string                  `json:"model"`
	Version          string                  `json:"version"`
	AudioStreamIndex *int                    `json:"audio_stream_index"`
	Segments         *[]transcriptionSegment `json:"segments"`
}

type transcriptionSegment struct {
	Start *float64             `json:"start"`
	End   *float64             `json:"end"`
	Text  string               `json:"text"`
	Words *[]transcriptionWord `json:"words"`
}

type transcriptionWord struct {
	Word  string   `json:"word"`
	Start *float64 `json:"start"`
	End   *float64 `json:"end"`
}

func validTranscriptionResult(raw json.RawMessage, step string) bool {
	var result transcriptionResult
	if json.Unmarshal(raw, &result) != nil || result.Provider == "" || result.Model == "" || result.Version == "" || result.AudioStreamIndex == nil || result.Segments == nil {
		return false
	}
	selected, err := strconv.Atoi(strings.TrimPrefix(step, "transcription.audio."))
	if err != nil || !strings.HasPrefix(step, "transcription.audio.") || selected != *result.AudioStreamIndex {
		return false
	}
	for _, segment := range *result.Segments {
		if segment.Start == nil || segment.End == nil || !validTime(*segment.Start) || !validTime(*segment.End) || *segment.End < *segment.Start || strings.TrimSpace(segment.Text) == "" || segment.Words == nil {
			return false
		}
		for _, word := range *segment.Words {
			if word.Start == nil || word.End == nil || !validTime(*word.Start) || !validTime(*word.End) || *word.End < *word.Start || strings.TrimSpace(word.Word) == "" || *word.Start < *segment.Start-0.05 || *word.End > *segment.End+0.05 {
				return false
			}
		}
	}
	return true
}

func validTime(seconds float64) bool {
	return seconds >= 0 && !math.IsNaN(seconds) && !math.IsInf(seconds, 0)
}
