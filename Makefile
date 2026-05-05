.PHONY: build build-bin test test-integration test-e2e coverage lint clean

build:
	go build ./...

build-bin:
	mkdir -p bin
	for d in cmd/*; do \
		if [ -f "$$d/main.go" ]; then \
			go build -o "bin/$$(basename "$$d")" "./$$d"; \
		fi; \
	done

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
	rm -rf coverage.out bin/
