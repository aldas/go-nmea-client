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
	./scripts/coverage.sh html;

actisense: # builds Actisense reader utility
	@go build -o actisense-reader cmd/actisense/main.go

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

download-canboat-pgns: # Downloads latest Canboat PNGs definitions (pgns.json) from Canboat repository
	@curl https://raw.githubusercontent.com/canboat/canboat/master/docs/canboat.json -o canboat/testdata/canboat.json
