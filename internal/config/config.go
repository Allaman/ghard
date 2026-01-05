// Package config provides configuration handling for ghard
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
)

const (
	ConfigDirName  = ".config"
	AppName        = "ghard"
	ConfigFileName = "ghard.toml"
)

type ConfigNotFoundError struct {
	Path string
}

func (e *ConfigNotFoundError) Error() string {
	return fmt.Sprintf("config file not found at %s", e.Path)
}

type ConfigValidationError struct {
	Underlying error
}

func (e *ConfigValidationError) Error() string {
	return fmt.Sprintf("configuration validation failed: %v", e.Underlying)
}

func (e *ConfigValidationError) Unwrap() error {
	return e.Underlying
}

type ConfigParseError struct {
	Underlying error
}

func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("failed to parse config file: %v", e.Underlying)
}

func (e *ConfigParseError) Unwrap() error {
	return e.Underlying
}

const (
	configNotFoundTemplate = `Configuration Error: {{.Error}}

To get started, create a configuration file at:
  {{.ConfigPath}}

Example configuration:
  [addressbook.personal]
  path = "~/contacts/personal"

  [addressbook.work]
  path = "/path/to/work/contacts"

Make sure the contact directories exist and contain .vcf files.
`

	validationFailedTemplate = `Configuration Error: {{.Error}}

Troubleshooting tips:
  • Ensure all addressbook paths exist
  • Check that paths are directories, not files
  • Verify you have read access to the directories
  • Use absolute paths or ~ for home directory

Use --debug for more detailed error information.
`

	parseFailedTemplate = `Configuration Error: {{.Error}}

The configuration file has invalid TOML syntax.
Please check the file format and fix any syntax errors.

Example of correct format:
  [addressbook.name]
  path = "/path/to/contacts"
`

	defaultErrorTemplate = `Configuration Error: {{.Error}}
Use --debug for more detailed error information.
`
)

// Config represents the application configuration
type Config struct {
	// AddressBooks maps addressbook names to their configurations
	AddressBooks map[string]AddressBook `toml:"addressbook"`
}

// AddressBook represents a single address book configuration
type AddressBook struct {
	// Path is the filesystem path to the directory containing vCard files
	Path string `toml:"path"`
}

func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ConfigDirName, AppName, ConfigFileName)
	slog.Debug("Looking for config file", "path", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, &ConfigNotFoundError{Path: configPath}
	}

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, &ConfigParseError{Underlying: err}
	}

	slog.Debug("Config loaded", "addressbooks", len(config.AddressBooks))
	for name, ab := range config.AddressBooks {
		slog.Debug("Address book", "name", name, "path", ab.Path)
	}

	if err := config.Validate(); err != nil {
		return nil, &ConfigValidationError{Underlying: err}
	}

	return &config, nil
}

// Validate checks the configuration for common issues and provides helpful error messages
func (c *Config) Validate() error {
	if len(c.AddressBooks) == 0 {
		return errors.New("no address books configured. Please add at least one [addressbook.name] section to your config file")
	}

	var validationErrors []string

	for name, ab := range c.AddressBooks {
		if err := c.validateAddressBook(name, ab); err != nil {
			validationErrors = append(validationErrors, err.Error())
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("address book validation errors:\n  - %s", strings.Join(validationErrors, "\n  - "))
	}

	return nil
}

// validateAddressBook validates a single address book configuration
func (c *Config) validateAddressBook(name string, ab AddressBook) error {
	if ab.Path == "" {
		return fmt.Errorf("address book '%s': path cannot be empty", name)
	}

	expandedPath := expandPath(ab.Path)

	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return fmt.Errorf("address book '%s': path does not exist: %s (expanded from: %s)", name, expandedPath, ab.Path)
	} else if err != nil {
		return fmt.Errorf("address book '%s': cannot access path %s: %w", name, expandedPath, err)
	}

	if info, err := os.Stat(expandedPath); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("address book '%s': path must be a directory, got file: %s", name, expandedPath)
		}
	}

	file, err := os.Open(expandedPath)
	if err != nil {
		return fmt.Errorf("address book '%s': directory is not readable: %s (%w)", name, expandedPath, err)
	}
	file.Close()

	return nil
}

func (c *Config) GetAddressBookPaths() []string {
	var paths []string
	for _, ab := range c.AddressBooks {
		path := expandPath(ab.Path)
		paths = append(paths, path)
	}
	return paths
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[2:])
		}
		// Log the error but continue with unexpanded path
		slog.Warn("Failed to get home directory for path expansion", "path", path, "error", err)
	}
	return path
}

// ErrorTemplateData holds data for error message templates
type ErrorTemplateData struct {
	Error      error
	ConfigPath string
}

// HandleError provides user-friendly error messages and guidance for configuration issues
func HandleError(err error) {
	var configNotFoundErr *ConfigNotFoundError
	var configValidationErr *ConfigValidationError
	var configParseErr *ConfigParseError

	switch {
	case errors.As(err, &configNotFoundErr):
		data := ErrorTemplateData{
			Error:      err,
			ConfigPath: configNotFoundErr.Path,
		}
		executeTemplate(configNotFoundTemplate, data)

	case errors.As(err, &configValidationErr):
		data := ErrorTemplateData{Error: err}
		executeTemplate(validationFailedTemplate, data)

	case errors.As(err, &configParseErr):
		data := ErrorTemplateData{Error: err}
		executeTemplate(parseFailedTemplate, data)

	default:
		// Fallback for other config errors
		data := ErrorTemplateData{Error: err}
		executeTemplate(defaultErrorTemplate, data)
	}
}

// getConfigPath returns the expected configuration file path
func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ConfigDirName, AppName, ConfigFileName)
}

// executeTemplate executes a template with the given data and writes to stderr
func executeTemplate(tmplStr string, data ErrorTemplateData) {
	tmpl, err := template.New("error").Parse(tmplStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration Error: %v\n", data.Error)
		return
	}

	if err := tmpl.Execute(os.Stderr, data); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration Error: %v\n", data.Error)
	}
}
