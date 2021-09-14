TOP_BUILDDIR = ../.
include ../global.mk

.PHONEY: all binary

all: test binary

binary:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o ../bin/protoc-gen-ybrpc main.go
