module github.com/yugabyte/yb-tools/yugatool

go 1.16

require (
	github.com/golangci/golangci-lint v1.41.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.16.4
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/yugabyte/yb-tools/protoc-gen-ybrpc v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.25.0
)

replace github.com/yugabyte/yb-tools/protoc-gen-ybrpc => ../protoc-gen-ybrpc
