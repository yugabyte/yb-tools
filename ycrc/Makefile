TOP_BUILDDIR = ../.
include ../global.mk

.PHONEY: all ycrc

BIN_NAME ?= ycrc
BIN_DIR := ${TOP_BUILDDIR}/bin

VERSION ?= $(shell git rev-parse HEAD)

all: tools test ycrc

ycrc:
	CGO_ENABLED=0 go build -a -ldflags=" -X 'github.com/yugabyte/yb-tools/ycrc/cmd.Version=${VERSION}'" -o ${BIN_DIR}/${BIN_NAME} main.go


all-distros:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags=" -X 'github.com/yugabyte/yb-tools/ycrc/cmd.Version=${VERSION}'" -o ${BIN_DIR}/${BIN_NAME} main.go
	zip ${BIN_DIR}/${BIN_NAME}-linux-amd64-${VERSION}-$$(date -I).zip ${BIN_DIR}/${BIN_NAME}
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -ldflags=" -X 'github.com/yugabyte/yb-tools/ycrc/cmd.Version=${VERSION}'" -o ${BIN_DIR}/${BIN_NAME}.exe main.go
	zip ${BIN_DIR}/${BIN_NAME}-windows-amd64-${VERSION}-$$(date -I).zip ${BIN_DIR}/${BIN_NAME}.exe
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -ldflags=" -X 'github.com/yugabyte/yb-tools/ycrc/cmd.Version=${VERSION}'" -o ${BIN_DIR}/${BIN_NAME} main.go
	zip ${BIN_DIR}/${BIN_NAME}-osx-amd64-${VERSION}-$$(date -I).zip ${BIN_DIR}/${BIN_NAME}
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -a -ldflags=" -X 'github.com/yugabyte/yb-tools/ycrc/cmd.Version=${VERSION}'" -o ${BIN_DIR}/${BIN_NAME} main.go
	zip ${BIN_DIR}/${BIN_NAME}-osx-arm64-${VERSION}-$$(date -I).zip ${BIN_DIR}/${BIN_NAME}
