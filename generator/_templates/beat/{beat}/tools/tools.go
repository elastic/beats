// +build tools

// This package contains the tool dependencies of the project.

package tools

import (
	_ "github.com/pierrre/gotestcover"
	_ "golang.org/x/tools/cmd/goimports"

	_ "github.com/mitchellh/gox"
	_ "github.com/reviewdog/reviewdog/cmd/reviewdog"
	_ "golang.org/x/lint/golint"
)
