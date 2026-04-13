BINARY := rulegen
MODULE := github.com/konveyor/ai-rule-gen
GOFLAGS := -trimpath

.PHONY: build test test-race test-integration test-e2e test-all lint vet clean

build:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/rulegen/

test:
	go test ./internal/... ./cmd/...

test-race:
	go test -race ./internal/... ./cmd/...

test-integration:
	go test -tags=integration ./test/integration/...

test-e2e:
	go test -tags=e2e ./test/e2e/...

test-all: vet test-race

vet:
	go vet ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out
