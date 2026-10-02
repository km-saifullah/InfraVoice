package speech

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestServiceRequiresProvider(t *testing.T) {
	service := NewService(nil, 0)

	_, err := service.Transcribe(
		context.Background(),
		TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.wav",
		},
	)

	if !errors.Is(err, ErrProviderNotConfigured) {
		t.Fatalf(
			"expected ErrProviderNotConfigured, got %v",
			err,
		)
	}
}

func TestServiceRejectsMissingReader(t *testing.T) {
	service := NewService(&MockProvider{}, 0)

	_, err := service.Transcribe(
		context.Background(),
		TranscribeRequest{
			Filename: "command.wav",
		},
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestServiceRejectsMissingFilename(t *testing.T) {
	service := NewService(&MockProvider{}, 0)

	_, err := service.Transcribe(
		context.Background(),
		TranscribeRequest{
			Reader: strings.NewReader("audio"),
		},
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestServiceRejectsUnsupportedExtension(t *testing.T) {
	service := NewService(&MockProvider{}, 0)

	_, err := service.Transcribe(
		context.Background(),
		TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.exe",
		},
	)

	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf(
			"expected ErrUnsupportedFormat, got %v",
			err,
		)
	}
}

func TestServiceRejectsOversizedFile(t *testing.T) {
	service := NewService(&MockProvider{}, 1024)

	_, err := service.Transcribe(
		context.Background(),
		TranscribeRequest{
			Reader:    strings.NewReader("audio"),
			Filename:  "command.wav",
			SizeBytes: 2048,
		},
	)

	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf(
			"expected ErrFileTooLarge, got %v",
			err,
		)
	}
}

func TestServiceRejectsEmptyTranscript(t *testing.T) {
	service := NewService(
		&MockProvider{
			TranscribeResult: TranscribeResult{
				Text: "   ",
			},
		},
		0,
	)

	_, err := service.Transcribe(
		context.Background(),
		TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.wav",
		},
	)

	if !errors.Is(err, ErrInvalidProviderResponse) {
		t.Fatalf(
			"expected ErrInvalidProviderResponse, got %v",
			err,
		)
	}
}

func TestServicePropagatesProviderError(t *testing.T) {
	sentinel := errors.New("boom")

	service := NewService(
		&MockProvider{
			TranscribeError: sentinel,
		},
		0,
	)

	_, err := service.Transcribe(
		context.Background(),
		TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.wav",
		},
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf(
			"expected the provider error to propagate, got %v",
			err,
		)
	}
}

func TestServiceAcceptsValidAudioAndTrimsText(t *testing.T) {
	service := NewService(
		&MockProvider{
			TranscribeResult: TranscribeResult{
				Text: "  create a t3 micro ec2 instance  ",
			},
		},
		0,
	)

	result, err := service.Transcribe(
		context.Background(),
		TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.webm",
		},
	)

	if err != nil {
		t.Fatalf(
			"Transcribe() returned unexpected error: %v",
			err,
		)
	}

	if result.Text != "create a t3 micro ec2 instance" {
		t.Fatalf(
			"expected trimmed text, got %q",
			result.Text,
		)
	}
}

func TestNewServiceFallsBackToDefaultMaxUploadBytes(t *testing.T) {
	service := NewService(&MockProvider{}, -5)

	if service.MaxUploadBytes() != DefaultMaxUploadBytes {
		t.Fatalf(
			"expected default max upload bytes, got %d",
			service.MaxUploadBytes(),
		)
	}
}

func TestIsSupportedExtensionIsCaseInsensitive(t *testing.T) {
	testCases := map[string]bool{
		"voice.WAV":    true,
		"voice.Mp3":    true,
		"voice.webm":   true,
		"voice.OGG":    true,
		"voice.flac":   true,
		"voice.txt":    false,
		"voice":        false,
		"voice.tar.gz": false,
	}

	for filename, expected := range testCases {
		if actual := IsSupportedExtension(filename); actual != expected {
			t.Errorf(
				"IsSupportedExtension(%q) = %v, want %v",
				filename,
				actual,
				expected,
			)
		}
	}
}
