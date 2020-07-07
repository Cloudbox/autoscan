.DEFAULT_GOAL  := build
CMD            := autoscan
TARGET         := $(shell go env GOOS)_$(shell go env GOARCH)
DIST_PATH      := dist
BUILD_PATH     := ${DIST_PATH}/${CMD}_${TARGET}
GO_FILES       := $(shell find . -path ./vendor -prune -or -type f -name '*.go' -print)
GIT_COMMIT     := $(shell git rev-parse --short HEAD)
TIMESTAMP      := $(shell date +%s)
VERSION        ?= 0.0.0-dev
CGO            := 1

# Deps
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