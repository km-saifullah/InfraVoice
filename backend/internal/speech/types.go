package speech

import (
	"context"
	"io"
)

// TranscribeRequest carries an audio upload through to a Provider.
// Reader is the raw audio bytes; the caller is responsible for
// closing the underlying source (e.g. the multipart file) once
// Transcribe returns.
type TranscribeRequest struct {
	Reader      io.Reader
	Filename    string
	ContentType string
	SizeBytes   int64

	// Language optionally overrides the provider's configured
	// default language (BCP-47, e.g. "en"). Leave empty to let the
	// provider auto-detect or fall back to its own default.
	Language string
}

type TranscribeResult struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

// Provider is implemented by a concrete speech-to-text backend
// (e.g. a local Whisper server). It mirrors the shape of
// ai.Provider so both AI integrations follow the same pattern.
type Provider interface {
	Name() string

	Transcribe(
		ctx context.Context,
		request TranscribeRequest,
	) (TranscribeResult, error)
}
