package ai

import "errors"

var (
	ErrInvalidInput = errors.New(
		"invalid AI input",
	)

	ErrProviderUnavailable = errors.New(
		"AI provider unavailable",
	)

	ErrProviderNotConfigured = errors.New(
		"AI provider not configured",
	)

	ErrInvalidProviderResponse = errors.New(
		"invalid AI provider response",
	)
)
