TOP_BUILDDIR ?= .
DEFAULT_TARGET ?= all
GOBIN_DIR := ${TOP_BUILDDIR}/bin
OUT_DIR := ${TOP_BUILDDIR}/out

.PHONY: default_target
_default_target: ${OUT_DIR} ${GOBIN_DIR} tools ${DEFAULT_TARGET}

${OUT_DIR}:
	@mkdir -p ${OUT_DIR}

${GOBIN_DIR}:
	@mkdir -p ${GOBIN_DIR}

${OUT_DIR}/creds: ${OUT_DIR}
	@mkdir -p ${OUT_DIR}/creds

# Run tests
test: fmt vet lint
	go test ./... -coverprofile cover.out

lint: tools
	golangci-lint run

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

protoc-gen-ybrpc: ${GOBIN}/protoc-gen-ybrpc

${GOBIN}/protoc-gen-ybrpc:
	${MAKE} -C ${TOP_BUILDDIR}/protoc-gen-ybrpc

####################################
#             TOOLS
####################################
.PHONY: tools golangci-lint ginkgo cobra swagger protoc-gen-go
tools: golangci-lint ginkgo cobra swagger protoc-gen-go

golangci-lint: ${GOBIN_DIR}/golangci-lint
${GOBIN_DIR}/golangci-lint:
	go install -modfile=${TOP_BUILDDIR}/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint

ginkgo: ${GOBIN}/ginkgo
${GOBIN}/ginkgo:
	go install -modfile=${TOP_BUILDDIR}/go.mod github.com/onsi/ginkgo/ginkgo

cobra: ${GOBIN}/cobra
${GOBIN}/cobra:
	go install -modfile=${TOP_BUILDDIR}/go.mod github.com/spf13/cobra/cobra

swagger: ${GOBIN}/swagger
${GOBIN}/swagger:
	go install -modfile=${TOP_BUILDDIR}/go.mod github.com/go-swagger/go-swagger/cmd/swagger

protoc-gen-go: ${GOBIN}/protoc-gen-go
${GOBIN}/protoc-gen-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
