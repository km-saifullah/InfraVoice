package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

func TestServiceRejectsEmptyCommand(t *testing.T) {
	service := NewService(&MockProvider{})

	_, err := service.ParseInfrastructure(
		context.Background(),
		ParseRequest{
			Command: "   ",
		},
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestServiceRequiresProvider(t *testing.T) {
	service := NewService(nil)

	_, err := service.ParseInfrastructure(
		context.Background(),
		ParseRequest{
			Command: "create an encrypted S3 bucket",
		},
	)

	if !errors.Is(err, ErrProviderNotConfigured) {
		t.Fatalf(
			"expected ErrProviderNotConfigured, got %v",
			err,
		)
	}
}

func TestServiceAcceptsValidSpecification(t *testing.T) {
	provider := &MockProvider{
		ParseResult: ParseResult{
			Specification: &infrastructure.InfrastructureSpec{
				Provider: "AWS",
				Version:  1,
				S3: []infrastructure.S3Spec{
					{
						Name:       "logs",
						BucketName: "infra-voice-logs",
						Encryption: true,
					},
				},
			},
		},
	}

	service := NewService(provider)

	result, err := service.ParseInfrastructure(
		context.Background(),
		ParseRequest{
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
		t.Fatal("expected infrastructure specification")
	}

	if result.Specification.Provider != infrastructure.ProviderAWS {
		t.Fatalf(
			"expected provider %q, got %q",
			infrastructure.ProviderAWS,
			result.Specification.Provider,
		)
	}
}

func TestServiceReturnsClarification(t *testing.T) {
	provider := &MockProvider{
		ParseResult: ParseResult{
			NeedsClarification: true,
			Clarification:      "Which AWS resource should be created?",
		},
	}

	service := NewService(provider)

	result, err := service.ParseInfrastructure(
		context.Background(),
		ParseRequest{
			Command: "create something in AWS",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if !result.NeedsClarification {
		t.Fatal("expected clarification response")
	}

	if result.Specification != nil {
		t.Fatal(
			"expected specification to be nil when clarification is required",
		)
	}
}

func TestServiceRejectsInvalidProviderSpecification(t *testing.T) {
	provider := &MockProvider{
		ParseResult: ParseResult{
			Specification: &infrastructure.InfrastructureSpec{
				Provider: "aws",
				Version:  1,
				S3: []infrastructure.S3Spec{
					{
						Name:       "logs",
						Encryption: false,
					},
				},
			},
		},
	}

	service := NewService(provider)

	_, err := service.ParseInfrastructure(
		context.Background(),
		ParseRequest{
			Command: "create an S3 bucket",
		},
	)

	if !errors.Is(err, ErrInvalidProviderResponse) {
		t.Fatalf(
			"expected ErrInvalidProviderResponse, got %v",
			err,
		)
	}
}
