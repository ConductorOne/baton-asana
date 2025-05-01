GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
BUILD_DIR = dist/${GOOS}_${GOARCH}

ifeq ($(GOOS),windows)
OUTPUT_PATH = ${BUILD_DIR}/baton-asana.exe
else
OUTPUT_PATH = ${BUILD_DIR}/baton-asana
endif

ifeq ($(GOOS),windows)
SCIM_TEST_OUTPUT_PATH = ${BUILD_DIR}/baton-asana-scim-test.exe
else
SCIM_TEST_OUTPUT_PATH = ${BUILD_DIR}/baton-asana-scim-test
endif

.PHONY: build
build:
	go build -o ${OUTPUT_PATH} ./cmd/baton-asana

.PHONY: build-scim-test
build-scim-test:
	go build -o ${SCIM_TEST_OUTPUT_PATH} ./cmd/baton-asana-scim-test

.PHONY: update-deps
update-deps:
	go get -d -u ./...
	go mod tidy -v
	go mod vendor

.PHONY: add-dep
add-dep:
	go mod tidy -v
	go mod vendor

.PHONY: lint
lint:
	golangci-lint run
