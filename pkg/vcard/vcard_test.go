package vcard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full address",
			input:    ";;123 Main St;Springfield;IL;62701;USA",
			expected: "123 Main St, Springfield, IL, 62701, USA",
		},
		{
			name:     "address with empty fields",
			input:    ";;Amalienstraße 60;Neuburg;;86666;",
			expected: "Amalienstraße 60, Neuburg, 86666",
		},
		{
			name:     "minimal address",
			input:    ";;;Berlin;;;",
			expected: "Berlin",
		},
		{
			name:     "empty address",
			input:    ";;;;;;",
			expected: "",
		},
		{
			name:     "street and postal code only",
			input:    ";;Main Street;;;12345;",
			expected: "Main Street, 12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAddress(tt.input)
			if result != tt.expected {
				t.Errorf("formatAddress(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadContactsFromPath(t *testing.T) {
	tempDir := t.TempDir()

	vcard1Content := `BEGIN:VCARD
VERSION:3.0
FN:John Doe
EMAIL;TYPE=HOME:john@example.com
EMAIL;TYPE=WORK:john@work.com
TEL;TYPE=HOME:+1234567890
TEL;TYPE=CELL:+1234567891
ORG:Example Corp
NOTE:Test contact
ADR:;;123 Main St;Springfield;IL;62701;USA
END:VCARD`

	vcard1Path := filepath.Join(tempDir, "john.vcf")
	err := os.WriteFile(vcard1Path, []byte(vcard1Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test vCard file: %v", err)
	}

	vcard2Content := `BEGIN:VCARD
VERSION:3.0
FN:Jane Smith
EMAIL:jane@example.com
TEL:+0987654321
ORG:Another Corp
END:VCARD`

	vcard2Path := filepath.Join(tempDir, "jane.vcf")
	err = os.WriteFile(vcard2Path, []byte(vcard2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create second test vCard file: %v", err)
	}

	nonVcardPath := filepath.Join(tempDir, "readme.txt")
	err = os.WriteFile(nonVcardPath, []byte("This is not a vCard"), 0644)
	if err != nil {
		t.Fatalf("Failed to create non-vCard file: %v", err)
	}

	contacts, err := loadContactsFromPath(tempDir)
	if err != nil {
		t.Fatalf("Failed to load contacts: %v", err)
	}

	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(contacts))
	}

	var johnContact *Contact
	for _, contact := range contacts {
		if contact.Name == "John Doe" {
			johnContact = &contact
			break
		}
	}

	if johnContact == nil {
		t.Error("John Doe contact not found")
	} else {
		if len(johnContact.Emails) != 2 {
			t.Errorf("Expected 2 emails, got %d", len(johnContact.Emails))
		}
		if johnContact.GetPrimaryEmail() != "john@example.com (HOME)" {
			t.Errorf("Expected primary email 'john@example.com (HOME)', got %q", johnContact.GetPrimaryEmail())
		}
		if johnContact.Emails[0].Type != "HOME" {
			t.Errorf("Expected first email type 'HOME', got %q", johnContact.Emails[0].Type)
		}
		if johnContact.Emails[1].Value != "john@work.com" {
			t.Errorf("Expected second email 'john@work.com', got %q", johnContact.Emails[1].Value)
		}
		if johnContact.Emails[1].Type != "WORK" {
			t.Errorf("Expected second email type 'WORK', got %q", johnContact.Emails[1].Type)
		}

		if len(johnContact.Phones) != 2 {
			t.Errorf("Expected 2 phones, got %d", len(johnContact.Phones))
		}
		if johnContact.GetPrimaryPhone() != "+1234567890 (HOME)" {
			t.Errorf("Expected primary phone '+1234567890 (HOME)', got %q", johnContact.GetPrimaryPhone())
		}
		if johnContact.Phones[0].Type != "HOME" {
			t.Errorf("Expected first phone type 'HOME', got %q", johnContact.Phones[0].Type)
		}
		if johnContact.Phones[1].Value != "+1234567891" {
			t.Errorf("Expected second phone '+1234567891', got %q", johnContact.Phones[1].Value)
		}
		if johnContact.Phones[1].Type != "CELL" {
			t.Errorf("Expected second phone type 'CELL', got %q", johnContact.Phones[1].Type)
		}

		expectedEmailFormat := "john@example.com (HOME), john@work.com (WORK)"
		if johnContact.FormatEmails() != expectedEmailFormat {
			t.Errorf("Expected formatted emails %q, got %q", expectedEmailFormat, johnContact.FormatEmails())
		}

		expectedPhoneFormat := "+1234567890 (HOME), +1234567891 (CELL)"
		if johnContact.FormatPhones() != expectedPhoneFormat {
			t.Errorf("Expected formatted phones %q, got %q", expectedPhoneFormat, johnContact.FormatPhones())
		}

		if johnContact.Organization != "Example Corp" {
			t.Errorf("Expected organization 'Example Corp', got %q", johnContact.Organization)
		}
		if johnContact.Note != "Test contact" {
			t.Errorf("Expected note 'Test contact', got %q", johnContact.Note)
		}
		if johnContact.Address != "123 Main St, Springfield, IL, 62701, USA" {
			t.Errorf("Expected formatted address, got %q", johnContact.Address)
		}
	}

	var janeContact *Contact
	for _, contact := range contacts {
		if contact.Name == "Jane Smith" {
			janeContact = &contact
			break
		}
	}

	if janeContact == nil {
		t.Error("Jane Smith contact not found")
	} else {
		if len(janeContact.Emails) != 1 {
			t.Errorf("Expected 1 email, got %d", len(janeContact.Emails))
		}
		if janeContact.GetPrimaryEmail() != "jane@example.com" {
			t.Errorf("Expected email 'jane@example.com', got %q", janeContact.GetPrimaryEmail())
		}
		if len(janeContact.Phones) != 1 {
			t.Errorf("Expected 1 phone, got %d", len(janeContact.Phones))
		}
		if janeContact.GetPrimaryPhone() != "+0987654321" {
			t.Errorf("Expected phone '+0987654321', got %q", janeContact.GetPrimaryPhone())
		}
		if janeContact.Organization != "Another Corp" {
			t.Errorf("Expected organization 'Another Corp', got %q", janeContact.Organization)
		}
		if janeContact.Note != "" {
			t.Errorf("Expected empty note, got %q", janeContact.Note)
		}
		if janeContact.Address != "" {
			t.Errorf("Expected empty address, got %q", janeContact.Address)
		}
	}
}

func TestLoadContactsFromNonexistentPath(t *testing.T) {
	nonexistentPath := "/path/that/does/not/exist"

	_, err := loadContactsFromPath(nonexistentPath)
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}

	if !strings.Contains(err.Error(), "path does not exist") {
		t.Errorf("Expected path does not exist error, got: %v", err)
	}
}

func TestLoadContactsFromEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	contacts, err := loadContactsFromPath(tempDir)
	if err == nil {
		t.Error("Expected error for directory with no .vcf files")
	}
	if !strings.Contains(err.Error(), "no .vcf files found") {
		t.Errorf("Expected 'no .vcf files found' error, got: %v", err)
	}

	if len(contacts) != 0 {
		t.Errorf("Expected 0 contacts from empty directory, got %d", len(contacts))
	}
}

