SHELL := /bin/bash

# output directory, where all artifacts will be created and managed
OUTPUT_DIR ?= build/_output
# relative path to operator binary
OPERATOR = $(OUTPUT_DIR)/bin/build-operator

# golang cache directory path
GOCACHE ?= $(shell echo ${PWD})/$(OUTPUT_DIR)/gocache
# golang target architecture
GOARCH ?= amd64
# golang global flags
GO_FLAGS ?= -v -mod=vendor
# golang test floags
GO_TEST_FLAGS ?= -failfast -timeout=20m

# configure zap based logr
OPERATOR_FLAGS ?= --zap-level=1 --zap-level=debug --zap-encoder=console
# extra flags passed to operator-sdk
OPERATOR_SDK_EXTRA_ARGS ?= --debug

# test namespace name
TEST_NAMESPACE ?= default

# CI: tekton pipelines operator version
TEKTON_VERSION ?= v0.10.1
# CI: operator-sdk version
SDK_VERSION ?= v0.15.2

# test repository to store images build during end-to-end tests
TEST_IMAGE_REPO ?= quay.io/redhat-developer/build-e2e
# test repository secret, must be defined during runtime
TEST_IMAGE_REPO_SECRET ?=

# enable private git repository tests
TEST_PRIVATE_REPO ?= false
# github private repository url
TEST_PRIVATE_GITHUB ?=
# gitlab private repository url
TEST_PRIVATE_GITLAB ?=
# private repository authentication secret
TEST_SOURCE_SECRET ?=

.EXPORT_ALL_VARIABLES:

default: build

.PHONY: vendor
vendor: go.mod go.sum
	go mod vendor

.PHONY: build
build: $(OPERATOR)

$(OPERATOR): vendor
	go build $(GO_FLAGS) -o $(OPERATOR) cmd/manager/main.go

install-ginkgo:
	go get -u github.com/onsi/ginkgo/ginkgo
	go get -u github.com/onsi/gomega/...

test: test-unit test-e2e

.PHONY: test-unit
test-unit:
	GO111MODULE=on ginkgo \
		-randomizeAllSpecs \
		-randomizeSuites \
		-failOnPending \
		-nodes=4 \
		-compilers=2 \
		-slowSpecThreshold=240 \
		-race \
		-cover \
		-trace \
		internal/... \
		pkg/...

.PHONY: test-e2e
test-e2e:
	operator-sdk --verbose test local ./test/e2e \
		--up-local \
		--namespace="$(TEST_NAMESPACE)" \
		--go-test-flags="$(GO_TEST_FLAGS)" \
		--local-operator-flags="$(OPERATOR_FLAGS)" \
			$(OPERATOR_SDK_EXTRA_ARGS)

crds:
	-hack/crd.sh uninstall
	@hack/crd.sh install

local: crds build
	operator-sdk run --local --operator-flags="$(ZAP_ENCODER_FLAG)" $(OPERATOR_SDK_EXTRA_ARGS)

clean:
	rm -rf $(OUTPUT_DIR)

gen-fakes:
	./hack/generate-fakes.sh

travis: install-ginkgo
	./hack/install-operator-sdk.sh
	./hack/install-kind.sh
	./hack/install-tekton.sh
