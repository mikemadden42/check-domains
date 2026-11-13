package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

// fakeLookup returns a lookup function that always gives addrs and err.
func fakeLookup(addrs []string, err error) func(string) ([]string, error) {
	return func(string) ([]string, error) { return addrs, err }
}

func TestCheckDomain(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer ok.Close()
	serverError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer serverError.Close()
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closed.Close()

	// Domains are host:port of the test servers, so "http://" + domain reaches them
	host := func(s *httptest.Server) string { return strings.TrimPrefix(s.URL, "http://") }
	resolved := fakeLookup([]string{"192.0.2.1", "2001:db8::1"}, nil)

	tests := []struct {
		name     string
		domain   string
		lookup   func(string) ([]string, error)
		wantOK   bool
		wantLine string
	}{
		{"success", host(ok), resolved, true, "  HTTP status code: 200\n"},
		// Only a failed request counts as a failure, not an error status
		{"server error status", host(serverError), resolved, true, "  HTTP status code: 500\n"},
		{"HTTP request fails", host(closed), resolved, false, "  HTTP request failed: "},
		{"domain not found", "missing.example", fakeLookup(nil, &net.DNSError{Err: "no such host", Name: "missing.example", IsNotFound: true}), false, "  Domain does not exist: missing.example\n"},
		{"wrapped not found", "missing.example", fakeLookup(nil, fmt.Errorf("wrapped: %w", &net.DNSError{IsNotFound: true})), false, "  Domain does not exist: missing.example\n"},
		{"other DNS error", "flaky.example", fakeLookup(nil, &net.DNSError{Err: "server misbehaving", Name: "flaky.example"}), false, "  DNS lookup failed: lookup flaky.example: server misbehaving\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			got := checkDomain(&buf, tt.domain, tt.lookup, &http.Client{Timeout: 5 * time.Second})
			out := buf.String()

			if got != tt.wantOK {
				t.Errorf("checkDomain() = %v, want %v", got, tt.wantOK)
			}
			if !strings.HasPrefix(out, "Checking domain: "+tt.domain+"\n") {
				t.Errorf("output missing header:\n%s", out)
			}
			if !strings.Contains(out, tt.wantLine) {
				t.Errorf("output missing %q:\n%s", tt.wantLine, out)
			}
		})
	}
}

func TestCheckDomainReportsAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	domain := strings.TrimPrefix(srv.URL, "http://")

	var buf bytes.Buffer
	checkDomain(&buf, domain, fakeLookup([]string{"192.0.2.1", "2001:db8::1"}, nil), &http.Client{Timeout: 5 * time.Second})

	want := "Checking domain: " + domain + "\n" +
		"  IP addresses:\n" +
		"    - 192.0.2.1\n" +
		"    - 2001:db8::1\n" +
		"  HTTP status code: 200\n"
	if got := buf.String(); got != want {
		t.Errorf("output =\n%s\nwant\n%s", got, want)
	}
}

func TestCheckDomainTimeout(t *testing.T) {
	// The handler never responds, so only the client timeout ends the request
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()
	domain := strings.TrimPrefix(srv.URL, "http://")

	var buf bytes.Buffer
	start := time.Now()
	ok := checkDomain(&buf, domain, fakeLookup([]string{"127.0.0.1"}, nil), &http.Client{Timeout: 100 * time.Millisecond})

	if ok {
		t.Error("checkDomain() = true, want false on timeout")
	}
	if !strings.Contains(buf.String(), "  HTTP request failed: ") {
		t.Errorf("output missing request failure:\n%s", buf.String())
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("checkDomain() took %v, timeout not applied", elapsed)
	}
}

func TestCheckAll(t *testing.T) {
	domains := []string{"a", "b", "c", "d", "e", "f"}
	const workers = 3

	var inFlight, maxInFlight atomic.Int32
	check := func(buf *bytes.Buffer, domain string) bool {
		n := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			m := maxInFlight.Load()
			if n <= m || maxInFlight.CompareAndSwap(m, n) {
				break
			}
		}

		// Earlier domains take longer, so they finish out of order
		i := slices.Index(domains, domain)
		time.Sleep(time.Duration(len(domains)-i) * 10 * time.Millisecond)
		buf.WriteString(domain + "\n")
		return domain != "d"
	}

	var out bytes.Buffer
	ok := checkAll(&out, domains, workers, check)

	if ok {
		t.Error("checkAll() = true, want false when one check fails")
	}
	if got, want := out.String(), "a\nb\nc\nd\ne\nf\n"; got != want {
		t.Errorf("output = %q, want %q (input order)", got, want)
	}
	if m := maxInFlight.Load(); m < 2 || m > workers {
		t.Errorf("max concurrent checks = %d, want 2..%d", m, workers)
	}
}

func TestCheckAllAllPass(t *testing.T) {
	check := func(*bytes.Buffer, string) bool { return true }
	if !checkAll(&bytes.Buffer{}, []string{"a", "b"}, 8, check) {
		t.Error("checkAll() = false, want true when every check passes")
	}
}

func TestCheckAllEmpty(t *testing.T) {
	check := func(*bytes.Buffer, string) bool {
		t.Error("check called with no domains")
		return false
	}
	var out bytes.Buffer
	if !checkAll(&out, nil, 8, check) {
		t.Error("checkAll() = false, want true for no domains")
	}
	if out.Len() != 0 {
		t.Errorf("output = %q, want empty", out.String())
	}
}