func TestLoadContacts(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	vcard1Content := `BEGIN:VCARD
VERSION:3.0
FN:Contact One
EMAIL:one@example.com
END:VCARD`

	vcard1Path := filepath.Join(tempDir1, "contact1.vcf")
	err := os.WriteFile(vcard1Path, []byte(vcard1Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create first vCard file: %v", err)
	}

	vcard2Content := `BEGIN:VCARD
VERSION:3.0
FN:Contact Two
EMAIL:two@example.com
END:VCARD`

	vcard2Path := filepath.Join(tempDir2, "contact2.vcf")
	err = os.WriteFile(vcard2Path, []byte(vcard2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create second vCard file: %v", err)
	}

	contacts, err := LoadContacts([]string{tempDir1, tempDir2})
	if err != nil {
		t.Fatalf("Failed to load contacts: %v", err)
	}

	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts from two directories, got %d", len(contacts))
	}

	names := make(map[string]bool)
	for _, contact := range contacts {
		names[contact.Name] = true
	}

	if !names["Contact One"] {
		t.Error("Contact One not found")
	}
	if !names["Contact Two"] {
		t.Error("Contact Two not found")
	}
}

func TestParseBirthday(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectValid   bool
		expectedMonth time.Month
		expectedDay   int
		expectedYear  int // 0 means don't check year (for year-unknown formats)
	}{
		{
			name:          "ISO date format",
			input:         "1990-01-15",
			expectValid:   true,
			expectedMonth: time.January,
			expectedDay:   15,
			expectedYear:  1990,
		},
		{
			name:          "compact date format",
			input:         "19901015",
			expectValid:   true,
			expectedMonth: time.October,
			expectedDay:   15,
			expectedYear:  1990,
		},
		{
			name:          "year unknown format",
			input:         "--03-25",
			expectValid:   true,
			expectedMonth: time.March,
			expectedDay:   25,
			expectedYear:  0, // Will be current year, don't check
		},
		{
			name:          "year unknown compact format",
			input:         "--1205",
			expectValid:   true,
			expectedMonth: time.December,
			expectedDay:   5,
			expectedYear:  0, // Will be current year, don't check
		},
		{
			name:          "ISO 8601 with time",
			input:         "1985-12-25T10:30:00Z",
			expectValid:   true,
			expectedMonth: time.December,
			expectedDay:   25,
			expectedYear:  1985,
		},
		{
			name:        "empty input",
			input:       "",
			expectValid: false,
		},
		{
			name:        "invalid format",
			input:       "not-a-date",
			expectValid: false,
		},
		{
			name:          "whitespace input",
			input:         "  1992-06-30  ",
			expectValid:   true,
			expectedMonth: time.June,
			expectedDay:   30,
			expectedYear:  1992,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBirthday(tt.input)
			
			if tt.expectValid {
				if result.IsZero() {
					t.Errorf("parseBirthday(%q) expected valid time, got zero time", tt.input)
				} else {
					if result.Month() != tt.expectedMonth {
						t.Errorf("parseBirthday(%q) month = %v, want %v", tt.input, result.Month(), tt.expectedMonth)
					}
					if result.Day() != tt.expectedDay {
						t.Errorf("parseBirthday(%q) day = %d, want %d", tt.input, result.Day(), tt.expectedDay)
					}
					if tt.expectedYear != 0 && result.Year() != tt.expectedYear {
						t.Errorf("parseBirthday(%q) year = %d, want %d", tt.input, result.Year(), tt.expectedYear)
					}
				}
			} else {
				if !result.IsZero() {
					t.Errorf("parseBirthday(%q) expected zero time, got %v", tt.input, result)
				}
			}
		})
	}
}

func TestFormatBirthdayForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "standard date with year",
			input:    time.Date(1990, time.January, 15, 0, 0, 0, 0, time.UTC),
			expected: "01/15/1990",
		},
		{
			name:     "current year date (year-unknown heuristic)",
			input:    time.Date(time.Now().Year(), time.March, 25, 0, 0, 0, 0, time.UTC),
			expected: "03/25",
		},
		{
			name:     "zero time",
			input:    time.Time{},
			expected: "",
		},
		{
			name:     "December date",
			input:    time.Date(1985, time.December, 25, 0, 0, 0, 0, time.UTC),
			expected: "12/25/1985",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBirthdayForDisplay(tt.input)
			if result != tt.expected {
				t.Errorf("FormatBirthdayForDisplay(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
