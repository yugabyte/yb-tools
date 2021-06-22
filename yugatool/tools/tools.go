// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "github.com/spf13/cobra/cobra"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)

// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
