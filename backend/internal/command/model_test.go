package command

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestCommandValidateValidCommand(t *testing.T) {
	currentCommand := &Command{
		ProjectID: bson.NewObjectID(),
		UserID:    bson.NewObjectID(),
		Input:     "Create a VPC with a public subnet.",
		Source:    SourceText,
		Status:    StatusReceived,
	}

	currentCommand.Normalize()

	if err := currentCommand.Validate(); err != nil {
		t.Fatalf(
			"expected valid command, got: %v",
			err,
		)
	}
}

func TestCommandValidateRejectsEmptyInput(t *testing.T) {
	currentCommand := &Command{
		ProjectID: bson.NewObjectID(),
		UserID:    bson.NewObjectID(),
		Source:    SourceText,
		Status:    StatusReceived,
	}

	currentCommand.Normalize()

	err := currentCommand.Validate()

	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(
		err.Error(),
		"command input is required",
	) {
		t.Fatalf(
			"expected command input validation error, got: %v",
			err,
		)
	}
}

func TestCommandValidateRejectsUnsupportedSource(t *testing.T) {
	currentCommand := &Command{
		ProjectID: bson.NewObjectID(),
		UserID:    bson.NewObjectID(),
		Input:     "Create an S3 bucket.",
		Source:    "email",
		Status:    StatusReceived,
	}

	currentCommand.Normalize()

	err := currentCommand.Validate()

	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(
		err.Error(),
		"command source must be text or voice",
	) {
		t.Fatalf(
			"expected source validation error, got: %v",
			err,
		)
	}
}

func TestCommandValidateRejectsOversizedInput(t *testing.T) {
	currentCommand := &Command{
		ProjectID: bson.NewObjectID(),
		UserID:    bson.NewObjectID(),
		Input:     strings.Repeat("a", MaxInputLength+1),
		Source:    SourceText,
		Status:    StatusReceived,
	}

	currentCommand.Normalize()

	err := currentCommand.Validate()

	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(
		err.Error(),
		"must not exceed",
	) {
		t.Fatalf(
			"expected input length validation error, got: %v",
			err,
		)
	}
}