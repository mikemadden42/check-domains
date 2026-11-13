# TODO

Findings from a project review (2026-11-12), in priority order.

## Bugs in `main.go`

- [x] **Early exit on body close error** (`main.go:50-53`): `return` from `main` skips all remaining domains silently with exit code 0. Log the error and `continue` instead.
- [x] **No HTTP timeout** (`main.go:45`): `http.Get` uses the default client, which never times out. Use `&http.Client{Timeout: 10 * time.Second}`.
- [x] **Error-string matching for NXDOMAIN** (`main.go:30`): fragile across resolvers and OSes. Use the typed error:
  ```go
  var dnsErr *net.DNSError
  if errors.As(err, &dnsErr) && dnsErr.IsNotFound { ... }
  ```
- [x] **Exit code is always 0** (`main.go:17-20`): print errors to stderr and `os.Exit(1)` on failure to read the file (and optionally if any domain fails).
- [ ] **Only plain HTTP is checked** (`main.go:45`): redirects are followed, so the reported status is the final hop's. Consider adding an `https://` check or reporting the redirect chain.

## Config: lint and release are broken

- [x] **Fix `.golangci.yml`** (fails `golangci-lint config verify` on v2.15). Or run `golangci-lint migrate`.
  - `version: 2` should be `version: "2"`
  - `disable-all: true` should be `default: none`
  - `linters-settings` should move under `linters.settings`, with goimports settings under `formatters.settings`
  - `issues.exclude-dirs` should be `linters.exclusions.paths`
- [x] **Resolve the 2 lint issues:**
  - G304 at `main.go:61`: annotate with `//nolint:gosec` and a reason once the path comes from user input
  - Add a package comment (`// Command check-domains ...`)
- [x] **Fix `.goreleaser.yaml`** (fails `goreleaser check`):
  - Add `version: 2`
  - `snapshot.name_template` should be `snapshot.version_template`
  - Drop the no-op `go generate ./...` hook

## Usability

- [x] Take the domains file as a flag/argument (and maybe stdin) instead of hardcoding `domains.txt`.
- [x] Check domains concurrently with a small worker pool.
- [x] Skip `#` comment lines in `readDomains`.
- [x] Replace stale sample entry `bard.google.com` in `domains.txt`.

## CI and tests

- [x] Bump `go.mod` from `go 1.21` to a supported version; test the current two releases in CI (`stable`/`oldstable` or `go-version-file: go.mod`) instead of 1.23–1.25.
- [x] Add tests for `readDomains` (blank lines, whitespace, CRLF, comments).
- [x] Split the per-domain check into a function taking a resolver/HTTP client so it can be tested without the network.
- [x] Run the linter in CI with `golangci/golangci-lint-action`.
- [x] Bump `actions/checkout` and `actions/setup-go` (now v7).
- [ ] Consider Dependabot for GitHub Actions.

## Minor

- [ ] Expand the README with usage and input format.
- [x] Add a `test` target to the Makefile.
