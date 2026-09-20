package project

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

type Project struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	OwnerID     bson.ObjectID `bson:"owner_id" json:"owner_id"`
	Name        string        `bson:"name" json:"name"`
	Description string        `bson:"description,omitempty" json:"description,omitempty"`
	Status      string        `bson:"status" json:"status"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at" json:"updated_at"`
}

func (p *Project) Normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
}

func (p *Project) IsActive() bool {
	return p.Status == StatusActive
}
