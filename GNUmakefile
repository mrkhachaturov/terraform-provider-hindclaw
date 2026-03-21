BINARY_NAME ?= terraform-provider-hindclaw
HOSTNAME ?= registry.terraform.io
NAMESPACE ?= mrkhachaturov
NAME ?= hindclaw
VERSION ?= 0.0.0-dev
OS_ARCH ?= $$(go env GOOS)_$$(go env GOARCH)
PLUGIN_DIR ?= $(HOME)/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)
TERRAFORM_BIN ?= $(shell bash -lc 'command -v terraform || mise which terraform 2>/dev/null || true')

default: build

build:
	go build -trimpath -o $(BINARY_NAME)

install: build
	mkdir -p $(PLUGIN_DIR)
	cp $(BINARY_NAME) $(PLUGIN_DIR)/

fmt:
	gofmt -w -s .

vet:
	go vet ./...

generate:
	test -n "$(TERRAFORM_BIN)"
	PATH="$$(dirname "$(TERRAFORM_BIN)"):$$PATH" go generate ./...

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./internal/provider/ -v -timeout 120m

release-check: vet test build generate

.PHONY: build install fmt vet generate test testacc release-check
