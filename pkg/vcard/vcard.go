package vcard

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-vcard"
)

type EmailEntry struct {
	Value string
	Type  string // HOME, WORK, etc.
}

type PhoneEntry struct {
	Value string
	Type  string // HOME, WORK, CELL, etc.
}

type Contact struct {
	Name         string // FN field - formatted name
	FamilyName   string // N field components
	GivenName    string
	MiddleName   string
	Prefix       string
	Suffix       string
	Emails       []EmailEntry
	Phones       []PhoneEntry
	Organization string
	Note         string
	Address      string
	Birthday     time.Time
	FileName     string
}

func LoadContacts(addressBookPaths []string) ([]Contact, error) {
	var contacts []Contact
	var loadErrors []string

	for _, path := range addressBookPaths {
		pathContacts, err := loadContactsFromPath(path)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		contacts = append(contacts, pathContacts...)
	}

	// If all paths failed to load, return an error
	if len(loadErrors) > 0 && len(contacts) == 0 {
		return nil, fmt.Errorf("failed to load contacts from any address book:\n  - %s", strings.Join(loadErrors, "\n  - "))
	}

	// If some paths failed but we got contacts, log warnings but continue
	if len(loadErrors) > 0 {
		for _, errMsg := range loadErrors {
			slog.Warn("Failed to load contacts from address book", "error", errMsg)
		}
	}

	if len(contacts) == 0 {
		slog.Info("No contacts found in any address book")
	} else {
		slog.Debug("Loaded contacts", "total", len(contacts), "from_paths", len(addressBookPaths))
	}

	return contacts, nil
}

func loadContactsFromPath(path string) ([]Contact, error) {
	var contacts []Contact

	slog.Debug("Loading contacts from path", "path", path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}

	var vcfCount int
	var processedCount int
	var errorCount int

	err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Warn("Error accessing file during directory walk", "path", filePath, "error", err)
			return nil // Continue processing other files
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(strings.ToLower(filePath), ".vcf") {
			slog.Debug("Skipping non-vcf file", "file", filePath)
			return nil
		}

		vcfCount++
		slog.Debug("Processing vCard file", "file", filePath)

		if err := processVCardFile(filePath, &contacts); err != nil {
			slog.Warn("Failed to process vCard file", "file", filePath, "error", err)
			errorCount++
			return nil // Continue processing other files
		}

		processedCount++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning directory %s: %w", path, err)
	}

	slog.Debug("Finished processing directory",
		"path", path,
		"vcf_files_found", vcfCount,
		"successfully_processed", processedCount,
		"errors", errorCount,
		"contacts_loaded", len(contacts))

	if vcfCount == 0 {
		return nil, fmt.Errorf("no .vcf files found in directory: %s", path)
	}

	if processedCount == 0 && errorCount > 0 {
		return nil, fmt.Errorf("failed to process any vCard files in directory: %s", path)
	}

	return contacts, err
}

// processVCardFile processes a single vCard file and adds contacts to the slice
func processVCardFile(filePath string, contacts *[]Contact) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := vcard.NewDecoder(file)
	fileContacts := 0

	for {
		card, err := decoder.Decode()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode vCard: %w", err)
		}

		contact := Contact{
			FileName: filePath,
		}

		if fn := card.Get(vcard.FieldFormattedName); fn != nil {
			contact.Name = fn.Value
		}

		// Extract structured name components (N field)
		if nameField := card.Get(vcard.FieldName); nameField != nil {
			// N field format: "Family;Given;Additional;Prefix;Suffix"
			parts := strings.Split(nameField.Value, ";")
			if len(parts) > 0 && parts[0] != "" {
				contact.FamilyName = parts[0]
			}
			if len(parts) > 1 && parts[1] != "" {
				contact.GivenName = parts[1]
			}
			if len(parts) > 2 && parts[2] != "" {
				contact.MiddleName = parts[2]
			}
			if len(parts) > 3 && parts[3] != "" {
				contact.Prefix = parts[3]
			}
			if len(parts) > 4 && parts[4] != "" {
				contact.Suffix = parts[4]
			}
		}

		// Collect all email entries
		if emailFields, exists := card[vcard.FieldEmail]; exists {
			for _, field := range emailFields {
				emailType := getFieldType(field.Params)
				contact.Emails = append(contact.Emails, EmailEntry{
					Value: field.Value,
					Type:  emailType,
				})
			}
		}

		// Collect all phone entries
		if phoneFields, exists := card[vcard.FieldTelephone]; exists {
			for _, field := range phoneFields {
				phoneType := getFieldType(field.Params)
				contact.Phones = append(contact.Phones, PhoneEntry{
					Value: field.Value,
					Type:  phoneType,
				})
			}
		}

		if org := card.Get(vcard.FieldOrganization); org != nil {
			contact.Organization = org.Value
		}

		if note := card.Get(vcard.FieldNote); note != nil {
			contact.Note = note.Value
		}

		if addr := card.Get(vcard.FieldAddress); addr != nil {
			contact.Address = formatAddress(addr.Value)
		}

		if bday := card.Get(vcard.FieldBirthday); bday != nil {
			contact.Birthday = parseBirthday(bday.Value)
		}

		slog.Debug("Loaded contact", "name", contact.Name, "file", filePath)
		*contacts = append(*contacts, contact)
		fileContacts++
	}

	if fileContacts == 0 {
		return errors.New("no valid contacts found in file")
	}

	return nil
}

