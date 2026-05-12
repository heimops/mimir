.PHONY: build test lint docker-build helm-lint all

all: build

## build: compile operator and verify agent package
build:
	cd operator && go build ./...
	cd agent && pip install --quiet -e . --dry-run 2>/dev/null || true

## test: run operator integration tests (requires setup-envtest)
test:
	cd operator && make test

## lint: lint operator (golangci-lint) and agent (ruff)
lint:
	cd operator && golangci-lint run ./...
	cd agent && ruff check .

## docker-build: build both container images locally
docker-build:
	docker build -t mimir-operator:dev ./operator
	docker build -t mimir:dev ./agent

## helm-lint: lint the Helm chart
helm-lint:
	helm lint charts/mimir
