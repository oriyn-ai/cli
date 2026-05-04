.PHONY: build test vet lint fmt fmt-check tidy install tools

build:
	go build ./...

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

# Format every Go file in-place using gofmt + goimports (with local prefix grouping).
fmt:
	gofmt -s -w .
	goimports -local github.com/oriyn-ai/cli -w .

# Verify formatting without writing — exits non-zero if any file would change.
fmt-check:
	@out=$$(gofmt -s -l .); if [ -n "$$out" ]; then echo "gofmt drift:"; echo "$$out"; exit 1; fi
	@out=$$(goimports -local github.com/oriyn-ai/cli -l .); if [ -n "$$out" ]; then echo "goimports drift:"; echo "$$out"; exit 1; fi

lint:
	golangci-lint run

tidy:
	go mod tidy

install:
	go install ./...

# Install/update local dev tooling. Requires Go 1.26+ on PATH.
tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
