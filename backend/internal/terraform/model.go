package terraform

const (
	FileProvider      = "provider.tf"
	FileVariables     = "variables.tf"
	FileVPC           = "vpc.tf"
	FileS3            = "s3.tf"
	FileEC2           = "ec2.tf"
	FileSNS           = "sns.tf"
	FileTerraformVars = "terraform.tfvars.example"
)

type Files map[string]string

type GenerateRequest struct {
	Region string
}
