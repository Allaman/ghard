package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/allaman/ghard/internal/config"
)

// Integration tests that test the complete pipeline from file I/O to output
// These tests are slower and more brittle, but verify end-to-end functionality

func TestIntegrationListContacts(t *testing.T) {
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

	configContent := `[addressbook.test]
path = "` + contactsDir + `"
`
	configPath := filepath.Join(configDir, "ghard.toml")
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	vcardContent := `BEGIN:VCARD
VERSION:3.0
FN:Integration Test
N:Test;Integration;;;
EMAIL:test@example.com
TEL:+1111111111
ORG:Test Corp
NOTE:Integration test contact
ADR:;;Test Street;Test City;TS;12345;Test Country
END:VCARD`

	vcardPath := filepath.Join(contactsDir, "test.vcf")
	err = os.WriteFile(vcardPath, []byte(vcardContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write vCard file: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	app := New(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.ListContacts(false, false, []string{})
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Test, Integration") {
		t.Error("Expected contact name not found in output")
	}
	if !strings.Contains(output, "test@example.com") {
		t.Error("Expected email not found in output")
	}
	if strings.Contains(output, "Integration test contact") {
		t.Error("Note should not appear in short format")
	}
}

func TestIntegrationListBirthdays(t *testing.T) {
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

	configContent := `[addressbook.test]
path = "` + contactsDir + `"
`
	configPath := filepath.Join(configDir, "ghard.toml")
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Copy testdata files to temp directory
	testdataDir := "testdata"
	files := []string{"john.vcf", "jane.vcf", "alice.vcf"}

	for _, file := range files {
		srcPath := filepath.Join(testdataDir, file)
		dstPath := filepath.Join(contactsDir, file)

		srcData, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("Failed to read testdata file %s: %v", file, err)
		}

		err = os.WriteFile(dstPath, srcData, 0o644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", file, err)
		}
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	app := New(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.ListBirthdays(false, []string{})
	if err != nil {
		t.Fatalf("ListBirthdays failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain header
	if !strings.Contains(output, "Name\tBirthday") {
		t.Error("Expected header 'Name\tBirthday' in output")
	}

	// Should contain contacts with birthdays
	if !strings.Contains(output, "Doe, John\t01/15/1990") {
		t.Error("Expected 'Doe, John\t01/15/1990' in output")
	}

	if !strings.Contains(output, "Brown, Alice\t03/10/1992") {
		t.Error("Expected 'Brown, Alice\t03/10/1992' in output")
	}

	// Should NOT contain contacts without birthdays
	if strings.Contains(output, "Jane Smith") {
		t.Error("Jane Smith should not appear in birthday list (no birthday)")
	}

	// Test sorting by month - should be in order: January (John), March (Alice)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var birthdayLines []string
	for _, line := range lines {
		if strings.Contains(line, "\t") && !strings.Contains(line, "Name\tBirthday") {
			birthdayLines = append(birthdayLines, line)
		}
	}

	if len(birthdayLines) != 2 {
		t.Errorf("Expected 2 birthday lines, got %d", len(birthdayLines))
	} else {
		// Check ordering: January (01/15), March (03/10)
		if !strings.Contains(birthdayLines[0], "01/15/1990") {
			t.Errorf("Expected January birthday first, got: %s", birthdayLines[0])
		}
		if !strings.Contains(birthdayLines[1], "03/10/1992") {
			t.Errorf("Expected March birthday second, got: %s", birthdayLines[1])
		}
	}
}

func TestIntegrationExportContactsWithFilter(t *testing.T) {
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

	configContent := `[addressbook.test]
path = "` + contactsDir + `"
`
	configPath := filepath.Join(configDir, "ghard.toml")
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Copy a couple testdata files
	testdataDir := "testdata"
	files := []string{"john.vcf", "jane.vcf"}

	for _, file := range files {
		srcPath := filepath.Join(testdataDir, file)
		dstPath := filepath.Join(contactsDir, file)

		srcData, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("Failed to read testdata file %s: %v", file, err)
		}

		err = os.WriteFile(dstPath, srcData, 0o644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", file, err)
		}
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	app := New(cfg)

	// Test that the integration works by calling ExportContacts with filter
	tempFile := filepath.Join(tempDir, "export_test.csv")
	err = app.ExportContacts("csv", ",", tempFile, []string{"john"})
	if err != nil {
		t.Fatalf("ExportContacts failed: %v", err)
	}

	// Read and verify the exported file
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read exported file: %v", err)
	}

	output := string(content)
	if !strings.Contains(output, "John Doe") {
		t.Error("Expected John Doe in filtered export")
	}
	if strings.Contains(output, "Jane Smith") {
		t.Error("Jane Smith should not appear in filtered export")
	}
}
