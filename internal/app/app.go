package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/allaman/ghard/internal/config"
	"github.com/allaman/ghard/internal/updater"
	"github.com/allaman/ghard/internal/version"
	"github.com/allaman/ghard/pkg/vcard"
)

const (
	defaultEmailType = "email"
	defaultPhoneType = "phone"
)

// CLI represents the command line interface structure
type CLI struct {
	Debug bool `help:"Enable debug logging"`

	List struct {
		Long    bool     `short:"l" help:"Show notes and addresses"`
		Reverse bool     `short:"r" help:"Reverse sort order"`
		Filter  []string `arg:"" optional:"" help:"Filter contacts by string (case insensitive)"`
	} `cmd:"" help:"List all contacts" default:"withargs"`

	Email struct {
		Parsable bool     `short:"p" help:"Output in parsable format (email<tab>name<tab>type)"`
		Reverse  bool     `short:"r" help:"Reverse sort order"`
		Filter   []string `arg:"" optional:"" help:"Filter contacts by string (case insensitive)"`
	} `cmd:"" help:"List only email entries"`

	Phone struct {
		Reverse bool     `short:"r" help:"Reverse sort order"`
		Filter  []string `arg:"" optional:"" help:"Filter contacts by string (case insensitive)"`
	} `cmd:"" help:"List only phone entries"`

	Birthday struct {
		Reverse bool     `short:"r" help:"Reverse sort order"`
		Filter  []string `arg:"" optional:"" help:"Filter contacts by string (case insensitive)"`
	} `cmd:"" help:"List contacts with birthdays"`

	Addressbooks struct {
	} `cmd:"" help:"List configured address books"`

	Export struct {
		Format    string   `help:"Export format (csv, json)" enum:"csv,json" default:"csv"`
		Delimiter string   `help:"CSV delimiter character" default:","`
		Output    string   `help:"Output file path (default: stdout)" type:"path"`
		Filter    []string `arg:"" optional:"" help:"Filter contacts by string (case insensitive)"`
	} `cmd:"" help:"Export contacts to various formats"`

	Update struct {
	} `cmd:"" help:"Update ghard to the latest version"`

	Version struct {
	} `cmd:"" help:"Show version information"`
}

// App represents the application instance
type App struct {
	config *config.Config
}

// New creates a new application instance
func New(cfg *config.Config) *App {
	return &App{
		config: cfg,
	}
}

