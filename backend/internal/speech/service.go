package speech

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// DefaultMaxUploadBytes is used when a Service is built without an
// explicit limit.
const DefaultMaxUploadBytes int64 = 25 * 1024 * 1024

// allowedExtensions covers both traditional audio containers (wav,
// mp3, m4a) and, importantly, the formats produced by a browser's
// MediaRecorder API (webm, ogg) — the default output of the React
// frontend this backend is meant to serve.
var allowedExtensions = map[string]struct{}{
	".wav":  {},
	".wave": {},
	".mp3":  {},
	".mpeg": {},
	".mp4":  {},
	".m4a":  {},
	".aac":  {},
	".ogg":  {},
	".oga":  {},
	".webm": {},
	".flac": {},
}

type Service struct {
	provider       Provider
	maxUploadBytes int64
}

func NewService(
	provider Provider,
	maxUploadBytes int64,
) *Service {
	if maxUploadBytes <= 0 {
		maxUploadBytes = DefaultMaxUploadBytes
	}

	return &Service{
		provider:       provider,
		maxUploadBytes: maxUploadBytes,
	}
}

// MaxUploadBytes reports the configured upload limit. Callers at the
// HTTP boundary use this to reject oversized bodies before reading
// them off the wire.
func (s *Service) MaxUploadBytes() int64 {
	if s == nil || s.maxUploadBytes <= 0 {
		return DefaultMaxUploadBytes
	}

	return s.maxUploadBytes
}

func (s *Service) Transcribe(
	ctx context.Context,
	request TranscribeRequest,
) (TranscribeResult, error) {
	if s == nil || s.provider == nil {
		return TranscribeResult{}, ErrProviderNotConfigured
	}

	if request.Reader == nil {
		return TranscribeResult{}, fmt.Errorf(
			"%w: audio file is required",
			ErrInvalidInput,
		)
	}

	filename := strings.TrimSpace(request.Filename)

	if filename == "" {
		return TranscribeResult{}, fmt.Errorf(
			"%w: audio filename is required",
			ErrInvalidInput,
		)
	}

	if !IsSupportedExtension(filename) {
		return TranscribeResult{}, fmt.Errorf(
			"%w: %s (supported: wav, mp3, mp4, m4a, aac, ogg, webm, flac)",
			ErrUnsupportedFormat,
			filepath.Ext(filename),
		)
	}

	if request.SizeBytes > 0 &&
		request.SizeBytes > s.MaxUploadBytes() {
		return TranscribeResult{}, fmt.Errorf(
			"%w: %d bytes exceeds the %d byte limit",
			ErrFileTooLarge,
			request.SizeBytes,
			s.MaxUploadBytes(),
		)
	}

	result, err := s.provider.Transcribe(
		ctx,
		request,
	)

	if err != nil {
		return TranscribeResult{}, err
	}

	result.Text = strings.TrimSpace(result.Text)

	if result.Text == "" {
		return TranscribeResult{}, fmt.Errorf(
			"%w: transcription produced no text (the audio may be silent or unintelligible)",
			ErrInvalidProviderResponse,
		)
	}

	return result, nil
}

// IsSupportedExtension reports whether filename's extension is one
// this service will forward to the speech provider.
func IsSupportedExtension(
	filename string,
) bool {
	extension := strings.ToLower(
		filepath.Ext(filename),
	)

	_, supported := allowedExtensions[extension]

	return supported
}
