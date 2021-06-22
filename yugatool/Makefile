.PHONEY: all yugatool

all: test yugatool

yugatool:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o ./bin/yugatool main.go

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
tools: ${GOBIN}/ginkgo ${GOBIN}/golangci-lint ${GOBIN}/cobra

${GOBIN}/ginkgo:
	go install github.com/onsi/ginkgo/ginkgo

${GOBIN}/golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

${GOBIN}/cobra:
	go install github.com/spf13/cobra/cobra

clean:
	rm -rf bin
