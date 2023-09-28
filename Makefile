SHELL := /bin/bash

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

GOPATH   ?= $(shell $(GO) env GOPATH)
GOBIN    ?= $(GOPATH)/bin
LOCALBIN := $(CURDIR)/bin

##@ Testing

K8S_VERSION  := 1.28
LINT_TIMEOUT := 10m

lint: ## Run linters.
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --timeout=$(LINT_TIMEOUT)

SETUP := $(GO) run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest use $(K8S_VERSION) --bin-dir $(LOCALBIN) -p path
setup-envtest: ## Setup envtest. This is automatically run by the test target.
	$(SETUP) 1> /dev/null

RICHGO       ?= $(GO) run github.com/kyoh86/richgo@v0.3.12
TEST_TIMEOUT ?= 180s
test: setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(SETUP))" \
		$(RICHGO) test -v -cover -covermode=atomic -coverprofile=cover.out -timeout=$(TEST_TIMEOUT) ./...

CI_TARGETS := lint test
ifeq ($(CI),true)
	CI_TARGETS := test
endif
ci-test: ## Run all CI tests.
	$(MAKE) $(CI_TARGETS)