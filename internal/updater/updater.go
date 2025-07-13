package updater

import (
	"context"
	"fmt"
	"os"

	"github.com/creativeprojects/go-selfupdate"
)

const repositorySlug = "allaman/ghard"

// Update performs a self-update of the ghard binary to the latest version
func Update() error {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{
		APIToken: os.Getenv("GITHUB_TOKEN"), // Optional: improves rate limiting
	})
	if err != nil {
		return fmt.Errorf("failed to create GitHub source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source: source,
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	fmt.Println("Checking for updates...")

	repository := selfupdate.ParseSlug(repositorySlug)
	latest, found, err := updater.DetectLatest(context.Background(), repository)
	if err != nil {
		return fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		fmt.Println("No releases found")
		return nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("could not locate executable path: %w", err)
	}

	err = updater.UpdateTo(context.Background(), latest, exe)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("Update completed successfully to version %s!\n", latest.Version())
	return nil
}
