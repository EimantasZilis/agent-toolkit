.PHONY: run test validate verify smoke-source

run:
	go run ./cmd/agent-toolkit

validate:
	go run ./cmd/agent-toolkit validate

verify: test
	go vet ./...
	go run ./cmd/agent-toolkit verify

test:
	go test ./...

smoke-source:
	go run ./cmd/agent-toolkit smoke-source
