package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "expands tilde path",
			input:    "~/Documents/contacts",
			expected: filepath.Join(os.Getenv("HOME"), "Documents/contacts"),
		},
		{
			name:     "leaves absolute path unchanged",
			input:    "/home/user/contacts",
			expected: "/home/user/contacts",
		},
		{
			name:     "leaves relative path unchanged",
			input:    "contacts",
			expected: "contacts",
		},
		{
			name:     "handles tilde only",
			input:    "~/",
			expected: os.Getenv("HOME"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetAddressBookPaths(t *testing.T) {
	config := &Config{
		AddressBooks: map[string]AddressBook{
			"personal": {Path: "~/contacts/personal"},
			"work":     {Path: "/opt/contacts/work"},
		},
	}

	paths := config.GetAddressBookPaths()

	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}

	homeDir := os.Getenv("HOME")
	expectedPersonal := filepath.Join(homeDir, "contacts/personal")

	if !slices.Contains(paths, expectedPersonal) {
		t.Errorf("Expected expanded path %q not found in paths %v", expectedPersonal, paths)
	}
}

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "ghard")
	personalDir := filepath.Join(tempDir, "contacts", "personal")
	workDir := filepath.Join(tempDir, "work")

	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	err = os.MkdirAll(personalDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create personal contacts directory: %v", err)
	}

	err = os.MkdirAll(workDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	configPath := filepath.Join(configDir, "ghard.toml")
	configContent := `[addressbook.personal]
path = "~/contacts/personal"

[addressbook.work]
path = "` + workDir + `"
`

	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.AddressBooks) != 2 {
		t.Errorf("Expected 2 address books, got %d", len(config.AddressBooks))
	}

	personal, exists := config.AddressBooks["personal"]
	if !exists {
		t.Error("Personal address book not found")
	}
	if personal.Path != "~/contacts/personal" {
		t.Errorf("Expected personal path '~/contacts/personal', got %q", personal.Path)
	}

	work, exists := config.AddressBooks["work"]
	if !exists {
		t.Error("Work address book not found")
	}
	if work.Path != workDir {
		t.Errorf("Expected work path '%s', got %q", workDir, work.Path)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	tempDir := t.TempDir()

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	_, err := Load()
	if err == nil {
		t.Error("Expected error when config file doesn't exist")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("Expected config file not found error, got: %v", err)
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("valid config with existing directories", func(t *testing.T) {
		tempDir := t.TempDir()
		contactsDir := filepath.Join(tempDir, "contacts")
		err := os.MkdirAll(contactsDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create contacts directory: %v", err)
		}

		config := &Config{
			AddressBooks: map[string]AddressBook{
				"test": {Path: contactsDir},
			},
		}

		if err := config.Validate(); err != nil {
			t.Errorf("Expected valid config to pass validation, got: %v", err)
		}
	})

	t.Run("empty address books", func(t *testing.T) {
		config := &Config{
			AddressBooks: map[string]AddressBook{},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected error for empty address books")
		}
		if !strings.Contains(err.Error(), "no address books configured") {
			t.Errorf("Expected 'no address books configured' error, got: %v", err)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		config := &Config{
			AddressBooks: map[string]AddressBook{
				"test": {Path: ""},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected error for empty path")
		}
		if !strings.Contains(err.Error(), "path cannot be empty") {
			t.Errorf("Expected 'path cannot be empty' error, got: %v", err)
		}
	})

	t.Run("nonexistent path", func(t *testing.T) {
		config := &Config{
			AddressBooks: map[string]AddressBook{
				"test": {Path: "/nonexistent/path"},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected error for nonexistent path")
		}
		if !strings.Contains(err.Error(), "path does not exist") {
			t.Errorf("Expected 'path does not exist' error, got: %v", err)
		}
	})

	t.Run("path is file not directory", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "notdir.txt")
		err := os.WriteFile(filePath, []byte("test"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := &Config{
			AddressBooks: map[string]AddressBook{
				"test": {Path: filePath},
			},
		}

		err = config.Validate()
		if err == nil {
			t.Error("Expected error for file instead of directory")
		}
		if !strings.Contains(err.Error(), "path must be a directory") {
			t.Errorf("Expected 'path must be a directory' error, got: %v", err)
		}
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		config := &Config{
			AddressBooks: map[string]AddressBook{
				"empty":    {Path: ""},
				"nonexist": {Path: "/nonexistent/path"},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected error for multiple validation issues")
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "path cannot be empty") {
			t.Error("Expected empty path error in combined error message")
		}
		if !strings.Contains(errStr, "path does not exist") {
			t.Error("Expected nonexistent path error in combined error message")
		}
	})

	t.Run("tilde expansion in validation", func(t *testing.T) {
		tempDir := t.TempDir()
		contactsDir := filepath.Join(tempDir, "contacts")
		err := os.MkdirAll(contactsDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create contacts directory: %v", err)
		}

		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", originalHome)

		config := &Config{
			AddressBooks: map[string]AddressBook{
				"test": {Path: "~/contacts"},
			},
		}

		if err := config.Validate(); err != nil {
			t.Errorf("Expected tilde expansion to work in validation, got: %v", err)
		}
	})
}

func TestLoadConfigWithValidation(t *testing.T) {
	t.Run("config with nonexistent path fails validation", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "ghard")
		err := os.MkdirAll(configDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		configPath := filepath.Join(configDir, "ghard.toml")
		configContent := `[addressbook.test]
path = "/nonexistent/path"
`

		err = os.WriteFile(configPath, []byte(configContent), 0o644)
		if err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", originalHome)

		_, err = Load()
		if err == nil {
			t.Error("Expected Load() to fail with validation error")
		}
		if !strings.Contains(err.Error(), "configuration validation failed") {
			t.Errorf("Expected configuration validation error, got: %v", err)
		}
	})

	t.Run("config with valid paths passes validation", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "ghard")
		contactsDir := filepath.Join(tempDir, "contacts")

		err := os.MkdirAll(configDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		err = os.MkdirAll(contactsDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create contacts directory: %v", err)
		}

		configPath := filepath.Join(configDir, "ghard.toml")
		configContent := `[addressbook.test]
path = "` + contactsDir + `"
`

		err = os.WriteFile(configPath, []byte(configContent), 0o644)
		if err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", originalHome)

		config, err := Load()
		if err != nil {
			t.Errorf("Expected Load() to succeed with valid config, got: %v", err)
		}
		if config == nil {
			t.Error("Expected config to be loaded")
		}
	})
}

func TestHandleError(t *testing.T) {
	// Capture stderr output for testing
	captureStderr := func(fn func()) string {
		// Create a pipe to capture stderr
		r, w, _ := os.Pipe()
		oldStderr := os.Stderr
		os.Stderr = w

		fn()

		w.Close()
		os.Stderr = oldStderr

		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		return string(buf[:n])
	}

	t.Run("config file not found error", func(t *testing.T) {
		err := &ConfigNotFoundError{Path: "/some/path"}

		output := captureStderr(func() {
			HandleError(err)
		})

		if !strings.Contains(output, "Configuration Error:") {
			t.Error("Expected 'Configuration Error:' in output")
		}
		if !strings.Contains(output, "To get started, create a configuration file at:") {
			t.Error("Expected configuration creation instructions")
		}
		if !strings.Contains(output, "[addressbook.personal]") {
			t.Error("Expected example configuration")
		}
		if !strings.Contains(output, "Make sure the contact directories exist") {
			t.Error("Expected directory creation instructions")
		}
		// Verify that the actual config path from the error is used
		if !strings.Contains(output, "/some/path") {
			t.Error("Expected the config path from the error to be included")
		}
	})

	t.Run("configuration validation failed error", func(t *testing.T) {
		underlyingErr := fmt.Errorf("some validation error")
		err := &ConfigValidationError{Underlying: underlyingErr}

		output := captureStderr(func() {
			HandleError(err)
		})

		if !strings.Contains(output, "Configuration Error:") {
			t.Error("Expected 'Configuration Error:' in output")
		}
		if !strings.Contains(output, "Troubleshooting tips:") {
			t.Error("Expected troubleshooting tips")
		}
		if !strings.Contains(output, "Ensure all addressbook paths exist") {
			t.Error("Expected path existence tip")
		}
		if !strings.Contains(output, "Use --debug for more detailed") {
			t.Error("Expected debug instruction")
		}
	})

	t.Run("failed to parse config file error", func(t *testing.T) {
		underlyingErr := fmt.Errorf("invalid TOML syntax")
		err := &ConfigParseError{Underlying: underlyingErr}

		output := captureStderr(func() {
			HandleError(err)
		})

		if !strings.Contains(output, "Configuration Error:") {
			t.Error("Expected 'Configuration Error:' in output")
		}
		if !strings.Contains(output, "invalid TOML syntax") {
			t.Error("Expected TOML syntax error message")
		}
		if !strings.Contains(output, "Example of correct format:") {
			t.Error("Expected format example")
		}
		if !strings.Contains(output, "[addressbook.name]") {
			t.Error("Expected TOML format example")
		}
	})

	t.Run("default error fallback", func(t *testing.T) {
		err := fmt.Errorf("some unknown configuration error")

		output := captureStderr(func() {
			HandleError(err)
		})

		if !strings.Contains(output, "Configuration Error:") {
			t.Error("Expected 'Configuration Error:' in output")
		}
		if !strings.Contains(output, "Use --debug for more detailed") {
			t.Error("Expected debug instruction")
		}
		if !strings.Contains(output, "some unknown configuration error") {
			t.Error("Expected original error message")
		}
	})
}

func TestExecuteTemplate(t *testing.T) {
	// Capture stderr output for testing
	captureStderr := func(fn func()) string {
		r, w, _ := os.Pipe()
		oldStderr := os.Stderr
		os.Stderr = w

		fn()

		w.Close()
		os.Stderr = oldStderr

		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		return string(buf[:n])
	}

	t.Run("valid template execution", func(t *testing.T) {
		template := "Error: {{.Error}}\nPath: {{.ConfigPath}}"
		data := ErrorTemplateData{
			Error:      fmt.Errorf("test error"),
			ConfigPath: "/test/path",
		}

		output := captureStderr(func() {
			executeTemplate(template, data)
		})

		if !strings.Contains(output, "Error: test error") {
			t.Error("Expected error message in template output")
		}
		if !strings.Contains(output, "Path: /test/path") {
			t.Error("Expected config path in template output")
		}
	})

	t.Run("invalid template fallback", func(t *testing.T) {
		template := "Invalid template {{.NonExistentField" // Missing closing brace
		data := ErrorTemplateData{
			Error: fmt.Errorf("test error"),
		}

		output := captureStderr(func() {
			executeTemplate(template, data)
		})

		if !strings.Contains(output, "Configuration Error: test error") {
			t.Error("Expected fallback error message")
		}
	})
}

func TestGetConfigPath(t *testing.T) {
	path := getConfigPath()

	if !strings.Contains(path, ConfigDirName) {
		t.Errorf("Expected config path to contain %s, got %s", ConfigDirName, path)
	}
	if !strings.Contains(path, AppName) {
		t.Errorf("Expected config path to contain %s, got %s", AppName, path)
	}
	if !strings.Contains(path, ConfigFileName) {
		t.Errorf("Expected config path to contain %s, got %s", ConfigFileName, path)
	}
}
