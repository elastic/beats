//go:build tools
// +build tools

// This package contains the tool dependencies of the project.

package tools

import (
	_ "github.com/blakesmith/ar"
	_ "github.com/cavaliergopher/rpm"
	_ "github.com/magefile/mage"
	_ "github.com/mitchellh/gox"
	_ "github.com/pierrre/gotestcover"
	_ "github.com/tsg/go-daemon"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/goimports"
	_ "gotest.tools/gotestsum/cmd"
)
