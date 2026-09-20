package infrastructure

import (
	"fmt"
	"strings"
)

type PolicyError struct {
	Resource string `json:"resource"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

type PolicyErrors struct {
	Errors []PolicyError `json:"errors"`
}

func (e *PolicyErrors) Error() string {
	if len(e.Errors) == 0 {
		return "policy validation failed"
	}

	messages := make([]string, 0, len(e.Errors))

	for _, policyError := range e.Errors {
		resource := policyError.Resource

		if policyError.Field != "" {
			resource = fmt.Sprintf(
				"%s.%s",
				resource,
				policyError.Field,
			)
		}

		messages = append(
			messages,
			fmt.Sprintf(
				"%s: %s",
				resource,
				policyError.Message,
			),
		)
	}

	return strings.Join(messages, "; ")
}

func ValidatePolicy(
	spec *InfrastructureSpec,
) error {
	if spec == nil {
		return &PolicyErrors{
			Errors: []PolicyError{
				{
					Resource: "spec",
					Message:  "infrastructure specification is required",
				},
			},
		}
	}

	errors := make([]PolicyError, 0)

	errors = append(
		errors,
		validateVPCEgressPolicy(spec.VPCs)...,
	)

	errors = append(
		errors,
		validateEC2Policy(spec.EC2)...,
	)

	errors = append(
		errors,
		validateS3Policy(spec.S3)...,
	)

	if len(errors) > 0 {
		return &PolicyErrors{
			Errors: errors,
		}
	}

	return nil
}

func validateVPCEgressPolicy(
	vpcs []VPCSpec,
) []PolicyError {
	errors := make([]PolicyError, 0)

	for index, vpc := range vpcs {
		resource := fmt.Sprintf(
			"vpcs[%d]",
			index,
		)

		for subnetIndex, subnet := range vpc.Subnets {
			subnetResource := fmt.Sprintf(
				"%s.subnets[%d]",
				resource,
				subnetIndex,
			)

			if subnet.Type == "private" &&
				strings.TrimSpace(
					subnet.AvailabilityZone,
				) == "" {
				// No public networking is implicitly enabled
				// for private subnets. This is intentionally
				// only a policy boundary; routing is handled
				// later by the Terraform layer.
				continue
			}

			if subnet.Type == "public" &&
				strings.TrimSpace(
					subnet.AvailabilityZone,
				) == "" {
				errors = append(
					errors,
					PolicyError{
						Resource: subnetResource,
						Field:    "availability_zone",
						Message:  "public subnets must explicitly define an availability zone",
					},
				)
			}
		}
	}

	return errors
}

func validateEC2Policy(
	instances []EC2Spec,
) []PolicyError {
	errors := make([]PolicyError, 0)

	for index, instance := range instances {
		resource := fmt.Sprintf(
			"ec2[%d]",
			index,
		)

		if instance.PublicIP &&
			strings.TrimSpace(instance.Subnet) == "" {
			errors = append(
				errors,
				PolicyError{
					Resource: resource,
					Field:    "subnet",
					Message:  "an EC2 instance with a public IP must explicitly specify a subnet",
				},
			)
		}
	}

	return errors
}

func validateS3Policy(
	buckets []S3Spec,
) []PolicyError {
	errors := make([]PolicyError, 0)

	for index, bucket := range buckets {
		resource := fmt.Sprintf(
			"s3[%d]",
			index,
		)

		if !bucket.Encryption {
			errors = append(
				errors,
				PolicyError{
					Resource: resource,
					Field:    "encryption",
					Message:  "S3 encryption must be enabled",
				},
			)
		}
	}

	return errors
}
