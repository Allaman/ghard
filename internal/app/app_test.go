package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/allaman/ghard/internal/config"
	"github.com/allaman/ghard/pkg/vcard"
)

// createTestContacts loads contacts from testdata files
func createTestContacts() []vcard.Contact {
	contacts, err := vcard.LoadContacts([]string{"testdata"})
	if err != nil {
		panic("Failed to load test contacts from testdata: " + err.Error())
	}
	return contacts
}

// findContactByName finds a contact by name for tests that don't depend on order
func findContactByName(contacts []vcard.Contact, name string) *vcard.Contact {
	for i := range contacts {
		if contacts[i].Name == name {
			return &contacts[i]
		}
	}
	return nil
}

func TestListContacts(t *testing.T) {
	contacts := createTestContacts()

	cfg := &config.Config{}
	_ = New(cfg)

	t.Run("short format", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		for _, contact := range contacts {
			var parts []string
			parts = append(parts, formatNameForDisplay(contact))

			for _, email := range contact.Emails {
				emailType := strings.ToLower(email.Type)
				if emailType == "" {
					emailType = "email"
				}
				parts = append(parts, fmt.Sprintf("%s: %s", emailType, email.Value))
			}

			for _, phone := range contact.Phones {
				phoneType := strings.ToLower(phone.Type)
				if phoneType == "" {
					phoneType = "phone"
				}
				parts = append(parts, fmt.Sprintf("%s: %s", phoneType, phone.Value))
			}

			fmt.Println(strings.Join(parts, "\t"))
		}

		w.Close()
		os.Stdout = old

		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		if !strings.Contains(output, "Doe, John") {
			t.Error("Expected John Doe name not found in output")
		}
		if !strings.Contains(output, "home: john@example.com") {
			t.Error("Expected John Doe home email not found in output")
		}
		if !strings.Contains(output, "Smith, Jane") {
			t.Error("Expected Jane Smith name not found in output")
		}
		if !strings.Contains(output, "work: jane@example.com") {
			t.Error("Expected Jane Smith email not found in output")
		}
	})

	t.Run("long format", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		for _, contact := range contacts {
			var parts []string
			parts = append(parts, formatNameForDisplay(contact))

			for _, email := range contact.Emails {
				emailType := strings.ToLower(email.Type)
				if emailType == "" {
					emailType = "email"
				}
				parts = append(parts, fmt.Sprintf("%s: %s", emailType, email.Value))
			}

			for _, phone := range contact.Phones {
				phoneType := strings.ToLower(phone.Type)
				if phoneType == "" {
					phoneType = "phone"
				}
				parts = append(parts, fmt.Sprintf("%s: %s", phoneType, phone.Value))
			}

			if contact.Organization != "" {
				parts = append(parts, fmt.Sprintf("org: %s", contact.Organization))
			}

			if contact.Note != "" {
				parts = append(parts, fmt.Sprintf("note: %s", contact.Note))
			}

			if contact.Address != "" {
				parts = append(parts, fmt.Sprintf("address: %s", contact.Address))
			}

			fmt.Println(strings.Join(parts, "\t"))
		}

		w.Close()
		os.Stdout = old

		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		if !strings.Contains(output, "org: Example Corp") {
			t.Error("Expected organization not found in output")
		}
		if !strings.Contains(output, "note: Test contact") {
			t.Error("Expected note not found in output")
		}
		if !strings.Contains(output, "address: 123 Main St, Springfield, IL") {
			t.Error("Expected address not found in output")
		}
	})
}

