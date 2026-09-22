package terraform

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

var (
	ErrInvalidInput = errors.New(
		"invalid terraform generation input",
	)

	ErrUnsupportedProvider = errors.New(
		"unsupported terraform provider",
	)

	ErrAmbiguousSubnet = errors.New(
		"ambiguous subnet reference",
	)

	ErrSubnetNotFound = errors.New(
		"subnet reference not found",
	)
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Generate(
	spec infrastructure.InfrastructureSpec,
	request GenerateRequest,
) (Files, error) {
	spec.Normalize()

	if err := infrastructure.Validate(&spec); err != nil {
		return nil, err
	}

	if err := infrastructure.ValidatePolicy(&spec); err != nil {
		return nil, err
	}

	if !spec.HasResources() {
		return nil, fmt.Errorf(
			"%w: infrastructure specification contains no resources",
			ErrInvalidInput,
		)
	}

	if spec.Provider != infrastructure.ProviderAWS {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrUnsupportedProvider,
			spec.Provider,
		)
	}

	region := strings.TrimSpace(request.Region)

	if region == "" {
		return nil, fmt.Errorf(
			"%w: AWS region is required",
			ErrInvalidInput,
		)
	}

	subnetReferences, err := buildSubnetReferences(
		spec.VPCs,
	)

	if err != nil {
		return nil, err
	}

	if err := validateEC2References(
		spec.EC2,
		subnetReferences,
	); err != nil {
		return nil, err
	}

	files := make(Files)

	files[FileProvider] = generateProvider()

	files[FileVariables] = generateVariables(
		spec,
		region,
	)

	files[FileTerraformVars] = generateTerraformVarsExample(
		spec,
		region,
	)

	if len(spec.VPCs) > 0 {
		files[FileVPC] = generateVPC(
			spec.VPCs,
		)
	}

	if len(spec.S3) > 0 {
		files[FileS3] = generateS3(
			spec.S3,
		)
	}

	if len(spec.EC2) > 0 {
		files[FileEC2] = generateEC2(
			spec.EC2,
			subnetReferences,
		)
	}

	if len(spec.SNS) > 0 {
		files[FileSNS] = generateSNS(
			spec.SNS,
		)
	}

	return files, nil
}

type subnetReference struct {
	ResourceName string
	VPCName      string
	SubnetName   string
}

func buildSubnetReferences(
	vpcs []infrastructure.VPCSpec,
) (map[string]subnetReference, error) {
	references := make(
		map[string]subnetReference,
	)

	for vpcIndex, vpc := range vpcs {
		vpcResourceName := terraformResourceName(
			vpc.Name,
			vpcIndex,
		)

		for subnetIndex, subnet := range vpc.Subnets {
			key := strings.TrimSpace(subnet.Name)

			if key == "" {
				continue
			}

			if _, exists := references[key]; exists {
				return nil, fmt.Errorf(
					"%w: subnet %q exists in multiple VPCs",
					ErrAmbiguousSubnet,
					key,
				)
			}

			references[key] = subnetReference{
				ResourceName: terraformResourceName(
					fmt.Sprintf(
						"%s_%s",
						vpc.Name,
						subnet.Name,
					),
					subnetIndex,
				),
				VPCName:    vpcResourceName,
				SubnetName: key,
			}
		}
	}

	return references, nil
}

func validateEC2References(
	instances []infrastructure.EC2Spec,
	subnetReferences map[string]subnetReference,
) error {
	for index, instance := range instances {
		subnetName := strings.TrimSpace(
			instance.Subnet,
		)

		if subnetName == "" {
			return fmt.Errorf(
				"%w: ec2[%d].subnet is required for deterministic Terraform generation",
				ErrInvalidInput,
				index,
			)
		}

		if _, exists := subnetReferences[subnetName]; !exists {
			return fmt.Errorf(
				"%w: ec2[%d].subnet %q",
				ErrSubnetNotFound,
				index,
				subnetName,
			)
		}
	}

	return nil
}

func generateProvider() string {
	return `terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}
`
}

