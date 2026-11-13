// Command check-domains resolves each domain listed in a file (domains.txt
// by default, or the path given with -f) and reports its IP addresses and
// HTTP status code.
package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// File of domains to check, one per line
	filePath := flag.String("f", "domains.txt", "file of domains to check, one per line")
	flag.Parse()

	// Read domain names from file
	domains, err := readDomains(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading domains: %v\n", err)
		os.Exit(1)
	}

	// HTTP client with a timeout so a slow host can't hang the run
	client := &http.Client{Timeout: 10 * time.Second}

	// Track whether any check failed so the exit code reflects it
	failed := false

	// Loop through each domain
	for _, domain := range domains {
		fmt.Printf("Checking domain: %s\n", domain)

		// DNS lookup
		addresses, err := net.LookupHost(domain)
		if err != nil {
			// Handle missing domain gracefully
			var dnsErr *net.DNSError
			if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
				fmt.Printf("  Domain does not exist: %s\n", domain)
			} else {
				fmt.Printf("  DNS lookup failed: %v\n", err)
			}
			failed = true
			continue // Skip to the next domain
		}

		// Print all resolved IP addresses
		fmt.Printf("  IP addresses:\n")
		for _, addr := range addresses {
			fmt.Printf("    - %s\n", addr)
		}

		// HTTP request
		resp, err := client.Get("http://" + domain)
		if err != nil {
			fmt.Printf("  HTTP request failed: %v\n", err)
			failed = true
		} else {
			fmt.Printf("  HTTP status code: %d\n", resp.StatusCode)
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("  Closing response body failed: %v\n", err)
			}
		}

		// Add more checks here (e.g., WHOIS lookup, MX records, etc.)
	}

	if failed {
		os.Exit(1)
	}
}

func readDomains(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath) //nolint:gosec // G304: the path comes from the user's -f flag by design
	if err != nil {
		return nil, err
	}

	// Split lines, trim whitespaces, and filter empty strings and # comments
	var domains []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			domains = append(domains, trimmedLine)
		}
	}
	return domains, nil
}
