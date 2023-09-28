SHELL := /bin/bash

NAME  ?= webmesh-k8s
REPO  ?= ghcr.io/webmeshproj
IMAGE ?= $(REPO)/$(NAME):latest
DISTROLESS_IMAGE ?= $(REPO)/$(NAME)-distroless:latest

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

GO    ?= go
ARCH  ?= $(shell $(GO) env GOARCH)
OS    ?= $(shell $(GO) env GOOS)

ifeq ($(OS),Windows_NT)
	OS := windows
endif

GOPATH ?= $(shell $(GO) env GOPATH)
GOBIN  ?= $(GOPATH)/bin

##@ Testing

LINT_TIMEOUT := 10m
lint: ## Run linters.
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --timeout=$(LINT_TIMEOUT)