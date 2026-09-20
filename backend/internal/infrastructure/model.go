package infrastructure

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	ResourceTypeVPC = "vpc"
	ResourceTypeS3  = "s3"
	ResourceTypeEC2 = "ec2"
	ResourceTypeSNS = "sns"
)

const (
	ProviderAWS = "aws"
)

type InfrastructureSpec struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ProjectID bson.ObjectID `bson:"project_id" json:"project_id"`
	Provider  string        `bson:"provider" json:"provider"`
	Version   int           `bson:"version" json:"version"`

	VPCs []VPCSpec `bson:"vpcs,omitempty" json:"vpcs,omitempty"`
	S3   []S3Spec  `bson:"s3,omitempty" json:"s3,omitempty"`
	EC2  []EC2Spec `bson:"ec2,omitempty" json:"ec2,omitempty"`
	SNS  []SNSSpec `bson:"sns,omitempty" json:"sns,omitempty"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

type VPCSpec struct {
	Name               string   `bson:"name" json:"name"`
	CIDR               string   `bson:"cidr" json:"cidr"`
	EnableDNS          bool     `bson:"enable_dns" json:"enable_dns"`
	EnableDNSHostnames bool     `bson:"enable_dns_hostnames" json:"enable_dns_hostnames"`
	Subnets            []Subnet `bson:"subnets,omitempty" json:"subnets,omitempty"`
}

type Subnet struct {
	Name             string `bson:"name" json:"name"`
	CIDR             string `bson:"cidr" json:"cidr"`
	Type             string `bson:"type" json:"type"`
	AvailabilityZone string `bson:"availability_zone,omitempty" json:"availability_zone,omitempty"`
}

type S3Spec struct {
	Name       string `bson:"name" json:"name"`
	BucketName string `bson:"bucket_name,omitempty" json:"bucket_name,omitempty"`
	Versioning bool   `bson:"versioning" json:"versioning"`
	Encryption bool   `bson:"encryption" json:"encryption"`
}

type EC2Spec struct {
	Name         string `bson:"name" json:"name"`
	InstanceType string `bson:"instance_type" json:"instance_type"`
	AMI          string `bson:"ami,omitempty" json:"ami,omitempty"`
	Subnet       string `bson:"subnet,omitempty" json:"subnet,omitempty"`
	PublicIP     bool   `bson:"public_ip" json:"public_ip"`
	RootVolumeGB int    `bson:"root_volume_gb" json:"root_volume_gb"`
	Count        int    `bson:"count" json:"count"`
}

type SNSSpec struct {
	Name        string `bson:"name" json:"name"`
	DisplayName string `bson:"display_name,omitempty" json:"display_name,omitempty"`
}

func (s *InfrastructureSpec) Normalize() {
	s.Provider = strings.ToLower(
		strings.TrimSpace(s.Provider),
	)

	for index := range s.VPCs {
		s.VPCs[index].Normalize()
	}

	for index := range s.S3 {
		s.S3[index].Normalize()
	}

	for index := range s.EC2 {
		s.EC2[index].Normalize()
	}

	for index := range s.SNS {
		s.SNS[index].Normalize()
	}
}

func (v *VPCSpec) Normalize() {
	v.Name = strings.TrimSpace(v.Name)
	v.CIDR = strings.TrimSpace(v.CIDR)

	for index := range v.Subnets {
		v.Subnets[index].Normalize()
	}
}

func (s *Subnet) Normalize() {
	s.Name = strings.TrimSpace(s.Name)
	s.CIDR = strings.TrimSpace(s.CIDR)
	s.Type = strings.ToLower(strings.TrimSpace(s.Type))
	s.AvailabilityZone = strings.TrimSpace(
		s.AvailabilityZone,
	)
}

func (s *S3Spec) Normalize() {
	s.Name = strings.TrimSpace(s.Name)
	s.BucketName = strings.ToLower(
		strings.TrimSpace(s.BucketName),
	)
}

func (e *EC2Spec) Normalize() {
	e.Name = strings.TrimSpace(e.Name)
	e.InstanceType = strings.TrimSpace(e.InstanceType)
	e.AMI = strings.TrimSpace(e.AMI)
	e.Subnet = strings.TrimSpace(e.Subnet)
}

func (s *SNSSpec) Normalize() {
	s.Name = strings.TrimSpace(s.Name)
	s.DisplayName = strings.TrimSpace(s.DisplayName)
}

func (s *InfrastructureSpec) HasResources() bool {
	return len(s.VPCs) > 0 ||
		len(s.S3) > 0 ||
		len(s.EC2) > 0 ||
		len(s.SNS) > 0
}
