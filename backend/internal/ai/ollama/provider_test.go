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

func TestProviderRepairsMissingRequestedResource(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				var request generateRequest

				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatalf("failed to decode request: %v", err)
				}

				if requestCount == 1 {
					if !strings.Contains(
						request.Prompt,
						"Never silently omit a requested resource type",
					) {
						t.Fatal(
							"expected resource preservation rule in initial prompt",
						)
					}

					response := generateResponse{
						Response: `{"needs_clarification":false,"specification":{"provider":"aws","version":1,"vpcs":[{"name":"main-vpc","cidr":"10.0.0.0/16","subnets":[{"name":"public-subnet","cidr":"10.0.1.0/24","type":"public","availability_zone":"ap-south-1a"}]}]}}`,
					}

					w.Header().Set(
						"Content-Type",
						"application/json",
					)

					_ = json.NewEncoder(
						w,
					).Encode(response)

					return
				}

				if !strings.Contains(
					request.Prompt,
					"Missing requested resource types:\ns3",
				) {
					t.Fatalf(
						"expected repair prompt to identify missing S3, got %q",
						request.Prompt,
					)
				}

				response := generateResponse{
					Response: `{"needs_clarification":false,"specification":{"provider":"aws","version":1,"vpcs":[{"name":"main-vpc","cidr":"10.0.0.0/16","subnets":[{"name":"public-subnet","cidr":"10.0.1.0/24","type":"public","availability_zone":"ap-south-1a"}]}],"s3":[{"name":"infravoice-demo-bucket","bucket_name":"infravoice-demo-bucket","versioning":true,"encryption":true}]}}`,
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
			Command: "Create an AWS VPC named main-vpc with CIDR 10.0.0.0/16 and one public subnet named public-subnet with CIDR 10.0.1.0/24 in ap-south-1a. Also create an S3 bucket named infravoice-demo-bucket with versioning and encryption enabled.",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if requestCount != 2 {
		t.Fatalf(
			"expected one repair request, got %d provider requests",
			requestCount,
		)
	}

	if result.Specification == nil {
		t.Fatal("expected specification")
	}

	if len(result.Specification.VPCs) != 1 {
		t.Fatalf(
			"expected one VPC, got %d",
			len(result.Specification.VPCs),
		)
	}

	if len(result.Specification.S3) != 1 {
		t.Fatalf(
			"expected one S3 resource, got %d",
			len(result.Specification.S3),
		)
	}

	bucket := result.Specification.S3[0]

	if bucket.BucketName != "infravoice-demo-bucket" {
		t.Fatalf(
			"unexpected bucket name: %q",
			bucket.BucketName,
		)
	}

	if !bucket.Versioning {
		t.Fatal(
			"expected S3 versioning to be enabled",
		)
	}

	if !bucket.Encryption {
		t.Fatal(
			"expected S3 encryption to be enabled",
		)
	}
}

func TestProviderRejectsResponseThatStillOmitsRequestedResource(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				response := generateResponse{
					Response: `{"needs_clarification":false,"specification":{"provider":"aws","version":1,"vpcs":[{"name":"main-vpc","cidr":"10.0.0.0/16"}]}}`,
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
			Command: "Create a VPC and an S3 bucket.",
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

	if !strings.Contains(
		err.Error(),
		"s3",
	) {
		t.Fatalf(
			"expected error to identify missing S3 resource, got %v",
			err,
		)
	}
}
