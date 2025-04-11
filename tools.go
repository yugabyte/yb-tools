//go:build tools
// +build tools

package tools

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	_ "github.com/go-swagger/go-swagger"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)

// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
