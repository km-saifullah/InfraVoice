package overview

type Response struct {
	Projects       ProjectsOverview       `json:"projects"`
	Infrastructure InfrastructureOverview `json:"infrastructure"`
	Commands       CommandsOverview       `json:"commands"`
	Terraform      TerraformOverview      `json:"terraform"`
}

type ProjectsOverview struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Archived int64 `json:"archived"`
}

type InfrastructureOverview struct {
	Specifications int64           `json:"specifications"`
	Resources      ResourcesByType `json:"resources"`
}

type ResourcesByType struct {
	VPC int64 `json:"vpc"`
	EC2 int64 `json:"ec2"`
	S3  int64 `json:"s3"`
	SNS int64 `json:"sns"`
}

type CommandsOverview struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
	BySource map[string]int64 `json:"by_source"`
}

type TerraformOverview struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
	ByType   map[string]int64 `json:"by_type"`
	ByRegion map[string]int64 `json:"by_region"`
}
