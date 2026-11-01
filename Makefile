GOFLAGS ?= -mod=readonly -buildvcs=false
export GOFLAGS
GOWORK := off
export GOWORK

IMG ?= ghcr.io/wyndure/octelium-operator:dev
BINARY ?= bin/manager
CONTAINER_TOOL ?= docker
PLATFORMS ?= linux/amd64,linux/arm64
YEAR ?= $(shell date +%Y)
LOCALBIN ?= $(CURDIR)/.tools
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
KUSTOMIZE ?= $(LOCALBIN)/kustomize
HELM ?= helm
KUBECTL ?= kubectl
CONTROLLER_TOOLS_VERSION := v0.22.0
ENVTEST_VERSION := v0.25.0
ENVTEST_K8S_VERSION ?= 1.37.0
GOLANGCI_LINT_VERSION := v2.15.0
KUSTOMIZE_VERSION := v5.8.1
SHELL := /bin/sh
.SHELLFLAGS := -ec

.DEFAULT_GOAL := build
.PHONY: help manifests generate fmt vet build run test test-race lint lint-fix \
 controller-gen envtest setup-envtest golangci-lint kustomize docker-build docker-push \
 docker-buildx build-installer install deploy helm-sync helm-lint helm-package clean

help: ## List available targets.
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*?##/ {printf "%-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

manifests: controller-gen ## Generate CRDs and controller RBAC.
	"$(CONTROLLER_GEN)" crd paths="./api/..." output:crd:artifacts:config=config/crd/bases
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role paths="./internal/controller/..." output:rbac:dir=config/rbac
	go run ./cmd/crd-schema

generate: controller-gen ## Generate Kubernetes deep copies.
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt",year=$(YEAR) paths="./api/..."

fmt: ## Format Go source.
	gofmt -w cmd api internal

vet: ## Run go vet.
	go vet ./...

build: ## Build the operator binary.
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$(BINARY)" ./cmd

run: ## Run using the current Kubernetes configuration.
	go run ./cmd

test: manifests generate setup-envtest ## Run unit and isolated Kubernetes API tests.
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -coverprofile=coverage.out ./...
	sh ci/test-image-tags.sh

test-race: manifests generate setup-envtest ## Run tests with the Go race detector (requires a C compiler).
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./...
	sh ci/test-image-tags.sh

lint: golangci-lint ## Lint Go source.
	"$(GOLANGCI_LINT)" run

lint-fix: golangci-lint ## Apply lint and formatting fixes.
	"$(GOLANGCI_LINT)" fmt
	"$(GOLANGCI_LINT)" run --fix

controller-gen: $(CONTROLLER_GEN)
envtest: $(ENVTEST)
golangci-lint: $(GOLANGCI_LINT)
kustomize: $(KUSTOMIZE)
setup-envtest: envtest
	"$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path

$(CONTROLLER_GEN):
	mkdir -p "$(dir $@)"
	GOBIN="$(abspath $(dir $@))" go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

$(ENVTEST):
	mkdir -p "$(dir $@)"
	GOBIN="$(abspath $(dir $@))" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

$(GOLANGCI_LINT):
	mkdir -p "$(dir $@)"
	GOBIN="$(abspath $(dir $@))" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(KUSTOMIZE):
	mkdir -p "$(dir $@)"
	GOBIN="$(abspath $(dir $@))" go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

docker-build: ## Build the runtime image.
	$(CONTAINER_TOOL) build --tag "$(IMG)" .

docker-push: ## Push the runtime image.
	$(CONTAINER_TOOL) push "$(IMG)"

docker-buildx: ## Build and push amd64 and arm64 images.
	$(CONTAINER_TOOL) buildx build --platform "$(PLATFORMS)" --push --tag "$(IMG)" .

build-installer: manifests generate kustomize ## Render an install bundle without modifying source manifests.
	mkdir -p dist
	mkdir -p "$(LOCALBIN)"
	@overlay=$$(mktemp -d "$(CURDIR)/config/.installer.XXXXXX"); \
	trap 'rm -rf "$$overlay"' EXIT HUP INT TERM; \
	printf 'resources:\n- ../default\n' > "$$overlay/kustomization.yaml"; \
	(cd "$$overlay" && "$(KUSTOMIZE)" edit set image "ghcr.io/wyndure/octelium-operator=$(IMG)"); \
	"$(KUSTOMIZE)" build "$$overlay" > dist/install.yaml

install: ## Install CRDs into the current cluster.
	$(KUBECTL) apply --server-side -f config/crd/bases

deploy: build-installer ## Apply the Kustomize installation bundle.
	$(KUBECTL) apply --server-side -f dist/install.yaml

helm-sync: manifests ## Refresh generated CRDs and RBAC in the Helm chart.
	./hack/sync-helm-chart.sh

helm-lint: helm-sync ## Validate the Helm chart.
	$(HELM) lint --strict charts/octelium-operator

helm-package: helm-lint ## Package the Helm chart into dist/.
	mkdir -p dist
	$(HELM) package charts/octelium-operator --destination dist

clean: ## Remove build artifacts.
	rm -rf bin dist coverage.out cover.out
