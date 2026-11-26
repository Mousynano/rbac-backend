.PHONY: fmt-check vet build test test-race test-integration vuln verify

fmt-check:
	@test -z "$$(gofmt -l cmd internal)"

vet:
	go vet ./...

build:
	go build ./cmd/...

test:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	@echo "Integration suite not implemented; gate cannot pass yet" && exit 1

vuln:
	govulncheck ./...

verify: fmt-check vet build test test-race test-integration vuln
