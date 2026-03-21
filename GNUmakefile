default: build

build:
	go build -o terraform-provider-hindclaw

install: build
	mkdir -p ~/.terraform.d/plugins/hindclaw.pro/mrkhachaturov/hindclaw/0.1.0/$$(go env GOOS)_$$(go env GOARCH)
	cp terraform-provider-hindclaw ~/.terraform.d/plugins/hindclaw.pro/mrkhachaturov/hindclaw/0.1.0/$$(go env GOOS)_$$(go env GOARCH)/

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./internal/provider/ -v -timeout 120s

.PHONY: build install test testacc
