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

##@ Testing

K8S_VERSION  := 1.28
LINT_TIMEOUT := 300s

lint: ## Run linters.
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --timeout=$(LINT_TIMEOUT)

SETUP := $(GO) run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest use $(K8S_VERSION) -p path
setup-envtest: ## Setup envtest. This is automatically run by the test target.
	$(SETUP) 1> /dev/null

RICHGO       ?= $(GO) run github.com/kyoh86/richgo@v0.3.12
TEST_TIMEOUT ?= 300s
TEST_ARGS    ?= -v -cover -covermode=atomic -coverprofile=cover.out -timeout=$(TEST_TIMEOUT)
test: generate setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(SETUP))" \
		$(RICHGO) test $(TEST_ARGS) ./...

CI_TARGETS := lint test
ifeq ($(CI),true)
	CI_TARGETS := test
endif
ci-test: $(CI_TARGETS) ## Run all CI tests.

##@ Development

generate: ## Run code generators.
	$(GO) generate ./...

RAW_REPO_URL ?= https://github.com/webmeshproj/storage-provider-k8s/raw/main
BUNDLE := deploy/bundle.yaml
bundle: generate ## Bundle a distribution manifest of CRDs and required Roles.
	@echo "+ Writing bundle to $(BUNDLE)"
	@echo "# Source: $(RAW_REPO_URL)/$(BUNDLE)" > $(BUNDLE)
	@for i in `find deploy/ -type f` ; do \
		[[ "$$i" =~ $(BUNDLE) ]] && continue ; \
		echo "---" >> $(BUNDLE) ; \
		echo "# Source: $$i" >> $(BUNDLE) ; \
		cat $$i | sed --posix -s -u 1,1d >> $(BUNDLE) ; \
	done