// Run executes the application based on the command
func (a *App) Run(command string, cli *CLI) error {
	// Handle Kong's command format for commands with arguments
	switch {
	case strings.HasPrefix(command, "list"):
		return a.ListContacts(cli.List.Long, cli.List.Reverse, cli.List.Filter)
	case strings.HasPrefix(command, "email"):
		return a.ListEmails(cli.Email.Parsable, cli.Email.Reverse, cli.Email.Filter)
	case strings.HasPrefix(command, "phone"):
		return a.ListPhones(cli.Phone.Reverse, cli.Phone.Filter)
	case strings.HasPrefix(command, "birthday"):
		return a.ListBirthdays(cli.Birthday.Reverse, cli.Birthday.Filter)
	case command == "addressbooks":
		return a.ListAddressbooks()
	case strings.HasPrefix(command, "export"):
		return a.ExportContacts(cli.Export.Format, cli.Export.Delimiter, cli.Export.Output, cli.Export.Filter)
	case command == "update":
		return updater.Update()
	case command == "version":
		version.ShowVersion()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// loadAndFilterContacts loads contacts and applies the given filter
func (a *App) loadAndFilterContacts(filter []string) ([]vcard.Contact, error) {
	contacts, err := vcard.LoadContacts(a.config.GetAddressBookPaths())
	if err != nil {
		return nil, fmt.Errorf("failed to load contacts: %w", err)
	}

	if len(filter) == 0 {
		return contacts, nil
	}

	var filteredContacts []vcard.Contact
	for _, contact := range contacts {
		if a.matchesFilter(contact, filter) {
			filteredContacts = append(filteredContacts, contact)
		}
	}

	return filteredContacts, nil
}

// matchesFilter checks if a contact matches the given filter strings (case insensitive)
func (a *App) matchesFilter(contact vcard.Contact, filter []string) bool {
	if len(filter) == 0 {
		return true
	}

	searchString := strings.ToLower(strings.Join(filter, " "))
	contactText := a.buildContactSearchText(contact)
	return strings.Contains(contactText, searchString)
}

// buildContactSearchText creates a searchable text representation of the contact
func (a *App) buildContactSearchText(contact vcard.Contact) string {
	var builder strings.Builder

	// Add name
	if contact.Name != "" {
		builder.WriteString(strings.ToLower(contact.Name))
		builder.WriteRune(' ')
	}

	// Add emails and types
	for _, email := range contact.Emails {
		builder.WriteString(strings.ToLower(email.Value))
		builder.WriteRune(' ')
		if email.Type != "" {
			builder.WriteString(strings.ToLower(email.Type))
			builder.WriteRune(' ')
		}
	}

	// Add phones and types
	for _, phone := range contact.Phones {
		builder.WriteString(strings.ToLower(phone.Value))
		builder.WriteRune(' ')
		if phone.Type != "" {
			builder.WriteString(strings.ToLower(phone.Type))
			builder.WriteRune(' ')
		}
	}

	// Add optional fields
	if contact.Organization != "" {
		builder.WriteString(strings.ToLower(contact.Organization))
		builder.WriteRune(' ')
	}
	if contact.Note != "" {
		builder.WriteString(strings.ToLower(contact.Note))
		builder.WriteRune(' ')
	}
	if contact.Address != "" {
		builder.WriteString(strings.ToLower(contact.Address))
		builder.WriteRune(' ')
	}

	return strings.TrimSpace(builder.String())
}

// formatEntryType returns the formatted type for display (email or phone)
func formatEntryType(entryType, defaultType string) string {
	if entryType == "" {
		return defaultType
	}
	return strings.ToLower(entryType)
}

// extractSortKey extracts the sort key from a contact using vCard structured fields
func extractSortKey(contact vcard.Contact) string {
	// If we have structured name data (FamilyName from N field), use it
	if contact.FamilyName != "" {
		return strings.ToLower(contact.FamilyName)
	}

	// If no structured name but we have an organization, it's likely a company
	if contact.Organization != "" && contact.Name != "" {
		return strings.ToLower(contact.Organization)
	}

	// Fallback to formatted name (FN field) for companies or single names
	if contact.Name != "" {
		return strings.ToLower(contact.Name)
	}

	return ""
}

// formatNameForDisplay formats names using vCard structured fields
func formatNameForDisplay(contact vcard.Contact) string {
	if contact.FamilyName != "" || contact.GivenName != "" {
		var parts []string

		// Start with family name
		if contact.FamilyName != "" {
			parts = append(parts, contact.FamilyName)
		}

		// Add given name and additional components after comma
		var givenParts []string
		if contact.Prefix != "" {
			givenParts = append(givenParts, contact.Prefix)
		}
		if contact.GivenName != "" {
			givenParts = append(givenParts, contact.GivenName)
		}
		if contact.MiddleName != "" {
			givenParts = append(givenParts, contact.MiddleName)
		}
		if contact.Suffix != "" {
			givenParts = append(givenParts, contact.Suffix)
		}

		if len(givenParts) > 0 {
			if len(parts) > 0 {
				return strings.Join(parts, "") + ", " + strings.Join(givenParts, " ")
			} else {
				// Only given name, no family name
				return strings.Join(givenParts, " ")
			}
		} else if len(parts) > 0 {
			// Only family name
			return strings.Join(parts, "")
		}
	}

	// Fallback to formatted name (FN) for companies or when N field is not available
	if contact.Name != "" {
		return contact.Name
	}

	return ""
}

// sortContacts sorts contacts by last name (for people) or full name (for companies)
func sortContacts(contacts []vcard.Contact, reverse bool) {
	sort.Slice(contacts, func(i, j int) bool {
		keyI := extractSortKey(contacts[i])
		keyJ := extractSortKey(contacts[j])

		if reverse {
			return keyI > keyJ
		}
		return keyI < keyJ
	})
}

// sortContactsByBirthday sorts contacts by birthday month and day
func sortContactsByBirthday(contacts []vcard.Contact, reverse bool) {
	sort.Slice(contacts, func(i, j int) bool {
		timeI := contacts[i].Birthday
		timeJ := contacts[j].Birthday

		// Handle zero times (unparseable dates) - put them at the end
		if timeI.IsZero() && timeJ.IsZero() {
			return false // Keep original order
		}
		if timeI.IsZero() {
			return false // i goes after j
		}
		if timeJ.IsZero() {
			return true // i goes before j
		}

		// Compare by month first, then by day
		monthI := int(timeI.Month())
		monthJ := int(timeJ.Month())
		
		if monthI != monthJ {
			if reverse {
				return monthI > monthJ
			}
			return monthI < monthJ
		}
		
		// If months are equal, compare by day
		dayI := timeI.Day()
		dayJ := timeJ.Day()
		
		if reverse {
			return dayI > dayJ
		}
		return dayI < dayJ
	})
}

// ListContacts lists all contacts with optional long format and filtering
func (a *App) ListContacts(long bool, reverse bool, filter []string) error {
	contacts, err := a.loadAndFilterContacts(filter)
	if err != nil {
		return err
	}

	if len(contacts) == 0 {
		fmt.Println("No contacts found")
		return nil
	}

	// Sort contacts by last name
	sortContacts(contacts, reverse)

	for _, contact := range contacts {
		parts := a.buildContactParts(contact, long)
		fmt.Println(strings.Join(parts, "\t"))
	}

	return nil
}

// buildContactParts creates the tab-separated parts for a contact display
func (a *App) buildContactParts(contact vcard.Contact, long bool) []string {
	var parts []string
	parts = append(parts, formatNameForDisplay(contact))

	// Add emails
	for _, email := range contact.Emails {
		emailType := formatEntryType(email.Type, defaultEmailType)
		parts = append(parts, fmt.Sprintf("%s: %s", emailType, email.Value))
	}

	// Add phones
	for _, phone := range contact.Phones {
		phoneType := formatEntryType(phone.Type, defaultPhoneType)
		parts = append(parts, fmt.Sprintf("%s: %s", phoneType, phone.Value))
	}

	// Add long format fields
	if long {
		if contact.Organization != "" {
			parts = append(parts, fmt.Sprintf("org: %s", contact.Organization))
		}
		if contact.Note != "" {
			parts = append(parts, fmt.Sprintf("note: %s", contact.Note))
		}
		if contact.Address != "" {
			parts = append(parts, fmt.Sprintf("address: %s", contact.Address))
		}
	}

	return parts
}

// ListEmails lists only email entries for all contacts with filtering
func (a *App) ListEmails(parsable bool, reverse bool, filter []string) error {
	contacts, err := a.loadAndFilterContacts(filter)
	if err != nil {
		return err
	}

	// Sort contacts by last name
	sortContacts(contacts, reverse)

	if parsable {
		// Parsable format: email<tab>name<tab>type (no header)
		for _, contact := range contacts {
			for _, email := range contact.Emails {
				emailType := formatEntryType(email.Type, defaultEmailType)
				fmt.Printf("%s\t%s\t%s\n",
					email.Value,
					contact.Name,
					emailType,
				)
			}
		}
	} else {
		// Standard format with header
		fmt.Println("Name\tEmail")
		for _, contact := range contacts {
			for _, email := range contact.Emails {
				emailType := formatEntryType(email.Type, defaultEmailType)
				fmt.Printf("%s\t%s: %s\n",
					formatNameForDisplay(contact),
					emailType,
					email.Value,
				)
			}
		}
	}

	return nil
}

// ListPhones lists only phone entries for all contacts with filtering
func (a *App) ListPhones(reverse bool, filter []string) error {
	contacts, err := a.loadAndFilterContacts(filter)
	if err != nil {
		return err
	}

	// Sort contacts by last name
	sortContacts(contacts, reverse)

	fmt.Println("Name\tPhone")

	for _, contact := range contacts {
		for _, phone := range contact.Phones {
			phoneType := formatEntryType(phone.Type, defaultPhoneType)
			fmt.Printf("%s\t%s: %s\n",
				formatNameForDisplay(contact),
				phoneType,
				phone.Value,
			)
		}
	}

	return nil
}

// ListBirthdays lists contacts with birthdays
// GetBirthdayContacts returns all contacts that have birthdays, sorted by month/day
func (a *App) GetBirthdayContacts(reverse bool, filter []string) ([]vcard.Contact, error) {
	contacts, err := a.loadAndFilterContacts(filter)
	if err != nil {
		return nil, err
	}

	// Filter to only contacts with birthdays
	var birthdayContacts []vcard.Contact
	for _, contact := range contacts {
		if !contact.Birthday.IsZero() {
			birthdayContacts = append(birthdayContacts, contact)
		}
	}

	// Sort contacts by birthday month and day
	sortContactsByBirthday(birthdayContacts, reverse)

	return birthdayContacts, nil
}

// PrintBirthdays displays contacts with birthdays in a formatted table
func (a *App) PrintBirthdays(reverse bool, filter []string) error {
	birthdayContacts, err := a.GetBirthdayContacts(reverse, filter)
	if err != nil {
		return err
	}

	if len(birthdayContacts) == 0 {
		fmt.Println("No contacts with birthdays found")
		return nil
	}

	fmt.Println("Name\tBirthday")

	for _, contact := range birthdayContacts {
		fmt.Printf("%s\t%s\n",
			formatNameForDisplay(contact),
			vcard.FormatBirthdayForDisplay(contact.Birthday),
		)
	}

	return nil
}

// ListBirthdays is kept for backward compatibility, calls PrintBirthdays
func (a *App) ListBirthdays(reverse bool, filter []string) error {
	return a.PrintBirthdays(reverse, filter)
}

// ListAddressbooks lists all configured address books
func (a *App) ListAddressbooks() error {
	if len(a.config.AddressBooks) == 0 {
		fmt.Println("No address books configured")
		return nil
	}

	fmt.Println("Name\tPath")
	for name, ab := range a.config.AddressBooks {
		fmt.Printf("%s\t%s\n", name, ab.Path)
	}

	return nil
}

// ExportContacts exports contacts to the specified format and output
func (a *App) ExportContacts(format, delimiter, output string, filter []string) error {
	contacts, err := a.loadAndFilterContacts(filter)
	if err != nil {
		return err
	}

	var writer io.Writer = os.Stdout
	if output != "" {
		if _, err := os.Stat(output); err == nil {
			return fmt.Errorf("output file already exists: %s", output)
		}

		file, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	switch format {
	case "csv":
		return a.exportCSV(writer, contacts, delimiter)
	case "json":
		return a.exportJSON(writer, contacts)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// exportCSV exports contacts to CSV format
func (a *App) exportCSV(writer io.Writer, contacts []vcard.Contact, delimiter string) error {
	csvWriter := csv.NewWriter(writer)
	if len(delimiter) > 0 {
		csvWriter.Comma = rune(delimiter[0])
	}
	defer csvWriter.Flush()

	// Write header
	header := []string{"Name", "Emails", "Phones", "Organization", "Note", "Address"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write contacts
	for _, contact := range contacts {
		record := []string{
			contact.Name,
			a.formatEmailsForExport(contact.Emails),
			a.formatPhonesForExport(contact.Phones),
			contact.Organization,
			contact.Note,
			contact.Address,
		}
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

// exportJSON exports contacts to JSON format
func (a *App) exportJSON(writer io.Writer, contacts []vcard.Contact) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(contacts); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// formatEmailsForExport formats emails for CSV export
func (a *App) formatEmailsForExport(emails []vcard.EmailEntry) string {
	if len(emails) == 0 {
		return ""
	}

	var parts []string
	for _, email := range emails {
		if email.Type != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", email.Value, strings.ToLower(email.Type)))
		} else {
			parts = append(parts, email.Value)
		}
	}
	return strings.Join(parts, "; ")
}

// formatPhonesForExport formats phones for CSV export
func (a *App) formatPhonesForExport(phones []vcard.PhoneEntry) string {
	if len(phones) == 0 {
		return ""
	}

	var parts []string
	for _, phone := range phones {
		if phone.Type != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", phone.Value, strings.ToLower(phone.Type)))
		} else {
			parts = append(parts, phone.Value)
		}
	}
	return strings.Join(parts, "; ")
}
