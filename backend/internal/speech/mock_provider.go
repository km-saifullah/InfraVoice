package speech

import "context"

type MockProvider struct {
	TranscribeResult TranscribeResult
	TranscribeError  error
}

func (p *MockProvider) Name() string {
	return "mock"
}

func (p *MockProvider) Transcribe(
	ctx context.Context,
	request TranscribeRequest,
) (TranscribeResult, error) {
	if p.TranscribeError != nil {
		return TranscribeResult{}, p.TranscribeError
	}

	return p.TranscribeResult, nil
}
