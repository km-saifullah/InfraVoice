package infrastructure

import (
	"fmt"
	"net"
	"strings"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}

	messages := make([]string, 0, len(e.Errors))

	for _, validationError := range e.Errors {
		if validationError.Field == "" {
			messages = append(
				messages,
				validationError.Message,
			)
			continue
		}

		messages = append(
			messages,
			fmt.Sprintf(
				"%s: %s",
				validationError.Field,
				validationError.Message,
			),
		)
	}

	return strings.Join(messages, "; ")
}

func Validate(spec *InfrastructureSpec) error {
	if spec == nil {
		return &ValidationErrors{
			Errors: []ValidationError{
				{
					Field:   "spec",
					Message: "infrastructure specification is required",
				},
			},
		}
	}

	spec.Normalize()

	errors := make([]ValidationError, 0)

	if spec.Provider == "" {
		errors = append(
			errors,
			ValidationError{
				Field:   "provider",
				Message: "provider is required",
			},
		)
	} else if spec.Provider != ProviderAWS {
		errors = append(
			errors,
			ValidationError{
				Field: "provider",
				Message: fmt.Sprintf(
					"unsupported provider %q",
					spec.Provider,
				),
			},
		)
	}

	if spec.Version <= 0 {
		errors = append(
			errors,
			ValidationError{
				Field:   "version",
				Message: "version must be greater than 0",
			},
		)
	}

	if !spec.HasResources() {
		errors = append(
			errors,
			ValidationError{
				Field:   "resources",
				Message: "at least one infrastructure resource is required",
			},
		)
	}

	errors = append(
		errors,
		validateVPCs(spec.VPCs)...,
	)

	errors = append(
		errors,
		validateS3(spec.S3)...,
	)

	errors = append(
		errors,
		validateEC2(spec.EC2)...,
	)

	errors = append(
		errors,
		validateSNS(spec.SNS)...,
	)

	if len(errors) > 0 {
		return &ValidationErrors{
			Errors: errors,
		}
	}

	return nil
}

func validateVPCs(
	vpcs []VPCSpec,
) []ValidationError {
	errors := make([]ValidationError, 0)

	names := make(map[string]struct{})

	for index, vpc := range vpcs {
		prefix := fmt.Sprintf(
			"vpcs[%d]",
			index,
		)

		if vpc.Name == "" {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "name is required",
				},
			)
		}

		if _, exists := names[vpc.Name]; exists {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "duplicate VPC name",
				},
			)
		}

		names[vpc.Name] = struct{}{}

		if err := validateCIDR(vpc.CIDR); err != nil {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".cidr",
					Message: err.Error(),
				},
			)
		}

		errors = append(
			errors,
			validateSubnets(
				prefix,
				vpc,
			)...,
		)
	}

	return errors
}

func validateSubnets(
	vpcPrefix string,
	vpc VPCSpec,
) []ValidationError {
	errors := make([]ValidationError, 0)

	names := make(map[string]struct{})

	var vpcNetwork *net.IPNet

	if vpc.CIDR != "" {
		_, network, err := net.ParseCIDR(vpc.CIDR)
		if err == nil {
			vpcNetwork = network
		}
	}

	for index, subnet := range vpc.Subnets {
		prefix := fmt.Sprintf(
			"%s.subnets[%d]",
			vpcPrefix,
			index,
		)

		if subnet.Name == "" {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "name is required",
				},
			)
		}

		if _, exists := names[subnet.Name]; exists {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "duplicate subnet name",
				},
			)
		}

		names[subnet.Name] = struct{}{}

		if err := validateCIDR(subnet.CIDR); err != nil {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".cidr",
					Message: err.Error(),
				},
			)
		} else if vpcNetwork != nil {
			_, subnetNetwork, _ := net.ParseCIDR(
				subnet.CIDR,
			)

			if !vpcNetwork.Contains(
				subnetNetwork.IP,
			) {
				errors = append(
					errors,
					ValidationError{
						Field:   prefix + ".cidr",
						Message: "subnet CIDR must be inside the VPC CIDR",
					},
				)
			}
		}

		if subnet.Type != "public" &&
			subnet.Type != "private" {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".type",
					Message: "subnet type must be public or private",
				},
			)
		}
	}

	return errors
}

func validateS3(
	buckets []S3Spec,
) []ValidationError {
	errors := make([]ValidationError, 0)

	names := make(map[string]struct{})

	for index, bucket := range buckets {
		prefix := fmt.Sprintf(
			"s3[%d]",
			index,
		)

		if bucket.Name == "" {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "name is required",
				},
			)
		}

		if _, exists := names[bucket.Name]; exists {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "duplicate S3 resource name",
				},
			)
		}

		names[bucket.Name] = struct{}{}

		if bucket.BucketName != "" {
			if len(bucket.BucketName) < 3 ||
				len(bucket.BucketName) > 63 {
				errors = append(
					errors,
					ValidationError{
						Field:   prefix + ".bucket_name",
						Message: "bucket name must be between 3 and 63 characters",
					},
				)
			}

			if strings.Contains(
				bucket.BucketName,
				"_",
			) {
				errors = append(
					errors,
					ValidationError{
						Field:   prefix + ".bucket_name",
						Message: "bucket name cannot contain underscores",
					},
				)
			}
		}
	}

	return errors
}

func validateEC2(
	instances []EC2Spec,
) []ValidationError {
	errors := make([]ValidationError, 0)

	names := make(map[string]struct{})

	for index, instance := range instances {
		prefix := fmt.Sprintf(
			"ec2[%d]",
			index,
		)

		if instance.Name == "" {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "name is required",
				},
			)
		}

		if _, exists := names[instance.Name]; exists {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "duplicate EC2 resource name",
				},
			)
		}

		names[instance.Name] = struct{}{}

		if instance.InstanceType == "" {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".instance_type",
					Message: "instance type is required",
				},
			)
		}

		if instance.RootVolumeGB <= 0 {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".root_volume_gb",
					Message: "root volume size must be greater than 0",
				},
			)
		}

		if instance.Count <= 0 {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".count",
					Message: "count must be greater than 0",
				},
			)
		}

		if instance.Count > 100 {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".count",
					Message: "count cannot exceed 100",
				},
			)
		}
	}

	return errors
}

func validateSNS(
	topics []SNSSpec,
) []ValidationError {
	errors := make([]ValidationError, 0)

	names := make(map[string]struct{})

	for index, topic := range topics {
		prefix := fmt.Sprintf(
			"sns[%d]",
			index,
		)

		if topic.Name == "" {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "name is required",
				},
			)
		}

		if _, exists := names[topic.Name]; exists {
			errors = append(
				errors,
				ValidationError{
					Field:   prefix + ".name",
					Message: "duplicate SNS resource name",
				},
			)
		}

		names[topic.Name] = struct{}{}
	}

	return errors
}

func validateCIDR(
	cidr string,
) error {
	if strings.TrimSpace(cidr) == "" {
		return fmt.Errorf("CIDR is required")
	}

	ip, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR")
	}

	if ip == nil || network == nil {
		return fmt.Errorf("invalid CIDR")
	}

	return nil
}
