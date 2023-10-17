TOP_BUILDDIR = ../.
include ../global.mk


.PHONEY: all yb-support-tool test

# go compile options
BIN_NAME ?= yb-support-tool
BIN_DIR := ${TOP_BUILDDIR}/bin


VERSION ?= $(shell git rev-parse HEAD)


all: test yb-support-tool

yb-support-tool: test ${BIN_DIR}/${BIN_NAME}
${BIN_DIR}/${BIN_NAME}:
	CGO_ENABLED=0  go build -a -o $@ -ldflags=" \
	-X 'github.com/yugabyte/yb-tools/yb-support-tool/cmd.Version=${VERSION}'"

all-distros: windows linux osx

windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -o ${BIN_DIR}/${BIN_NAME}.exe -ldflags=" \
	-X 'github.com/yugabyte/yb-tools/yb-support-tool/cmd.Version=${VERSION}'"
	zip -j ${BIN_DIR}/${BIN_NAME}-WINDOWS-x86_64-${VERSION}-$$(date -I).zip ${BIN_DIR}/${BIN_NAME}.exe


linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ${BIN_DIR}/${BIN_NAME} -ldflags=" \
	-X 'github.com/yugabyte/yb-tools/yb-support-tool/cmd.Version=${VERSION}'"
	zip -j ${BIN_DIR}/${BIN_NAME}-LINUX-x86_64-${VERSION}-$$(date -I).zip ${BIN_DIR}/${BIN_NAME}


osx:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -o ${BIN_DIR}/${BIN_NAME} -ldflags=" \
	-X 'github.com/yugabyte/yb-tools/yb-support-tool/cmd.Version=${VERSION}'"
	zip -j ${BIN_DIR}/${BIN_NAME}-MacOS-x86_64-${VERSION}-$$(date -I).zip ${BIN_DIR}/${BIN_NAME}
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -a -o ${BIN_DIR}/${BIN_NAME} -ldflags=" \
	-X 'github.com/yugabyte/yb-tools/yb-support-tool/cmd.Version=${VERSION}'"
	zip -j ${BIN_DIR}/${BIN_NAME}-MacOS-arm64-${VERSION}-$$(date -I).zip ${BIN_DIR}/${BIN_NAME}
