BIN     := ./tmp/server
MODULE  := github.com/thiagotn/football-manager/football-api-go
MIGRATE_PATH := ../football-api/migrations

# Coverage packages: exclude internal/db (sqlc-generated, testable only via integration)
COVPKG := ./internal/apierror/...,./internal/config/...,./internal/handlers/...,./internal/middleware/...,./internal/services/...

.PHONY: run build test test-integration test-race test-all lint generate migrate clean coverage

## Run the server with live-reload (requires air)
run:
	air

## Build production binary
build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o $(BIN) ./cmd/server/main.go

## Run unit tests (no database required)
test:
	go test ./tests/unit/... -coverprofile=coverage-unit.out -covermode=atomic -coverpkg=$(COVPKG) -count=1

## Run tests with race detector
test-race:
	go test -race ./internal/... ./tests/unit/... -count=1

## Run integration tests (requires DATABASE_URL pointing to a test database)
test-integration:
	go test ./tests/integration/... -coverprofile=coverage-integration.out -covermode=atomic -coverpkg=$(COVPKG) -v -timeout 120s -count=1

## Run all tests and merge coverage profiles
test-all: test test-integration
	@head -1 coverage-unit.out > coverage.out
	@tail -n +2 coverage-unit.out >> coverage.out
	@tail -n +2 coverage-integration.out >> coverage.out

## Run linter
lint:
	golangci-lint run ./...

## Generate sqlc code from sql/queries/
generate:
	sqlc generate
	go generate ./...

## Apply database migrations
migrate:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL not set"; exit 1; fi
	migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" up

## Generate OpenAPI spec for Mintlify
docs:
	swag init -g cmd/server/main.go --output mintlify/ --outputTypes yaml --parseInternal
	@echo "openapi.yaml generated in mintlify/"

## Download dependencies
deps:
	go mod download
	go mod tidy

## Clean build artifacts
clean:
	rm -rf ./tmp ./coverage.out

## Show coverage in browser (unit tests only)
coverage: test
	go tool cover -html=coverage-unit.out

## Show combined coverage in browser (unit + integration)
coverage-all: test-all
	go tool cover -html=coverage.out
