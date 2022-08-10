TOP_BUILDDIR = ../.
include ../global.mk


.PHONEY: all yb-support-tool test

# go compile options
GOOS ?= linux
GOARCH ?= amd64
BIN_NAME ?= yb-support-tool
BIN_DIR := ${TOP_BUILDDIR}/bin


VERSION ?= $(shell git rev-parse HEAD)


all: test yb-support-tool

yb-support-tool: test ${BIN_DIR}/${BIN_NAME}
${BIN_DIR}/${BIN_NAME}:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -a -o $@ -ldflags=" \
	-X 'github.com/yugabyte/yb-tools/yb-support-tool/cmd.Version=${VERSION}'"


