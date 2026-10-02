package whisper

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/km-saifullah/infra-voice/backend/internal/speech"
)

func TestProviderSendsAudioAndParsesTranscript(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf(
						"expected POST, got %s",
						r.Method,
					)
				}

				if r.URL.Path != "/v1/audio/transcriptions" {
					t.Fatalf(
						"expected /v1/audio/transcriptions, got %s",
						r.URL.Path,
					)
				}

				if err := r.ParseMultipartForm(
					10 << 20,
				); err != nil {
					t.Fatalf(
						"failed to parse multipart form: %v",
						err,
					)
				}

				file, header, err := r.FormFile("file")

				if err != nil {
					t.Fatalf(
						"expected a file field: %v",
						err,
					)
				}

				defer file.Close()

				if header.Filename != "command.webm" {
					t.Fatalf(
						"expected command.webm, got %s",
						header.Filename,
					)
				}

				content, err := io.ReadAll(file)

				if err != nil {
					t.Fatalf(
						"failed to read uploaded audio: %v",
						err,
					)
				}

				if string(content) != "fake-audio-bytes" {
					t.Fatalf(
						"unexpected audio content: %q",
						content,
					)
				}

				if got := r.FormValue("model"); got != "base.en" {
					t.Fatalf(
						"expected model base.en, got %q",
						got,
					)
				}

				if got := r.FormValue("language"); got != "en" {
					t.Fatalf(
						"expected language en, got %q",
						got,
					)
				}

				if got := r.FormValue("response_format"); got != "json" {
					t.Fatalf(
						"expected response_format json, got %q",
						got,
					)
				}

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				_ = json.NewEncoder(w).Encode(
					transcriptionResponse{
						Text:     "create a t3 micro ec2 instance",
						Language: "en",
					},
				)
			},
		),
	)

	defer server.Close()

	provider := NewProvider(
		server.URL,
		"base.en",
		"en",
		nil,
	)

	if provider.Name() != "whisper" {
		t.Fatalf(
			"expected provider name whisper, got %s",
			provider.Name(),
		)
	}

	result, err := provider.Transcribe(
		context.Background(),
		speech.TranscribeRequest{
			Reader:   strings.NewReader("fake-audio-bytes"),
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
			"unexpected transcript: %q",
			result.Text,
		)
	}

	if result.Language != "en" {
		t.Fatalf(
			"unexpected language: %q",
			result.Language,
		)
	}
}

func TestProviderRequestLanguageOverridesConfiguredDefault(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_ = r.ParseMultipartForm(10 << 20)

				if got := r.FormValue("language"); got != "bn" {
					t.Fatalf(
						"expected the per-request language to win, got %q",
						got,
					)
				}

				_ = json.NewEncoder(w).Encode(
					transcriptionResponse{Text: "ok"},
				)
			},
		),
	)

	defer server.Close()

	provider := NewProvider(server.URL, "", "en", nil)

	if _, err := provider.Transcribe(
		context.Background(),
		speech.TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.wav",
			Language: "bn",
		},
	); err != nil {
		t.Fatalf(
			"Transcribe() returned unexpected error: %v",
			err,
		)
	}
}

func TestProviderReturnsProviderUnavailableOnHTTPError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)

				_, _ = w.Write(
					[]byte(`{"error":{"message":"model failed to load"}}`),
				)
			},
		),
	)

	defer server.Close()

	provider := NewProvider(server.URL, "", "", nil)

	_, err := provider.Transcribe(
		context.Background(),
		speech.TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.wav",
		},
	)

	if !errors.Is(err, speech.ErrProviderUnavailable) {
		t.Fatalf(
			"expected ErrProviderUnavailable, got %v",
			err,
		)
	}

	if !strings.Contains(err.Error(), "model failed to load") {
		t.Fatalf(
			"expected the upstream error message to be included, got %v",
			err,
		)
	}
}

func TestProviderReturnsProviderUnavailableOnPlainStringError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)

				_, _ = w.Write(
					[]byte(`{"error":"unsupported audio codec"}`),
				)
			},
		),
	)

	defer server.Close()

	provider := NewProvider(server.URL, "", "", nil)

	_, err := provider.Transcribe(
		context.Background(),
		speech.TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.wav",
		},
	)

	if !strings.Contains(err.Error(), "unsupported audio codec") {
		t.Fatalf(
			"expected the plain-string error to be extracted, got %v",
			err,
		)
	}
}

func TestProviderReturnsInvalidProviderResponseOnMalformedJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)

				_, _ = w.Write([]byte("not json"))
			},
		),
	)

	defer server.Close()

	provider := NewProvider(server.URL, "", "", nil)

	_, err := provider.Transcribe(
		context.Background(),
		speech.TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.wav",
		},
	)

	if !errors.Is(err, speech.ErrInvalidProviderResponse) {
		t.Fatalf(
			"expected ErrInvalidProviderResponse, got %v",
			err,
		)
	}
}

func TestProviderRequiresBaseURL(t *testing.T) {
	provider := NewProvider("", "", "", nil)

	_, err := provider.Transcribe(
		context.Background(),
		speech.TranscribeRequest{
			Reader:   strings.NewReader("audio"),
			Filename: "command.wav",
		},
	)

	if !errors.Is(err, speech.ErrProviderNotConfigured) {
		t.Fatalf(
			"expected ErrProviderNotConfigured, got %v",
			err,
		)
	}
}

type erroringReader struct{}

func (erroringReader) Read([]byte) (int, error) {
	return 0, errors.New("simulated read failure")
}

func TestProviderSurfacesReaderFailures(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				// The provider should fail before a valid response
				// is needed; if this handler is reached at all, still
				// respond so the test does not hang.
				_ = json.NewEncoder(w).Encode(
					transcriptionResponse{Text: "unused"},
				)
			},
		),
	)

	defer server.Close()

	provider := NewProvider(server.URL, "", "", nil)

	_, err := provider.Transcribe(
		context.Background(),
		speech.TranscribeRequest{
			Reader:   erroringReader{},
			Filename: "command.wav",
		},
	)

	if !errors.Is(err, speech.ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestProviderSendsBearerAPIKey(t *testing.T) {
	const expectedAPIKey = "test-whisper-key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+expectedAPIKey {
			t.Errorf("expected Bearer authorization header, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(transcriptionResponse{Text: "authenticated"})
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "", "", nil, expectedAPIKey)
	result, err := provider.Transcribe(context.Background(), speech.TranscribeRequest{
		Reader: strings.NewReader("audio"), Filename: "command.wav",
	})
	if err != nil {
		t.Fatalf("Transcribe() returned unexpected error: %v", err)
	}
	if result.Text != "authenticated" {
		t.Fatalf("expected authenticated transcript, got %q", result.Text)
	}
}
