TOP_BUILDDIR = ../.
include ../global.mk

.PHONEY: all protobuf yugatool

all: test yugatool

yugatool:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -ldflags=" -X 'github.com/yugabyte/yb-tools/yugatool/cmd.Version=$$(git rev-parse HEAD)'" -o ${TOP_BUILDDIR}/bin/yugatool main.go

# Generate protobufs
protobuf: tools protoc-gen-ybrpc
	make -C protobuf/
