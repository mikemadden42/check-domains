package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestReadDomains(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"one per line", "a.com\nb.com\n", []string{"a.com", "b.com"}},
		{"no trailing newline", "a.com\nb.com", []string{"a.com", "b.com"}},
		{"blank lines", "\na.com\n\n\nb.com\n\n", []string{"a.com", "b.com"}},
		{"surrounding whitespace", "  a.com\t\n\tb.com  \n", []string{"a.com", "b.com"}},
		{"CRLF line endings", "a.com\r\nb.com\r\n", []string{"a.com", "b.com"}},
		{"comment lines", "# list\na.com\n#b.com\n  # indented\nc.com\n", []string{"a.com", "c.com"}},
		{"empty file", "", nil},
		{"only comments and blanks", "# nothing\n\n   \n", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "domains.txt")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}

			got, err := readDomains(path)
			if err != nil {
				t.Fatalf("readDomains() error = %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("readDomains() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadDomainsMissingFile(t *testing.T) {
	_, err := readDomains(filepath.Join(t.TempDir(), "missing.txt"))
	if !os.IsNotExist(err) {
		t.Errorf("readDomains() error = %v, want a not-exist error", err)
	}
}
