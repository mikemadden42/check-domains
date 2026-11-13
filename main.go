// Command check-domains resolves each domain listed in a file (domains.txt
// by default, or the path given with -f) and reports its IP addresses and
// HTTP status code.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// maxConcurrent is how many domains are checked at the same time
const maxConcurrent = 8

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

	check := func(buf *bytes.Buffer, domain string) bool {
		return checkDomain(buf, domain, net.LookupHost, client)
	}
	if !checkAll(os.Stdout, domains, maxConcurrent, check) {
		os.Exit(1)
	}
}

// checkAll runs check on each domain, at most workers at a time, and writes
// each domain's report to out in input order. It returns false if any check
// failed.
func checkAll(out io.Writer, domains []string, workers int, check func(buf *bytes.Buffer, domain string) bool) bool {
	type result struct {
		report []byte
		ok     bool
	}

	// One buffered channel per domain so results can be printed in order
	results := make([]chan result, len(domains))
	for i := range results {
		results[i] = make(chan result, 1)
	}

	jobs := make(chan int)
	go func() {
		for i := range domains {
			jobs <- i
		}
		close(jobs)
	}()

	for range min(workers, len(domains)) {
		go func() {
			for i := range jobs {
				var buf bytes.Buffer
				ok := check(&buf, domains[i])
				results[i] <- result{buf.Bytes(), ok}
			}
		}()
	}

	// Print each report as soon as it and all earlier ones are done
	allOK := true
	for _, ch := range results {
		r := <-ch
		if _, err := out.Write(r.report); err != nil {
			fmt.Fprintf(os.Stderr, "Writing output failed: %v\n", err)
		}
		allOK = allOK && r.ok
	}
	return allOK
}

// checkDomain runs the DNS and HTTP checks for one domain and writes its
// report to buf. It returns false if any check failed.
func checkDomain(buf *bytes.Buffer, domain string, lookup func(host string) ([]string, error), client *http.Client) bool {
	fmt.Fprintf(buf, "Checking domain: %s\n", domain)

	// DNS lookup
	addresses, err := lookup(domain)
	if err != nil {
		// Handle missing domain gracefully
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			fmt.Fprintf(buf, "  Domain does not exist: %s\n", domain)
		} else {
			fmt.Fprintf(buf, "  DNS lookup failed: %v\n", err)
		}
		return false
	}

	// Print all resolved IP addresses
	fmt.Fprintf(buf, "  IP addresses:\n")
	for _, addr := range addresses {
		fmt.Fprintf(buf, "    - %s\n", addr)
	}

	// HTTP request
	ok := true
	resp, err := client.Get("http://" + domain)
	if err != nil {
		fmt.Fprintf(buf, "  HTTP request failed: %v\n", err)
		ok = false
	} else {
		fmt.Fprintf(buf, "  HTTP status code: %d\n", resp.StatusCode)
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(buf, "  Closing response body failed: %v\n", err)
		}
	}

	// Add more checks here (e.g., WHOIS lookup, MX records, etc.)
	return ok
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