func formatAddress(address string) string {
	// vCard address format: post-office-box;extended-address;street-address;locality;region;postal-code;country
	parts := strings.Split(address, ";")

	var addressParts []string

	// Extract important parts and skip empty ones
	if len(parts) > 2 && parts[2] != "" {
		addressParts = append(addressParts, parts[2]) // street-address
	}
	if len(parts) > 3 && parts[3] != "" {
		addressParts = append(addressParts, parts[3]) // locality (city)
	}
	if len(parts) > 4 && parts[4] != "" {
		addressParts = append(addressParts, parts[4]) // region (state)
	}
	if len(parts) > 5 && parts[5] != "" {
		addressParts = append(addressParts, parts[5]) // postal-code
	}
	if len(parts) > 6 && parts[6] != "" {
		addressParts = append(addressParts, parts[6]) // country
	}

	return strings.Join(addressParts, ", ")
}

func parseBirthday(birthday string) time.Time {
	birthday = strings.TrimSpace(birthday)
	if birthday == "" {
		return time.Time{}
	}

	// Try various vCard birthday formats
	formats := []string{
		"2006-01-02",           // YYYY-MM-DD (ISO date)
		"20060102",             // YYYYMMDD (compact format)
		"--01-02",              // --MM-DD (year unknown, month-day only)
		"--0102",               // --MMDD (year unknown, compact month-day)
		"2006-01-02T15:04:05Z", // ISO 8601 with time
		"2006-01-02T15:04:05",  // ISO 8601 without timezone
	}

	var parsedTime time.Time
	var err error

	for _, format := range formats {
		parsedTime, err = time.Parse(format, birthday)
		if err == nil {
			// Successfully parsed
			if strings.HasPrefix(birthday, "--") {
				// For year-unknown formats, use current year for sorting purposes
				currentYear := time.Now().Year()
				parsedTime = time.Date(currentYear, parsedTime.Month(), parsedTime.Day(), 0, 0, 0, 0, time.UTC)
			}
			return parsedTime
		}
	}

	// If no format matches, try to extract just the date part if it has extra content
	if len(birthday) >= 10 {
		datePart := birthday[:10]
		parsedTime, err = time.Parse("2006-01-02", datePart)
		if err == nil {
			return parsedTime
		}
	}

	// If all parsing fails, return zero time
	return time.Time{}
}

// FormatBirthdayForDisplay formats a birthday time.Time for display as mm/dd/yyyy or mm/dd
func FormatBirthdayForDisplay(birthday time.Time) string {
	if birthday.IsZero() {
		return ""
	}

	// Check if this was originally a year-unknown format by checking if year is current year
	// and month/day are not today (heuristic for year-unknown dates)
	currentYear := time.Now().Year()
	if birthday.Year() == currentYear {
		// This might be a year-unknown format, show without year
		return fmt.Sprintf("%02d/%02d", birthday.Month(), birthday.Day())
	}

	// Standard format with year
	return fmt.Sprintf("%02d/%02d/%04d", birthday.Month(), birthday.Day(), birthday.Year())
}

func getFieldType(params map[string][]string) string {
	// Check for TYPE parameter
	if types, exists := params["TYPE"]; exists && len(types) > 0 {
		return strings.ToUpper(types[0])
	}

	// Check for legacy vCard 2.1 style parameters (HOME, WORK, etc.)
	for param := range params {
		switch strings.ToUpper(param) {
		case "HOME", "WORK", "CELL", "FAX", "PAGER", "VOICE", "MSG":
			return strings.ToUpper(param)
		}
	}

	return ""
}

// GetPrimaryEmail is a helper functions for display formatting
func (c *Contact) GetPrimaryEmail() string {
	if len(c.Emails) == 0 {
		return ""
	}
	email := c.Emails[0]
	if email.Type != "" {
		return fmt.Sprintf("%s (%s)", email.Value, email.Type)
	}
	return email.Value
}

func (c *Contact) GetPrimaryPhone() string {
	if len(c.Phones) == 0 {
		return ""
	}
	phone := c.Phones[0]
	if phone.Type != "" {
		return fmt.Sprintf("%s (%s)", phone.Value, phone.Type)
	}
	return phone.Value
}

func (c *Contact) FormatEmails() string {
	if len(c.Emails) == 0 {
		return ""
	}

	var parts []string
	for _, email := range c.Emails {
		if email.Type != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", email.Value, email.Type))
		} else {
			parts = append(parts, email.Value)
		}
	}
	return strings.Join(parts, ", ")
}

func (c *Contact) FormatPhones() string {
	if len(c.Phones) == 0 {
		return ""
	}

	var parts []string
	for _, phone := range c.Phones {
		if phone.Type != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", phone.Value, phone.Type))
		} else {
			parts = append(parts, phone.Value)
		}
	}
	return strings.Join(parts, ", ")
}
