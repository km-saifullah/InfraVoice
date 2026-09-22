package terraform

import (
	"errors"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

func TestGeneratorGenerateVPCAndPublicSubnet(t *testing.T) {
	generator := NewGenerator()

	spec := infrastructure.InfrastructureSpec{
		ID:       bson.NewObjectID(),
		Provider: infrastructure.ProviderAWS,
		Version:  1,
		VPCs: []infrastructure.VPCSpec{
			{
				Name:               "main-vpc",
				CIDR:               "10.0.0.0/16",
				EnableDNS:          true,
				EnableDNSHostnames: true,
				Subnets: []infrastructure.Subnet{
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
	}

	files, err := generator.Generate(
		spec,
		GenerateRequest{
			Region: "ap-south-1",
		},
	)

	if err != nil {
		t.Fatalf(
			"Generate() returned unexpected error: %v",
			err,
		)
	}

	vpcFile := files[FileVPC]

	requiredFragments := []string{
		`resource "aws_vpc"`,
		`resource "aws_subnet"`,
		`10.0.0.0/16`,
		`10.0.1.0/24`,
		`10.0.2.0/24`,
		`aws_internet_gateway`,
		`aws_route_table`,
		`aws_route_table_association`,
		`map_public_ip_on_launch = true`,
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(vpcFile, fragment) {
			t.Fatalf(
				"generated VPC Terraform does not contain %q",
				fragment,
			)
		}
	}

	if strings.Contains(
		vpcFile,
		"aws_nat_gateway",
	) {
		t.Fatal(
			"generated VPC Terraform must not create a NAT gateway",
		)
	}
}

func TestGeneratorGenerateS3(t *testing.T) {
	generator := NewGenerator()

	spec := infrastructure.InfrastructureSpec{
		Provider: infrastructure.ProviderAWS,
		Version:  1,
		S3: []infrastructure.S3Spec{
			{
				Name:       "logs-bucket",
				BucketName: "infra-voice-test-logs",
				Versioning: true,
				Encryption: true,
			},
		},
	}

	files, err := generator.Generate(
		spec,
		GenerateRequest{
			Region: "ap-south-1",
		},
	)

	if err != nil {
		t.Fatalf(
			"Generate() returned unexpected error: %v",
			err,
		)
	}

	s3File := files[FileS3]

	requiredFragments := []string{
		`resource "aws_s3_bucket"`,
		`infra-voice-test-logs`,
		`aws_s3_bucket_versioning`,
		`status = "Enabled"`,
		`aws_s3_bucket_server_side_encryption_configuration`,
		`sse_algorithm = "AES256"`,
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(s3File, fragment) {
			t.Fatalf(
				"generated S3 Terraform does not contain %q",
				fragment,
			)
		}
	}
}

func TestGeneratorGenerateEC2UsesSubnetReference(t *testing.T) {
	generator := NewGenerator()

	spec := infrastructure.InfrastructureSpec{
		Provider: infrastructure.ProviderAWS,
		Version:  1,
		VPCs: []infrastructure.VPCSpec{
			{
				Name:               "main-vpc",
				CIDR:               "10.0.0.0/16",
				EnableDNS:          true,
				EnableDNSHostnames: true,
				Subnets: []infrastructure.Subnet{
					{
						Name: "private-app",
						CIDR: "10.0.1.0/24",
						Type: "private",
					},
				},
			},
		},
		EC2: []infrastructure.EC2Spec{
			{
				Name:         "app-server",
				InstanceType: "t4g.micro",
				AMI:          "ami-1234567890abcdef0",
				Subnet:       "private-app",
				PublicIP:     false,
				RootVolumeGB: 20,
				Count:        1,
			},
		},
	}

	files, err := generator.Generate(
		spec,
		GenerateRequest{
			Region: "ap-south-1",
		},
	)

	if err != nil {
		t.Fatalf(
			"Generate() returned unexpected error: %v",
			err,
		)
	}

	ec2File := files[FileEC2]

	if !strings.Contains(
		ec2File,
		`ami = var.app_server_ami`,
	) {
		t.Fatal(
			"EC2 Terraform does not use the generated AMI variable",
		)
	}

	if !strings.Contains(
		ec2File,
		`subnet_id = aws_subnet.main_vpc_private_app.id`,
	) {
		t.Fatal(
			"EC2 Terraform does not reference the generated subnet",
		)
	}

	if !strings.Contains(
		ec2File,
		`instance_type = "t4g.micro"`,
	) {
		t.Fatal(
			"EC2 Terraform does not contain the expected instance type",
		)
	}
}

func TestGeneratorRejectsMissingEC2Subnet(t *testing.T) {
	generator := NewGenerator()

	spec := infrastructure.InfrastructureSpec{
		Provider: infrastructure.ProviderAWS,
		Version:  1,
		EC2: []infrastructure.EC2Spec{
			{
				Name:         "app-server",
				InstanceType: "t4g.micro",
				AMI:          "ami-1234567890abcdef0",
				PublicIP:     false,
				RootVolumeGB: 20,
				Count:        1,
			},
		},
	}

	_, err := generator.Generate(
		spec,
		GenerateRequest{
			Region: "ap-south-1",
		},
	)

	if err == nil {
		t.Fatal(
			"expected missing subnet error",
		)
	}

	if !errors.Is(
		err,
		ErrInvalidInput,
	) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestGeneratorRejectsUnknownEC2Subnet(t *testing.T) {
	generator := NewGenerator()

	spec := infrastructure.InfrastructureSpec{
		Provider: infrastructure.ProviderAWS,
		Version:  1,
		VPCs: []infrastructure.VPCSpec{
			{
				Name:               "main-vpc",
				CIDR:               "10.0.0.0/16",
				EnableDNS:          true,
				EnableDNSHostnames: true,
				Subnets: []infrastructure.Subnet{
					{
						Name: "private-app",
						CIDR: "10.0.1.0/24",
						Type: "private",
					},
				},
			},
		},
		EC2: []infrastructure.EC2Spec{
			{
				Name:         "app-server",
				InstanceType: "t4g.micro",
				AMI:          "ami-1234567890abcdef0",
				Subnet:       "does-not-exist",
				PublicIP:     false,
				RootVolumeGB: 20,
				Count:        1,
			},
		},
	}

	_, err := generator.Generate(
		spec,
		GenerateRequest{
			Region: "ap-south-1",
		},
	)

	if err == nil {
		t.Fatal(
			"expected unknown subnet error",
		)
	}

	if !errors.Is(
		err,
		ErrSubnetNotFound,
	) {
		t.Fatalf(
			"expected ErrSubnetNotFound, got %v",
			err,
		)
	}
}

func TestGeneratorRejectsAmbiguousSubnet(t *testing.T) {
	generator := NewGenerator()

	spec := infrastructure.InfrastructureSpec{
		Provider: infrastructure.ProviderAWS,
		Version:  1,
		VPCs: []infrastructure.VPCSpec{
			{
				Name:               "vpc-one",
				CIDR:               "10.0.0.0/16",
				EnableDNS:          true,
				EnableDNSHostnames: true,
				Subnets: []infrastructure.Subnet{
					{
						Name: "app",
						CIDR: "10.0.1.0/24",
						Type: "private",
					},
				},
			},
			{
				Name:               "vpc-two",
				CIDR:               "10.1.0.0/16",
				EnableDNS:          true,
				EnableDNSHostnames: true,
				Subnets: []infrastructure.Subnet{
					{
						Name: "app",
						CIDR: "10.1.1.0/24",
						Type: "private",
					},
				},
			},
		},
		EC2: []infrastructure.EC2Spec{
			{
				Name:         "app-server",
				InstanceType: "t4g.micro",
				AMI:          "ami-1234567890abcdef0",
				Subnet:       "app",
				PublicIP:     false,
				RootVolumeGB: 20,
				Count:        1,
			},
		},
	}

	_, err := generator.Generate(
		spec,
		GenerateRequest{
			Region: "ap-south-1",
		},
	)

	if err == nil {
		t.Fatal(
			"expected ambiguous subnet error",
		)
	}

	if !errors.Is(
		err,
		ErrAmbiguousSubnet,
	) {
		t.Fatalf(
			"expected ErrAmbiguousSubnet, got %v",
			err,
		)
	}
}

func TestGeneratorRejectsUnsupportedProvider(t *testing.T) {
	generator := NewGenerator()

	spec := infrastructure.InfrastructureSpec{
		Provider: "azure",
		Version:  1,
		S3: []infrastructure.S3Spec{
			{
				Name:       "test",
				BucketName: "infra-voice-test",
				Encryption: true,
			},
		},
	}

	_, err := generator.Generate(
		spec,
		GenerateRequest{
			Region: "ap-south-1",
		},
	)

	if err == nil {
		t.Fatal(
			"expected unsupported provider error",
		)
	}

	if !errors.Is(
		err,
		ErrUnsupportedProvider,
	) {
		t.Fatalf(
			"expected ErrUnsupportedProvider, got %v",
			err,
		)
	}
}

func TestGeneratorRejectsMissingRegion(t *testing.T) {
	generator := NewGenerator()

	spec := infrastructure.InfrastructureSpec{
		Provider: infrastructure.ProviderAWS,
		Version:  1,
		S3: []infrastructure.S3Spec{
			{
				Name:       "test",
				BucketName: "infra-voice-test",
				Encryption: true,
			},
		},
	}

	_, err := generator.Generate(
		spec,
		GenerateRequest{},
	)

	if err == nil {
		t.Fatal(
			"expected missing region error",
		)
	}

	if !errors.Is(
		err,
		ErrInvalidInput,
	) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestTerraformResourceName(t *testing.T) {
	testCases := []struct {
		input    string
		index    int
		expected string
	}{
		{
			input:    "main-vpc",
			index:    0,
			expected: "main_vpc",
		},
		{
			input:    "Public Subnet",
			index:    0,
			expected: "public_subnet",
		},
		{
			input:    "123-server",
			index:    0,
			expected: "resource_123_server",
		},
		{
			input:    "app-server",
			index:    1,
			expected: "app_server_2",
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.input,
			func(t *testing.T) {
				actual := terraformResourceName(
					testCase.input,
					testCase.index,
				)

				if actual != testCase.expected {
					t.Fatalf(
						"expected %q, got %q",
						testCase.expected,
						actual,
					)
				}
			},
		)
	}
}