func generateVariables(
	spec infrastructure.InfrastructureSpec,
	region string,
) string {
	var builder strings.Builder

	builder.WriteString(`variable "aws_region" {
  description = "AWS region where resources will be created."
  type        = string
`)

	builder.WriteString(
		fmt.Sprintf(
			"  default     = %q\n",
			region,
		),
	)

	builder.WriteString("}\n")

	for index, instance := range spec.EC2 {
		name := terraformResourceName(
			instance.Name,
			index,
		)

		builder.WriteString("\n")

		builder.WriteString(
			fmt.Sprintf(
				`variable "%s_ami" {
  description = "AMI ID for EC2 resource %s."
  type        = string
`,
				name,
				instance.Name,
			),
		)

		if instance.AMI != "" {
			builder.WriteString(
				fmt.Sprintf(
					"  default     = %q\n",
					instance.AMI,
				),
			)
		}

		builder.WriteString("}\n")
	}

	return builder.String()
}

func generateTerraformVarsExample(
	spec infrastructure.InfrastructureSpec,
	region string,
) string {
	var builder strings.Builder

	builder.WriteString(
		fmt.Sprintf(
			"aws_region = %q\n",
			region,
		),
	)

	for index, instance := range spec.EC2 {
		if instance.AMI == "" {
			name := terraformResourceName(
				instance.Name,
				index,
			)

			builder.WriteString("\n")

			builder.WriteString(
				fmt.Sprintf(
					"%s_ami = \"ami-REPLACE_WITH_VALID_AMI_ID\"\n",
					name,
				),
			)
		}
	}

	return builder.String()
}

func generateVPC(
	vpcs []infrastructure.VPCSpec,
) string {
	var builder strings.Builder

	for vpcIndex, vpc := range vpcs {
		vpcResourceName := terraformResourceName(
			vpc.Name,
			vpcIndex,
		)

		if vpcIndex > 0 {
			builder.WriteString("\n")
		}

		builder.WriteString(
			fmt.Sprintf(
				`resource "aws_vpc" "%s" {
  cidr_block           = %q
  enable_dns_support   = %t
  enable_dns_hostnames = %t

  tags = {
    Name = %q
  }
}
`,
				vpcResourceName,
				vpc.CIDR,
				vpc.EnableDNS,
				vpc.EnableDNSHostnames,
				vpc.Name,
			),
		)

		for subnetIndex, subnet := range vpc.Subnets {
			subnetResourceName := terraformResourceName(
				fmt.Sprintf(
					"%s_%s",
					vpc.Name,
					subnet.Name,
				),
				subnetIndex,
			)

			builder.WriteString("\n")

			builder.WriteString(
				fmt.Sprintf(
					`resource "aws_subnet" "%s" {
  vpc_id            = aws_vpc.%s.id
  cidr_block        = %q
`,
					subnetResourceName,
					vpcResourceName,
					subnet.CIDR,
				),
			)

			if subnet.AvailabilityZone != "" {
				builder.WriteString(
					fmt.Sprintf(
						"  availability_zone = %q\n",
						subnet.AvailabilityZone,
					),
				)
			}

			if subnet.Type == "public" {
				builder.WriteString(
					"  map_public_ip_on_launch = true\n",
				)
			}

			builder.WriteString(
				fmt.Sprintf(
					`
  tags = {
    Name = %q
    Type = %q
  }
}
`,
					subnet.Name,
					subnet.Type,
				),
			)
		}

		for subnetIndex, subnet := range vpc.Subnets {
			if subnet.Type != "public" {
				continue
			}

			subnetResourceName := terraformResourceName(
				fmt.Sprintf(
					"%s_%s",
					vpc.Name,
					subnet.Name,
				),
				subnetIndex,
			)

			routeTableName := terraformResourceName(
				fmt.Sprintf(
					"%s_%s_public",
					vpc.Name,
					subnet.Name,
				),
				subnetIndex,
			)

			igwName := terraformResourceName(
				fmt.Sprintf(
					"%s_igw",
					vpc.Name,
				),
				vpcIndex,
			)

			builder.WriteString("\n")

			builder.WriteString(
				fmt.Sprintf(
					`resource "aws_internet_gateway" "%s" {
  vpc_id = aws_vpc.%s.id

  tags = {
    Name = %q
  }
}
`,
					igwName,
					vpcResourceName,
					fmt.Sprintf(
						"%s-igw",
						vpc.Name,
					),
				),
			)

			builder.WriteString("\n")

			builder.WriteString(
				fmt.Sprintf(
					`resource "aws_route_table" "%s" {
  vpc_id = aws_vpc.%s.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.%s.id
  }

  tags = {
    Name = %q
  }
}
`,
					routeTableName,
					vpcResourceName,
					igwName,
					fmt.Sprintf(
						"%s-public",
						subnet.Name,
					),
				),
			)

			builder.WriteString("\n")

			builder.WriteString(
				fmt.Sprintf(
					`resource "aws_route_table_association" "%s" {
  subnet_id      = aws_subnet.%s.id
  route_table_id = aws_route_table.%s.id
}
`,
					terraformResourceName(
						fmt.Sprintf(
							"%s_%s_association",
							vpc.Name,
							subnet.Name,
						),
						subnetIndex,
					),
					subnetResourceName,
					routeTableName,
				),
			)
		}
	}

	return builder.String()
}

