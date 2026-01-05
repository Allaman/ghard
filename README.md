<h1 align="center">ghard</h1>
<div align="center">
  <img width="256" height="256" alt="gopher" src="https://github.com/user-attachments/assets/e2caaa83-f1c2-44fe-acb1-9cc2dc43f11f" />
  <p>
    <img src="https://github.com/Allaman/ghard/actions/workflows/release.yaml/badge.svg" alt="Release"/>
    <img src="https://github.com/Allaman/ghard/actions/workflows/govulncheck.yaml/badge.svg" alt="Vulnerabilities"/>
    <img src="https://img.shields.io/github/repo-size/Allaman/ghard" alt="size"/>
    <img src="https://img.shields.io/github/issues/Allaman/ghard" alt="issues"/>
    <img src="https://img.shields.io/github/last-commit/Allaman/ghard" alt="last commit"/>
    <img src="https://img.shields.io/github/license/Allaman/ghard" alt="license"/>
    <img src="https://img.shields.io/github/v/release/Allaman/ghard?sort=semver" alt="last release"/>
  </p>
</div>

A basic Go port of [khard](https://github.com/lucc/khard) - a command-line vCard address book manager.

## Motivation

Though Khard is actively developed, more mature, and has more features, for me, it was often a struggle to get the Python components installed.
So, I wanted to port the parts that are important to me (reading and (Neo)mutt integration) to a language that I can distribute as a single binary.

> [!NOTE]
> This is by no means a full replacement; it's only a lightweight, easy-to-install (single binary 🚀) option for reading folders with VCF files.
> Credits to the [authors](https://github.com/lucc/khard?tab=readme-ov-file#authors) of khard.

## Features

- **Read vCard files** from local directories
- **Multiple address books** support via TOML configuration
- **Unix-friendly output** with tab-separated values for easy piping
- **Flexible display** with short and long format options
- **Case-insensitive filtering** by name, email, phone, organization, or any other supported field
- **Contact export** to CSV and JSON formats
- **(Neo)mutt compatible** to look up email addresses

## Installation

### Binary Releases

Download the latest binary for your platform from the [releases page](https://github.com/allaman/ghard/releases).

**Linux/macOS:**

```bash
VERSION=$(curl -s https://api.github.com/repos/allaman/ghard/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -sLo ghard https://github.com/Allaman/ghard/releases/download/${VERSION}/ghard${VERSION}_$(uname -s)_$(uname -m)
chmod +x ghard
sudo mv ghard /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/allaman/ghard
cd ghard
go build -o ghard ./cmd/ghard/
```

Have a look at the provided [Taskfile](./Taskfile.yml) for shortcuts.

## Configuration

Create a configuration file at `~/.config/ghard/ghard.toml`:

```toml
[addressbook.personal]
path = "~/contacts/personal"

[addressbook.work]
path = "/opt/contacts/work"
```

Each `[addressbook.<name>]` section defines an address book with the `path`, a directory containing `.vcf` files.

## Usage

### List all contacts (default command)

```bash
# Basic format: Name, Email, Phone
ghard
# or explicitly
ghard list
```

### List with detailed information

```bash
# Long format: Name, Email, Phone, Organization, Note, Address
ghard list --long
# or
ghard list -l
```

### List only email entries

```bash
ghard email

# Output in parsable format (email<tab>name<tab>type) e.g. for (neo)mutt
ghard email --parsable
```

### List only phone entries

```bash
ghard phone
```

### List configured address books

```bash
ghard addressbooks
```

### Debug mode

```bash
ghard --debug [command]
```

### Filtering contacts

All main commands support case-insensitive filtering by providing search terms as arguments:

```bash
ghard jane smith
ghard phone max mustermann
ghard email --parsable john doe
```

The filter searches across all contact fields including:

- Names
- Email addresses and types
- Phone numbers and types
- Organizations
- Notes
- Addresses

### Exporting contacts

Export your contacts to various formats for backup, data portability, or integration with other tools:

```bash
# Export all contacts to CSV (default format) and stdout
ghard export

# Export to JSON format
ghard export --format json --output contacts.json

# Export with custom CSV delimiter
ghard export --format csv --delimiter ";" --output contacts.csv

# Export filtered contacts
ghard export john doe --output john_contacts.csv
```

## Output Formats

### Standard Format

The default output is tab-separated for easy processing with Unix tools like `cut`, `grep`, `awk`, `sort`, etc.

### Parsable (Email) Format

The email command supports a `--parsable` format compatible with khard:

```bash
ghard email --parsable
```

Output format: `email<tab>name<tab>type`

Usage in `muttrc`:

```
set query_command= "ghard email --parsable %s"
```

## vCard Support

ghard reads standard vCard (.vcf) files and extracts:

- **Name** (FN field)
- **Email** (EMAIL field) - supports multiple entries with types (HOME, WORK, etc.)
- **Phone** (TEL field) - supports multiple entries with types (HOME, WORK, CELL, etc.)
- **Organization** (ORG field)
- **Note** (NOTE field)
- **Address** (ADR field, formatted for readability)

## License

MIT License
