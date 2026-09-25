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

	operationContext, cancel := context.WithTimeout(
		ctx,
		110*time.Second,
	)
	defer cancel()

	result, rawResponse, err := p.generate(
		operationContext,
		buildPrompt(command),
	)
	if err != nil {
		return ai.ParseResult{}, err
	}

	if result.NeedsClarification {
		return result, nil
	}

	if result.Specification == nil {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: specification is missing",
			ai.ErrInvalidProviderResponse,
		)
	}

	missingResources := missingRequestedResources(
		command,
		result.Specification,
	)

	if len(missingResources) == 0 {
		return result, nil
	}

	repairPrompt := buildRepairPrompt(
		command,
		rawResponse,
		missingResources,
	)

	repairedResult, _, repairErr := p.generate(
		operationContext,
		repairPrompt,
	)
	if repairErr != nil {
		return ai.ParseResult{}, repairErr
	}

	if repairedResult.NeedsClarification {
		return repairedResult, nil
	}

	if repairedResult.Specification == nil {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: repair response is missing specification",
			ai.ErrInvalidProviderResponse,
		)
	}

	remainingResources := missingRequestedResources(
		command,
		repairedResult.Specification,
	)

	if len(remainingResources) > 0 {
		return ai.ParseResult{}, fmt.Errorf(
			"%w: model response omitted requested resource types: %s",
			ai.ErrInvalidProviderResponse,
			strings.Join(remainingResources, ", "),
		)
	}

	return repairedResult, nil
}

func (p *Provider) generate(
	ctx context.Context,
	prompt string,
) (ai.ParseResult, string, error) {
	payload := generateRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ai.ParseResult{}, "", fmt.Errorf(
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
		return ai.ParseResult{}, "", fmt.Errorf(
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
		return ai.ParseResult{}, "", fmt.Errorf(
			"%w: %v",
			ai.ErrProviderUnavailable,
			err,
		)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ai.ParseResult{}, "", fmt.Errorf(
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
			return ai.ParseResult{}, "", fmt.Errorf(
				"%w: %s",
				ai.ErrProviderUnavailable,
				providerResponse.Error,
			)
		}

		return ai.ParseResult{}, "", fmt.Errorf(
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
		return ai.ParseResult{}, "", fmt.Errorf(
			"%w: invalid Ollama response: %v",
			ai.ErrInvalidProviderResponse,
			err,
		)
	}

	if strings.TrimSpace(
		providerResponse.Error,
	) != "" {
		return ai.ParseResult{}, "", fmt.Errorf(
			"%w: %s",
			ai.ErrProviderUnavailable,
			providerResponse.Error,
		)
	}

	responseJSON := strings.TrimSpace(
		providerResponse.Response,
	)

	if responseJSON == "" {
		return ai.ParseResult{}, "", fmt.Errorf(
			"%w: Ollama returned an empty response",
			ai.ErrInvalidProviderResponse,
		)
	}

	var parsed modelResponse

	if err := json.Unmarshal(
		[]byte(responseJSON),
		&parsed,
	); err != nil {
		return ai.ParseResult{}, responseJSON, fmt.Errorf(
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
			return ai.ParseResult{}, responseJSON, fmt.Errorf(
				"%w: clarification response is missing clarification text",
				ai.ErrInvalidProviderResponse,
			)
		}

		return ai.ParseResult{
			NeedsClarification: true,
			Clarification:      clarification,
		}, responseJSON, nil
	}

	if parsed.Specification == nil {
		return ai.ParseResult{}, responseJSON, fmt.Errorf(
			"%w: specification is missing",
			ai.ErrInvalidProviderResponse,
		)
	}

	return ai.ParseResult{
		Specification: parsed.Specification,
	}, responseJSON, nil
}

func missingRequestedResources(
	command string,
	specification *infrastructure.InfrastructureSpec,
) []string {
	if specification == nil {
		return []string{"infrastructure"}
	}

	command = strings.ToLower(
		strings.TrimSpace(command),
	)

	missing := make([]string, 0, 4)

	if mentionsVPC(command) && len(specification.VPCs) == 0 {
		missing = append(missing, "vpc")
	}

	if mentionsS3(command) && len(specification.S3) == 0 {
		missing = append(missing, "s3")
	}

	if mentionsEC2(command) && len(specification.EC2) == 0 {
		missing = append(missing, "ec2")
	}

	if mentionsSNS(command) && len(specification.SNS) == 0 {
		missing = append(missing, "sns")
	}

	return missing
}

func mentionsVPC(command string) bool {
	return containsAnyWord(
		command,
		"vpc",
		"vpcs",
		"virtual private cloud",
		"subnet",
		"subnets",
	)
}

func mentionsS3(command string) bool {
	return containsAnyWord(
		command,
		"s3",
		"bucket",
		"buckets",
	)
}

func mentionsEC2(command string) bool {
	return containsAnyWord(
		command,
		"ec2",
		"instance",
		"instances",
	)
}

func mentionsSNS(command string) bool {
	return containsAnyWord(
		command,
		"sns",
		"topic",
		"topics",
	)
}

func containsAnyWord(
	value string,
	terms ...string,
) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}

	return false
}

