package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

type MockProvider struct {
	Result ParseResult
	Err    error
}

func NewMockProvider(
	result ParseResult,
) *MockProvider {
	return &MockProvider{
		Result: result,
	}
}

func (p *MockProvider) Name() string {
	return "mock"
}

func (p *MockProvider) ParseInfrastructure(
	ctx context.Context,
	request ParseRequest,
) (ParseResult, error) {
	if err := ctx.Err(); err != nil {
		return ParseResult{}, err
	}

	if strings.TrimSpace(request.Command) == "" {
		return ParseResult{}, fmt.Errorf(
			"%w: command cannot be empty",
			ErrInvalidInput,
		)
	}

	if p.Err != nil {
		return ParseResult{}, p.Err
	}

	return p.Result, nil
}

func NewMockVPCResult() ParseResult {
	return ParseResult{
		Specification: &infrastructure.InfrastructureSpec{
			Provider: infrastructure.ProviderAWS,
			Version:  1,
			VPCs: []infrastructure.VPCSpec{
				{
					Name:               "main-vpc",
					CIDR:               "10.0.0.0/16",
					EnableDNS:          true,
					EnableDNSHostnames: true,
					Subnets: []infrastructure.Subnet{
						{
							Name:             "public-subnet",
							CIDR:             "10.0.1.0/24",
							Type:             "public",
							AvailabilityZone: "ap-south-1a",
						},
					},
				},
			},
		},
	}
}
