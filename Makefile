install:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	wget -O $(HOME)/.local/bin/sleek https://github.com/nrempel/sleek/releases/download/v0.5.0/sleek-linux-x86_64
	chmod +x $(HOME)/.local/bin/sleek

vet: fmt
	go vet ./...
	staticcheck ./...

fmt:
	go fmt ./...
	sleek *.sql

sqlc: fmt
	sqlc generate

test:
	go test -v -timeout=10m ./...

test-short:
	go test -v -short ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

check: fmt vet test-build
	@echo "✓ All checks passed"
