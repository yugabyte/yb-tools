//go:build tools
// +build tools

package tools

import (
	_ "github.com/go-swagger/go-swagger"
	_ "github.com/oapi-codegen/oapi-codegen/cmd/oapi-codegen"
)

// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
