package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

func TestServiceParseInfrastructure(t *testing.T) {
	provider := NewMockProvider(
		NewMockVPCResult(),
	)

	service := NewService(provider)

	result, err := service.ParseInfrastructure(
		context.Background(),
		ParseRequest{
			Command: "Create a VPC with a public subnet.",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected successful parsing, got: %v",
			err,
		)
	}

	if result.Specification == nil {
		t.Fatal("expected infrastructure specification")
	}

	if result.Specification.Provider !=
		infrastructure.ProviderAWS {
		t.Fatalf(
			"expected AWS provider, got: %s",
			result.Specification.Provider,
		)
	}

	if len(result.Specification.VPCs) != 1 {
		t.Fatalf(
			"expected one VPC, got: %d",
			len(result.Specification.VPCs),
		)
	}
}

func TestServiceRejectsEmptyCommand(t *testing.T) {
	service := NewService(
		NewMockProvider(
			NewMockVPCResult(),
		),
	)

	_, err := service.ParseInfrastructure(
		context.Background(),
		ParseRequest{
			Command: "   ",
		},
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got: %v",
			err,
		)
	}
}

func TestServiceRejectsInvalidProviderOutput(t *testing.T) {
	provider := NewMockProvider(
		ParseResult{
			Specification: &infrastructure.InfrastructureSpec{
				Provider: infrastructure.ProviderAWS,
				Version:  1,
			},
		},
	)

	service := NewService(provider)

	_, err := service.ParseInfrastructure(
		context.Background(),
		ParseRequest{
			Command: "Create infrastructure.",
		},
	)

	if err == nil {
		t.Fatal("expected provider output validation error")
	}

	if !errors.Is(
		err,
		ErrInvalidProviderResponse,
	) {
		t.Fatalf(
			"expected ErrInvalidProviderResponse, got: %v",
			err,
		)
	}
}

func TestServiceSupportsClarification(t *testing.T) {
	provider := NewMockProvider(
		ParseResult{
			NeedsClarification: true,
			Clarification:      "Which AWS region should be used?",
		},
	)

	service := NewService(provider)

	result, err := service.ParseInfrastructure(
		context.Background(),
		ParseRequest{
			Command: "Create a VPC.",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected clarification result, got: %v",
			err,
		)
	}

	if !result.NeedsClarification {
		t.Fatal("expected clarification to be required")
	}

	if result.Specification != nil {
		t.Fatal(
			"expected specification to be nil when clarification is required",
		)
	}

	if !strings.Contains(
		result.Clarification,
		"region",
	) {
		t.Fatalf(
			"expected region clarification, got: %s",
			result.Clarification,
		)
	}
}
