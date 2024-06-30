package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	// Specify the file path here
	filePath := "domains.txt"

	// Read domain names from file
	domains, err := readDomains(filePath)
	if err != nil {
		fmt.Printf("Error reading domains: %v\n", err)
		return
	}

	// Loop through each domain
	for _, domain := range domains {
		fmt.Printf("Checking domain: %s\n", domain)

		// DNS lookup
		addresses, err := net.LookupHost(domain)
		if err != nil {
			// Handle missing domain gracefully
			if strings.HasPrefix(err.Error(), "lookup ") && strings.Contains(err.Error(), ": no such host") {
				fmt.Printf("  Domain does not exist: %s\n", domain)
			} else {
				fmt.Printf("  DNS lookup failed: %v\n", err)
			}
			continue // Skip to the next domain
		}

		// Print all resolved IP addresses
		fmt.Printf("  IP addresses:\n")
		for _, addr := range addresses {
			fmt.Printf("    - %s\n", addr)
		}

		// HTTP request
		resp, err := http.Get("http://" + domain)
		if err != nil {
			fmt.Printf("  HTTP request failed: %v\n", err)
		} else {
			fmt.Printf("  HTTP status code: %d\n", resp.StatusCode)
			err := resp.Body.Close()
			if err != nil {
				return
			}
		}

		// Add more checks here (e.g., WHOIS lookup, MX records, etc.)
	}
}

func readDomains(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Split lines, trim whitespaces, and filter empty strings
	var domains []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			domains = append(domains, trimmedLine)
		}
	}
	return domains, nil
}
