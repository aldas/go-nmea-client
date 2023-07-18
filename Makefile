PROJECT_NAME := go-nmea-client
PKG := "github.com/aldas/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/...)

.PHONY: init lint test coverage coverhtml

.DEFAULT_GOAL := check

check: lint vet race ## check project

init:
	git config core.hooksPath ./scripts/.githooks
	@go install honnef.co/go/tools/cmd/staticcheck@latest

lint: ## Lint the files
	@staticcheck -tests=false ${PKG_LIST}

vet: ## Vet the files
	@go vet ${PKG_LIST}

test: ## Run unittests
	@go test -short ${PKG_LIST}

goversion ?= "1.19"
test_version: ## Run tests inside Docker with given version (defaults to 1.19 oldest supported). Example: make test_version goversion=1.19
	@docker run --rm -it -v $(shell pwd):/project golang:$(goversion) /bin/sh -c "cd /project && make init check"

race: ## Run data race detector
	@go test -race -short ${PKG_LIST}

benchmark: ## Run benchmarks
	@go test -run="-" -bench=".*" ${PKG_LIST}

coverage: ## Generate global code coverage report
	./scripts/coverage.sh;

coverhtml: ## Generate global code coverage report in HTML
	./scripts/coverage.sh html

build-reader: ## builds Actisense reader utility (for current architecture)
	@go build -ldflags="-s -w" -o n2k-reader cmd/n2kreader/main.go cmd/n2kreader/csv.go

build-reader-all: ## builds NMEA2000 reader utility (for different architectures)
	# Compiling binary file suitable for AMD64
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o n2k-reader-amd64 cmd/n2kreader/main.go cmd/n2kreader/csv.go
	# Compiling binary file suitable for MIPS32 (softfloat)
	@GOOS=linux GOARCH=mips GOMIPS=softfloat go build -ldflags="-s -w" -o n2k-reader-mips32 cmd/n2kreader/main.go cmd/n2kreader/csv.go
	# Compiling binary file suitable for ARM32v6 (Raspberry PI zero)
	@GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o n2k-reader-arm32v6 cmd/n2kreader/main.go cmd/n2kreader/csv.go
	# Compiling binary file suitable for ARM32v7 (Raspberry 2/3/+)
	@GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o n2k-reader-arm32v7 cmd/n2kreader/main.go cmd/n2kreader/csv.go
	# Compiling binary file suitable for ARM64 (Raspberry 64bit OS)
	@GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o n2k-reader-arm64 cmd/n2kreader/main.go cmd/n2kreader/csv.go

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

download-canboat-pgns: # Downloads Canboat PNG definitions (pgns.json) from Canboat repository
	# download canboat v4.10.0 PGNs
	@wget -O canboat/testdata/canboat.json https://raw.githubusercontent.com/canboat/canboat/v4.10.0/docs/canboat.json
	@cp canboat/testdata/canboat.json cmd/n2kreader/canboat.json