func buildRepairPrompt(
	command string,
	previousResponse string,
	missingResources []string,
) string {
	return fmt.Sprintf(`The previous infrastructure parsing response was incomplete.

You must repair it without dropping any resource explicitly requested by the user.

Original user request:
%s

Previous model response:
%s

Missing requested resource types:
%s

Return the complete corrected JSON response using exactly the same response schema.
Do not remove resources that were already present in the previous response.
Do not invent unspecified infrastructure values.
If required information is genuinely missing, return needs_clarification=true with one concise question instead.

Resource preservation is mandatory:
- If the user requested a VPC or subnet, specification.vpcs must contain the requested VPC resource(s).
- If the user requested S3 or a bucket, specification.s3 must contain the requested bucket resource(s).
- If the user requested EC2 or an instance, specification.ec2 must contain the requested instance resource(s).
- If the user requested SNS or a topic, specification.sns must contain the requested topic resource(s).
- Every explicitly requested resource type must be represented in its corresponding array.

Return JSON only.`, command, previousResponse, strings.Join(missingResources, ", "))
}

func buildPrompt(command string) string {
	return fmt.Sprintf(`You are the infrastructure planning component of InfraVoice.

Convert the user's natural-language infrastructure request into one complete, safe, structured AWS infrastructure specification.

Security and correctness rules:
- Preserve every infrastructure resource explicitly requested by the user.
- Never silently omit a requested resource type.
- Never invent missing infrastructure requirements.
- Never invent AWS resource names, CIDRs, availability zones, instance types, AMIs, subnet relationships, or other values that materially affect the requested infrastructure.
- If information required to produce a valid specification is missing, return needs_clarification=true and ask one concise question.
- Never return shell commands, Terraform code, AWS CLI commands, credentials, secrets, or executable instructions.
- Return only JSON matching the response schema below.
- Supported provider: aws.
- Supported resources: VPC, S3, EC2, SNS.
- S3 encryption must be true.
- Public subnets must explicitly include an availability zone.
- EC2 instances with public_ip=true must explicitly specify a subnet.

Resource preservation is mandatory:
- If the user requests a VPC or subnet, specification.vpcs must contain the requested VPC resource(s).
- If the user requests S3 or a bucket, specification.s3 must contain the requested bucket resource(s).
- If the user requests EC2 or an instance, specification.ec2 must contain the requested instance resource(s).
- If the user requests SNS or a topic, specification.sns must contain the requested topic resource(s).
- A request can contain multiple resource types. Preserve all of them in the same response.
- Never return only the first resource type mentioned by the user.
- Before producing the final JSON, compare the user request against all four resource arrays and make sure every explicitly requested resource type is represented.

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

Important:
The following user request may contain multiple independent resource types. All requested resource types must appear in the final specification.

User request:
%s`, command)
}
