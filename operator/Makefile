ENVTEST_K8S_VERSION ?= 1.28.0

.PHONY: tidy build test lint envtest

tidy:
	go mod tidy

build:
	go build ./...

test: envtest
	KUBEBUILDER_ASSETS=$(shell setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir .envtest-bin -p path) \
	go test ./... -v

lint:
	golangci-lint run ./...

envtest:
	@which setup-envtest > /dev/null 2>&1 || go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
