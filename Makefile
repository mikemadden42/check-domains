build:
	goreleaser build --snapshot --clean

check:
	golangci-lint run

release:
	goreleaser release --snapshot --clean

test:
	go test ./...

.PHONY: build check release test
