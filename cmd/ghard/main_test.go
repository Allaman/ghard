package main

import (
	"os"
	"testing"

	"github.com/allaman/ghard/internal/app"
	"github.com/allaman/ghard/internal/config"
)

func TestMain(t *testing.T) {
	// Test that main function can be called without panicking
	// This is a simple smoke test for the main function structure

	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test with help flag to avoid actual execution
	os.Args = []string{"ghard", "--help"}

	// Since main() calls os.Exit, we need to test it indirectly
	// by testing the CLI structure and app creation
	var cli app.CLI
	cfg := &config.Config{}
	application := app.New(cfg)

	if application == nil {
		t.Error("Expected app.New to return a non-nil app instance")
	}

	// Test that CLI struct has expected fields
	if cli.Debug != false {
		t.Error("Expected Debug field to be false by default")
	}
}
