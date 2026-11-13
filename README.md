# check-domains

Check a list of domains from the command line. For each domain, `check-domains`:

1. Resolves it with DNS and lists its IP addresses.
2. Sends an HTTP request to `http://<domain>` and reports the status code.

## Install

Requires Go 1.26 or later.

```sh
go install github.com/mikemadden42/check-domains@latest
```

This puts `check-domains` in `$(go env GOPATH)/bin`.

Or build from source:

```sh
git clone https://github.com/mikemadden42/check-domains.git
cd check-domains
go build
```

## Usage

```sh
./check-domains                  # reads domains.txt in the current directory
./check-domains -f my-list.txt   # reads another file
```

### Input format

One domain per line. Blank lines, surrounding whitespace, and lines starting with `#` are ignored.

```text
# Production
example.com
www.example.com

# Staging
staging.example.com
```

### Output

```text
Checking domain: example.com
  IP addresses:
    - 192.0.2.10
    - 2001:db8::10
  HTTP status code: 200
Checking domain: missing.example.com
  Domain does not exist: missing.example.com
```

Up to 8 domains are checked at a time, and results are always printed in the same order as the input file. HTTP requests time out after 10 seconds. Redirects are followed, so the status code is the final response's.

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Every domain resolved and answered an HTTP request. |
| 1 | The input file couldn't be read, or at least one domain failed to resolve or answer. |

An HTTP error status such as 404 or 500 is reported but doesn't count as a failure.

## Development

```sh
make test    # run tests
make check   # run golangci-lint
make build   # snapshot build of all release targets with goreleaser
```

## License

MIT. See [LICENSE](LICENSE).
