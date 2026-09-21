package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/km-saifullah/infra-voice/backend/internal/ai"
	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

type Provider struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewProvider(
	baseURL string,
	model string,
	httpClient *http.Client,
) *Provider {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 90 * time.Second,
		}
	}

	return &Provider{
		baseURL: strings.TrimRight(
			strings.TrimSpace(baseURL),
			"/",
		),
		model:      strings.TrimSpace(model),
		httpClient: httpClient,
	}
}

func (p *Provider) Name() string {
	return "ollama"
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

type generateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

type modelResponse struct {
	NeedsClarification bool                               `json:"needs_clarification"`
	Clarification      string                             `json:"clarification,omitempty"`
	Specification      *infrastructure.InfrastructureSpec `json:"specification,omitempty"`
}

func (p *Provider) ParseInfrastructure(
	ctx context.Context,
	request ai.ParseRequest,
) (ai.ParseResult, error) {
	if p == nil {
		return ai.ParseResult{}, ai.ErrProviderNotConfigured
	}

	if p.baseURL == "" || p.model == "" {
		return ai.ParseResult{}, ai.ErrProviderNotConfigured
	}

	command := strings.TrimSpace(request.Command)

	if command == "" {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: command cannot be empty",
			ai.ErrInvalidInput,
		)
	}

	payload := generateRequest{
		Model:  p.model,
		Prompt: buildPrompt(command),
		Stream: false,
		Format: "json",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: failed to encode request",
			ai.ErrInvalidProviderResponse,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+"/api/generate",
		bytes.NewReader(body),
	)
	if err != nil {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: failed to create request: %v",
			ai.ErrProviderUnavailable,
			err,
		)
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: %v",
			ai.ErrProviderUnavailable,
			err,
		)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: failed to read provider response: %v",
			ai.ErrProviderUnavailable,
			err,
		)
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {

		var providerResponse generateResponse

		if json.Unmarshal(
			responseBody,
			&providerResponse,
		) == nil &&
			strings.TrimSpace(
				providerResponse.Error,
			) != "" {

			return ai.ParseResult{}, fmt.Errorf(
				"%w: %s",
				ai.ErrProviderUnavailable,
				providerResponse.Error,
			)
		}

		return ai.ParseResult{}, fmt.Errorf(
			"%w: ollama returned HTTP %d",
			ai.ErrProviderUnavailable,
			resp.StatusCode,
		)
	}

	var providerResponse generateResponse

	if err := json.Unmarshal(
		responseBody,
		&providerResponse,
	); err != nil {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: invalid Ollama response: %v",
			ai.ErrInvalidProviderResponse,
			err,
		)
	}

	if strings.TrimSpace(
		providerResponse.Error,
	) != "" {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: %s",
			ai.ErrProviderUnavailable,
			providerResponse.Error,
		)
	}

	responseJSON := strings.TrimSpace(
		providerResponse.Response,
	)

	if responseJSON == "" {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: Ollama returned an empty response",
			ai.ErrInvalidProviderResponse,
		)
	}

	var parsed modelResponse

	if err := json.Unmarshal(
		[]byte(responseJSON),
		&parsed,
	); err != nil {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: model response is not valid JSON: %v",
			ai.ErrInvalidProviderResponse,
			err,
		)
	}

	if parsed.NeedsClarification {
		clarification := strings.TrimSpace(
			parsed.Clarification,
		)

		if clarification == "" {
			return ai.ParseResult{}, fmt.Errorf(
				"%w: clarification response is missing clarification text",
				ai.ErrInvalidProviderResponse,
			)
		}

		return ai.ParseResult{
			NeedsClarification: true,
			Clarification:      clarification,
		}, nil
	}

	if parsed.Specification == nil {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: specification is missing",
			ai.ErrInvalidProviderResponse,
		)
	}

	return ai.ParseResult{
		Specification: parsed.Specification,
	}, nil
}

func buildPrompt(command string) string {
	return fmt.Sprintf(`You are the infrastructure planning component of InfraVoice.

Convert the user's natural-language infrastructure request into a safe, structured AWS infrastructure specification.

Security rules:
- Never invent missing infrastructure requirements.
- Never invent AWS resource names, CIDRs, availability zones, instance types, AMIs, subnet relationships, or other values that materially affect the requested infrastructure.
- If information required to produce a valid specification is missing, return needs_clarification=true and ask one concise question.
- Never return shell commands, Terraform code, AWS CLI commands, credentials, secrets, or executable instructions.
- Return only JSON matching the response schema below.
- Supported provider: aws.
- Supported resources: VPC, S3, EC2, SNS.
- S3 encryption must be enabled.
- Public subnets must explicitly include an availability zone.
- EC2 instances with public_ip=true must explicitly specify a subnet.

Response schema:
{
  "needs_clarification": false,
  "clarification": "",
  "specification": {
    "provider": "aws",
    "version": 1,
    "vpcs": [],
    "s3": [],
    "ec2": [],
    "sns": []
  }
}

If clarification is required, use:
{
  "needs_clarification": true,
  "clarification": "Your concise question here",
  "specification": null
}

Infrastructure specification fields:
- vpcs[].name: logical VPC name
- vpcs[].cidr: VPC CIDR
- vpcs[].enable_dns: whether VPC DNS support is enabled
- vpcs[].enable_dns_hostnames: whether VPC DNS hostnames are enabled
- vpcs[].subnets[].name: logical subnet name
- vpcs[].subnets[].cidr: subnet CIDR
- vpcs[].subnets[].type: public or private
- vpcs[].subnets[].availability_zone: required for public subnets
- s3[].name: logical resource name
- s3[].bucket_name: explicit bucket name when the user provides one
- s3[].versioning: whether versioning is enabled
- s3[].encryption: must be true
- ec2[].name: logical resource name
- ec2[].instance_type: instance type when explicitly provided
- ec2[].ami: AMI when explicitly provided
- ec2[].subnet: subnet name when explicitly provided
- ec2[].public_ip: whether a public IP is requested
- ec2[].root_volume_gb: root volume size when explicitly provided
- ec2[].count: requested instance count
- sns[].name: logical topic name
- sns[].display_name: display name when explicitly provided

User request:
%s`, command)
}
