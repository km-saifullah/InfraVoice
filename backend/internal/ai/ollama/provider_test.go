package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/km-saifullah/infra-voice/backend/internal/ai"
)

func TestProviderParsesInfrastructureResponse(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf(
						"expected POST, got %s",
						r.Method,
					)
				}

				if r.URL.Path != "/api/generate" {
					t.Fatalf(
						"expected /api/generate, got %s",
						r.URL.Path,
					)
				}

				var request generateRequest

				if err := json.NewDecoder(
					r.Body,
				).Decode(&request); err != nil {
					t.Fatalf(
						"failed to decode request: %v",
						err,
					)
				}

				if request.Model != "qwen2.5:7b" {
					t.Fatalf(
						"expected qwen2.5:7b, got %s",
						request.Model,
					)
				}

				if !strings.Contains(
					request.Prompt,
					"create an encrypted S3 bucket",
				) {
					t.Fatalf(
						"expected user command in prompt, got %q",
						request.Prompt,
					)
				}

				if request.Stream {
					t.Fatal(
						"expected stream=false",
					)
				}

				if request.Format != "json" {
					t.Fatalf(
						"expected json format, got %s",
						request.Format,
					)
				}

				response := generateResponse{
					Response: `{"needs_clarification":false,"specification":{"provider":"aws","version":1,"s3":[{"name":"logs","bucket_name":"infra-voice-logs","versioning":true,"encryption":true}]}}`,
				}

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				if err := json.NewEncoder(
					w,
				).Encode(response); err != nil {
					t.Fatalf(
						"failed to encode response: %v",
						err,
					)
				}
			},
		),
	)

	defer server.Close()

	provider := NewProvider(
		server.URL,
		"qwen2.5:7b",
		server.Client(),
	)

	result, err := provider.ParseInfrastructure(
		context.Background(),
		ai.ParseRequest{
			Command: "create an encrypted S3 bucket named infra-voice-logs",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result.Specification == nil {
		t.Fatal("expected specification")
	}

	if len(result.Specification.S3) != 1 {
		t.Fatalf(
			"expected one S3 resource, got %d",
			len(result.Specification.S3),
		)
	}

	if !result.Specification.S3[0].Encryption {
		t.Fatal(
			"expected S3 encryption to be enabled",
		)
	}
}

func TestProviderParsesClarification(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				response := generateResponse{
					Response: `{"needs_clarification":true,"clarification":"What CIDR should the VPC use?","specification":null}`,
				}

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				_ = json.NewEncoder(
					w,
				).Encode(response)
			},
		),
	)

	defer server.Close()

	provider := NewProvider(
		server.URL,
		"qwen2.5:7b",
		server.Client(),
	)

	result, err := provider.ParseInfrastructure(
		context.Background(),
		ai.ParseRequest{
			Command: "create a VPC",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if !result.NeedsClarification {
		t.Fatal(
			"expected clarification response",
		)
	}

	if result.Clarification != "What CIDR should the VPC use?" {
		t.Fatalf(
			"unexpected clarification: %q",
			result.Clarification,
		)
	}
}

func TestProviderMapsHTTPFailure(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(
					w,
					`{"error":"model not found"}`,
					http.StatusNotFound,
				)
			},
		),
	)

	defer server.Close()

	provider := NewProvider(
		server.URL,
		"qwen2.5:7b",
		server.Client(),
	)

	_, err := provider.ParseInfrastructure(
		context.Background(),
		ai.ParseRequest{
			Command: "create an S3 bucket",
		},
	)

	if !errors.Is(
		err,
		ai.ErrProviderUnavailable,
	) {
		t.Fatalf(
			"expected ErrProviderUnavailable, got %v",
			err,
		)
	}
}

func TestProviderRejectsInvalidModelJSON(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				response := generateResponse{
					Response: "not-json",
				}

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				_ = json.NewEncoder(
					w,
				).Encode(response)
			},
		),
	)

	defer server.Close()

	provider := NewProvider(
		server.URL,
		"qwen2.5:7b",
		server.Client(),
	)

	_, err := provider.ParseInfrastructure(
		context.Background(),
		ai.ParseRequest{
			Command: "create an S3 bucket",
		},
	)

	if !errors.Is(
		err,
		ai.ErrInvalidProviderResponse,
	) {
		t.Fatalf(
			"expected ErrInvalidProviderResponse, got %v",
			err,
		)
	}
}