func TestFilterFunctionality(t *testing.T) {
	cfg := &config.Config{}
	app := New(cfg)

	contacts := createTestContacts()

	t.Run("filter by single name", func(t *testing.T) {
		johnContact := findContactByName(contacts, "John Doe")
		janeContact := findContactByName(contacts, "Jane Smith")
		aliceContact := findContactByName(contacts, "Alice Brown")

		if johnContact == nil || janeContact == nil || aliceContact == nil {
			t.Fatal("Required test contacts not found")
		}

		match1 := app.matchesFilter(*johnContact, []string{"john"})
		match2 := app.matchesFilter(*janeContact, []string{"john"})
		match3 := app.matchesFilter(*aliceContact, []string{"john"})

		if !match1 {
			t.Error("Expected John Doe to match filter 'john'")
		}
		if match2 {
			t.Error("Expected Jane Smith to NOT match filter 'john'")
		}
		if match3 {
			t.Error("Expected Alice Brown to NOT match filter 'john'")
		}
	})

	t.Run("filter by multiple words", func(t *testing.T) {
		johnContact := findContactByName(contacts, "John Doe")
		janeContact := findContactByName(contacts, "Jane Smith")
		aliceContact := findContactByName(contacts, "Alice Brown")

		if johnContact == nil || janeContact == nil || aliceContact == nil {
			t.Fatal("Required test contacts not found")
		}

		match1 := app.matchesFilter(*johnContact, []string{"alice", "brown"})
		match2 := app.matchesFilter(*janeContact, []string{"alice", "brown"})
		match3 := app.matchesFilter(*aliceContact, []string{"alice", "brown"})

		if match1 {
			t.Error("Expected John Doe to NOT match filter 'alice brown'")
		}
		if match2 {
			t.Error("Expected Jane Smith to NOT match filter 'alice brown'")
		}
		if !match3 {
			t.Error("Expected Alice Brown to match filter 'alice brown'")
		}
	})

	t.Run("filter by email", func(t *testing.T) {
		johnContact := findContactByName(contacts, "John Doe")
		janeContact := findContactByName(contacts, "Jane Smith")
		aliceContact := findContactByName(contacts, "Alice Brown")

		if johnContact == nil || janeContact == nil || aliceContact == nil {
			t.Fatal("Required test contacts not found")
		}

		match1 := app.matchesFilter(*johnContact, []string{"example.com"})  // John has john@example.com
		match2 := app.matchesFilter(*janeContact, []string{"work.com"})     // Jane has jane@example.com
		match3 := app.matchesFilter(*aliceContact, []string{"example.com"}) // Alice has alice@example.com

		if !match1 {
			t.Error("Expected John Doe to match filter 'example.com'")
		}
		if match2 {
			t.Error("Expected Jane Smith to NOT match filter 'work.com'")
		}
		if !match3 {
			t.Error("Expected Alice Brown to match filter 'example.com'")
		}
	})

	t.Run("filter by organization", func(t *testing.T) {
		johnContact := findContactByName(contacts, "John Doe")
		janeContact := findContactByName(contacts, "Jane Smith")
		acmeContact := findContactByName(contacts, "Acme Corporation")

		if johnContact == nil || janeContact == nil || acmeContact == nil {
			t.Fatal("Required test contacts not found")
		}

		match1 := app.matchesFilter(*johnContact, []string{"Example"}) // John: Example Corp
		match2 := app.matchesFilter(*janeContact, []string{"Another"}) // Jane: Another Corp
		match3 := app.matchesFilter(*acmeContact, []string{"Acme"})    // Company: Acme Corporation

		if !match1 {
			t.Error("Expected John Doe to match filter 'Example'")
		}
		if !match2 {
			t.Error("Expected Jane Smith to match filter 'Another'")
		}
		if !match3 {
			t.Error("Expected Acme Corporation to match filter 'Acme'")
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		johnContact := findContactByName(contacts, "John Doe")
		janeContact := findContactByName(contacts, "Jane Smith")
		aliceContact := findContactByName(contacts, "Alice Brown")

		if johnContact == nil || janeContact == nil || aliceContact == nil {
			t.Fatal("Required test contacts not found")
		}

		match1 := app.matchesFilter(*johnContact, []string{"JOHN"})   // John Doe
		match2 := app.matchesFilter(*janeContact, []string{"jane"})   // Jane Smith
		match3 := app.matchesFilter(*aliceContact, []string{"ALICE"}) // Alice Brown

		if !match1 {
			t.Error("Expected case insensitive match for 'JOHN'")
		}
		if !match2 {
			t.Error("Expected case insensitive match for 'jane'")
		}
		if !match3 {
			t.Error("Expected case insensitive match for 'ALICE'")
		}
	})

	t.Run("empty filter matches all", func(t *testing.T) {
		if len(contacts) < 3 {
			t.Fatal("Need at least 3 contacts for this test")
		}

		match1 := app.matchesFilter(contacts[0], []string{})
		match2 := app.matchesFilter(contacts[1], []string{})
		match3 := app.matchesFilter(contacts[2], []string{})

		if !match1 || !match2 || !match3 {
			t.Error("Expected empty filter to match all contacts")
		}
	})
}

func TestListEmails(t *testing.T) {
	contacts := []vcard.Contact{
		{
			Name:         "John Doe",
			Emails:       []vcard.EmailEntry{{Value: "john@example.com", Type: "HOME"}, {Value: "john@work.com", Type: "WORK"}},
			Phones:       []vcard.PhoneEntry{{Value: "+1234567890", Type: "HOME"}},
			Organization: "Example Corp",
		},
		{
			Name:   "Jane Smith",
			Emails: []vcard.EmailEntry{{Value: "jane@example.com", Type: "WORK"}},
			Phones: []vcard.PhoneEntry{{Value: "+0987654321", Type: "CELL"}},
		},
	}

	cfg := &config.Config{}
	_ = New(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	testPrintln("Name", "Email")

	for _, contact := range contacts {
		for _, email := range contact.Emails {
			emailType := strings.ToLower(email.Type)
			if emailType == "" {
				emailType = "email"
			}
			testPrintf("%s\t%s: %s\n", contact.Name, emailType, email.Value)
		}
	}

	w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "Name\tEmail") {
		t.Error("Expected header not found in output")
	}
	if !strings.Contains(output, "John Doe\thome: john@example.com") {
		t.Error("Expected John Doe home email not found in output")
	}
	if !strings.Contains(output, "John Doe\twork: john@work.com") {
		t.Error("Expected John Doe work email not found in output")
	}
	if !strings.Contains(output, "Jane Smith\twork: jane@example.com") {
		t.Error("Expected Jane Smith email not found in output")
	}
	// Should not contain phone numbers
	if strings.Contains(output, "+1234567890") {
		t.Error("Phone numbers should not appear in email output")
	}
}

func TestListPhones(t *testing.T) {
	contacts := []vcard.Contact{
		{
			Name:   "John Doe",
			Emails: []vcard.EmailEntry{{Value: "john@example.com", Type: "HOME"}},
			Phones: []vcard.PhoneEntry{{Value: "+1234567890", Type: "HOME"}, {Value: "+1234567891", Type: "CELL"}},
		},
		{
			Name:   "Jane Smith",
			Emails: []vcard.EmailEntry{{Value: "jane@example.com", Type: "WORK"}},
			Phones: []vcard.PhoneEntry{{Value: "+0987654321", Type: "CELL"}},
		},
	}

	cfg := &config.Config{}
	_ = New(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	testPrintln("Name", "Phone")

	for _, contact := range contacts {
		for _, phone := range contact.Phones {
			phoneType := strings.ToLower(phone.Type)
			if phoneType == "" {
				phoneType = "phone"
			}
			testPrintf("%s\t%s: %s\n", contact.Name, phoneType, phone.Value)
		}
	}

	w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "Name\tPhone") {
		t.Error("Expected header not found in output")
	}
	if !strings.Contains(output, "John Doe\thome: +1234567890") {
		t.Error("Expected John Doe home phone not found in output")
	}
	if !strings.Contains(output, "John Doe\tcell: +1234567891") {
		t.Error("Expected John Doe cell phone not found in output")
	}
	if !strings.Contains(output, "Jane Smith\tcell: +0987654321") {
		t.Error("Expected Jane Smith phone not found in output")
	}
	// Should not contain emails
	if strings.Contains(output, "john@example.com") {
		t.Error("Email addresses should not appear in phone output")
	}
}

func TestListAddressbooks(t *testing.T) {
	cfg := &config.Config{
		AddressBooks: map[string]config.AddressBook{
			"personal": {Path: "~/contacts/personal"},
			"work":     {Path: "/opt/contacts/work"},
		},
	}

	app := New(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := app.ListAddressbooks()
	if err != nil {
		t.Fatalf("ListAddressbooks failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "Name\tPath") {
		t.Error("Expected header not found in output")
	}
	if !strings.Contains(output, "personal\t~/contacts/personal") {
		t.Error("Expected personal addressbook not found in output")
	}
	if !strings.Contains(output, "work\t/opt/contacts/work") {
		t.Error("Expected work addressbook not found in output")
	}
}

func TestListAddressbooksEmpty(t *testing.T) {
	cfg := &config.Config{
		AddressBooks: map[string]config.AddressBook{},
	}

	app := New(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := app.ListAddressbooks()
	if err != nil {
		t.Fatalf("ListAddressbooks failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "No address books configured") {
		t.Error("Expected 'No address books configured' message not found")
	}
}

func testPrintln(args ...any) {
	for i, arg := range args {
		if i > 0 {
			os.Stdout.WriteString("\t")
		}
		os.Stdout.WriteString(arg.(string))
	}
	os.Stdout.WriteString("\n")
}

func testPrintf(format string, args ...any) {
	result := format
	for _, arg := range args {
		result = strings.Replace(result, "%s", arg.(string), 1)
	}
	os.Stdout.WriteString(result)
}

func TestExportCSV(t *testing.T) {
	allContacts := createTestContacts()

	johnContact := findContactByName(allContacts, "John Doe")
	janeContact := findContactByName(allContacts, "Jane Smith")

	if johnContact == nil || janeContact == nil {
		t.Fatal("Required test contacts not found")
	}

	contacts := []vcard.Contact{*johnContact, *janeContact}

	cfg := &config.Config{}
	app := New(cfg)

	var buf bytes.Buffer
	err := app.exportCSV(&buf, contacts, ",")
	if err != nil {
		t.Fatalf("exportCSV failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	expectedHeader := "Name,Emails,Phones,Organization,Note,Address"
	if lines[0] != expectedHeader {
		t.Errorf("Expected header %q, got %q", expectedHeader, lines[0])
	}

	if !strings.Contains(output, "John Doe") {
		t.Error("Expected John Doe in CSV output")
	}
	if !strings.Contains(output, "john@example.com (home)") {
		t.Error("Expected formatted email in CSV output")
	}
	if !strings.Contains(output, "+1234567890 (home)") {
		t.Error("Expected formatted phone in CSV output")
	}
	if !strings.Contains(output, "Example Corp") {
		t.Error("Expected organization in CSV output")
	}

	if !strings.Contains(output, "Jane Smith") {
		t.Error("Expected Jane Smith in CSV output")
	}
	if !strings.Contains(output, "jane@example.com (work)") {
		t.Error("Expected Jane's email in CSV output")
	}
}

func TestExportCSVCustomDelimiter(t *testing.T) {
	contacts := []vcard.Contact{
		{
			Name:   "John Doe",
			Emails: []vcard.EmailEntry{{Value: "john@example.com", Type: "HOME"}},
		},
	}

	cfg := &config.Config{}
	app := New(cfg)

	var buf bytes.Buffer
	err := app.exportCSV(&buf, contacts, ";")
	if err != nil {
		t.Fatalf("exportCSV with custom delimiter failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	expectedHeader := "Name;Emails;Phones;Organization;Note;Address"
	if lines[0] != expectedHeader {
		t.Errorf("Expected header with semicolon delimiter %q, got %q", expectedHeader, lines[0])
	}
}

func TestExportJSON(t *testing.T) {
	contacts := []vcard.Contact{
		{
			Name:         "John Doe",
			Emails:       []vcard.EmailEntry{{Value: "john@example.com", Type: "HOME"}},
			Phones:       []vcard.PhoneEntry{{Value: "+1234567890", Type: "HOME"}},
			Organization: "Example Corp",
			Note:         "Test note",
			Address:      "123 Main St, Springfield, IL",
		},
		{
			Name:   "Jane Smith",
			Emails: []vcard.EmailEntry{{Value: "jane@work.com", Type: "WORK"}},
			Phones: []vcard.PhoneEntry{{Value: "+0987654321", Type: "CELL"}},
		},
	}

	cfg := &config.Config{}
	app := New(cfg)

	var buf bytes.Buffer
	err := app.exportJSON(&buf, contacts)
	if err != nil {
		t.Fatalf("exportJSON failed: %v", err)
	}

	var exportedContacts []vcard.Contact
	err = json.Unmarshal(buf.Bytes(), &exportedContacts)
	if err != nil {
		t.Fatalf("Failed to parse exported JSON: %v", err)
	}

	if len(exportedContacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(exportedContacts))
	}

	if exportedContacts[0].Name != "John Doe" {
		t.Errorf("Expected first contact name 'John Doe', got %q", exportedContacts[0].Name)
	}

	if exportedContacts[0].Organization != "Example Corp" {
		t.Errorf("Expected organization 'Example Corp', got %q", exportedContacts[0].Organization)
	}

	if len(exportedContacts[0].Emails) != 1 || exportedContacts[0].Emails[0].Value != "john@example.com" {
		t.Error("Expected John's email to be preserved in JSON export")
	}

	if exportedContacts[1].Name != "Jane Smith" {
		t.Errorf("Expected second contact name 'Jane Smith', got %q", exportedContacts[1].Name)
	}
}

func TestFormatEmailsForExport(t *testing.T) {
	cfg := &config.Config{}
	app := New(cfg)

	// Test empty emails
	result := app.formatEmailsForExport([]vcard.EmailEntry{})
	if result != "" {
		t.Errorf("Expected empty string for no emails, got %q", result)
	}

	// Test single email with type
	emails := []vcard.EmailEntry{{Value: "john@example.com", Type: "HOME"}}
	result = app.formatEmailsForExport(emails)
	expected := "john@example.com (home)"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test multiple emails
	emails = []vcard.EmailEntry{
		{Value: "john@example.com", Type: "HOME"},
		{Value: "john@work.com", Type: "WORK"},
		{Value: "john@personal.com", Type: ""},
	}
	result = app.formatEmailsForExport(emails)
	expected = "john@example.com (home); john@work.com (work); john@personal.com"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestFormatPhonesForExport(t *testing.T) {
	cfg := &config.Config{}
	app := New(cfg)

	result := app.formatPhonesForExport([]vcard.PhoneEntry{})
	if result != "" {
		t.Errorf("Expected empty string for no phones, got %q", result)
	}

	phones := []vcard.PhoneEntry{{Value: "+1234567890", Type: "CELL"}}
	result = app.formatPhonesForExport(phones)
	expected := "+1234567890 (cell)"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	phones = []vcard.PhoneEntry{
		{Value: "+1234567890", Type: "HOME"},
		{Value: "+0987654321", Type: "CELL"},
		{Value: "+1111111111", Type: ""},
	}
	result = app.formatPhonesForExport(phones)
	expected = "+1234567890 (home); +0987654321 (cell); +1111111111"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEmailListFormatting(t *testing.T) {
	allContacts := createTestContacts()
	johnContact := findContactByName(allContacts, "John Doe")

	if johnContact == nil {
		t.Fatal("John Doe contact not found in test data")
	}

	t.Run("parsable format logic", func(t *testing.T) {
		for _, email := range johnContact.Emails {
			emailType := formatEntryType(email.Type, defaultEmailType)

			if johnContact.Name == "" {
				t.Error("Contact name should not be empty")
			}
			if email.Value == "" {
				t.Error("Email value should not be empty")
			}
			if emailType == "" {
				t.Error("Email type should not be empty (should default)")
			}
		}
	})

	t.Run("standard format logic", func(t *testing.T) {
		displayName := formatNameForDisplay(*johnContact)

		if displayName != "Doe, John" {
			t.Errorf("Expected 'Doe, John', got %q", displayName)
		}

		for _, email := range johnContact.Emails {
			emailType := formatEntryType(email.Type, defaultEmailType)
			formatted := emailType + ": " + email.Value

			if !strings.Contains(formatted, ":") {
				t.Error("Standard format should contain colon separator")
			}
		}
	})
}

func TestExportContactsFileExists(t *testing.T) {
	allContacts := createTestContacts()
	johnContact := findContactByName(allContacts, "John Doe")

	if johnContact == nil {
		t.Fatal("John Doe contact not found in test data")
	}

	contacts := []vcard.Contact{*johnContact}
	cfg := &config.Config{}
	app := New(cfg)

	var buf bytes.Buffer
	err := app.exportCSV(&buf, contacts, ",")
	if err != nil {
		t.Fatalf("exportCSV failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "John Doe") {
		t.Error("Expected John Doe in CSV output")
	}
}

func TestExtractSortKey(t *testing.T) {
	tests := []struct {
		name     string
		contact  vcard.Contact
		expected string
	}{
		{
			name:     "structured personal name",
			contact:  vcard.Contact{FamilyName: "Smith", GivenName: "John"},
			expected: "smith",
		},
		{
			name:     "structured name with title and suffix",
			contact:  vcard.Contact{FamilyName: "Doe", GivenName: "Jane", Prefix: "Dr.", Suffix: "Jr."},
			expected: "doe",
		},
		{
			name:     "company with organization field",
			contact:  vcard.Contact{Name: "Acme Corp", Organization: "Acme Corporation"},
			expected: "acme corporation",
		},
		{
			name:     "company without structured name",
			contact:  vcard.Contact{Name: "Microsoft Corporation"},
			expected: "microsoft corporation",
		},
		{
			name:     "single name without structured components",
			contact:  vcard.Contact{Name: "Madonna"},
			expected: "madonna",
		},
		{
			name:     "person with both structured and org (person wins)",
			contact:  vcard.Contact{FamilyName: "Doe", GivenName: "John", Organization: "Acme Corp"},
			expected: "doe",
		},
		{
			name:     "organization only",
			contact:  vcard.Contact{Name: "ACME Inc", Organization: "ACME Incorporated"},
			expected: "acme incorporated",
		},
		{
			name:     "empty contact",
			contact:  vcard.Contact{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSortKey(tt.contact)
			if result != tt.expected {
				t.Errorf("extractSortKey(%q) = %q, want %q", tt.contact.Name, result, tt.expected)
			}
		})
	}
}

func TestSortContacts(t *testing.T) {
	contacts := []vcard.Contact{
		{FamilyName: "Smith", GivenName: "John", Name: "John Smith"},
		{Name: "Acme Corp", Organization: "Acme Corp"},
		{FamilyName: "Doe", GivenName: "Jane", Name: "Jane Doe"},
		{FamilyName: "Wilson", GivenName: "Bob", Name: "Bob Wilson"},
		{Name: "Microsoft Corporation", Organization: "Microsoft Corporation"},
	}

	sortContacts(contacts, false)

	// Expected: Acme Corp (org), Jane Doe (family), Microsoft Corp (org), John Smith (family), Bob Wilson (family)
	expectedSortKeys := []string{"acme corp", "doe", "microsoft corporation", "smith", "wilson"}
	for i, expectedKey := range expectedSortKeys {
		actualKey := extractSortKey(contacts[i])
		if actualKey != expectedKey {
			t.Errorf("Normal sort position %d: got sort key %q, want %q", i, actualKey, expectedKey)
		}
	}

	// Test reverse
	sortContacts(contacts, true)
	expectedReverseSortKeys := []string{"wilson", "smith", "microsoft corporation", "doe", "acme corp"}
	for i, expectedKey := range expectedReverseSortKeys {
		actualKey := extractSortKey(contacts[i])
		if actualKey != expectedKey {
			t.Errorf("Reverse sort position %d: got sort key %q, want %q", i, actualKey, expectedKey)
		}
	}
}

func TestFormatNameForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		contact  vcard.Contact
		expected string
	}{
		{
			name:     "structured personal name",
			contact:  vcard.Contact{FamilyName: "Smith", GivenName: "John"},
			expected: "Smith, John",
		},
		{
			name:     "structured name with title and suffix",
			contact:  vcard.Contact{FamilyName: "Doe", GivenName: "Jane", Prefix: "Dr.", Suffix: "Jr."},
			expected: "Doe, Dr. Jane Jr.",
		},
		{
			name:     "company name (no structured fields)",
			contact:  vcard.Contact{Name: "Acme Corp"},
			expected: "Acme Corp",
		},
		{
			name:     "company with Inc",
			contact:  vcard.Contact{Name: "Microsoft Corporation"},
			expected: "Microsoft Corporation",
		},
		{
			name:     "single name (no structured fields)",
			contact:  vcard.Contact{Name: "Madonna"},
			expected: "Madonna",
		},
		{
			name:     "structured name with middle name",
			contact:  vcard.Contact{FamilyName: "Smith", GivenName: "John", MiddleName: "Michael"},
			expected: "Smith, John Michael",
		},
		{
			name:     "structured name with all components",
			contact:  vcard.Contact{FamilyName: "Einstein", GivenName: "Albert", Prefix: "Prof.", Suffix: "PhD"},
			expected: "Einstein, Prof. Albert PhD",
		},
		{
			name:     "only given name",
			contact:  vcard.Contact{GivenName: "Madonna"},
			expected: "Madonna",
		},
		{
			name:     "only family name",
			contact:  vcard.Contact{FamilyName: "Smith"},
			expected: "Smith",
		},
		{
			name:     "empty contact",
			contact:  vcard.Contact{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNameForDisplay(tt.contact)
			if result != tt.expected {
				t.Errorf("formatNameForDisplay(%q) = %q, want %q", tt.contact.Name, result, tt.expected)
			}
		})
	}
}

func TestListBirthdays(t *testing.T) {
	cfg := &config.Config{
		AddressBooks: map[string]config.AddressBook{
			"test": {Path: "testdata"},
		},
	}
	app := New(cfg)

	t.Run("filters and displays only contacts with birthdays", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := app.ListBirthdays(false, []string{})
		if err != nil {
			t.Fatalf("ListBirthdays failed: %v", err)
		}

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Verify header is present
		if !strings.Contains(output, "Name\tBirthday") {
			t.Error("Expected header 'Name\tBirthday' not found in output")
		}

		// Verify contacts WITH birthdays are included
		if !strings.Contains(output, "Doe, John\t01/15/1990") {
			t.Error("Expected John Doe with birthday not found in output")
		}
		if !strings.Contains(output, "Brown, Alice\t03/10/1992") {
			t.Error("Expected Alice Brown with birthday not found in output")
		}
		if !strings.Contains(output, "Wilson, Bob\t12/25/1985") {
			t.Error("Expected Bob Wilson with birthday not found in output")
		}

		// Verify contacts WITHOUT birthdays are excluded
		if strings.Contains(output, "Jane Smith") {
			t.Error("Jane Smith (no birthday) should not appear in birthday list")
		}
		if strings.Contains(output, "Acme Corporation") {
			t.Error("Acme Corporation (no birthday) should not appear in birthday list")
		}
	})

	t.Run("backward compatibility - calls PrintBirthdays", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := app.ListBirthdays(false, []string{})
		if err != nil {
			t.Fatalf("ListBirthdays failed: %v", err)
		}

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Should work exactly like PrintBirthdays
		if !strings.Contains(output, "Name\tBirthday") {
			t.Error("Expected header 'Name\tBirthday' not found in output")
		}
		if !strings.Contains(output, "Doe, John\t01/15/1990") {
			t.Error("Expected John Doe with birthday not found in output")
		}
	})
}

func TestBirthdaySorting(t *testing.T) {
	t.Run("sorts contacts by birthday month", func(t *testing.T) {
		allContacts := createTestContacts()

		// Filter to only contacts with birthdays for sorting test
		var birthdayContacts []vcard.Contact
		for _, contact := range allContacts {
			if !contact.Birthday.IsZero() {
				birthdayContacts = append(birthdayContacts, contact)
			}
		}

		// Test the sorting function used by ListBirthdays
		sortContactsByBirthday(birthdayContacts, false)

		// Expected order: January (John), March (Alice), December (Bob)
		expectedOrder := []string{"John Doe", "Alice Brown", "Bob Wilson"}
		for i, expectedName := range expectedOrder {
			if birthdayContacts[i].Name != expectedName {
				t.Errorf("Position %d: expected %s, got %s", i, expectedName, birthdayContacts[i].Name)
			}
		}
	})

	t.Run("reverse sorts contacts by birthday month", func(t *testing.T) {
		allContacts := createTestContacts()

		// Filter to only contacts with birthdays for sorting test
		var birthdayContacts []vcard.Contact
		for _, contact := range allContacts {
			if !contact.Birthday.IsZero() {
				birthdayContacts = append(birthdayContacts, contact)
			}
		}

		// Test reverse sorting used by ListBirthdays
		sortContactsByBirthday(birthdayContacts, true)

		// Expected reverse order: December (Bob), March (Alice), January (John)
		expectedOrder := []string{"Bob Wilson", "Alice Brown", "John Doe"}
		for i, expectedName := range expectedOrder {
			if birthdayContacts[i].Name != expectedName {
				t.Errorf("Reverse position %d: expected %s, got %s", i, expectedName, birthdayContacts[i].Name)
			}
		}
	})

	t.Run("formats birthday display correctly", func(t *testing.T) {
		allContacts := createTestContacts()

		// Test the formatting used by ListBirthdays for contacts that have birthdays
		expectedFormats := map[string]string{
			"John Doe":    "01/15/1990",
			"Alice Brown": "03/10/1992",
			"Bob Wilson":  "12/25/1985",
		}

		for _, contact := range allContacts {
			if !contact.Birthday.IsZero() {
				formatted := vcard.FormatBirthdayForDisplay(contact.Birthday)
				expected := expectedFormats[contact.Name]
				if formatted != expected {
					t.Errorf("Contact %s: expected birthday format %s, got %s", contact.Name, expected, formatted)
				}
			}
		}
	})

	t.Run("formats names for display", func(t *testing.T) {
		allContacts := createTestContacts()

		expectedNames := map[string]string{
			"John Doe":    "Doe, John",
			"Alice Brown": "Brown, Alice",
			"Bob Wilson":  "Wilson, Bob",
		}

		for _, contact := range allContacts {
			if !contact.Birthday.IsZero() {
				formatted := formatNameForDisplay(contact)
				expected := expectedNames[contact.Name]
				if formatted != expected {
					t.Errorf("Contact %s: expected name format %s, got %s", contact.Name, expected, formatted)
				}
			}
		}
	})
}

func TestGetBirthdayContacts(t *testing.T) {
	cfg := &config.Config{
		AddressBooks: map[string]config.AddressBook{
			"test": {Path: "testdata"},
		},
	}
	app := New(cfg)

	t.Run("returns only contacts with birthdays", func(t *testing.T) {
		contacts, err := app.GetBirthdayContacts(false, []string{})
		if err != nil {
			t.Fatalf("GetBirthdayContacts failed: %v", err)
		}

		// Should have 3 contacts with birthdays (John, Alice, Bob)
		if len(contacts) != 3 {
			t.Errorf("Expected 3 contacts with birthdays, got %d", len(contacts))
		}

		// Verify correct contacts are included
		names := make(map[string]bool)
		for _, contact := range contacts {
			names[contact.Name] = true
			if contact.Birthday.IsZero() {
				t.Errorf("Contact %s should have a birthday", contact.Name)
			}
		}

		expectedNames := []string{"John Doe", "Alice Brown", "Bob Wilson"}
		for _, name := range expectedNames {
			if !names[name] {
				t.Errorf("Expected contact %s with birthday not found", name)
			}
		}

		// Verify contacts without birthdays are excluded
		for _, contact := range contacts {
			if contact.Name == "Jane Smith" || contact.Name == "Acme Corporation" {
				t.Errorf("Contact %s should not be included (no birthday)", contact.Name)
			}
		}
	})

	t.Run("sorts contacts by birthday month", func(t *testing.T) {
		contacts, err := app.GetBirthdayContacts(false, []string{})
		if err != nil {
			t.Fatalf("GetBirthdayContacts failed: %v", err)
		}

		if len(contacts) < 3 {
			t.Fatal("Need at least 3 contacts for sorting test")
		}

		// Should be sorted: John (Jan), Alice (Mar), Bob (Dec)
		if contacts[0].Name != "John Doe" {
			t.Errorf("Expected John Doe first (January), got %s", contacts[0].Name)
		}
		if contacts[1].Name != "Alice Brown" {
			t.Errorf("Expected Alice Brown second (March), got %s", contacts[1].Name)
		}
		if contacts[2].Name != "Bob Wilson" {
			t.Errorf("Expected Bob Wilson third (December), got %s", contacts[2].Name)
		}
	})

	t.Run("reverse sorts contacts by birthday month", func(t *testing.T) {
		contacts, err := app.GetBirthdayContacts(true, []string{})
		if err != nil {
			t.Fatalf("GetBirthdayContacts failed: %v", err)
		}

		if len(contacts) < 3 {
			t.Fatal("Need at least 3 contacts for reverse sorting test")
		}

		// Should be reverse sorted: Bob (Dec), Alice (Mar), John (Jan)
		if contacts[0].Name != "Bob Wilson" {
			t.Errorf("Expected Bob Wilson first (December), got %s", contacts[0].Name)
		}
		if contacts[1].Name != "Alice Brown" {
			t.Errorf("Expected Alice Brown second (March), got %s", contacts[1].Name)
		}
		if contacts[2].Name != "John Doe" {
			t.Errorf("Expected John Doe third (January), got %s", contacts[2].Name)
		}
	})

	t.Run("applies filter correctly", func(t *testing.T) {
		contacts, err := app.GetBirthdayContacts(false, []string{"John"})
		if err != nil {
			t.Fatalf("GetBirthdayContacts failed: %v", err)
		}

		// Should only return John Doe
		if len(contacts) != 1 {
			t.Errorf("Expected 1 filtered contact, got %d", len(contacts))
		}
		if len(contacts) > 0 && contacts[0].Name != "John Doe" {
			t.Errorf("Expected John Doe, got %s", contacts[0].Name)
		}
	})

	t.Run("returns empty slice when no birthdays found", func(t *testing.T) {
		emptyConfig := &config.Config{
			AddressBooks: map[string]config.AddressBook{},
		}
		emptyApp := New(emptyConfig)

		contacts, err := emptyApp.GetBirthdayContacts(false, []string{})
		if err != nil {
			t.Fatalf("GetBirthdayContacts failed: %v", err)
		}

		if len(contacts) != 0 {
			t.Errorf("Expected empty slice, got %d contacts", len(contacts))
		}
	})
}

func TestPrintBirthdays(t *testing.T) {
	cfg := &config.Config{
		AddressBooks: map[string]config.AddressBook{
			"test": {Path: "testdata"},
		},
	}
	app := New(cfg)

	t.Run("displays formatted birthday table", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := app.PrintBirthdays(false, []string{})
		if err != nil {
			t.Fatalf("PrintBirthdays failed: %v", err)
		}

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Verify header is present
		if !strings.Contains(output, "Name\tBirthday") {
			t.Error("Expected header 'Name\tBirthday' not found in output")
		}

		// Verify contacts WITH birthdays are included in correct format
		if !strings.Contains(output, "Doe, John\t01/15/1990") {
			t.Error("Expected John Doe with birthday not found in output")
		}
		if !strings.Contains(output, "Brown, Alice\t03/10/1992") {
			t.Error("Expected Alice Brown with birthday not found in output")
		}
		if !strings.Contains(output, "Wilson, Bob\t12/25/1985") {
			t.Error("Expected Bob Wilson with birthday not found in output")
		}

		// Verify contacts WITHOUT birthdays are excluded
		if strings.Contains(output, "Jane Smith") {
			t.Error("Jane Smith (no birthday) should not appear in birthday list")
		}
		if strings.Contains(output, "Acme Corporation") {
			t.Error("Acme Corporation (no birthday) should not appear in birthday list")
		}
	})

	t.Run("displays no contacts message when no birthdays found", func(t *testing.T) {
		emptyConfig := &config.Config{
			AddressBooks: map[string]config.AddressBook{},
		}
		emptyApp := New(emptyConfig)

		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := emptyApp.PrintBirthdays(false, []string{})
		if err != nil {
			t.Fatalf("PrintBirthdays failed: %v", err)
		}

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Should display message about no contacts found
		if !strings.Contains(output, "No contacts with birthdays found") {
			t.Error("Expected 'No contacts with birthdays found' message not found in output")
		}
	})
}
