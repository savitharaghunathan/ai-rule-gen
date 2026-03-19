BINARY := rulegen
MODULE := github.com/konveyor/ai-rule-gen
GOFLAGS := -trimpath

.PHONY: build test test-integration test-e2e lint clean

build:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/rulegen/

test:
	go test ./internal/...

test-integration:
	go test -tags=integration ./internal/integration/...

test-e2e:
	go test -tags=e2e ./test/e2e/...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out
