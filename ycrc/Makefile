TOP_BUILDDIR = ../.
include ../global.mk

.PHONEY: all ycrc

all: tools test ycrc

ycrc:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -ldflags=" -X 'github.com/yugabyte/yb-tools/ycrc/cmd.Version=$$(git rev-parse HEAD)'" -o ${TOP_BUILDDIR}/bin/ycrc main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 GO111MODULE=on go build -a -ldflags=" -X 'github.com/yugabyte/yb-tools/ycrc/cmd.Version=$$(git rev-parse HEAD)'" -o ${TOP_BUILDDIR}/bin/ycrc.exe main.go

