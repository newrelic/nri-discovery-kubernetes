# Copyright 2019 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
NATIVEOS	 := $(shell go env GOOS)
NATIVEARCH	 := $(shell go env GOARCH)
PROJECT      := nri-discovery-kubernetes
BINARY_NAME   = $(PROJECT)
IMAGE_NAME   ?= newrelic/nri-discovery-kubernetes
GOPATH := $(shell go env GOPATH)
GORELEASER_VERSION := v2.4.4
GORELEASER_BIN ?= bin/goreleaser
GO_VERSION 		?= $(shell grep '^go ' go.mod | awk '{print $$2}')
BUILDER_IMAGE 	?= "ghcr.io/newrelic/coreint-automation:latest-go$(GO_VERSION)-ubuntu16.04"

all: build

build: check-version clean test compile

clean:
	@echo "=== $(PROJECT) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin
	@rm -rfv target

fmt:
	@go fmt ./...

deps:
	@echo "=== $(PROJECT) === [ deps ]: Installing package dependencies required by the project..."
	@go mod download

compile: deps
	@echo "=== $(PROJECT) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./cmd/discovery/

compile-only: deps-only
	@echo "=== $(PROJECT) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./cmd/discovery

test: deps
	@echo "=== $(PROJECT) === [ test ]: Running unit tests..."
	@go test -race ./...

test-integration: ## Runs all integration tests. Expects kubeconfig pointing to cluster with single node.
	@go test -tags integration ./...

test/skaffold:
	@skaffold dev -f ./deploy/skaffold.yaml

test/skaffold/gcp:
	@skaffold dev -f ./deploy/skaffold.yaml -p gcp

check-version:
ifdef GOOS
ifneq ("$(GOOS)", "$(NATIVEOS)")
	$(error GOOS is not $(NATIVEOS). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif
ifdef GOARCH
ifneq ("$(GOARCH)", "$(NATIVEARCH)")
	$(error GOARCH variable is not $(NATIVEARCH). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif

# rt-update-changelog runs the release-toolkit run.sh script by piping it into bash to update the CHANGELOG.md.
# It also passes down to the script all the flags added to the make target. To check all the accepted flags,
# see: https://github.com/newrelic/release-toolkit/blob/main/contrib/ohi-release-notes/run.sh
#  e.g. `make rt-update-changelog -- -v`
rt-update-changelog:
	curl "https://raw.githubusercontent.com/newrelic/release-toolkit/v1/contrib/ohi-release-notes/run.sh" | bash -s -- $(filter-out $@,$(MAKECMDGOALS))

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk

.PHONY: all fmt build clean tools tools-update deps deps-only compile compile-only test check-version docker-build release release/deps release/test snyk snyk/monitor docker-release rt-update-changelog
