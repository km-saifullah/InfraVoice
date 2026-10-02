package whisper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/km-saifullah/infra-voice/backend/internal/speech"
)

type Provider struct {
	baseURL    string
	model      string
	language   string
	apiKey     string
	httpClient *http.Client
}

func NewProvider(
	baseURL string,
	model string,
	language string,
	httpClient *http.Client,
	apiKeys ...string,
) *Provider {
	if httpClient == nil {
		httpClient = &http.Client{
			// Transcription of a longer voice command can take a
			// while on CPU-only local hardware.
			Timeout: 120 * time.Second,
		}
	}

	apiKey := ""
	if len(apiKeys) > 0 {
		apiKey = strings.TrimSpace(apiKeys[0])
	}

	return &Provider{
		baseURL: strings.TrimRight(
			strings.TrimSpace(baseURL),
			"/",
		),
		model:      strings.TrimSpace(model),
		language:   strings.TrimSpace(language),
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

func (p *Provider) Name() string {
	return "whisper"
}

type transcriptionResponse struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

func (p *Provider) Transcribe(
	ctx context.Context,
	request speech.TranscribeRequest,
) (speech.TranscribeResult, error) {
	if p == nil || p.baseURL == "" {
		return speech.TranscribeResult{}, speech.ErrProviderNotConfigured
	}

	if request.Reader == nil {
		return speech.TranscribeResult{}, fmt.Errorf(
			"%w: audio is required",
			speech.ErrInvalidInput,
		)
	}

	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+"/v1/audio/transcriptions",
		pipeReader,
	)

	if err != nil {
		_ = pipeReader.Close()

		return speech.TranscribeResult{}, fmt.Errorf(
			"%w: failed to create request: %v",
			speech.ErrProviderUnavailable,
			err,
		)
	}

	httpRequest.Header.Set(
		"Content-Type",
		multipartWriter.FormDataContentType(),
	)
	if p.apiKey != "" {
		httpRequest.Header.Set(
			"Authorization",
			"Bearer "+p.apiKey,
		)
	}

	language := request.Language

	if strings.TrimSpace(language) == "" {
		language = p.language
	}

	writeErrors := make(chan error, 1)

	go func() {
		writeErrors <- writeMultipartBody(
			multipartWriter,
			pipeWriter,
			p.model,
			language,
			request,
		)
	}()

	response, doErr := p.httpClient.Do(httpRequest)

	if writeErr := <-writeErrors; writeErr != nil {
		return speech.TranscribeResult{}, fmt.Errorf(
			"%w: failed to encode audio upload: %v",
			speech.ErrInvalidInput,
			writeErr,
		)
	}

	if doErr != nil {
		return speech.TranscribeResult{}, fmt.Errorf(
			"%w: %v",
			speech.ErrProviderUnavailable,
			doErr,
		)
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		return speech.TranscribeResult{}, fmt.Errorf(
			"%w: failed to read provider response: %v",
			speech.ErrProviderUnavailable,
			err,
		)
	}

	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return speech.TranscribeResult{}, fmt.Errorf(
			"%w: whisper returned HTTP %d: %s",
			speech.ErrProviderUnavailable,
			response.StatusCode,
			extractErrorMessage(body),
		)
	}

	var parsed transcriptionResponse

	if err := json.Unmarshal(body, &parsed); err != nil {
		return speech.TranscribeResult{}, fmt.Errorf(
			"%w: invalid whisper response: %v",
			speech.ErrInvalidProviderResponse,
			err,
		)
	}

	return speech.TranscribeResult{
		Text:     parsed.Text,
		Language: parsed.Language,
	}, nil
}

func writeMultipartBody(
	multipartWriter *multipart.Writer,
	pipeWriter *io.PipeWriter,
	model string,
	language string,
	request speech.TranscribeRequest,
) error {
	err := func() error {
		part, err := multipartWriter.CreateFormFile(
			"file",
			fileNameOrDefault(request.Filename),
		)

		if err != nil {
			return err
		}

		if _, err := io.Copy(part, request.Reader); err != nil {
			return err
		}

		if model != "" {
			if err := multipartWriter.WriteField(
				"model",
				model,
			); err != nil {
				return err
			}
		}

		if strings.TrimSpace(language) != "" {
			if err := multipartWriter.WriteField(
				"language",
				strings.TrimSpace(language),
			); err != nil {
				return err
			}
		}

		if err := multipartWriter.WriteField(
			"response_format",
			"json",
		); err != nil {
			return err
		}

		return multipartWriter.Close()
	}()

	if err != nil {
		_ = pipeWriter.CloseWithError(err)

		return err
	}

	return pipeWriter.Close()
}

func fileNameOrDefault(
	filename string,
) string {
	filename = strings.TrimSpace(filename)

	if filename == "" {
		return "audio.wav"
	}

	return filename
}

func extractErrorMessage(
	body []byte,
) string {
	var structuredError struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if json.Unmarshal(body, &structuredError) == nil &&
		strings.TrimSpace(structuredError.Error.Message) != "" {
		return structuredError.Error.Message
	}

	var stringError struct {
		Error string `json:"error"`
	}

	if json.Unmarshal(body, &stringError) == nil &&
		strings.TrimSpace(stringError.Error) != "" {
		return stringError.Error
	}

	trimmed := strings.TrimSpace(string(body))

	if trimmed == "" {
		return "no additional details"
	}

	if len(trimmed) > 200 {
		trimmed = trimmed[:200] + "..."
	}

	return trimmed
}
