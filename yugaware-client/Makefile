TOP_BUILDDIR = ../.
include ../global.mk

.PHONEY: all yugaware-client

all: tools test yugaware-client

yugaware-client:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -ldflags=" -X 'github.com/yugabyte/yb-tools/yugaware-client/cmd.Version=$$(git rev-parse HEAD)'" -o ${TOP_BUILDDIR}/bin/yugaware-client main.go

swaggerclient: tools
	swagger generate client -f hack/platform.swagger.json -t pkg/client/swagger