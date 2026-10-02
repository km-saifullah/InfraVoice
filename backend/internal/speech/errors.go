package speech

import "errors"

var (
	ErrInvalidInput = errors.New(
		"invalid speech input",
	)

	ErrProviderUnavailable = errors.New(
		"speech provider unavailable",
	)

	ErrProviderNotConfigured = errors.New(
		"speech provider not configured",
	)

	ErrInvalidProviderResponse = errors.New(
		"invalid speech provider response",
	)

	ErrUnsupportedFormat = errors.New(
		"unsupported audio format",
	)

	ErrFileTooLarge = errors.New(
		"audio file exceeds the maximum allowed size",
	)
)
