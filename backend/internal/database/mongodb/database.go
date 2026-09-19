package mongodb

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	CollectionUsers               = "users"
	CollectionProjects            = "projects"
	CollectionCommands            = "commands"
	CollectionInfrastructureSpecs = "infrastructure_specs"
	CollectionTerraformRuns       = "terraform_runs"
	CollectionAuditLogs           = "audit_logs"
)

type Database struct {
	client *Client
	db     *mongo.Database
}

func NewDatabase(client *Client) (*Database, error) {
	if client == nil {
		return nil, fmt.Errorf("mongodb client is required")
	}

	database := client.Database()

	if database == nil {
		return nil, fmt.Errorf("mongodb database is not initialized")
	}

	return &Database{
		client: client,
		db:     database,
	}, nil
}

func (d *Database) Client() *Client {
	return d.client
}

func (d *Database) Database() *mongo.Database {
	return d.db
}

func (d *Database) Collection(name string) *mongo.Collection {
	return d.db.Collection(name)
}

func (d *Database) Users() *mongo.Collection {
	return d.Collection(CollectionUsers)
}

func (d *Database) Projects() *mongo.Collection {
	return d.Collection(CollectionProjects)
}

func (d *Database) Commands() *mongo.Collection {
	return d.Collection(CollectionCommands)
}

func (d *Database) InfrastructureSpecs() *mongo.Collection {
	return d.Collection(CollectionInfrastructureSpecs)
}

func (d *Database) TerraformRuns() *mongo.Collection {
	return d.Collection(CollectionTerraformRuns)
}

func (d *Database) AuditLogs() *mongo.Collection {
	return d.Collection(CollectionAuditLogs)
}
