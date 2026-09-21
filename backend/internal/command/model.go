package command

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	SourceText  = "text"
	SourceVoice = "voice"
)

const (
	StatusReceived   = "received"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

const MaxInputLength = 10000

type Command struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ProjectID bson.ObjectID `bson:"project_id" json:"project_id"`
	UserID    bson.ObjectID `bson:"user_id" json:"user_id"`
	Input     string        `bson:"input" json:"input"`
	Source    string        `bson:"source" json:"source"`
	Status    string        `bson:"status" json:"status"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

type CreateInput struct {
	Input  string
	Source string
}

func (c *Command) Normalize() {
	c.Input = strings.TrimSpace(c.Input)
	c.Source = strings.ToLower(strings.TrimSpace(c.Source))
	c.Status = strings.ToLower(strings.TrimSpace(c.Status))
}

func (c *Command) Validate() error {
	if c == nil {
		return fmt.Errorf("command is required")
	}

	if c.ProjectID.IsZero() {
		return fmt.Errorf("project is required")
	}

	if c.UserID.IsZero() {
		return fmt.Errorf("user is required")
	}

	if c.Input == "" {
		return fmt.Errorf("command input is required")
	}

	if len([]byte(c.Input)) > MaxInputLength {
		return fmt.Errorf(
			"command input must not exceed %d bytes",
			MaxInputLength,
		)
	}

	if c.Source != SourceText && c.Source != SourceVoice {
		return fmt.Errorf("command source must be text or voice")
	}

	if c.Status != StatusReceived &&
		c.Status != StatusProcessing &&
		c.Status != StatusCompleted &&
		c.Status != StatusFailed {
		return fmt.Errorf("invalid command status")
	}

	return nil
}
