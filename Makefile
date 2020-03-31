# Copyright 2019 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
NATIVEOS	 := $(shell go env GOOS)
NATIVEARCH	 := $(shell go env GOARCH)
PROJECT      := nri-discovery-kubernetes
BINARY_NAME   = $(PROJECT)
IMAGE_NAME   ?= newrelic/nri-discovery-kubernetes
GOPATH := $(shell go env GOPATH)
GORELEASER_VERSION := v0.129.0
GORELEASER_SHA256 := e9e61de6565ad4acbe33a944abbeaf0d75582c10b89b793c99acd41a0846c166
GORELEASER_BIN ?= bin/goreleaser
GOLANGCILINT_VERSION = v1.24.0
GOLANGCI_LINT_BIN = bin/golangci-lint

all: build

build: check-version clean validate test compile

clean:
	@echo "=== $(PROJECT) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin
	@rm -rfv target

tools: bin check-version tools-golangci-lint
	@echo "=== $(PROJECT) === [ tools ]: Installing tools required by the project..."

$(GOLANGCI_LINT_BIN):
	@echo "installing GolangCI lint"
	@(wget -qO - https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s $(GOLANGCILINT_VERSION) )

tools-golangci-lint: $(GOLANGCI_LINT_BIN)

fmt:
	@go fmt ./...

bin:
	@mkdir -p bin

deps: tools deps-only

deps-only:
	@echo "=== $(PROJECT) === [ deps ]: Installing package dependencies required by the project..."
	@go mod download

validate: deps
	@echo "=== $(PROJECT) === [ validate ]: Validating source code running golangci-lint..."
	@${GOLANGCI_LINT_BIN} --version
	@${GOLANGCI_LINT_BIN} run

compile: deps
	@echo "=== $(PROJECT) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./cmd/discovery/

compile-only: deps-only
	@echo "=== $(PROJECT) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./cmd/discovery

test: deps
	@echo "=== $(PROJECT) === [ test ]: Running unit tests..."
	@go test -race ./...

test/skaffold:
	@skaffold dev -f ./deploy/skaffold.yaml

test/skaffold/gcp:
	@skaffold dev -f ./deploy/skaffold.yaml -p gcp

check-version:
ifdef GOOS
ifneq ("$(GOOS)" "$(NATIVEOS)")
	$(error GOOS is not $(NATIVEOS). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif
ifdef GOARCH
ifneq ("$(GOARCH)" "$(NATIVEARCH)")
	$(error GOARCH variable is not $(NATIVEARCH). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif

$(GORELEASER_BIN): bin
	@echo "=== $(PROJECT) === [ release/deps ]: Installing goreleaser"
	@(wget -qO /tmp/goreleaser.tar.gz https://github.com/goreleaser/goreleaser/releases/download/$(GORELEASER_VERSION)/goreleaser_Linux_x86_64.tar.gz)
	@(tar -xf  /tmp/goreleaser.tar.gz -C bin/)
	@(rm -f /tmp/goreleaser.tar.gz)

release/deps: $(GORELEASER_BIN)

release: release/deps
	@echo "=== $(PROJECT) === [ release ]: Releasing new version..."
	@$(GORELEASER_BIN) release
	@$(MAKE) snyk/monitor

release/test: release/deps
	@echo "=== $(PROJECT) === [ release/test ]: Testing releasing new version..."
	@$(GORELEASER_BIN) release --snapshot --skip-publish --rm-dist

snyk: deps-only
	@echo "=== $(PROJECT) === [ snyk ]: Running snyk..."
	@snyk test --file=go.mod --org=ohai

snyk/monitor: deps-only
	@echo "=== $(PROJECT) === [ snyk/monitor ]: Running snyk..."
	@snyk monitor --file=go.mod --org=ohai

.PHONY: all fmt build clean tools tools-update deps deps-only validate compile compile-only test check-version tools-golangci-lint docker-build release release/deps release/test snyk snyk/monitor docker-release
