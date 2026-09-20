package infrastructure

import (
	"strings"
	"testing"
)

func TestValidateValidSpecification(t *testing.T) {
	spec := &InfrastructureSpec{
		Provider: ProviderAWS,
		Version:  1,
		VPCs: []VPCSpec{
			{
				Name:               "main-vpc",
				CIDR:               "10.0.0.0/16",
				EnableDNS:          true,
				EnableDNSHostnames: true,
				Subnets: []Subnet{
					{
						Name:             "public-subnet",
						CIDR:             "10.0.1.0/24",
						Type:             "public",
						AvailabilityZone: "ap-south-1a",
					},
					{
						Name: "private-subnet",
						CIDR: "10.0.2.0/24",
						Type: "private",
					},
				},
			},
		},
		S3: []S3Spec{
			{
				Name:       "application-assets",
				BucketName: "infra-voice-application-assets",
				Versioning: true,
				Encryption: true,
			},
		},
		EC2: []EC2Spec{
			{
				Name:         "application-server",
				InstanceType: "t3.micro",
				AMI:          "ami-12345678",
				Subnet:       "public-subnet",
				PublicIP:     true,
				RootVolumeGB: 20,
				Count:        1,
			},
		},
		SNS: []SNSSpec{
			{
				Name:        "application-alerts",
				DisplayName: "Application Alerts",
			},
		},
	}

	if err := Validate(spec); err != nil {
		t.Fatalf(
			"expected valid specification, got: %v",
			err,
		)
	}

	if err := ValidatePolicy(spec); err != nil {
		t.Fatalf(
			"expected valid policy, got: %v",
			err,
		)
	}
}

func TestValidateRejectsUnsupportedProvider(t *testing.T) {
	spec := &InfrastructureSpec{
		Provider: "azure",
		Version:  1,
		VPCs: []VPCSpec{
			{
				Name: "main-vpc",
				CIDR: "10.0.0.0/16",
			},
		},
	}

	err := Validate(spec)

	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(
		err.Error(),
		"unsupported provider",
	) {
		t.Fatalf(
			"expected unsupported provider error, got: %v",
			err,
		)
	}
}

func TestValidateRejectsSubnetOutsideVPC(t *testing.T) {
	spec := &InfrastructureSpec{
		Provider: ProviderAWS,
		Version:  1,
		VPCs: []VPCSpec{
			{
				Name: "main-vpc",
				CIDR: "10.0.0.0/16",
				Subnets: []Subnet{
					{
						Name: "invalid-subnet",
						CIDR: "10.1.1.0/24",
						Type: "private",
					},
				},
			},
		},
	}

	err := Validate(spec)

	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(
		err.Error(),
		"subnet CIDR must be inside the VPC CIDR",
	) {
		t.Fatalf(
			"expected subnet CIDR validation error, got: %v",
			err,
		)
	}
}

func TestValidateRejectsInvalidEC2Count(t *testing.T) {
	spec := &InfrastructureSpec{
		Provider: ProviderAWS,
		Version:  1,
		EC2: []EC2Spec{
			{
				Name:         "application-server",
				InstanceType: "t3.micro",
				RootVolumeGB: 20,
				Count:        0,
			},
		},
	}

	err := Validate(spec)

	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(
		err.Error(),
		"count must be greater than 0",
	) {
		t.Fatalf(
			"expected EC2 count validation error, got: %v",
			err,
		)
	}
}

func TestValidatePolicyRejectsUnencryptedS3(t *testing.T) {
	spec := &InfrastructureSpec{
		Provider: ProviderAWS,
		Version:  1,
		S3: []S3Spec{
			{
				Name:       "application-assets",
				BucketName: "infra-voice-application-assets",
				Encryption: false,
			},
		},
	}

	if err := Validate(spec); err != nil {
		t.Fatalf(
			"expected structural validation to pass, got: %v",
			err,
		)
	}

	err := ValidatePolicy(spec)

	if err == nil {
		t.Fatal("expected policy validation error")
	}

	if !strings.Contains(
		err.Error(),
		"S3 encryption must be enabled",
	) {
		t.Fatalf(
			"expected S3 encryption policy error, got: %v",
			err,
		)
	}
}
