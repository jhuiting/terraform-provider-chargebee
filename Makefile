MODULE := github.com/jhuiting/terraform-provider-chargebee

# Local test-site credentials (gitignored). Plain KEY=value lines, no quotes.
-include .env
export CHARGEBEE_SITE CHARGEBEE_API_KEY

.PHONY: default build install lint fmt verify test testacc docs deps

default: verify

build:
	@go build ./...

install:
	@go install .

lint:
	@golangci-lint run

fmt:
	@go fmt ./...
	@goimports -w -local $(MODULE) .

verify: fmt lint build test
	@echo "✅ Code verification complete"

test:
	@go test ./... -count=1

testacc:
	@TF_ACC=1 go test ./internal/provider/ -v -count=1 -timeout 30m

docs:
	@cd tools && go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir .. --provider-name chargebee

deps:
	@go mod download
	@go mod tidy