func generateS3(
	buckets []infrastructure.S3Spec,
) string {
	var builder strings.Builder

	for index, bucket := range buckets {
		resourceName := terraformResourceName(
			bucket.Name,
			index,
		)

		bucketName := bucket.BucketName

		if bucketName == "" {
			bucketName = bucket.Name
		}

		if index > 0 {
			builder.WriteString("\n")
		}

		builder.WriteString(
			fmt.Sprintf(
				`resource "aws_s3_bucket" "%s" {
  bucket = %q

  tags = {
    Name = %q
  }
}
`,
				resourceName,
				bucketName,
				bucket.Name,
			),
		)

		if bucket.Versioning {
			builder.WriteString("\n")

			builder.WriteString(
				fmt.Sprintf(
					`resource "aws_s3_bucket_versioning" "%s" {
  bucket = aws_s3_bucket.%s.id

  versioning_configuration {
    status = "Enabled"
  }
}
`,
					resourceName,
					resourceName,
				),
			)
		}

		builder.WriteString("\n")

		builder.WriteString(
			fmt.Sprintf(
				`resource "aws_s3_bucket_server_side_encryption_configuration" "%s" {
  bucket = aws_s3_bucket.%s.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}
`,
				resourceName,
				resourceName,
			),
		)
	}

	return builder.String()
}

func generateEC2(
	instances []infrastructure.EC2Spec,
	subnetReferences map[string]subnetReference,
) string {
	var builder strings.Builder

	for index, instance := range instances {
		if index > 0 {
			builder.WriteString("\n")
		}

		resourceName := terraformResourceName(
			instance.Name,
			index,
		)

		subnet := subnetReferences[strings.TrimSpace(instance.Subnet)]

		builder.WriteString(
			fmt.Sprintf(
				`resource "aws_instance" "%s" {
  count = %d

  ami           = var.%s_ami
  instance_type = %q
  subnet_id     = aws_subnet.%s.id

  associate_public_ip_address = %t

`,
				resourceName,
				instance.Count,
				resourceName,
				instance.InstanceType,
				subnet.ResourceName,
				instance.PublicIP,
			),
		)

		builder.WriteString(
			fmt.Sprintf(
				`  root_block_device {
    volume_size = %d
  }

`,
				instance.RootVolumeGB,
			),
		)

		builder.WriteString(
			fmt.Sprintf(
				`  tags = {
    Name = %q
  }
}
`,
				instance.Name,
			),
		)
	}

	return builder.String()
}

func generateSNS(
	topics []infrastructure.SNSSpec,
) string {
	var builder strings.Builder

	for index, topic := range topics {
		if index > 0 {
			builder.WriteString("\n")
		}

		resourceName := terraformResourceName(
			topic.Name,
			index,
		)

		builder.WriteString(
			fmt.Sprintf(
				`resource "aws_sns_topic" "%s" {
  name = %q
`,
				resourceName,
				topic.Name,
			),
		)

		if topic.DisplayName != "" {
			builder.WriteString(
				fmt.Sprintf(
					"  display_name = %q\n",
					topic.DisplayName,
				),
			)
		}

		builder.WriteString("}\n")
	}

	return builder.String()
}

func terraformResourceName(
	value string,
	index int,
) string {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	var builder strings.Builder

	for _, character := range value {
		switch {
		case character >= 'a' &&
			character <= 'z':
			builder.WriteRune(character)

		case character >= '0' &&
			character <= '9':
			builder.WriteRune(character)

		default:
			builder.WriteRune('_')
		}
	}

	result := strings.Trim(
		builder.String(),
		"_",
	)

	if result == "" {
		result = "resource"
	}

	if character := result[0]; character >= '0' &&
		character <= '9' {
		result = "resource_" + result
	}

	if index > 0 {
		result += "_" + strconv.Itoa(index+1)
	}

	return result
}

func sortedFileNames(
	files Files,
) []string {
	names := make([]string, 0, len(files))

	for name := range files {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
