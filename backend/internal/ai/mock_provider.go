package ai

import "context"

type MockProvider struct {
	ParseResult ParseResult
	ParseError  error
}

func (p *MockProvider) Name() string {
	return "mock"
}

func (p *MockProvider) ParseInfrastructure(
	ctx context.Context,
	request ParseRequest,
) (ParseResult, error) {
	if p.ParseError != nil {
		return ParseResult{}, p.ParseError
	}

	return p.ParseResult, nil
}
