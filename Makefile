.DEFAULT_GOAL  := build
CMD            := autoscan
GOARCH         := $(shell go env GOARCH)
GOOS           := $(shell go env GOOS)
TARGET         := ${GOOS}_${GOARCH}
DIST_PATH      := dist
BUILD_PATH     := ${DIST_PATH}/${CMD}_${TARGET}
GO_FILES       := $(shell find . -path ./vendor -prune -or -type f -name '*.go' -print)
GO_PACKAGES    := $(shell go list -mod vendor ./...)
GIT_COMMIT     := $(shell git rev-parse --short HEAD)
TIMESTAMP      := $(shell date +%s)
VERSION        ?= 0.0.0-dev
CGO            := 1

# Deps
.PHONY: check_golangci
check_golangci:
	@command -v golangci-lint >/dev/null || (echo "golangci-lint is required."; exit 1)

.PHONY: test
test: ## Run tests
	@echo "*** go test ***"
	go test -cover -v -race ${GO_PACKAGES}

.PHONY: lint
lint: check_golangci ## Run linting
	@echo "*** golangci-lint ***"
	golangci-lint run --timeout 10m

.PHONY: vendor
vendor: ## Vendor files and tidy go.mod
	go mod vendor
	go mod tidy

.PHONY: vendor_update
vendor_update: ## Update vendor dependencies
	go get -u ./...
	${MAKE} vendor

.PHONY: build
build: vendor ${BUILD_PATH}/${CMD} ## Build application

# Binary
${BUILD_PATH}/${CMD}: ${GO_FILES} go.sum
	@echo "Building for ${TARGET}..." && \
	mkdir -p ${BUILD_PATH} && \
	CGO_ENABLED=${CGO} go build \
		-mod vendor \
		-trimpath \
		-ldflags "-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.Timestamp=${TIMESTAMP}" \
		-o ${BUILD_PATH}/${CMD} \
		./cmd/autoscan