.PHONEY: all protoc-gen-ybrpc

all: test protoc-gen-ybrpc

protoc-gen-ybrpc:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o ./bin/protoc-gen-ybrpc main.go

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

.PHONY: tools
tools: ${GOBIN}/ginkgo ${GOBIN}/golangci-lint

${GOBIN}/ginkgo:
	go install github.com/onsi/ginkgo/ginkgo

${GOBIN}/golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

clean:
	rm -rf bin
