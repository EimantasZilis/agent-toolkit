run:
	go run ./cmd/agent-toolkit

install:
	bash ./scripts/setup.sh

install-cli:
	bash ./scripts/install.sh

update:
	bash ./scripts/update.sh

uninstall:
	./scripts/uninstall.sh

validate:
	go run ./cmd/agent-toolkit validate

verify: test
	go vet ./...
	go run ./cmd/agent-toolkit verify

test:
	go test ./...

smoke-source:
	go run ./cmd/agent-toolkit smoke-source
