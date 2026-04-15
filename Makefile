.PHONY: test test-integration test-e2e coverage lint clean

test:
	go test ./internal/...

test-integration:
	go test -tags=integration ./internal/integration/...

test-e2e:
	go test -tags=e2e ./test/e2e/...

coverage:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./internal/... ./cmd/...

clean:
	rm -rf coverage.out
